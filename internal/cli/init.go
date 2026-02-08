package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chaz8081/positive-vibes/internal/engine"
	"github.com/chaz8081/positive-vibes/internal/manifest"
	_ "github.com/chaz8081/positive-vibes/internal/registry"
	"github.com/spf13/cobra"
)

// bootstrapHeader is the comment block written at the top of a new vibes.yml.
const bootstrapHeader = `# vibes.yml - positive-vibes project configuration
#
# This file defines your AI tooling setup. Run 'vibes apply' to sync
# skills and instructions to all configured targets.
#
# registries:
#   Remote git repositories containing skill collections.
#   Each registry has a name, URL, and optional paths map.
#   Example:
#     - name: awesome-copilot
#       url: https://github.com/github/awesome-copilot
#       paths:
#         skills: skills/    # subdirectory within the repo
#
# skills:
#   Skills to install. Reference by name (from a registry) or by local path.
#   Example:
#     - name: conventional-commits          # from registry
#     - name: my-custom-skill
#       path: ./local-skills/my-custom-skill # local directory
#
# instructions:
#   Free-form instructions appended to each target's configuration.
#   Example:
#     - "Always use TypeScript for frontend code"
#
# targets:
#   AI tools to sync skills into. Valid values:
#     vscode-copilot, opencode, cursor
#
# Global config: ~/.config/positive-vibes/vibes.yml (merged with this file)
#   Global registries, skills, and instructions are combined with project config.
#   Project targets override global targets.
#

`

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
		if err := manifest.SaveManifestWithComments(m, manifestPath, bootstrapHeader); err != nil {
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
