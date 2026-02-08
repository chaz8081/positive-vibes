package target

import (
	"os"
	"path/filepath"

	"github.com/chaz8081/positive-vibes/pkg/schema"
)

type CopilotTarget struct{}

func (CopilotTarget) Name() string     { return "vscode-copilot" }
func (CopilotTarget) SkillDir() string { return filepath.Join(".github", "skills") }

func (t CopilotTarget) Install(skill *schema.Skill, sourceDir string, projectRoot string, opts InstallOpts) error {
	return installGeneric(skill, sourceDir, projectRoot, t.SkillDir(), opts)
}

func (CopilotTarget) SkillExists(skillName string, projectRoot string) bool {
	dest := skillPath(projectRoot, filepath.Join(".github", "skills"), skillName)
	f := filepath.Join(dest, "SKILL.md")
	if _, err := os.Stat(f); err == nil {
		return true
	}
	return false
}
