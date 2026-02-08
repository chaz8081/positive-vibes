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

// renderBootstrapManifest builds the vibes.yml content with inline comments
// per section, proper 2-space indent, and blank lines between sections.
func renderBootstrapManifest(m *manifest.Manifest) string {
	var b strings.Builder

	b.WriteString("# vibes.yml - positive-vibes project configuration\n")
	b.WriteString("# Run 'vibes apply' to sync skills and instructions to all targets.\n")
	b.WriteString("# Global config (~/.config/positive-vibes/vibes.yml) is merged with this file.\n")
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
	b.WriteString("skills:\n")
	for _, s := range m.Skills {
		if s.Path != "" {
			b.WriteString(fmt.Sprintf("  - name: %s\n", s.Name))
			b.WriteString(fmt.Sprintf("    path: %s\n", s.Path))
		} else {
			b.WriteString(fmt.Sprintf("  - name: %s\n", s.Name))
		}
	}
	b.WriteString("\n")

	// Instructions (commented-out example since init doesn't generate any)
	if len(m.Instructions) > 0 {
		b.WriteString("# Free-form instructions appended to each target.\n")
		b.WriteString("instructions:\n")
		for _, inst := range m.Instructions {
			b.WriteString(fmt.Sprintf("  - %q\n", inst))
		}
	} else {
		b.WriteString("# Free-form instructions appended to each target.\n")
		b.WriteString("# instructions:\n")
		b.WriteString("#   - \"Always use TypeScript for frontend code\"\n")
	}
	b.WriteString("\n")

	// Targets
	b.WriteString("# AI tools to sync into. Valid: vscode-copilot, opencode, cursor\n")
	b.WriteString("targets:\n")
	for _, t := range m.Targets {
		b.WriteString(fmt.Sprintf("  - %s\n", t))
	}

	return b.String()
}

// initTarget represents which manifest file(s) to create.
type initTarget int

const (
	initTargetLocal  initTarget = iota + 1 // project-level vibes.yml only
	initTargetGlobal                       // global (~/.config/positive-vibes/vibes.yml) only
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

// writeInitManifest writes a bootstrap vibes.yml to path.
// It creates parent directories as needed and refuses to overwrite an existing file.
func writeInitManifest(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	content := renderBootstrapManifest(&manifest.Manifest{})
	return os.WriteFile(path, []byte(content), 0o644)
}

// promptInitTarget reads an interactive choice from stdin: (L)ocal, (G)lobal, or (B)oth.
func promptInitTarget() (initTarget, error) {
	fmt.Println("No vibes.yml found (local or global).")
	fmt.Println("Where would you like to create one?")
	fmt.Println("  [L] Local  (project-level vibes.yml)")
	fmt.Println("  [G] Global (~/.config/positive-vibes/vibes.yml)")
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

// localManifestExists checks whether a project-level vibes.yml or vibes.yaml exists.
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
	Short: "Create a starter vibes.yml",
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
			localPath := filepath.Join(project, "vibes.yml")

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

		fmt.Println("\nRun 'vibes config validate' to verify your setup.")
		fmt.Println("Run 'vibes apply' to sync your tools!")
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
