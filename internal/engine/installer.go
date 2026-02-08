package engine

import (
	"fmt"

	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/chaz8081/positive-vibes/internal/registry"
)

type Installer struct {
	Registries []registry.SkillSource
}

func NewInstaller(regs []registry.SkillSource) *Installer {
	return &Installer{Registries: regs}
}

// Install adds a skill ref to the manifest if it exists in a registry.
func (i *Installer) Install(skillName string, manifestPath string) error {
	m, err := manifest.LoadManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	// check exists
	for _, s := range m.Skills {
		if s.Name == skillName {
			return fmt.Errorf("skill already in manifest")
		}
	}

	// verify in registry
	found := false
	for _, r := range i.Registries {
		if _, _, err := r.Fetch(skillName); err == nil {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("skill not found: %s", skillName)
	}

	m.Skills = append(m.Skills, manifest.SkillRef{Name: skillName})
	if err := manifest.SaveManifest(m, manifestPath); err != nil {
		return fmt.Errorf("save manifest: %w", err)
	}
	return nil
}
