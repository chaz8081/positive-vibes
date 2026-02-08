package target

import (
	"fmt"
	"path/filepath"

	"os"

	"github.com/chaz8081/positive-vibes/pkg/schema"
)

// InstallOpts controls how skills are installed.
type InstallOpts struct {
	Force bool // overwrite existing skills
	Link  bool // create symlinks instead of copies
}

// Target knows how to install a skill for a specific AI tool.
type Target interface {
	// Name returns the target identifier (e.g., "vscode-copilot").
	Name() string
	// SkillDir returns the base directory for skills relative to project root.
	SkillDir() string
	// Install writes the skill to the tool's expected location.
	Install(skill *schema.Skill, sourceDir string, projectRoot string, opts InstallOpts) error
	// SkillExists checks if a skill is already installed for this target.
	SkillExists(skillName string, projectRoot string) bool
}

// ResolveTargets maps target name strings to Target implementations.
func ResolveTargets(names []string) ([]Target, error) {
	var out []Target
	for _, n := range names {
		switch n {
		case "vscode-copilot":
			out = append(out, CopilotTarget{})
		case "opencode":
			out = append(out, OpenCodeTarget{})
		case "cursor":
			out = append(out, CursorTarget{})
		default:
			return nil, fmt.Errorf("unknown target: %s", n)
		}
	}
	return out, nil
}

// helper to compute skill path
func skillPath(projectRoot, skillDir, skillName string) string {
	return filepath.Join(projectRoot, skillDir, skillName)
}

// installGeneric contains shared installation logic for targets.
func installGeneric(skill *schema.Skill, sourceDir, projectRoot, skillDir string, opts InstallOpts) error {
	dest := skillPath(projectRoot, skillDir, skill.Name)

	// check exists
	if _, err := os.Stat(dest); err == nil {
		if !opts.Force {
			return fmt.Errorf("skill '%s' already exists for %s (use --force to overwrite)", skill.Name, skillDir)
		}
		// remove
		if err := os.RemoveAll(dest); err != nil {
			return err
		}
	}

	parent := filepath.Dir(dest)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}

	if opts.Link {
		// create symlink
		if err := os.Symlink(sourceDir, dest); err != nil {
			return err
		}
		return nil
	}

	// copy mode: create dest and write SKILL.md
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	content, err := schema.RenderSkillFile(skill)
	if err != nil {
		return err
	}
	f := filepath.Join(dest, "SKILL.md")
	if err := os.WriteFile(f, []byte(content), 0o644); err != nil {
		return err
	}

	// copy additional files from sourceDir (simple recursive copy)
	// if sourceDir doesn't exist or is same as dest, skip
	if sourceDir != "" {
		// walk sourceDir and copy files except SKILL.md
		filepath.WalkDir(sourceDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(sourceDir, path)
			if err != nil {
				return err
			}
			if rel == "" || rel == "SKILL.md" {
				return nil
			}
			targetPath := filepath.Join(dest, rel)
			if d.IsDir() {
				return os.MkdirAll(targetPath, 0o755)
			}
			// file: copy
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(targetPath, data, 0o644)
		})
	}

	return nil
}
