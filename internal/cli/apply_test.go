package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chaz8081/positive-vibes/internal/manifest"
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

func TestResolveManifestForApply_GlobalModeRequiresGlobalManifest(t *testing.T) {
	projectDir := t.TempDir()
	globalPath := filepath.Join(t.TempDir(), "vibes.yaml")

	_, err := resolveManifestForApply(projectDir, globalPath, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no global manifest found")
}

func TestFormatOverrideWarnings(t *testing.T) {
	d := manifest.OverrideDiagnostics{
		Registries:   []string{"shared-reg"},
		Skills:       []string{"shared-skill"},
		Instructions: []string{"shared-inst"},
		Agents:       []string{"shared-agent"},
	}
	out := formatOverrideWarnings(d)

	assert.Contains(t, out, "Warning: local config overrides global entries")
	assert.Contains(t, out, "registries: shared-reg")
	assert.Contains(t, out, "skills: shared-skill")
	assert.Contains(t, out, "instructions: shared-inst")
	assert.Contains(t, out, "agents: shared-agent")
}
