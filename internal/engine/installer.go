package engine

import (
	"errors"
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

// unwrapPathError checks whether the error chain contains a *os.PathError
// indicating the file does not exist.
func unwrapPathError(err error) error {
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		return pathErr.Err
	}
	return err
}

// Install adds a skill ref to the manifest. It checks local skills first
// (project/skills/<name>/SKILL.md), then falls back to registries.
// If the manifest file does not exist, a new empty manifest is created.
func (i *Installer) Install(skillName string, manifestPath string) error {
	m, err := manifest.LoadManifest(manifestPath)
	if err != nil {
		if !os.IsNotExist(unwrapPathError(err)) {
			return fmt.Errorf("load manifest: %w", err)
		}
		// Manifest doesn't exist yet — start fresh.
		m = &manifest.Manifest{}
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

// Remove removes a skill ref from the manifest by name.
// Returns an error if the manifest cannot be loaded or the skill is not found.
func (i *Installer) Remove(skillName string, manifestPath string) error {
	m, err := manifest.LoadManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	found := -1
	for idx, s := range m.Skills {
		if s.Name == skillName {
			found = idx
			break
		}
	}
	if found < 0 {
		return fmt.Errorf("skill not found in manifest: %s", skillName)
	}

	m.Skills = append(m.Skills[:found], m.Skills[found+1:]...)
	if err := manifest.SaveManifest(m, manifestPath); err != nil {
		return fmt.Errorf("save manifest: %w", err)
	}
	return nil
}
