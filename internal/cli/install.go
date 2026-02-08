package cli

import (
	"fmt"
	"path/filepath"

	"github.com/chaz8081/positive-vibes/internal/engine"
	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/chaz8081/positive-vibes/internal/registry"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install <skill-name>",
	Short: "Add a skill to vibes.yaml 🔧",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		project := ProjectDir()
		manifestPath := filepath.Join(project, "vibes.yaml")

		fmt.Printf("🔎 Looking for '%s'...\n", name)

		// prepare registries: embedded + ones from manifest
		regs := []registry.SkillSource{registry.NewEmbeddedRegistry()}
		if m, err := manifest.LoadManifest(manifestPath); err == nil {
			for _, r := range m.Registries {
				// create GitRegistry entries for now
				regs = append(regs, &registry.GitRegistry{RegistryName: r.Name, URL: r.URL})
			}
		}

		inst := engine.NewInstaller(regs)
		if err := inst.Install(name, manifestPath); err != nil {
			fmt.Printf("error: %v\n", err)
			return
		}

		fmt.Println("✅ Found it! Adding to your vibes...")
		fmt.Println("📝 Updated vibes.yaml")
		fmt.Println("Run 'vibes apply' to install it everywhere! 🎯")
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
