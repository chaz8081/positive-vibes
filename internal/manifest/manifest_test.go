package manifest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const exampleYAML = `registries:
  - name: awesome-copilot
    url: https://github.com/github/awesome-copilot

skills:
  - name: conventional-commits
  - name: react-expert
    version: "1.0"
  - name: my-custom-skill
    path: ./local-skills/my-custom-skill

instructions:
  - "Always use TypeScript for frontend code"
  - "Prefer functional components over class components"

targets:
  - vscode-copilot
  - opencode
  - cursor
`

func TestLoadManifest_Valid(t *testing.T) {
	m, err := LoadManifestFromBytes([]byte(exampleYAML))
	require.NoError(t, err)
	require.NotNil(t, m)

	assert.Len(t, m.Registries, 1)
	assert.Equal(t, "awesome-copilot", m.Registries[0].Name)
	assert.Equal(t, "https://github.com/github/awesome-copilot", m.Registries[0].URL)

	assert.Len(t, m.Skills, 3)
	assert.Equal(t, "conventional-commits", m.Skills[0].Name)
	assert.Equal(t, "react-expert", m.Skills[1].Name)
	assert.Equal(t, "1.0", m.Skills[1].Version)
	assert.Equal(t, "my-custom-skill", m.Skills[2].Name)
	assert.Equal(t, "./local-skills/my-custom-skill", m.Skills[2].Path)

	assert.Len(t, m.Instructions, 2)
	assert.Contains(t, m.Instructions[0], "TypeScript")

	assert.Len(t, m.Targets, 3)
	assert.Contains(t, m.Targets, "vscode-copilot")
}

func TestLoadManifest_MinimalValid(t *testing.T) {
	yaml := `skills:
  - name: foo
targets:
  - opencode
`
	m, err := LoadManifestFromBytes([]byte(yaml))
	require.NoError(t, err)
	require.NotNil(t, m)

	assert.Len(t, m.Skills, 1)
	assert.Equal(t, "foo", m.Skills[0].Name)
	assert.Len(t, m.Targets, 1)
	assert.Equal(t, "opencode", m.Targets[0])
}

func TestLoadManifest_FileNotFound(t *testing.T) {
	_, err := LoadManifest(filepath.Join(t.TempDir(), "nope.yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read manifest")
}

func TestValidate_NoSkills(t *testing.T) {
	m := &Manifest{
		Targets: []string{"opencode"},
	}
	err := m.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one skill")
}

func TestValidate_InvalidTarget(t *testing.T) {
	m := &Manifest{
		Skills:  []SkillRef{{Name: "x"}},
		Targets: []string{"notepad"},
	}
	err := m.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid target")
}

func TestValidate_Valid(t *testing.T) {
	m := &Manifest{
		Skills:  []SkillRef{{Name: "x"}},
		Targets: []string{"opencode", "cursor"},
	}
	err := m.Validate()
	require.NoError(t, err)
}

func TestSaveManifest(t *testing.T) {
	m := &Manifest{
		Registries:   []RegistryRef{{Name: "r", URL: "https://r"}},
		Skills:       []SkillRef{{Name: "s", Version: "v1", Path: "./p"}},
		Instructions: []string{"do the thing"},
		Targets:      []string{"vscode-copilot"},
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "vibes.yaml")

	require.NoError(t, SaveManifest(m, p))

	m2, err := LoadManifest(p)
	require.NoError(t, err)
	require.NotNil(t, m2)

	assert.Equal(t, m.Registries[0].Name, m2.Registries[0].Name)
	assert.Equal(t, m.Skills[0].Name, m2.Skills[0].Name)
	assert.Equal(t, m.Instructions[0], m2.Instructions[0])
	assert.Equal(t, m.Targets[0], m2.Targets[0])

	// ensure file exists
	_, err = os.Stat(p)
	require.NoError(t, err)
}
