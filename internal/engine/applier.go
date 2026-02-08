package engine

import (
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

// ApplyOp records the result of installing one skill to one target.
type ApplyOp struct {
	SkillName  string
	TargetName string
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
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("validate manifest: %w", err)
	}

	targets, err := target.ResolveTargets(m.Targets)
	if err != nil {
		return nil, fmt.Errorf("resolve targets: %w", err)
	}

	res := &ApplyResult{}

	projectDir := filepath.Dir(manifestPath)

	// iterate skills
	for _, s := range m.Skills {
		var sk *schema.Skill
		var srcDir string
		// local override path -- resolve relative to project directory
		if s.Path != "" {
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
				Status:    OpNotFound,
				Error:     fmt.Sprintf("skill not found: %s", s.Name),
			})
			continue
		}

		// install to each target
		for _, t := range targets {
			if t.SkillExists(sk.Name, filepath.Dir(manifestPath)) {
				if !opts.Force {
					res.Skipped++
					res.Ops = append(res.Ops, ApplyOp{
						SkillName:  sk.Name,
						TargetName: t.Name(),
						Status:     OpSkipped,
					})
					continue
				}
			}
			if err := t.Install(sk, srcDir, filepath.Dir(manifestPath), opts); err != nil {
				errMsg := fmt.Sprintf("install %s -> %s: %v", sk.Name, t.Name(), err)
				res.Errors = append(res.Errors, errMsg)
				res.Ops = append(res.Ops, ApplyOp{
					SkillName:  sk.Name,
					TargetName: t.Name(),
					Status:     OpError,
					Error:      errMsg,
				})
			} else {
				res.Installed++
				res.Ops = append(res.Ops, ApplyOp{
					SkillName:  sk.Name,
					TargetName: t.Name(),
					Status:     OpInstalled,
				})
			}
		}
	}

	return res, nil
}
