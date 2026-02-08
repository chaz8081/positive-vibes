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

// ApplyResult summarizes installation results.
type ApplyResult struct {
	Installed int
	Skipped   int
	Errors    []string
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

	// iterate skills
	for _, s := range m.Skills {
		var sk *schema.Skill
		var srcDir string
		// local override path
		if s.Path != "" {
			// parse SKILL.md from path
			p := filepath.Join(s.Path, "SKILL.md")
			data, err := os.ReadFile(p)
			if err == nil {
				sk, err = schema.ParseSkillFile(data)
				if err == nil {
					srcDir = s.Path
				}
			}
			_ = p
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
			continue
		}

		// install to each target
		for _, t := range targets {
			if t.SkillExists(sk.Name, filepath.Dir(manifestPath)) {
				if !opts.Force {
					res.Skipped++
					continue
				}
			}
			if err := t.Install(sk, srcDir, filepath.Dir(manifestPath), opts); err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("install %s -> %s: %v", sk.Name, t.Name(), err))
			} else {
				res.Installed++
			}
		}
	}

	return res, nil
}
