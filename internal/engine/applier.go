package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/chaz8081/positive-vibes/internal/registry"
	"github.com/chaz8081/positive-vibes/internal/target"
	"github.com/chaz8081/positive-vibes/pkg/schema"
)

// ApplyOpStatus represents the outcome of a single skill-target operation.
type ApplyOpStatus string

const (
	OpInstalled ApplyOpStatus = "installed"
	OpSkipped   ApplyOpStatus = "skipped"
	OpError     ApplyOpStatus = "error"
	OpNotFound  ApplyOpStatus = "not_found"
)

// ApplyOpKind distinguishes the type of item that was applied.
type ApplyOpKind string

const (
	KindSkill       ApplyOpKind = "skill"
	KindInstruction ApplyOpKind = "instruction"
	KindAgent       ApplyOpKind = "agent"
)

// ApplyOp records the result of installing one item to one target.
type ApplyOp struct {
	SkillName  string
	TargetName string
	Kind       ApplyOpKind
	Status     ApplyOpStatus
	Error      string
}

// ApplyResult summarizes installation results.
type ApplyResult struct {
	Installed int
	Skipped   int
	Errors    []string
	Ops       []ApplyOp
}

type Applier struct {
	Registries []registry.SkillSource
}

func NewApplier(regs []registry.SkillSource) *Applier {
	return &Applier{Registries: regs}
}

// Apply loads a manifest and installs each skill to each target.
func (a *Applier) Apply(manifestPath string, opts target.InstallOpts) (*ApplyResult, error) {
	m, err := manifest.LoadManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("load manifest: %w", err)
	}
	projectDir := filepath.Dir(manifestPath)
	return a.ApplyManifest(m, projectDir, opts)
}

// ApplyManifest installs resources from an already-loaded manifest.
// projectDir is used as the base for resolving relative resource paths.
func (a *Applier) ApplyManifest(m *manifest.Manifest, projectDir string, opts target.InstallOpts) (*ApplyResult, error) {
	if m == nil {
		return nil, fmt.Errorf("manifest is nil")
	}
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("validate manifest: %w", err)
	}

	targets, err := target.ResolveTargets(m.Targets)
	if err != nil {
		return nil, fmt.Errorf("resolve targets: %w", err)
	}

	res := &ApplyResult{}

	// iterate skills
	for _, s := range m.Skills {
		var sk *schema.Skill
		var srcDir string

		if s.Registry != "" {
			skillPath := s.Path
			if skillPath == "" {
				skillPath = s.Name
			}
			got, dir, err := a.fetchSkillFromRegistry(s.Registry, skillPath)
			if err == nil {
				sk = got
				srcDir = dir
			}
		} else if s.Path != "" {
			// local override path -- resolve relative to project directory
			resolvedPath := s.Path
			if !filepath.IsAbs(resolvedPath) {
				resolvedPath = filepath.Join(projectDir, resolvedPath)
			}
			p := filepath.Join(resolvedPath, "SKILL.md")
			data, err := os.ReadFile(p)
			if err == nil {
				sk, err = schema.ParseSkillFile(data)
				if err == nil {
					srcDir = resolvedPath
				}
			}
		}

		// if not local, search registries
		if sk == nil {
			for _, r := range a.Registries {
				got, dir, err := r.Fetch(s.Name)
				if err == nil {
					sk = got
					srcDir = dir
					break
				}
			}
		}

		if sk == nil {
			res.Errors = append(res.Errors, fmt.Sprintf("skill not found: %s", s.Name))
			res.Ops = append(res.Ops, ApplyOp{
				SkillName: s.Name,
				Kind:      KindSkill,
				Status:    OpNotFound,
				Error:     fmt.Sprintf("skill not found: %s", s.Name),
			})
			continue
		}

		// install to each target
		for _, t := range targets {
			if t.SkillExists(sk.Name, projectDir) {
				if !opts.Force {
					res.Skipped++
					res.Ops = append(res.Ops, ApplyOp{
						SkillName:  sk.Name,
						TargetName: t.Name(),
						Kind:       KindSkill,
						Status:     OpSkipped,
					})
					continue
				}
			}
			if err := t.Install(sk, srcDir, projectDir, opts); err != nil {
				errMsg := fmt.Sprintf("install %s -> %s: %v", sk.Name, t.Name(), err)
				res.Errors = append(res.Errors, errMsg)
				res.Ops = append(res.Ops, ApplyOp{
					SkillName:  sk.Name,
					TargetName: t.Name(),
					Kind:       KindSkill,
					Status:     OpError,
					Error:      errMsg,
				})
			} else {
				res.Installed++
				res.Ops = append(res.Ops, ApplyOp{
					SkillName:  sk.Name,
					TargetName: t.Name(),
					Kind:       KindSkill,
					Status:     OpInstalled,
				})
			}
		}
	}

	// iterate instructions
	for _, inst := range m.Instructions {
		sourcePath := ""
		tempFile := ""
		if inst.Registry != "" {
			skillName, relPath, parseErr := splitRegistryFilePath(inst.Path)
			if parseErr != nil {
				errMsg := fmt.Sprintf("instruction %s: invalid registry path %q: %v", inst.Name, inst.Path, parseErr)
				res.Errors = append(res.Errors, errMsg)
				res.Ops = append(res.Ops, ApplyOp{SkillName: inst.Name, Kind: KindInstruction, Status: OpError, Error: errMsg})
				continue
			}
			data, fetchErr := a.fetchFileFromRegistry(inst.Registry, skillName, relPath)
			if fetchErr != nil {
				errMsg := fmt.Sprintf("instruction %s: fetch from registry: %v", inst.Name, fetchErr)
				res.Errors = append(res.Errors, errMsg)
				res.Ops = append(res.Ops, ApplyOp{SkillName: inst.Name, Kind: KindInstruction, Status: OpError, Error: errMsg})
				continue
			}
			tmp, tmpErr := writeTempResourceFile(projectDir, "pv-inst-*", data)
			if tmpErr != nil {
				errMsg := fmt.Sprintf("instruction %s: create temp file: %v", inst.Name, tmpErr)
				res.Errors = append(res.Errors, errMsg)
				res.Ops = append(res.Ops, ApplyOp{SkillName: inst.Name, Kind: KindInstruction, Status: OpError, Error: errMsg})
				continue
			}
			tempFile = tmp
			sourcePath = tempFile
		} else {
			// Resolve source path relative to project directory
			sourcePath = inst.Path
			if sourcePath != "" && !filepath.IsAbs(sourcePath) {
				sourcePath = filepath.Join(projectDir, sourcePath)
			}
		}

		for _, t := range targets {
			// If ApplyTo is set, only install to matching target
			if inst.ApplyTo != "" && inst.ApplyTo != t.Name() {
				continue
			}

			if err := t.InstallInstruction(inst.Name, inst.Content, sourcePath, projectDir, opts); err != nil {
				errMsg := fmt.Sprintf("install instruction %s -> %s: %v", inst.Name, t.Name(), err)
				res.Errors = append(res.Errors, errMsg)
				res.Ops = append(res.Ops, ApplyOp{
					SkillName:  inst.Name,
					TargetName: t.Name(),
					Kind:       KindInstruction,
					Status:     OpError,
					Error:      errMsg,
				})
			} else {
				res.Installed++
				res.Ops = append(res.Ops, ApplyOp{
					SkillName:  inst.Name,
					TargetName: t.Name(),
					Kind:       KindInstruction,
					Status:     OpInstalled,
				})
			}
		}
		if tempFile != "" {
			_ = os.Remove(tempFile)
		}
	}

	// iterate agents
	for _, agent := range m.Agents {
		// Resolve source path: local path or registry fetch
		sourcePath := agent.Path
		if agent.Registry == "" && sourcePath != "" && !filepath.IsAbs(sourcePath) {
			sourcePath = filepath.Join(projectDir, sourcePath)
		}

		// If agent.Registry is set, fetch the file from the registry
		var tempFile string
		if agent.Registry != "" {
			skillName, relPath, parseErr := splitRegistryFilePath(agent.Path)
			if parseErr != nil {
				errMsg := fmt.Sprintf("agent %s: invalid registry path %q: %v", agent.Name, agent.Path, parseErr)
				res.Errors = append(res.Errors, errMsg)
				res.Ops = append(res.Ops, ApplyOp{
					SkillName: agent.Name,
					Kind:      KindAgent,
					Status:    OpError,
					Error:     errMsg,
				})
				continue
			}

			data, fetchErr := a.fetchFileFromRegistry(agent.Registry, skillName, relPath)
			if fetchErr != nil {
				errMsg := fmt.Sprintf("agent %s: fetch from registry: %v", agent.Name, fetchErr)
				res.Errors = append(res.Errors, errMsg)
				res.Ops = append(res.Ops, ApplyOp{
					SkillName: agent.Name,
					Kind:      KindAgent,
					Status:    OpError,
					Error:     errMsg,
				})
				continue
			}

			tmp, tmpErr := writeTempResourceFile(projectDir, "pv-agent-*", data)
			if tmpErr != nil {
				errMsg := fmt.Sprintf("agent %s: create temp file: %v", agent.Name, tmpErr)
				res.Errors = append(res.Errors, errMsg)
				res.Ops = append(res.Ops, ApplyOp{
					SkillName: agent.Name,
					Kind:      KindAgent,
					Status:    OpError,
					Error:     errMsg,
				})
				continue
			}
			tempFile = tmp
			sourcePath = tempFile
		}

		for _, t := range targets {
			if err := t.InstallAgent(agent.Name, sourcePath, projectDir, opts); err != nil {
				errMsg := fmt.Sprintf("install agent %s -> %s: %v", agent.Name, t.Name(), err)
				res.Errors = append(res.Errors, errMsg)
				res.Ops = append(res.Ops, ApplyOp{
					SkillName:  agent.Name,
					TargetName: t.Name(),
					Kind:       KindAgent,
					Status:     OpError,
					Error:      errMsg,
				})
			} else {
				res.Installed++
				res.Ops = append(res.Ops, ApplyOp{
					SkillName:  agent.Name,
					TargetName: t.Name(),
					Kind:       KindAgent,
					Status:     OpInstalled,
				})
			}
		}

		// Clean up temp file after installing to all targets
		if tempFile != "" {
			_ = os.Remove(tempFile)
		}
	}

	return res, nil
}

func splitRegistryFilePath(path string) (skillName, relPath string, err error) {
	slashIdx := strings.Index(path, "/")
	if slashIdx < 1 || slashIdx == len(path)-1 {
		return "", "", fmt.Errorf("expected format 'skillName/relative/file.md'")
	}
	skillName = path[:slashIdx]
	relPath = path[slashIdx+1:]
	return skillName, relPath, nil
}

func writeTempResourceFile(projectDir, pattern string, data []byte) (string, error) {
	tmp, err := os.CreateTemp(projectDir, pattern)
	if err != nil {
		return "", err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return "", err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return "", err
	}
	return tmp.Name(), nil
}

func (a *Applier) fetchSkillFromRegistry(regName, skillName string) (*schema.Skill, string, error) {
	for _, r := range a.Registries {
		if r.Name() != regName {
			continue
		}
		return r.Fetch(skillName)
	}
	return nil, "", fmt.Errorf("registry %q not found", regName)
}

// fetchFileFromRegistry looks up a registry by name, asserts it supports
// FileSource, and fetches the requested file.
func (a *Applier) fetchFileFromRegistry(regName, skillName, relPath string) ([]byte, error) {
	for _, r := range a.Registries {
		if r.Name() != regName {
			continue
		}
		fs, ok := r.(registry.FileSource)
		if !ok {
			return nil, fmt.Errorf("registry %q does not support file access", regName)
		}
		return fs.FetchFile(skillName, relPath)
	}
	return nil, fmt.Errorf("registry %q not found", regName)
}
