package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chaz8081/positive-vibes/internal/engine"
	"github.com/chaz8081/positive-vibes/pkg/schema"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate <description>",
	Short: "Generate a starter skill from a description 🌀",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		desc := strings.Join(args, " ")
		project := ProjectDir()

		gen := engine.NewMockGenerator()
		sk, err := gen.Generate(desc)
		if err != nil {
			fmt.Printf("error generating skill: %v\n", err)
			return
		}

		// render file
		content, err := schema.RenderSkillFile(sk)
		if err != nil {
			fmt.Printf("error rendering skill file: %v\n", err)
			return
		}

		dir := filepath.Join(project, "skills", sk.Name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Printf("error creating skill dir: %v\n", err)
			return
		}
		path := filepath.Join(dir, "SKILL.md")
		if err := os.WriteFile(path, content, 0o644); err != nil {
			fmt.Printf("error writing SKILL.md: %v\n", err)
			return
		}

		fmt.Println("🌀 Channeling the vibes...")
		fmt.Printf("✨ Generated skill: '%s'\n", sk.Name)
		fmt.Printf("📄 Written to skills/%s/SKILL.md\n\n", sk.Name)
		fmt.Println("Edit it to customize, then 'vibes install' to add it! 🎨")
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
}
