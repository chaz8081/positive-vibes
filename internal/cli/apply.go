package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	applyGlobal  bool
)

func resolveManifestForApply(project, globalPath string, globalOnly bool) (*manifest.Manifest, error) {
	if globalOnly {
		if _, err := os.Stat(globalPath); err != nil {
			return nil, fmt.Errorf("no global manifest found at %s", globalPath)
		}
		m, err := manifest.LoadManifest(globalPath)
		if err != nil {
			return nil, fmt.Errorf("error loading global manifest: %w", err)
		}
		manifest.ResolveManifestPaths(m, filepath.Dir(globalPath))
		return m, nil
	}

	if _, _, err := manifest.LoadManifestFromProject(project); err != nil {
		return nil, fmt.Errorf("no manifest found in %s - run 'positive-vibes init' first", project)
	}

	merged, err := manifest.LoadMergedManifest(project, globalPath)
	if err != nil {
		return nil, fmt.Errorf("error loading manifest: %w", err)
	}

	return merged, nil
}

func formatOverrideWarnings(d manifest.RiskyOverrideDiagnostics) string {
	var lines []string
	if len(d.Skills) > 0 {
		lines = append(lines, "- skills: "+strings.Join(d.Skills, ", "))
	}
	if len(d.Instructions) > 0 {
		lines = append(lines, "- instructions: "+strings.Join(d.Instructions, ", "))
	}
	if len(d.Agents) > 0 {
		lines = append(lines, "- agents: "+strings.Join(d.Agents, ", "))
	}
	if len(lines) == 0 {
		return ""
	}
	return "Warning: local config overrides change resource source type:\n" + strings.Join(lines, "\n") + "\n"
}

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply manifest to all targets",
	Run: func(cmd *cobra.Command, args []string) {
		project := ProjectDir()
		globalPath := defaultGlobalManifestPath()
		merged, err := resolveManifestForApply(project, globalPath, applyGlobal)
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}

		if !applyGlobal {
			var globalM, localM *manifest.Manifest
			if data, readErr := os.ReadFile(globalPath); readErr == nil {
				globalM, _ = manifest.LoadManifestFromBytes(data)
			}
			if m, _, loadErr := manifest.LoadManifestFromProject(project); loadErr == nil {
				localM = m
			}
			if warning := formatOverrideWarnings(manifest.ComputeRiskyOverrideDiagnostics(globalM, localM)); warning != "" {
				fmt.Print(warning)
			}
		}

		// registries
		regs := []registry.SkillSource{registry.NewEmbeddedRegistry()}
		regs = append(regs, gitRegistriesFromManifest(merged)...)

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

		fmt.Println("Aligning your AI tools...")
		fmt.Println()
		res, err := applier.ApplyManifest(merged, project, opts)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			return
		}

		// Print per-operation lines
		for _, op := range res.Ops {
			kind := string(op.Kind)
			if kind == "" {
				kind = "skill"
			}
			switch op.Status {
			case engine.OpInstalled:
				fmt.Printf("  installed %s: %s -> %s\n", kind, op.SkillName, op.TargetName)
			case engine.OpSkipped:
				fmt.Printf("  skipped %s:   %s -> %s (already exists)\n", kind, op.SkillName, op.TargetName)
			case engine.OpNotFound:
				fmt.Printf("  not found %s: %s\n", kind, op.SkillName)
			case engine.OpError:
				fmt.Printf("  error %s:     %s -> %s: %s\n", kind, op.SkillName, op.TargetName, op.Error)
			}
		}

		// Summary line
		fmt.Println()
		if res.Installed > 0 {
			fmt.Printf("Done. Installed %d, skipped %d, errors %d.\n", res.Installed, res.Skipped, len(res.Errors))
		} else if res.Skipped > 0 {
			fmt.Printf("Already in sync. %d items up to date. Use --force to reinstall.\n", res.Skipped)
		} else {
			fmt.Println("Nothing to install. Check your manifest.")
		}
	},
}

func init() {
	applyCmd.Flags().BoolVarP(&applyForce, "force", "f", false, "overwrite existing skills")
	applyCmd.Flags().BoolVarP(&applyLink, "link", "l", false, "symlink skills instead of copying")
	applyCmd.Flags().BoolVar(&applyRefresh, "refresh", false, "pull latest from git registries before applying")
	applyCmd.Flags().BoolVar(&applyGlobal, "global", false, "apply only global config to current project targets")
	rootCmd.AddCommand(applyCmd)
}
