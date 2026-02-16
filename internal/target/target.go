package target

import (
	"errors"
	"fmt"
	"path/filepath"

	"os"

	"github.com/chaz8081/positive-vibes/internal/fsutil"
	"github.com/chaz8081/positive-vibes/pkg/schema"
)

// InstallOpts controls how skills are installed.
type InstallOpts struct {
	Force bool // overwrite existing skills
	Link  bool // create symlinks instead of copies
}

// ErrPromptInstallUnsupported indicates the target does not support prompts.
var ErrPromptInstallUnsupported = errors.New("prompt install unsupported for target")

// Target knows how to install a skill for a specific AI tool.
type Target interface {
	// Name returns the target identifier (e.g., "vscode-copilot").
	Name() string
	// SkillDir returns the base directory for skills relative to project root.
	SkillDir() string
	// InstructionDir returns the base directory for instructions relative to project root.
	InstructionDir() string
	// AgentDir returns the base directory for agents relative to project root.
	AgentDir() string
	// PromptDir returns the base directory for prompts relative to project root.
	PromptDir() string
	// Install writes the skill to the tool's expected location.
	Install(skill *schema.Skill, sourceDir string, projectRoot string, opts InstallOpts) error
	// SkillExists checks if a skill is already installed for this target.
	SkillExists(skillName string, projectRoot string) bool
	// InstallInstruction writes an instruction file to the target's instruction directory.
	// Either content (inline text) or sourcePath (file to copy) must be provided.
	InstallInstruction(name string, content string, sourcePath string, projectRoot string, opts InstallOpts) error
	// InstallAgent writes an agent file to the target's agent directory.
	// sourcePath is the path to the agent file to copy.
	InstallAgent(name string, sourcePath string, projectRoot string, opts InstallOpts) error
	// InstallPrompt writes a prompt file to the target's prompt/command directory.
	InstallPrompt(name string, sourcePath string, projectRoot string, opts InstallOpts) error
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
func skillPath(projectRoot, skillDir, skillName string) (string, error) {
	return fsutil.ResolveWithinRoot(filepath.Join(projectRoot, skillDir), skillName)
}

// installGeneric contains shared installation logic for targets.
func installGeneric(skill *schema.Skill, sourceDir, projectRoot, skillDir string, opts InstallOpts) error {
	dest, err := skillPath(projectRoot, skillDir, skill.Name)
	if err != nil {
		return fmt.Errorf("invalid skill name %q: %w", skill.Name, err)
	}

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
		if err := filepath.WalkDir(sourceDir, func(path string, d os.DirEntry, err error) error {
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
		}); err != nil {
			return err
		}
	}

	return nil
}

// installInstructionGeneric writes an instruction file as <name>.md into the
// target's instruction directory. Either content or sourcePath must be provided.
func installInstructionGeneric(name, content, sourcePath, projectRoot, instDir string, opts InstallOpts) error {
	dest, err := fsutil.ResolveWithinRoot(filepath.Join(projectRoot, instDir), name+".md")
	if err != nil {
		return fmt.Errorf("invalid instruction name %q: %w", name, err)
	}

	if _, err := os.Stat(dest); err == nil {
		if !opts.Force {
			return fmt.Errorf("instruction '%s' already exists for %s (use --force to overwrite)", name, instDir)
		}
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}

	var data []byte
	if content != "" {
		data = []byte(content)
	} else if sourcePath != "" {
		var err error
		data, err = os.ReadFile(sourcePath)
		if err != nil {
			return fmt.Errorf("read instruction source: %w", err)
		}
	} else {
		return fmt.Errorf("instruction '%s': no content or source path provided", name)
	}

	return os.WriteFile(dest, data, 0o644)
}

// installAgentGeneric writes an agent file as <name>.md into the target's agent
// directory by copying the content from sourcePath.
func installAgentGeneric(name, sourcePath, projectRoot, agentDir string, opts InstallOpts) error {
	dest, err := fsutil.ResolveWithinRoot(filepath.Join(projectRoot, agentDir), name+".md")
	if err != nil {
		return fmt.Errorf("invalid agent name %q: %w", name, err)
	}

	if _, err := os.Stat(dest); err == nil {
		if !opts.Force {
			return fmt.Errorf("agent '%s' already exists for %s (use --force to overwrite)", name, agentDir)
		}
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}

	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read agent source: %w", err)
	}

	return os.WriteFile(dest, data, 0o644)
}

// installPromptGeneric writes a prompt file to the provided directory with
// target-specific suffix.
func installPromptGeneric(name, sourcePath, projectRoot, promptDir, suffix string, opts InstallOpts) error {
	dest, err := fsutil.ResolveWithinRoot(filepath.Join(projectRoot, promptDir), name+suffix)
	if err != nil {
		return fmt.Errorf("invalid prompt name %q: %w", name, err)
	}

	if _, err := os.Stat(dest); err == nil {
		if !opts.Force {
			return fmt.Errorf("prompt '%s' already exists for %s (use --force to overwrite)", name, promptDir)
		}
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}

	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read prompt source: %w", err)
	}

	return os.WriteFile(dest, data, 0o644)
}
