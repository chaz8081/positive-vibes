package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chaz8081/positive-vibes/internal/engine"
	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/chaz8081/positive-vibes/internal/registry"
	"github.com/chaz8081/positive-vibes/internal/target"
	"github.com/spf13/cobra"
)

var (
	applyForce bool
	applyLink  bool
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply vibes.yaml to all targets 🧘",
	Run: func(cmd *cobra.Command, args []string) {
		project := ProjectDir()
		manifestPath := filepath.Join(project, "vibes.yaml")

		if _, err := os.Stat(manifestPath); err != nil {
			fmt.Printf("no vibes.yaml found at %s - run 'vibes init' first\n", manifestPath)
			return
		}

		// registries
		regs := []registry.SkillSource{registry.NewEmbeddedRegistry()}
		if m, err := manifest.LoadManifest(manifestPath); err == nil {
			for _, r := range m.Registries {
				regs = append(regs, &registry.GitRegistry{RegistryName: r.Name, URL: r.URL})
			}
		}

		applier := engine.NewApplier(regs)
		opts := target.InstallOpts{Force: applyForce, Link: applyLink}

		fmt.Println("🧘 Aligning your AI tools...")
		res, err := applier.Apply(manifestPath, opts)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			return
		}

		// Print individual results are available via res.Errors and counts
		if len(res.Errors) > 0 {
			for _, e := range res.Errors {
				fmt.Printf("⚠️  %s\n", e)
			}
		}

		// try to read manifest to count targets
		tgtCount := 0
		if m, err := manifest.LoadManifest(manifestPath); err == nil {
			tgtCount = len(m.Targets)
		}

		fmt.Printf("\n✨ Vibe check passed! Installed %d skills across %d targets.\n", res.Installed, tgtCount)
		fmt.Println("🎵 Your tools are in harmony.")
	},
}

// osStat is a tiny wrapper to avoid importing os in top-level var section
func osStat(p string) (bool, error) {
	if _, err := osStatImpl(p); err == nil {
		return true, nil
	} else {
		return false, err
	}
}

func osStatImpl(p string) (bool, error) { // actual os.Stat usage
	_, err := os.Stat(p)
	if err != nil {
		return false, err
	}
	return true, nil
}

func init() {
	applyCmd.Flags().BoolVarP(&applyForce, "force", "f", false, "overwrite existing skills")
	applyCmd.Flags().BoolVarP(&applyLink, "link", "l", false, "symlink skills instead of copying")
	rootCmd.AddCommand(applyCmd)
}
