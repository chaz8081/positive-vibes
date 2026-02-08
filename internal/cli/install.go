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
	Short: "Add a skill to the manifest",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		project := ProjectDir()

		// Find existing manifest (vibes.yml preferred, vibes.yaml fallback)
		_, manifestPath, findErr := manifest.LoadManifestFromProject(project)
		if findErr != nil {
			// No manifest yet — default to vibes.yml
			manifestPath = filepath.Join(project, "vibes.yml")
		}

		fmt.Printf("Looking for '%s'...\n", name)

		// prepare registries: embedded + ones from manifest (if loadable)
		regs := []registry.SkillSource{registry.NewEmbeddedRegistry()}
		if m, err := manifest.LoadManifest(manifestPath); err == nil {
			regs = append(regs, gitRegistriesFromManifest(m)...)
		}

		inst := engine.NewInstaller(regs)
		if err := inst.Install(name, manifestPath); err != nil {
			fmt.Printf("error: %v\n", err)
			return
		}

		// Check what was installed to provide the right feedback
		m, err := manifest.LoadManifest(manifestPath)
		if err == nil {
			for _, s := range m.Skills {
				if s.Name == name && s.Path != "" {
					fmt.Printf("Found local skill at %s\n", s.Path)
					fmt.Println("Added to manifest with local path")
					fmt.Println("Run 'vibes apply' to install it everywhere!")
					return
				}
			}
		}

		fmt.Println("Found it! Adding to your manifest...")
		fmt.Printf("Updated %s\n", filepath.Base(manifestPath))
		fmt.Println("Run 'vibes apply' to install it everywhere!")
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
