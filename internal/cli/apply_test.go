package cli

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveManifestForApply_DefaultRequiresLocalManifest(t *testing.T) {
	projectDir := t.TempDir()
	globalDir := t.TempDir()
	globalPath := filepath.Join(globalDir, "vibes.yaml")

	globalContent := `skills:
  - name: conventional-commits
targets:
  - opencode
`
	require.NoError(t, os.WriteFile(globalPath, []byte(globalContent), 0o644))

	_, err := resolveManifestForApply(projectDir, globalPath, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no manifest found in")
}

func TestResolveManifestForApply_GlobalModeUsesGlobalManifest(t *testing.T) {
	projectDir := t.TempDir()
	globalDir := t.TempDir()
	globalPath := filepath.Join(globalDir, "vibes.yaml")

	globalContent := `skills:
  - name: conventional-commits
targets:
  - opencode
`
	require.NoError(t, os.WriteFile(globalPath, []byte(globalContent), 0o644))

	m, err := resolveManifestForApply(projectDir, globalPath, true)
	require.NoError(t, err)
	require.NotNil(t, m)
	assert.Len(t, m.Skills, 1)
	assert.Equal(t, "conventional-commits", m.Skills[0].Name)
}

func TestResolveManifestForApply_GlobalModeResolvesRelativePaths(t *testing.T) {
	projectDir := t.TempDir()
	globalDir := t.TempDir()
	globalPath := filepath.Join(globalDir, "vibes.yaml")

	globalContent := `skills:
  - name: local-skill
    path: ./skills/local-skill
targets:
  - opencode
`
	require.NoError(t, os.WriteFile(globalPath, []byte(globalContent), 0o644))

	m, err := resolveManifestForApply(projectDir, globalPath, true)
	require.NoError(t, err)
	require.NotNil(t, m)
	assert.Equal(t, filepath.Join(globalDir, "skills", "local-skill"), m.Skills[0].Path)
}

func TestResolveManifestForApply_GlobalModeRequiresGlobalManifest(t *testing.T) {
	projectDir := t.TempDir()
	globalPath := filepath.Join(t.TempDir(), "vibes.yaml")

	_, err := resolveManifestForApply(projectDir, globalPath, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no global manifest found")
}

func TestFormatOverrideWarnings(t *testing.T) {
	d := manifest.RiskyOverrideDiagnostics{
		Skills:       []string{"shared-skill"},
		Instructions: []string{"shared-inst"},
		Agents:       []string{"shared-agent"},
	}
	out := formatOverrideWarnings(d)

	assert.Contains(t, out, "Warning: local config overrides change resource source type")
	assert.Contains(t, out, "skills: shared-skill")
	assert.Contains(t, out, "instructions: shared-inst")
	assert.Contains(t, out, "agents: shared-agent")
}

func TestApplyCommand_HasGlobalFlag(t *testing.T) {
	f := applyCmd.Flags().Lookup("global")
	require.NotNil(t, f)
	assert.Equal(t, "bool", f.Value.Type())
}

func TestGlobalApplyNoOpMessage_WhenNoInstallableResources(t *testing.T) {
	m := &manifest.Manifest{
		Registries: []manifest.RegistryRef{{Name: "r", URL: "https://example.com/r", Ref: "latest"}},
		Targets:    []string{"opencode"},
	}

	msg, skip := globalApplyNoOpMessage(m)
	assert.True(t, skip)
	assert.Contains(t, msg, "No-op")
	assert.Contains(t, msg, "global config has no installable resources")
}

func TestRootNoArgs_TTYLaunchesUISuccessfully(t *testing.T) {
	originalHelpFn := rootCmd.HelpFunc()
	originalLaunchUI := launchUI
	originalIsInteractiveTTY := isInteractiveTTY
	t.Cleanup(func() {
		rootCmd.SetHelpFunc(originalHelpFn)
		launchUI = originalLaunchUI
		isInteractiveTTY = originalIsInteractiveTTY
	})

	calledLaunchUI := false
	launchUI = func() error {
		calledLaunchUI = true
		return nil
	}

	isInteractiveTTY = func() bool {
		return true
	}

	helpCalled := false
	rootCmd.SetHelpFunc(func(*cobra.Command, []string) {
		helpCalled = true
	})

	require.NotNil(t, rootCmd.RunE)
	err := rootCmd.RunE(rootCmd, []string{})
	require.NoError(t, err)

	assert.True(t, calledLaunchUI)
	assert.False(t, helpCalled)
}

func TestRootNoArgs_NonTTYShowsHelp(t *testing.T) {
	originalHelpFn := rootCmd.HelpFunc()
	originalLaunchUI := launchUI
	originalIsInteractiveTTY := isInteractiveTTY
	t.Cleanup(func() {
		rootCmd.SetHelpFunc(originalHelpFn)
		launchUI = originalLaunchUI
		isInteractiveTTY = originalIsInteractiveTTY
	})

	launchCalled := false
	launchUI = func() error {
		launchCalled = true
		return nil
	}

	isInteractiveTTY = func() bool {
		return false
	}

	helpCalled := false
	rootCmd.SetHelpFunc(func(*cobra.Command, []string) {
		helpCalled = true
	})

	require.NotNil(t, rootCmd.RunE)
	err := rootCmd.RunE(rootCmd, []string{})
	require.NoError(t, err)

	assert.False(t, launchCalled)
	assert.True(t, helpCalled)
}

func TestRootNoArgs_TTYLaunchFailureReturnsError(t *testing.T) {
	originalHelpFn := rootCmd.HelpFunc()
	originalLaunchUI := launchUI
	originalIsInteractiveTTY := isInteractiveTTY
	t.Cleanup(func() {
		rootCmd.SetHelpFunc(originalHelpFn)
		launchUI = originalLaunchUI
		isInteractiveTTY = originalIsInteractiveTTY
	})

	launchErr := errors.New("boom")
	launchCalled := false
	launchUI = func() error {
		launchCalled = true
		return launchErr
	}

	isInteractiveTTY = func() bool {
		return true
	}

	helpCalled := false
	rootCmd.SetHelpFunc(func(*cobra.Command, []string) {
		helpCalled = true
	})

	require.NotNil(t, rootCmd.RunE)
	err := rootCmd.RunE(rootCmd, []string{})
	require.ErrorIs(t, err, launchErr)

	assert.True(t, launchCalled)
	assert.False(t, helpCalled)
}
