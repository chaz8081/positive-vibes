package target

import (
	"os"
	"path/filepath"

	"github.com/chaz8081/positive-vibes/pkg/schema"
)

type OpenCodeTarget struct{}

func (OpenCodeTarget) Name() string     { return "opencode" }
func (OpenCodeTarget) SkillDir() string { return filepath.Join(".opencode", "skills") }

func (t OpenCodeTarget) Install(skill *schema.Skill, sourceDir string, projectRoot string, opts InstallOpts) error {
	return installGeneric(skill, sourceDir, projectRoot, t.SkillDir(), opts)
}

func (OpenCodeTarget) SkillExists(skillName string, projectRoot string) bool {
	dest := skillPath(projectRoot, filepath.Join(".opencode", "skills"), skillName)
	f := filepath.Join(dest, "SKILL.md")
	if _, err := os.Stat(f); err == nil {
		return true
	}
	return false
}
