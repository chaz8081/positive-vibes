package engine

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
	KindPrompt      ApplyOpKind = "prompt"
)

// ApplyOp records the result of installing one item to one target.
type ApplyOp struct {
	ResourceName string
	TargetName   string
	Kind         ApplyOpKind
	Status       ApplyOpStatus
	Error        string
}

// ApplyResult summarizes installation results.
type ApplyResult struct {
	Installed int
	Skipped   int
	Errors    []string
	Ops       []ApplyOp
	DryRunOps []DryRunOp
}

type resourceContext struct {
	projectDir string
	opts       target.InstallOpts
	res        *ApplyResult
	fetcher    *Applier
}

type resourceSpec struct {
	kind      ApplyOpKind
	name      string
	registry  string
	path      string
	content   string
	suffix    string
	applyTo   string
	kindLabel string
}

type applyCallbacks struct {
	resolvePath     func(projectDir, sourcePath string) string
	fetchRegistry   func(spec resourceSpec) ([]byte, error)
	preview         func(spec resourceSpec, content, sourcePath string, t target.Target) (DryRunOp, error)
	install         func(spec resourceSpec, sourcePath string, t target.Target) error
	dryRunSkip      func(spec resourceSpec, t target.Target) (DryRunOp, bool)
	postTargetError func(spec resourceSpec, t target.Target, err error) *ApplyOp
}

func applyResource(ctx resourceContext, targets []target.Target, spec resourceSpec, callbacks applyCallbacks) {
	sourcePath := ""
	tempFile := ""
	previewContent := ""

	if spec.registry != "" {
		data, fetchErr := callbacks.fetchRegistry(spec)
		if fetchErr != nil {
			errMsg := fmt.Sprintf("%s %s: fetch from registry: %v", spec.kindLabel, spec.name, fetchErr)
			ctx.res.Errors = append(ctx.res.Errors, errMsg)
			ctx.res.Ops = append(ctx.res.Ops, ApplyOp{ResourceName: spec.name, Kind: spec.kind, Status: OpError, Error: errMsg})
			return
		}
		if ctx.opts.DryRun {
			previewContent = string(data)
		} else {
			tmp, tmpErr := writeTempResourceFile(ctx.projectDir, "pv-"+spec.kindLabel+"-*", data)
			if tmpErr != nil {
				errMsg := fmt.Sprintf("%s %s: create temp file: %v", spec.kindLabel, spec.name, tmpErr)
				ctx.res.Errors = append(ctx.res.Errors, errMsg)
				ctx.res.Ops = append(ctx.res.Ops, ApplyOp{ResourceName: spec.name, Kind: spec.kind, Status: OpError, Error: errMsg})
				return
			}
			tempFile = tmp
			sourcePath = tempFile
		}
	} else {
		sourcePath = callbacks.resolvePath(ctx.projectDir, spec.path)
	}

	for _, t := range targets {
		if spec.applyTo != "" && spec.applyTo != t.Name() {
			continue
		}

		if ctx.opts.DryRun {
			if callbacks.dryRunSkip != nil {
				if op, ok := callbacks.dryRunSkip(spec, t); ok {
					ctx.res.DryRunOps = append(ctx.res.DryRunOps, op)
					continue
				}
			}

			content := spec.content
			if content == "" {
				content = previewContent
			}
			op, previewErr := callbacks.preview(spec, content, sourcePath, t)
			if previewErr != nil {
				errMsg := fmt.Sprintf("dry-run preview %s %s -> %s: %v", spec.kindLabel, spec.name, t.Name(), previewErr)
				ctx.res.Errors = append(ctx.res.Errors, errMsg)
			} else {
				ctx.res.DryRunOps = append(ctx.res.DryRunOps, op)
			}
			continue
		}

		if err := callbacks.install(spec, sourcePath, t); err != nil {
			if callbacks.postTargetError != nil {
				if op := callbacks.postTargetError(spec, t, err); op != nil {
					ctx.res.Ops = append(ctx.res.Ops, *op)
					ctx.res.Skipped++
					continue
				}
			}
			errMsg := fmt.Sprintf("install %s %s -> %s: %v", spec.kindLabel, spec.name, t.Name(), err)
			ctx.res.Errors = append(ctx.res.Errors, errMsg)
			ctx.res.Ops = append(ctx.res.Ops, ApplyOp{ResourceName: spec.name, TargetName: t.Name(), Kind: spec.kind, Status: OpError, Error: errMsg})
		} else {
			ctx.res.Installed++
			ctx.res.Ops = append(ctx.res.Ops, ApplyOp{ResourceName: spec.name, TargetName: t.Name(), Kind: spec.kind, Status: OpInstalled})
		}
	}

	if tempFile != "" {
		_ = os.Remove(tempFile)
	}
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
		var localPathErr error
		var cleanupFn func()

		if s.Registry != "" {
			skillPath := s.Path
			if skillPath == "" {
				skillPath = s.Name
			}
			got, dir, cleanup, err := a.fetchSkillFromRegistryWithCleanup(s.Registry, skillPath)
			if err == nil {
				sk = got
				srcDir = dir
				cleanupFn = cleanup
			}
		} else if s.Path != "" {
			// local override path -- resolve relative to project directory
			resolvedPath := s.Path
			if !filepath.IsAbs(resolvedPath) {
				resolvedPath = filepath.Join(projectDir, resolvedPath)
			}
			p := filepath.Join(resolvedPath, "SKILL.md")
			data, err := os.ReadFile(p)
			if err != nil {
				localPathErr = fmt.Errorf("skill %s: read local skill file: %w", s.Name, err)
			} else {
				sk, err = schema.ParseSkillFile(data)
				if err != nil {
					localPathErr = fmt.Errorf("skill %s: parse local skill file: %w", s.Name, err)
				} else {
					srcDir = resolvedPath
				}
			}
		}

		if localPathErr != nil {
			errMsg := localPathErr.Error()
			res.Errors = append(res.Errors, errMsg)
			res.Ops = append(res.Ops, ApplyOp{
				ResourceName: s.Name,
				Kind:         KindSkill,
				Status:       OpError,
				Error:        errMsg,
			})
			continue
		}

		// if not local, search registries
		if sk == nil {
			for _, r := range a.Registries {
				got, dir, cleanup, err := fetchWithCleanup(r, s.Name)
				if err == nil {
					sk = got
					srcDir = dir
					cleanupFn = cleanup
					break
				}
			}
		}

		if sk == nil {
			res.Errors = append(res.Errors, fmt.Sprintf("skill not found: %s", s.Name))
			res.Ops = append(res.Ops, ApplyOp{
				ResourceName: s.Name,
				Kind:         KindSkill,
				Status:       OpNotFound,
				Error:        fmt.Sprintf("skill not found: %s", s.Name),
			})
			continue
		}

		// install to each target
		for _, t := range targets {
			if opts.DryRun {
				ops, previewErr := previewSkillInstall(sk, srcDir, projectDir, t)
				if previewErr != nil {
					errMsg := fmt.Sprintf("dry-run preview %s -> %s: %v", sk.Name, t.Name(), previewErr)
					res.Errors = append(res.Errors, errMsg)
				} else {
					res.DryRunOps = append(res.DryRunOps, ops...)
				}
				continue
			}
			if t.SkillExists(sk.Name, projectDir) {
				if !opts.Force {
					res.Skipped++
					res.Ops = append(res.Ops, ApplyOp{
						ResourceName: sk.Name,
						TargetName:   t.Name(),
						Kind:         KindSkill,
						Status:       OpSkipped,
					})
					continue
				}
			}
			if err := t.Install(sk, srcDir, projectDir, opts); err != nil {
				errMsg := fmt.Sprintf("install %s -> %s: %v", sk.Name, t.Name(), err)
				res.Errors = append(res.Errors, errMsg)
				res.Ops = append(res.Ops, ApplyOp{
					ResourceName: sk.Name,
					TargetName:   t.Name(),
					Kind:         KindSkill,
					Status:       OpError,
					Error:        errMsg,
				})
			} else {
				res.Installed++
				res.Ops = append(res.Ops, ApplyOp{
					ResourceName: sk.Name,
					TargetName:   t.Name(),
					Kind:         KindSkill,
					Status:       OpInstalled,
				})
			}
		}

		// Clean up temp source directory (e.g. from embedded registry)
		if cleanupFn != nil {
			cleanupFn()
		}
	}

	// iterate instructions
	ctx := resourceContext{projectDir: projectDir, opts: opts, res: res, fetcher: a}
	resolveRelative := func(projectDir, sourcePath string) string {
		if sourcePath != "" && !filepath.IsAbs(sourcePath) {
			return filepath.Join(projectDir, sourcePath)
		}
		return sourcePath
	}

	for _, inst := range m.Instructions {
		spec := resourceSpec{
			kind:      KindInstruction,
			name:      inst.Name,
			registry:  inst.Registry,
			path:      inst.Path,
			content:   inst.Content,
			suffix:    ".md",
			applyTo:   inst.ApplyTo,
			kindLabel: "instruction",
		}
		applyResource(ctx, targets, spec, applyCallbacks{
			resolvePath: resolveRelative,
			fetchRegistry: func(spec resourceSpec) ([]byte, error) {
				return a.fetchResourceFileFromRegistry(spec.registry, "instructions", spec.path)
			},
			preview: func(spec resourceSpec, content, sourcePath string, t target.Target) (DryRunOp, error) {
				return previewSingleFileInstall(spec.name, content, sourcePath, projectDir, t.InstructionDir(), spec.suffix, t, KindInstruction)
			},
			install: func(spec resourceSpec, sourcePath string, t target.Target) error {
				return t.InstallInstruction(spec.name, spec.content, sourcePath, projectDir, opts)
			},
		})
	}

	for _, agent := range m.Agents {
		spec := resourceSpec{
			kind:      KindAgent,
			name:      agent.Name,
			registry:  agent.Registry,
			path:      agent.Path,
			suffix:    ".md",
			kindLabel: "agent",
		}
		applyResource(ctx, targets, spec, applyCallbacks{
			resolvePath: resolveRelative,
			fetchRegistry: func(spec resourceSpec) ([]byte, error) {
				return a.fetchResourceFileFromRegistry(spec.registry, "agents", spec.path)
			},
			preview: func(spec resourceSpec, content, sourcePath string, t target.Target) (DryRunOp, error) {
				return previewSingleFileInstall(spec.name, content, sourcePath, projectDir, t.AgentDir(), spec.suffix, t, KindAgent)
			},
			install: func(spec resourceSpec, sourcePath string, t target.Target) error {
				return t.InstallAgent(spec.name, sourcePath, projectDir, opts)
			},
		})
	}

	for _, prompt := range m.Prompts {
		spec := resourceSpec{
			kind:      KindPrompt,
			name:      prompt.Name,
			registry:  prompt.Registry,
			path:      prompt.Path,
			suffix:    "",
			kindLabel: "prompt",
		}
		applyResource(ctx, targets, spec, applyCallbacks{
			resolvePath: resolveRelative,
			fetchRegistry: func(spec resourceSpec) ([]byte, error) {
				return a.fetchResourceFileFromRegistry(spec.registry, "prompts", spec.path)
			},
			preview: func(spec resourceSpec, content, sourcePath string, t target.Target) (DryRunOp, error) {
				suffix := t.PromptSuffix()
				return previewSingleFileInstall(spec.name, content, sourcePath, projectDir, t.PromptDir(), suffix, t, KindPrompt)
			},
			install: func(spec resourceSpec, sourcePath string, t target.Target) error {
				return t.InstallPrompt(spec.name, sourcePath, projectDir, opts)
			},
			dryRunSkip: func(spec resourceSpec, t target.Target) (DryRunOp, bool) {
				if !t.SupportsPrompts() {
					promptRelPath := filepath.Join(t.PromptDir(), spec.name+t.PromptSuffix())
					return DryRunOp{
						Action:   DryRunSkip,
						RelPath:  promptRelPath,
						Target:   t.Name(),
						Kind:     KindPrompt,
						Resource: spec.name,
						Reason:   "unsupported",
					}, true
				}
				return DryRunOp{}, false
			},
			postTargetError: func(spec resourceSpec, t target.Target, err error) *ApplyOp {
				if errors.Is(err, target.ErrPromptInstallUnsupported) {
					return &ApplyOp{
						ResourceName: spec.name,
						TargetName:   t.Name(),
						Kind:         KindPrompt,
						Status:       OpSkipped,
						Error:        err.Error(),
					}
				}
				return nil
			},
		})
	}

	return res, nil
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

// fetchSkillFromRegistryWithCleanup is like fetchSkillFromRegistry but returns
// a cleanup function when the underlying registry supports CleanableFetcher.
func (a *Applier) fetchSkillFromRegistryWithCleanup(regName, skillName string) (*schema.Skill, string, func(), error) {
	for _, r := range a.Registries {
		if r.Name() != regName {
			continue
		}
		return fetchWithCleanup(r, skillName)
	}
	return nil, "", nil, fmt.Errorf("registry %q not found", regName)
}

// fetchWithCleanup fetches a skill from a source, returning a cleanup function
// if the source implements CleanableFetcher. For sources that don't, cleanup is nil.
func fetchWithCleanup(src registry.SkillSource, name string) (*schema.Skill, string, func(), error) {
	if cf, ok := src.(registry.CleanableFetcher); ok {
		return cf.FetchWithCleanup(name)
	}
	sk, dir, err := src.Fetch(name)
	return sk, dir, nil, err
}

// fetchResourceFileFromRegistry looks up a registry by name, asserts it
// supports resource file access, and fetches the requested file.
func (a *Applier) fetchResourceFileFromRegistry(regName, kind, relPath string) ([]byte, error) {
	for _, r := range a.Registries {
		if r.Name() != regName {
			continue
		}
		fs, ok := r.(registry.ResourceSource)
		if !ok {
			return nil, fmt.Errorf("registry %q does not support file access", regName)
		}
		return fs.FetchResourceFile(kind, relPath)
	}
	return nil, fmt.Errorf("registry %q not found", regName)
}
