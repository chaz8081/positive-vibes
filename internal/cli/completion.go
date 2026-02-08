package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var completionShellOverride string

// supportedShells lists shells we can generate completions for.
var supportedShells = []string{"zsh", "bash", "fish", "powershell"}

// detectShell returns the normalized shell name. If override is non-empty it
// is used directly; otherwise $SHELL is read. Returns error if the shell
// cannot be determined or is unsupported.
func detectShell(override string) (string, error) {
	name := override
	if name == "" {
		name = os.Getenv("SHELL")
		if name == "" {
			return "", fmt.Errorf("could not detect shell: $SHELL is not set (use --shell to specify)")
		}
		name = filepath.Base(name) // /usr/bin/zsh -> zsh
	}

	// Normalize powershell variants.
	if name == "pwsh" {
		name = "powershell"
	}

	for _, s := range supportedShells {
		if name == s {
			return name, nil
		}
	}
	return "", fmt.Errorf("unsupported shell: %s (supported: %s)", name, strings.Join(supportedShells, ", "))
}

// completionPath returns the conventional filesystem path where a completion
// script should be installed for the given shell and OS.
//
// The hintDir parameter is used as a preferred directory for zsh (e.g. the
// first writable $fpath entry). Pass "" to use defaults.
func completionPath(shell string, hintDir string) string {
	switch shell {
	case "zsh":
		return zshCompletionPath(hintDir)
	case "bash":
		return bashCompletionPath()
	case "fish":
		return fishCompletionPath()
	case "powershell":
		return powershellCompletionPath()
	default:
		return ""
	}
}

func zshCompletionPath(hintDir string) string {
	if hintDir != "" {
		return filepath.Join(hintDir, "_vibes")
	}
	// macOS with Homebrew
	if runtime.GOOS == "darwin" {
		// Try brew --prefix first (common: /opt/homebrew or /usr/local)
		for _, prefix := range []string{"/opt/homebrew", "/usr/local"} {
			dir := filepath.Join(prefix, "share", "zsh", "site-functions")
			if info, err := os.Stat(dir); err == nil && info.IsDir() {
				return filepath.Join(dir, "_vibes")
			}
		}
	}
	// Fallback: ~/.local/share/zsh/site-functions (works with oh-my-zsh & standard zsh)
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "zsh", "site-functions", "_vibes")
}

func bashCompletionPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "bash-completion", "completions", "vibes")
}

func fishCompletionPath() string {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "fish", "completions", "vibes.fish")
}

func powershellCompletionPath() string {
	home, _ := os.UserHomeDir()
	// Standard cross-platform location for PowerShell modules
	return filepath.Join(home, ".config", "powershell", "vibes.ps1")
}

// installCompletionFile writes data to destPath, creating parent directories
// as needed.
func installCompletionFile(destPath string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	if err := os.WriteFile(destPath, data, 0o644); err != nil {
		return fmt.Errorf("write completion file: %w", err)
	}
	return nil
}

// uninstallCompletionFile removes the completion file at destPath. It is not
// an error if the file does not exist.
func uninstallCompletionFile(destPath string) error {
	if err := os.Remove(destPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove completion file: %w", err)
	}
	return nil
}

// generateCompletionScript produces the shell completion script for the given
// shell by calling Cobra's built-in generators.
func generateCompletionScript(cmd *cobra.Command, shell string) ([]byte, error) {
	var buf bytes.Buffer
	var err error
	switch shell {
	case "zsh":
		err = cmd.Root().GenZshCompletion(&buf)
	case "bash":
		err = cmd.Root().GenBashCompletion(&buf)
	case "fish":
		err = cmd.Root().GenFishCompletion(&buf, true)
	case "powershell":
		err = cmd.Root().GenPowerShellCompletionWithDesc(&buf)
	default:
		return nil, fmt.Errorf("unknown shell: %s", shell)
	}
	if err != nil {
		return nil, fmt.Errorf("generate %s completion: %w", shell, err)
	}
	return buf.Bytes(), nil
}

var completionInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Auto-detect your shell and install completions",
	Long: `Detects your current shell from $SHELL and writes a completion
script to the conventional system path. Use --shell to override detection.

Supported shells: zsh, bash, fish, powershell`,
	RunE: func(cmd *cobra.Command, args []string) error {
		shell, err := detectShell(completionShellOverride)
		if err != nil {
			return err
		}

		destPath := completionPath(shell, "")
		if destPath == "" {
			return fmt.Errorf("could not determine completion path for %s", shell)
		}

		script, err := generateCompletionScript(cmd, shell)
		if err != nil {
			return err
		}

		if err := installCompletionFile(destPath, script); err != nil {
			return err
		}

		fmt.Printf("Detected shell: %s\n", shell)
		fmt.Printf("Wrote completions to %s\n", destPath)
		fmt.Println("Restart your shell to activate completions.")
		return nil
	},
}

var completionUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove installed shell completions",
	Long: `Detects your current shell from $SHELL and removes the completion
script from the conventional system path. Use --shell to override detection.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		shell, err := detectShell(completionShellOverride)
		if err != nil {
			return err
		}

		destPath := completionPath(shell, "")
		if destPath == "" {
			return fmt.Errorf("could not determine completion path for %s", shell)
		}

		if _, statErr := os.Stat(destPath); os.IsNotExist(statErr) {
			fmt.Printf("No completion file found at %s (nothing to remove)\n", destPath)
			return nil
		}

		if err := uninstallCompletionFile(destPath); err != nil {
			return err
		}

		fmt.Printf("Removed %s completions from %s\n", shell, destPath)
		return nil
	},
}

func init() {
	completionInstallCmd.Flags().StringVar(&completionShellOverride, "shell", "", "override shell detection (zsh, bash, fish, powershell)")
	completionUninstallCmd.Flags().StringVar(&completionShellOverride, "shell", "", "override shell detection (zsh, bash, fish, powershell)")

	// Disable Cobra's auto-generated completion command so we can provide
	// our own with install/uninstall subcommands alongside the standard
	// print-to-stdout ones.
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	completionCmd := &cobra.Command{
		Use:   "completion [command]",
		Short: "Generate or install shell completions",
		Long: `Generate or install shell completion scripts.

Use "install" to auto-detect your shell and write completions to the right
system path. Use the shell-specific subcommands (bash, zsh, fish, powershell)
to print the script to stdout for manual setup.`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	// Standard Cobra print-to-stdout subcommands
	completionCmd.AddCommand(&cobra.Command{
		Use:   "bash",
		Short: "Print bash completion script to stdout",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenBashCompletion(os.Stdout)
		},
	})
	completionCmd.AddCommand(&cobra.Command{
		Use:   "zsh",
		Short: "Print zsh completion script to stdout",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenZshCompletion(os.Stdout)
		},
	})
	completionCmd.AddCommand(&cobra.Command{
		Use:   "fish",
		Short: "Print fish completion script to stdout",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		},
	})
	completionCmd.AddCommand(&cobra.Command{
		Use:   "powershell",
		Short: "Print powershell completion script to stdout",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		},
	})

	// Our install/uninstall subcommands
	completionCmd.AddCommand(completionInstallCmd)
	completionCmd.AddCommand(completionUninstallCmd)

	rootCmd.AddCommand(completionCmd)
}
