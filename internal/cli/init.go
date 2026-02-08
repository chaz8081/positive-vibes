package cli

import (
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

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a starter vibes.yml",
	Run: func(cmd *cobra.Command, args []string) {
		project := ProjectDir()

		// check for existing manifest (either filename)
		for _, name := range manifest.ManifestFilenames {
			p := filepath.Join(project, name)
			if _, err := os.Stat(p); err == nil {
				fmt.Printf("manifest already exists at %s\n", p)
				return
			}
		}

		fmt.Println("Scanning your project...")
		res, err := engine.ScanProject(project)
		if err != nil {
			fmt.Printf("error scanning project: %v\n", err)
			return
		}

		// build manifest
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
			Paths: map[string]string{"skills": "skills/"},
		}}

		manifestPath := filepath.Join(project, "vibes.yml")
		content := renderBootstrapManifest(m)
		if err := os.WriteFile(manifestPath, []byte(content), 0o644); err != nil {
			fmt.Printf("error writing vibes.yml: %v\n", err)
			return
		}

		fmt.Printf("Detected: %s project\n", res.Language)
		fmt.Printf("Created vibes.yml with %d recommended skills\n", len(res.RecommendedSkills))
		fmt.Printf("Targets: %s\n\n", join(res.SuggestedTargets, ", "))
		fmt.Println("Run 'vibes apply' to align your tools!")
	},
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
