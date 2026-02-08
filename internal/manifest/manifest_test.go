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

// --- LoadManifestFromProject tests ---

func TestLoadManifestFromProject_PrefersVibesYml(t *testing.T) {
	dir := t.TempDir()

	// Create both vibes.yml and vibes.yaml with different content
	ymlContent := `skills:
  - name: from-yml
targets:
  - opencode
`
	yamlContent := `skills:
  - name: from-yaml
targets:
  - cursor
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "vibes.yml"), []byte(ymlContent), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "vibes.yaml"), []byte(yamlContent), 0o644))

	m, path, err := LoadManifestFromProject(dir)
	require.NoError(t, err)
	require.NotNil(t, m)

	// Should prefer vibes.yml
	assert.Equal(t, "from-yml", m.Skills[0].Name)
	assert.Equal(t, filepath.Join(dir, "vibes.yml"), path)
}

func TestLoadManifestFromProject_FallsBackToVibesYaml(t *testing.T) {
	dir := t.TempDir()

	yamlContent := `skills:
  - name: from-yaml
targets:
  - cursor
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "vibes.yaml"), []byte(yamlContent), 0o644))

	m, path, err := LoadManifestFromProject(dir)
	require.NoError(t, err)
	require.NotNil(t, m)

	assert.Equal(t, "from-yaml", m.Skills[0].Name)
	assert.Equal(t, filepath.Join(dir, "vibes.yaml"), path)
}

func TestLoadManifestFromProject_NoManifestReturnsError(t *testing.T) {
	dir := t.TempDir()

	_, _, err := LoadManifestFromProject(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no manifest found")
}

// --- SaveManifestWithComments tests ---

func TestSaveManifestWithComments_WritesHeaderAndValidYAML(t *testing.T) {
	m := &Manifest{
		Skills:  []SkillRef{{Name: "test-skill"}},
		Targets: []string{"opencode"},
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "vibes.yml")

	header := "# vibes.yml - Project configuration\n# See docs for details\n"

	require.NoError(t, SaveManifestWithComments(m, p, header))

	// Read raw file content
	data, err := os.ReadFile(p)
	require.NoError(t, err)
	content := string(data)

	// File should start with the comment header
	assert.True(t, len(content) > len(header))
	assert.Equal(t, header, content[:len(header)])

	// Should still be loadable as valid YAML
	m2, err := LoadManifest(p)
	require.NoError(t, err)
	assert.Equal(t, "test-skill", m2.Skills[0].Name)
	assert.Equal(t, "opencode", m2.Targets[0])
}

func TestSaveManifestWithComments_EmptyHeaderStillWorks(t *testing.T) {
	m := &Manifest{
		Skills:  []SkillRef{{Name: "s"}},
		Targets: []string{"cursor"},
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "vibes.yml")

	require.NoError(t, SaveManifestWithComments(m, p, ""))

	m2, err := LoadManifest(p)
	require.NoError(t, err)
	assert.Equal(t, "s", m2.Skills[0].Name)
}

// --- Merged manifest tests ---

func TestLoadMergedManifest_ProjectOnly(t *testing.T) {
	projectDir := t.TempDir()
	globalDir := t.TempDir() // empty, no global config

	content := `skills:
  - name: project-skill
targets:
  - opencode
`
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "vibes.yml"), []byte(content), 0o644))

	m, err := LoadMergedManifest(projectDir, filepath.Join(globalDir, "vibes.yml"))
	require.NoError(t, err)
	require.NotNil(t, m)

	assert.Len(t, m.Skills, 1)
	assert.Equal(t, "project-skill", m.Skills[0].Name)
}

func TestLoadMergedManifest_GlobalOnly(t *testing.T) {
	projectDir := t.TempDir() // empty, no project config
	globalDir := t.TempDir()

	content := `registries:
  - name: global-reg
    url: https://example.com/global
skills:
  - name: global-skill
targets:
  - cursor
instructions:
  - "global instruction"
`
	globalPath := filepath.Join(globalDir, "vibes.yml")
	require.NoError(t, os.WriteFile(globalPath, []byte(content), 0o644))

	m, err := LoadMergedManifest(projectDir, globalPath)
	require.NoError(t, err)
	require.NotNil(t, m)

	assert.Len(t, m.Skills, 1)
	assert.Equal(t, "global-skill", m.Skills[0].Name)
	assert.Len(t, m.Registries, 1)
	assert.Equal(t, "global-reg", m.Registries[0].Name)
	assert.Equal(t, []string{"cursor"}, m.Targets)
	assert.Equal(t, []string{"global instruction"}, m.Instructions)
}

func TestLoadMergedManifest_MergesRegistriesByName(t *testing.T) {
	projectDir := t.TempDir()
	globalDir := t.TempDir()

	globalContent := `registries:
  - name: shared-reg
    url: https://global.example.com
  - name: global-only
    url: https://global-only.example.com
skills:
  - name: placeholder
targets:
  - opencode
`
	projectContent := `registries:
  - name: shared-reg
    url: https://project.example.com
  - name: project-only
    url: https://project-only.example.com
skills:
  - name: placeholder
targets:
  - opencode
`
	globalPath := filepath.Join(globalDir, "vibes.yml")
	require.NoError(t, os.WriteFile(globalPath, []byte(globalContent), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "vibes.yml"), []byte(projectContent), 0o644))

	m, err := LoadMergedManifest(projectDir, globalPath)
	require.NoError(t, err)

	// Should have 3 registries: shared-reg (project wins), global-only, project-only
	assert.Len(t, m.Registries, 3)

	regMap := map[string]string{}
	for _, r := range m.Registries {
		regMap[r.Name] = r.URL
	}
	// shared-reg should use project URL (project overrides)
	assert.Equal(t, "https://project.example.com", regMap["shared-reg"])
	assert.Equal(t, "https://global-only.example.com", regMap["global-only"])
	assert.Equal(t, "https://project-only.example.com", regMap["project-only"])
}

func TestLoadMergedManifest_DeduplicatesSkillsByName(t *testing.T) {
	projectDir := t.TempDir()
	globalDir := t.TempDir()

	globalContent := `skills:
  - name: shared-skill
    version: "1.0"
  - name: global-only-skill
targets:
  - opencode
`
	projectContent := `skills:
  - name: shared-skill
    path: ./local/shared
  - name: project-only-skill
targets:
  - cursor
`
	globalPath := filepath.Join(globalDir, "vibes.yml")
	require.NoError(t, os.WriteFile(globalPath, []byte(globalContent), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "vibes.yml"), []byte(projectContent), 0o644))

	m, err := LoadMergedManifest(projectDir, globalPath)
	require.NoError(t, err)

	assert.Len(t, m.Skills, 3)

	skillMap := map[string]SkillRef{}
	for _, s := range m.Skills {
		skillMap[s.Name] = s
	}
	// shared-skill should use project version (project overrides)
	assert.Equal(t, "./local/shared", skillMap["shared-skill"].Path)
	assert.Contains(t, skillMap, "global-only-skill")
	assert.Contains(t, skillMap, "project-only-skill")
}

func TestLoadMergedManifest_ProjectTargetsOverride(t *testing.T) {
	projectDir := t.TempDir()
	globalDir := t.TempDir()

	globalContent := `skills:
  - name: s
targets:
  - opencode
  - cursor
`
	projectContent := `skills:
  - name: s
targets:
  - vscode-copilot
`
	globalPath := filepath.Join(globalDir, "vibes.yml")
	require.NoError(t, os.WriteFile(globalPath, []byte(globalContent), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "vibes.yml"), []byte(projectContent), 0o644))

	m, err := LoadMergedManifest(projectDir, globalPath)
	require.NoError(t, err)

	// Project targets override global
	assert.Equal(t, []string{"vscode-copilot"}, m.Targets)
}

func TestLoadMergedManifest_InstructionsConcatenated(t *testing.T) {
	projectDir := t.TempDir()
	globalDir := t.TempDir()

	globalContent := `skills:
  - name: s
targets:
  - opencode
instructions:
  - "global first"
  - "global second"
`
	projectContent := `skills:
  - name: s
targets:
  - opencode
instructions:
  - "project first"
`
	globalPath := filepath.Join(globalDir, "vibes.yml")
	require.NoError(t, os.WriteFile(globalPath, []byte(globalContent), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "vibes.yml"), []byte(projectContent), 0o644))

	m, err := LoadMergedManifest(projectDir, globalPath)
	require.NoError(t, err)

	// Instructions concatenated: global first, then project
	assert.Equal(t, []string{"global first", "global second", "project first"}, m.Instructions)
}

func TestLoadMergedManifest_NeitherExists(t *testing.T) {
	projectDir := t.TempDir()
	globalPath := filepath.Join(t.TempDir(), "vibes.yml")

	_, err := LoadMergedManifest(projectDir, globalPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no manifest found")
}
