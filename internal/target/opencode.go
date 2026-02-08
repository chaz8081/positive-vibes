package target

import (
	"os"
	"path/filepath"

	"github.com/chaz8081/positive-vibes/pkg/schema"
)

type OpenCodeTarget struct{}

func (OpenCodeTarget) Name() string           { return "opencode" }
func (OpenCodeTarget) SkillDir() string       { return filepath.Join(".opencode", "skills") }
func (OpenCodeTarget) InstructionDir() string { return filepath.Join(".opencode", "instructions") }
func (OpenCodeTarget) AgentDir() string       { return filepath.Join(".opencode", "agents") }

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

func (t OpenCodeTarget) InstallInstruction(name, content, sourcePath, projectRoot string, opts InstallOpts) error {
	return installInstructionGeneric(name, content, sourcePath, projectRoot, t.InstructionDir(), opts)
}

func (t OpenCodeTarget) InstallAgent(name, sourcePath, projectRoot string, opts InstallOpts) error {
	return installAgentGeneric(name, sourcePath, projectRoot, t.AgentDir(), opts)
}
