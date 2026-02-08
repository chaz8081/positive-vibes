package engine

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/chaz8081/positive-vibes/internal/registry"
)

type Installer struct {
	Registries []registry.SkillSource
}

func NewInstaller(regs []registry.SkillSource) *Installer {
	return &Installer{Registries: regs}
}

// Install adds a skill ref to the manifest. It checks local skills first
// (project/skills/<name>/SKILL.md), then falls back to registries.
func (i *Installer) Install(skillName string, manifestPath string) error {
	m, err := manifest.LoadManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	// check if already in manifest
	for _, s := range m.Skills {
		if s.Name == skillName {
			return fmt.Errorf("skill already in manifest")
		}
	}

	projectDir := filepath.Dir(manifestPath)

	// check local skills directory first
	localPath := filepath.Join(projectDir, "skills", skillName, "SKILL.md")
	if _, err := os.Stat(localPath); err == nil {
		// local skill found -- add with relative path
		ref := manifest.SkillRef{
			Name: skillName,
			Path: "./skills/" + skillName,
		}
		m.Skills = append(m.Skills, ref)
		if err := manifest.SaveManifest(m, manifestPath); err != nil {
			return fmt.Errorf("save manifest: %w", err)
		}
		return nil
	}

	// fall back to registries
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
