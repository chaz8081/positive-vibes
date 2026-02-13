package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chaz8081/positive-vibes/internal/cli/ui"
	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/chaz8081/positive-vibes/internal/registry"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	rootCmd = &cobra.Command{
		Use:   "positive-vibes",
		Short: "positive-vibes - harmonize your AI tooling",
		Long: `positive-vibes helps align your AI tooling across platforms.
Manage Agent Skills and Instructions from a single source of truth
(vibes.yaml) and keep your dev setup in sync.

  Examples:
  positive-vibes init    # create a vibes.yaml
  positive-vibes apply   # push local vibes to supported platforms
`,
		// Default action launches TUI for interactive terminals and falls back to help otherwise.
		RunE: func(cmd *cobra.Command, args []string) error {
			if isInteractiveTTY() {
				return launchUI()
			}
			return cmd.Help()
		},
	}

	projectDir       string
	verbose          bool
	launchUI         = ui.Run
	isInteractiveTTY = func() bool {
		return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
	}
)

func init() {
	// Persistent flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
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
		fmt.Fprintf(os.Stderr, "[positive-vibes] "+format+"\n", a...)
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

// defaultGlobalManifestPath returns the path to the user-level global config.
// Uses $XDG_CONFIG_HOME/positive-vibes/vibes.yaml, falling back to
// ~/.config/positive-vibes/vibes.yaml.
func defaultGlobalManifestPath() string {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "positive-vibes", "vibes.yaml")
}

// gitRegistriesFromManifest builds GitRegistry sources for each registry in the manifest.
func gitRegistriesFromManifest(m *manifest.Manifest) []registry.SkillSource {
	var sources []registry.SkillSource
	for _, r := range m.Registries {
		sources = append(sources, &registry.GitRegistry{
			RegistryName:     r.Name,
			URL:              r.URL,
			CachePath:        defaultCachePath(r.Name),
			SkillsPath:       r.SkillsPath(),
			InstructionsPath: r.InstructionsPath(),
			AgentsPath:       r.AgentsPath(),
			Ref:              r.Ref,
		})
	}
	return sources
}
