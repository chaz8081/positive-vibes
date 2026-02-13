package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chaz8081/positive-vibes/internal/engine"
	"github.com/chaz8081/positive-vibes/internal/manifest"
	_ "github.com/chaz8081/positive-vibes/internal/registry"
	"github.com/spf13/cobra"
)

// renderBootstrapManifest builds the vibes.yaml content with inline comments
// per section, proper 2-space indent, and blank lines between sections.
func renderBootstrapManifest(m *manifest.Manifest) string {
	var b strings.Builder

	b.WriteString("# vibes.yaml - positive-vibes configuration\n")
	b.WriteString("# Run 'positive-vibes apply' to sync skills and instructions to all targets.\n")
	b.WriteString("# Global (~/.config/positive-vibes/vibes.yaml) and project configs are merged automatically; project values take priority.\n")
	b.WriteString("\n")

	// Registries
	b.WriteString("# Remote skill registries (git repos). Project entries override global by name.\n")
	b.WriteString("registries:\n")
	for _, r := range m.Registries {
		b.WriteString(fmt.Sprintf("  - name: %s\n", r.Name))
		b.WriteString(fmt.Sprintf("    url: %s\n", r.URL))
		b.WriteString(fmt.Sprintf("    ref: %s\n", r.Ref))
		if len(r.Paths) > 0 {
			b.WriteString("    paths:\n")
			for k, v := range r.Paths {
				b.WriteString(fmt.Sprintf("      %s: %s\n", k, v))
			}
		}
	}
	b.WriteString("\n")

	// Skills
	b.WriteString("# Skills to install. Use name (from registry) or path (local directory).\n")
	if len(m.Skills) > 0 {
		b.WriteString("skills:\n")
		for _, s := range m.Skills {
			if s.Path != "" {
				b.WriteString(fmt.Sprintf("  - name: %s\n", s.Name))
				b.WriteString(fmt.Sprintf("    path: %s\n", s.Path))
			} else {
				b.WriteString(fmt.Sprintf("  - name: %s\n", s.Name))
			}
		}
	} else {
		b.WriteString("# skills:\n")
		b.WriteString("#   - name: conventional-commits\n")
		b.WriteString("#   - name: my-custom-skill\n")
		b.WriteString("#     path: ./local-skills/my-custom-skill\n")
	}
	b.WriteString("\n")

	// Instructions
	b.WriteString("# Instructions appended to each target. Use name + content (inline) or path (file).\n")
	if len(m.Instructions) > 0 {
		b.WriteString("instructions:\n")
		for _, inst := range m.Instructions {
			b.WriteString(fmt.Sprintf("  - name: %s\n", inst.Name))
			if inst.Content != "" {
				b.WriteString(fmt.Sprintf("    content: %q\n", inst.Content))
			} else if inst.Path != "" {
				b.WriteString(fmt.Sprintf("    path: %s\n", inst.Path))
			}
			if inst.ApplyTo != "" {
				b.WriteString(fmt.Sprintf("    apply_to: %q\n", inst.ApplyTo))
			}
		}
	} else {
		b.WriteString("# instructions:\n")
		b.WriteString("#   - name: coding-style\n")
		b.WriteString("#     content: \"Always use TypeScript for frontend code\"\n")
		b.WriteString("#   - name: project-guide\n")
		b.WriteString("#     path: ./instructions/guide.md\n")
		b.WriteString("#     apply_to: opencode\n")
	}
	b.WriteString("\n")

	// Agents
	b.WriteString("# Agents to install. Use path (local file) or registry (remote).\n")
	if len(m.Agents) > 0 {
		b.WriteString("agents:\n")
		for _, a := range m.Agents {
			b.WriteString(fmt.Sprintf("  - name: %s\n", a.Name))
			if a.Path != "" {
				b.WriteString(fmt.Sprintf("    path: %s\n", a.Path))
			} else if a.Registry != "" {
				b.WriteString(fmt.Sprintf("    registry: %s\n", a.Registry))
			}
		}
	} else {
		b.WriteString("# agents:\n")
		b.WriteString("#   - name: code-reviewer\n")
		b.WriteString("#     path: ./agents/reviewer.md\n")
		b.WriteString("#   - name: registry-agent\n")
		b.WriteString("#     registry: awesome-copilot\n")
		b.WriteString("#     path: my-skill/agents/reviewer.md\n")
	}
	b.WriteString("\n")

	// Targets
	b.WriteString("# AI tools to sync into. Valid: vscode-copilot, opencode, cursor\n")
	if len(m.Targets) > 0 {
		b.WriteString("targets:\n")
		for _, t := range m.Targets {
			b.WriteString(fmt.Sprintf("  - %s\n", t))
		}
	} else {
		b.WriteString("# targets:\n")
		b.WriteString("#   - vscode-copilot\n")
		b.WriteString("#   - opencode\n")
		b.WriteString("#   - cursor\n")
	}

	return b.String()
}

// initTarget represents which manifest file(s) to create.
type initTarget int

const (
	initTargetLocal  initTarget = iota + 1 // project-level vibes.yaml only
	initTargetGlobal                       // global (~/.config/positive-vibes/vibes.yaml) only
	initTargetBoth                         // both local and global
)

// resolveInitAction determines which manifest(s) to create based on what
// already exists. When neither exists, it calls prompt to ask the user.
func resolveInitAction(globalExists, localExists bool, prompt func() (initTarget, error)) (initTarget, error) {
	switch {
	case globalExists && localExists:
		return 0, fmt.Errorf("both manifests already exist; nothing to do")
	case globalExists && !localExists:
		return initTargetLocal, nil
	case !globalExists && localExists:
		return initTargetGlobal, nil
	default: // neither exists
		return prompt()
	}
}

// writeInitManifest writes a bootstrap vibes.yaml to path.
// It creates parent directories as needed and refuses to overwrite an existing file.
// Uses buildGlobalDefaults to include sensible defaults (e.g. awesome-copilot registry).
func writeInitManifest(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	content := renderBootstrapManifest(buildGlobalDefaults())
	return os.WriteFile(path, []byte(content), 0o644)
}

// promptInitTarget reads an interactive choice from stdin: (L)ocal, (G)lobal, or (B)oth.
func promptInitTarget() (initTarget, error) {
	fmt.Println("No vibes.yaml found (local or global).")
	fmt.Println("Where would you like to create one?")
	fmt.Println("  [L] Local  (project-level vibes.yaml)")
	fmt.Println("  [G] Global (~/.config/positive-vibes/vibes.yaml)")
	fmt.Println("  [B] Both")
	fmt.Print("Choice [L]: ")

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return 0, fmt.Errorf("read input: %w", err)
	}
	line = strings.TrimSpace(strings.ToLower(line))
	switch line {
	case "", "l":
		return initTargetLocal, nil
	case "g":
		return initTargetGlobal, nil
	case "b":
		return initTargetBoth, nil
	default:
		return 0, fmt.Errorf("invalid choice: %q (expected L, G, or B)", line)
	}
}

// localManifestExists checks whether a project-level vibes.yaml or vibes.yml exists.
func localManifestExists(projectDir string) bool {
	for _, name := range manifest.ManifestFilenames {
		if _, err := os.Stat(filepath.Join(projectDir, name)); err == nil {
			return true
		}
	}
	return false
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a starter vibes.yaml",
	Run: func(cmd *cobra.Command, args []string) {
		project := ProjectDir()
		globalPath := defaultGlobalManifestPath()

		// Determine what already exists.
		globalExists := false
		if _, err := os.Stat(globalPath); err == nil {
			globalExists = true
		}
		localExists := localManifestExists(project)

		// Decide what to create.
		action, err := resolveInitAction(globalExists, localExists, promptInitTarget)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Write the requested manifest(s).
		switch action {
		case initTargetLocal, initTargetBoth:
			localPath := filepath.Join(project, "vibes.yaml")

			// Scan project for smart defaults (skills, targets).
			fmt.Println("Scanning your project...")
			res, scanErr := engine.ScanProject(project)
			if scanErr != nil {
				fmt.Printf("error scanning project: %v\n", scanErr)
				return
			}

			m := buildManifestFromScan(res)
			content := renderBootstrapManifest(m)
			if err := writeInitManifestContent(localPath, content); err != nil {
				fmt.Printf("error writing local manifest: %v\n", err)
				return
			}
			fmt.Printf("Created %s (%s project, %d skills)\n", localPath, res.Language, len(res.RecommendedSkills))

			if action == initTargetBoth {
				if err := writeInitManifest(globalPath); err != nil {
					fmt.Printf("error writing global manifest: %v\n", err)
					return
				}
				fmt.Printf("Created %s\n", globalPath)
			}

		case initTargetGlobal:
			if err := writeInitManifest(globalPath); err != nil {
				fmt.Printf("error writing global manifest: %v\n", err)
				return
			}
			fmt.Printf("Created %s\n", globalPath)
		}

		fmt.Println("\nRun 'positive-vibes config validate' to verify your setup.")
		fmt.Println("Run 'positive-vibes apply' to sync your tools!")
	},
}

// buildManifestFromScan creates a Manifest populated from project scan results.
func buildManifestFromScan(res *engine.ScanResult) *manifest.Manifest {
	m := &manifest.Manifest{}
	for _, s := range res.RecommendedSkills {
		m.Skills = append(m.Skills, manifest.SkillRef{Name: s})
	}
	for _, t := range res.SuggestedTargets {
		m.Targets = append(m.Targets, t)
	}
	m.Registries = []manifest.RegistryRef{{
		Name:  "awesome-copilot",
		URL:   "https://github.com/github/awesome-copilot",
		Ref:   "latest",
		Paths: map[string]string{"skills": "skills/"},
	}}
	return m
}

// buildGlobalDefaults creates a Manifest with sensible defaults for the global config.
// Includes the awesome-copilot registry but no skills or targets (those are project-specific).
func buildGlobalDefaults() *manifest.Manifest {
	return &manifest.Manifest{
		Registries: []manifest.RegistryRef{{
			Name:  "awesome-copilot",
			URL:   "https://github.com/github/awesome-copilot",
			Ref:   "latest",
			Paths: map[string]string{"skills": "skills/"},
		}},
	}
}

// writeInitManifestContent writes pre-rendered content to path, creating
// parent directories as needed. Refuses to overwrite an existing file.
func writeInitManifestContent(path, content string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func join(ss []string, sep string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += sep
			out += " "
		}
		out += s
	}
	return out
}

func init() {
	rootCmd.AddCommand(initCmd)
}
