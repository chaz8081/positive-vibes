package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chaz8081/positive-vibes/internal/engine"
	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveInitAction_NeitherExists_UserPicksLocal(t *testing.T) {
	action, err := resolveInitAction(false, false, func() (initTarget, error) {
		return initTargetLocal, nil
	})
	require.NoError(t, err)
	assert.Equal(t, initTargetLocal, action)
}

func TestResolveInitAction_NeitherExists_UserPicksGlobal(t *testing.T) {
	action, err := resolveInitAction(false, false, func() (initTarget, error) {
		return initTargetGlobal, nil
	})
	require.NoError(t, err)
	assert.Equal(t, initTargetGlobal, action)
}

func TestResolveInitAction_NeitherExists_UserPicksBoth(t *testing.T) {
	action, err := resolveInitAction(false, false, func() (initTarget, error) {
		return initTargetBoth, nil
	})
	require.NoError(t, err)
	assert.Equal(t, initTargetBoth, action)
}

func TestResolveInitAction_GlobalExists_NoLocal(t *testing.T) {
	promptCalled := false
	action, err := resolveInitAction(true, false, func() (initTarget, error) {
		promptCalled = true
		return 0, nil
	})
	require.NoError(t, err)
	assert.Equal(t, initTargetLocal, action)
	assert.False(t, promptCalled, "should not prompt when action is obvious")
}

func TestResolveInitAction_NoGlobal_LocalExists(t *testing.T) {
	promptCalled := false
	action, err := resolveInitAction(false, true, func() (initTarget, error) {
		promptCalled = true
		return 0, nil
	})
	require.NoError(t, err)
	assert.Equal(t, initTargetGlobal, action)
	assert.False(t, promptCalled, "should not prompt when action is obvious")
}

func TestResolveInitAction_BothExist(t *testing.T) {
	_, err := resolveInitAction(true, true, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exist")
}

// --- writeInitManifest tests ---

func TestWriteInitManifest_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vibes.yaml")

	err := writeInitManifest(path)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)

	assert.Contains(t, content, "registries:")
	assert.Contains(t, content, "skills:")
	assert.Contains(t, content, "targets:")
}

func TestWriteInitManifest_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deep", "nested", "vibes.yml")

	err := writeInitManifest(path)
	require.NoError(t, err)

	_, err = os.Stat(path)
	require.NoError(t, err)
}

func TestWriteInitManifest_DoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vibes.yml")
	require.NoError(t, os.WriteFile(path, []byte("existing"), 0o644))

	err := writeInitManifest(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// Original content preserved
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "existing", string(data))
}

// --- buildManifestFromScan tests ---

func TestBuildManifestFromScan_SetsRefLatest(t *testing.T) {
	res := &engine.ScanResult{
		Language:          "go",
		RecommendedSkills: []string{"tdd"},
		SuggestedTargets:  []string{"opencode"},
	}
	m := buildManifestFromScan(res)

	require.Len(t, m.Registries, 1)
	assert.Equal(t, "latest", m.Registries[0].Ref, "default registry should have ref set to 'latest'")
}

func TestBuildManifestFromScan_PassesValidation(t *testing.T) {
	res := &engine.ScanResult{
		Language:          "go",
		RecommendedSkills: []string{"tdd"},
		SuggestedTargets:  []string{"opencode"},
	}
	m := buildManifestFromScan(res)
	err := m.Validate()
	require.NoError(t, err, "manifest from scan should pass validation")
}

// --- renderBootstrapManifest template content tests ---

func TestRenderBootstrapManifest_Header_UsesVibesYaml(t *testing.T) {
	content := renderBootstrapManifest(&manifest.Manifest{})
	assert.Contains(t, content, "vibes.yaml")
	assert.NotContains(t, content, "vibes.yml")
}

func TestRenderBootstrapManifest_Header_IsGeneric(t *testing.T) {
	// The header should work for both global and local configs
	content := renderBootstrapManifest(&manifest.Manifest{})
	// Should mention merge behavior
	assert.Contains(t, content, "project values take priority")
	// Should NOT say "project configuration" (too local-specific)
	assert.NotContains(t, content, "project configuration")
}

func TestRenderBootstrapManifest_EmptySkills_ShowsCommentedExamples(t *testing.T) {
	content := renderBootstrapManifest(&manifest.Manifest{})
	// When no skills are provided, should show commented-out examples
	assert.Contains(t, content, "# skills:")
	assert.Contains(t, content, "#   - name: conventional-commits")
}

func TestRenderBootstrapManifest_EmptyTargets_ShowsCommentedExamples(t *testing.T) {
	content := renderBootstrapManifest(&manifest.Manifest{})
	// When no targets are provided, should show commented-out examples
	assert.Contains(t, content, "# targets:")
	assert.Contains(t, content, "#   - vscode-copilot")
	assert.Contains(t, content, "#   - opencode")
	assert.Contains(t, content, "#   - cursor")
}

func TestRenderBootstrapManifest_EmptyAgents_ShowsBothFormats(t *testing.T) {
	content := renderBootstrapManifest(&manifest.Manifest{})
	// Agent examples should show both path and registry formats
	assert.Contains(t, content, "#     path:")
	assert.Contains(t, content, "#     registry:")
}

func TestRenderBootstrapManifest_EmptyInstructions_ShowsApplyTo(t *testing.T) {
	content := renderBootstrapManifest(&manifest.Manifest{})
	// Instruction examples should show apply_to field
	assert.Contains(t, content, "#     apply_to:")
}

func TestRenderBootstrapManifest_PopulatedSkills_NotCommented(t *testing.T) {
	m := &manifest.Manifest{
		Skills: []manifest.SkillRef{{Name: "tdd"}},
	}
	content := renderBootstrapManifest(m)
	// Populated skills should NOT be commented out
	assert.Contains(t, content, "skills:\n  - name: tdd")
	assert.NotContains(t, content, "# skills:")
}

func TestRenderBootstrapManifest_PopulatedTargets_NotCommented(t *testing.T) {
	m := &manifest.Manifest{
		Targets: []string{"opencode"},
	}
	content := renderBootstrapManifest(m)
	// Populated targets should NOT be commented out
	assert.Contains(t, content, "targets:\n  - opencode")
	assert.NotContains(t, content, "# targets:")
}

// --- buildGlobalDefaults tests ---

func TestBuildGlobalDefaults_HasAwesomeCopilotRegistry(t *testing.T) {
	m := buildGlobalDefaults()
	require.Len(t, m.Registries, 1)
	assert.Equal(t, "awesome-copilot", m.Registries[0].Name)
	assert.Equal(t, "https://github.com/github/awesome-copilot", m.Registries[0].URL)
	assert.Equal(t, "latest", m.Registries[0].Ref)
}

func TestBuildGlobalDefaults_NoSkillsOrTargets(t *testing.T) {
	m := buildGlobalDefaults()
	// Global defaults should have no skills or targets (those are project-specific)
	assert.Empty(t, m.Skills)
	assert.Empty(t, m.Targets)
}

func TestWriteInitManifest_GlobalContent_HasRegistry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vibes.yaml")

	err := writeInitManifest(path)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)

	// Global init should include the awesome-copilot registry
	assert.Contains(t, content, "awesome-copilot")
	assert.Contains(t, content, "https://github.com/github/awesome-copilot")
}
