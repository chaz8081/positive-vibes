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

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a starter vibes.yaml ✨",
	Run: func(cmd *cobra.Command, args []string) {
		project := ProjectDir()

		// check for existing manifest
		manifestPath := filepath.Join(project, "vibes.yaml")
		if _, err := os.Stat(manifestPath); err == nil {
			fmt.Printf("⚠️  vibes.yaml already exists at %s\n", manifestPath)
			return
		}

		fmt.Println("🔍 Scanning your project's energy...")
		res, err := engine.ScanProject(project)
		if err != nil {
			fmt.Printf("error scanning project: %v\n", err)
			return
		}

		// build manifest
		m := &manifest.Manifest{}
		// recommended skills
		for _, s := range res.RecommendedSkills {
			m.Skills = append(m.Skills, manifest.SkillRef{Name: s})
		}
		// targets
		for _, t := range res.SuggestedTargets {
			m.Targets = append(m.Targets, t)
		}
		// add default registry
		m.Registries = []manifest.RegistryRef{{
			Name:  "awesome-copilot",
			URL:   "https://github.com/github/awesome-copilot",
			Paths: map[string]string{"skills": "skills/"},
		}}

		if err := manifest.SaveManifest(m, manifestPath); err != nil {
			fmt.Printf("error writing vibes.yaml: %v\n", err)
			return
		}

		fmt.Printf("✨ Detected: %s project\n", res.Language)
		fmt.Printf("📝 Created vibes.yaml with %d recommended skills\n", len(res.RecommendedSkills))
		fmt.Printf("🎯 Targets: %s\n\n", join(res.SuggestedTargets, ", "))
		fmt.Println("Run 'vibes apply' to align your tools! 🚀")
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
