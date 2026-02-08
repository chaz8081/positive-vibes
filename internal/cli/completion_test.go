package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Shell detection tests ---

func TestDetectShell_FromEnvVar(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")
	shell, err := detectShell("")
	require.NoError(t, err)
	assert.Equal(t, "zsh", shell)
}

func TestDetectShell_FromEnvVar_FullPath(t *testing.T) {
	t.Setenv("SHELL", "/usr/local/bin/bash")
	shell, err := detectShell("")
	require.NoError(t, err)
	assert.Equal(t, "bash", shell)
}

func TestDetectShell_OverrideWins(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")
	shell, err := detectShell("fish")
	require.NoError(t, err)
	assert.Equal(t, "fish", shell)
}

func TestDetectShell_UnsupportedShell(t *testing.T) {
	t.Setenv("SHELL", "/bin/csh")
	_, err := detectShell("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported shell")
}

func TestDetectShell_NoShellEnv(t *testing.T) {
	t.Setenv("SHELL", "")
	_, err := detectShell("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not detect shell")
}

func TestDetectShell_PowershellVariants(t *testing.T) {
	for _, name := range []string{"powershell", "pwsh"} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("SHELL", "/usr/bin/"+name)
			shell, err := detectShell("")
			require.NoError(t, err)
			assert.Equal(t, "powershell", shell)
		})
	}
}

// --- Completion path resolution tests ---

func TestCompletionPath_Zsh_Linux(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("unix-only test")
	}
	// Provide a writable fpath dir to avoid needing real system paths
	dir := t.TempDir()
	path := completionPath("zsh", dir)
	assert.Equal(t, filepath.Join(dir, "_vibes"), path)
}

func TestCompletionPath_Bash(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := completionPath("bash", "")
	assert.Equal(t, filepath.Join(home, ".local", "share", "bash-completion", "completions", "vibes"), path)
}

func TestCompletionPath_Fish(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	path := completionPath("fish", "")
	assert.Equal(t, filepath.Join(configDir, "fish", "completions", "vibes.fish"), path)
}

func TestCompletionPath_Fish_DefaultConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	path := completionPath("fish", "")
	assert.Equal(t, filepath.Join(home, ".config", "fish", "completions", "vibes.fish"), path)
}

func TestCompletionPath_Powershell(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := completionPath("powershell", "")
	// Should be under home somewhere
	assert.Contains(t, path, "vibes.ps1")
}

func TestCompletionPath_UnknownShell(t *testing.T) {
	path := completionPath("tcsh", "")
	assert.Equal(t, "", path)
}

// --- Install/uninstall file operation tests ---

func TestInstallCompletion_WritesFile(t *testing.T) {
	dir := t.TempDir()
	destPath := filepath.Join(dir, "_vibes")

	err := installCompletionFile(destPath, []byte("# completion script content"))
	require.NoError(t, err)

	data, err := os.ReadFile(destPath)
	require.NoError(t, err)
	assert.Equal(t, "# completion script content", string(data))
}

func TestInstallCompletion_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	destPath := filepath.Join(dir, "deep", "nested", "_vibes")

	err := installCompletionFile(destPath, []byte("content"))
	require.NoError(t, err)

	data, err := os.ReadFile(destPath)
	require.NoError(t, err)
	assert.Equal(t, "content", string(data))
}

func TestUninstallCompletion_RemovesFile(t *testing.T) {
	dir := t.TempDir()
	destPath := filepath.Join(dir, "_vibes")
	require.NoError(t, os.WriteFile(destPath, []byte("content"), 0o644))

	err := uninstallCompletionFile(destPath)
	require.NoError(t, err)

	_, err = os.Stat(destPath)
	assert.True(t, os.IsNotExist(err))
}

func TestUninstallCompletion_NoFileIsNotError(t *testing.T) {
	dir := t.TempDir()
	destPath := filepath.Join(dir, "_vibes")

	err := uninstallCompletionFile(destPath)
	require.NoError(t, err)
}
