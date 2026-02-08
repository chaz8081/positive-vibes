package target

import (
	"os"
	"path/filepath"

	"github.com/chaz8081/positive-vibes/pkg/schema"
)

type CursorTarget struct{}

func (CursorTarget) Name() string           { return "cursor" }
func (CursorTarget) SkillDir() string       { return filepath.Join(".cursor", "skills") }
func (CursorTarget) InstructionDir() string { return filepath.Join(".cursor", "instructions") }
func (CursorTarget) AgentDir() string       { return filepath.Join(".cursor", "agents") }

func (t CursorTarget) Install(skill *schema.Skill, sourceDir string, projectRoot string, opts InstallOpts) error {
	return installGeneric(skill, sourceDir, projectRoot, t.SkillDir(), opts)
}

func (CursorTarget) SkillExists(skillName string, projectRoot string) bool {
	dest := skillPath(projectRoot, filepath.Join(".cursor", "skills"), skillName)
	f := filepath.Join(dest, "SKILL.md")
	if _, err := os.Stat(f); err == nil {
		return true
	}
	return false
}

func (t CursorTarget) InstallInstruction(name, content, sourcePath, projectRoot string, opts InstallOpts) error {
	return installInstructionGeneric(name, content, sourcePath, projectRoot, t.InstructionDir(), opts)
}

func (t CursorTarget) InstallAgent(name, sourcePath, projectRoot string, opts InstallOpts) error {
	return installAgentGeneric(name, sourcePath, projectRoot, t.AgentDir(), opts)
}
