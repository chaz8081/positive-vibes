package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chaz8081/positive-vibes/internal/engine"
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

func TestWriteInitManifest_CreatesLocalFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vibes.yml")

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
