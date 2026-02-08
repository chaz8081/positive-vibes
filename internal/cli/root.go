package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/chaz8081/positive-vibes/internal/registry"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "vibes",
		Short: "positive-vibes - harmonize your AI tooling ✨",
		Long: `positive-vibes helps align your AI tooling across platforms.
It's playful, helpful, and chill — manage Agent Skills and Instructions
from a single source of truth (vibes.yaml) and keep your dev setup groovy.

  Examples:
  vibes init    # create a vibes.yaml
  vibes apply   # push local vibes to supported platforms
`,
		// Default action shows help
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	projectDir string
	verbose    bool
)

func init() {
	// Persistent flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output 📣")
	rootCmd.PersistentFlags().StringVarP(&projectDir, "project-dir", "p", ".", "override project root directory")
}

// Execute runs the root cobra command
func Execute() error {
	return rootCmd.Execute()
}

// ProjectDir returns the configured project directory
func ProjectDir() string { return projectDir }

// Verbose returns whether verbose mode is enabled
func Verbose() bool { return verbose }

// helper for internal debug prints
func debugf(format string, a ...interface{}) {
	if verbose {
		fmt.Fprintf(os.Stderr, "[vibes] "+format+"\n", a...)
	}
}

// defaultCachePath returns ~/.positive-vibes/cache/<name>.
func defaultCachePath(name string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".positive-vibes", "cache", name)
}

// gitRegistriesFromManifest builds GitRegistry sources for each registry in the manifest.
func gitRegistriesFromManifest(m *manifest.Manifest) []registry.SkillSource {
	var sources []registry.SkillSource
	for _, r := range m.Registries {
		sources = append(sources, &registry.GitRegistry{
			RegistryName: r.Name,
			URL:          r.URL,
			CachePath:    defaultCachePath(r.Name),
			SkillsPath:   r.SkillsPath(),
		})
	}
	return sources
}
