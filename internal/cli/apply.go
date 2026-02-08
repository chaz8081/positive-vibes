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
	applyForce   bool
	applyLink    bool
	applyRefresh bool
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
			regs = append(regs, gitRegistriesFromManifest(m)...)
		}

		// Refresh git registries if requested
		if applyRefresh {
			for _, r := range regs {
				if gr, ok := r.(*registry.GitRegistry); ok {
					debugf("refreshing registry %s ...", gr.Name())
					if err := gr.Refresh(); err != nil {
						fmt.Printf("warning: refresh %s failed: %v\n", gr.Name(), err)
					}
				}
			}
		}

		applier := engine.NewApplier(regs)
		opts := target.InstallOpts{Force: applyForce, Link: applyLink}

		fmt.Println("🧘 Aligning your AI tools...")
		fmt.Println()
		res, err := applier.Apply(manifestPath, opts)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			return
		}

		// Print per-operation lines
		for _, op := range res.Ops {
			switch op.Status {
			case engine.OpInstalled:
				fmt.Printf("  ✅ %s -> %s\n", op.SkillName, op.TargetName)
			case engine.OpSkipped:
				fmt.Printf("  ⏭️  %s -> %s (already exists)\n", op.SkillName, op.TargetName)
			case engine.OpNotFound:
				fmt.Printf("  ⚠️  %s (not found)\n", op.SkillName)
			case engine.OpError:
				fmt.Printf("  ❌ %s -> %s: %s\n", op.SkillName, op.TargetName, op.Error)
			}
		}

		// Summary line
		fmt.Println()
		if res.Installed > 0 {
			fmt.Printf("✨ Vibe check passed! Installed %d, skipped %d, errors %d.\n", res.Installed, res.Skipped, len(res.Errors))
			fmt.Println("🎵 Your tools are in harmony.")
		} else if res.Skipped > 0 {
			fmt.Printf("😎 Already in sync! %d skills up to date. Use --force to reinstall.\n", res.Skipped)
		} else {
			fmt.Println("🤔 Nothing to install. Check your vibes.yaml.")
		}
	},
}

func init() {
	applyCmd.Flags().BoolVarP(&applyForce, "force", "f", false, "overwrite existing skills")
	applyCmd.Flags().BoolVarP(&applyLink, "link", "l", false, "symlink skills instead of copying")
	applyCmd.Flags().BoolVar(&applyRefresh, "refresh", false, "pull latest from git registries before applying")
	rootCmd.AddCommand(applyCmd)
}
