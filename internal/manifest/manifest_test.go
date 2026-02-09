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
    ref: latest

skills:
  - name: conventional-commits
  - name: react-expert
    version: "1.0"
  - name: my-custom-skill
    path: ./local-skills/my-custom-skill

instructions:
  - name: typescript-preference
    content: "Always use TypeScript for frontend code"
  - name: functional-components
    content: "Prefer functional components over class components"
    apply_to: "**/*.tsx"

agents:
  - name: reviewer
    registry: awesome-copilot
  - name: local-agent
    path: ./agents/local-agent.md

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
	assert.Equal(t, "typescript-preference", m.Instructions[0].Name)
	assert.Equal(t, "Always use TypeScript for frontend code", m.Instructions[0].Content)
	assert.Empty(t, m.Instructions[0].ApplyTo)
	assert.Equal(t, "functional-components", m.Instructions[1].Name)
	assert.Equal(t, "**/*.tsx", m.Instructions[1].ApplyTo)

	assert.Len(t, m.Agents, 2)
	assert.Equal(t, "reviewer", m.Agents[0].Name)
	assert.Equal(t, "awesome-copilot", m.Agents[0].Registry)
	assert.Equal(t, "local-agent", m.Agents[1].Name)
	assert.Equal(t, "./agents/local-agent.md", m.Agents[1].Path)

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

func TestValidate_RegistryMissingRef(t *testing.T) {
	m := &Manifest{
		Registries: []RegistryRef{{Name: "r", URL: "https://example.com"}},
		Skills:     []SkillRef{{Name: "x"}},
		Targets:    []string{"opencode"},
	}
	err := m.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ref")
}

func TestValidate_RegistryWithRef(t *testing.T) {
	m := &Manifest{
		Registries: []RegistryRef{{Name: "r", URL: "https://example.com", Ref: "latest"}},
		Skills:     []SkillRef{{Name: "x"}},
		Targets:    []string{"opencode"},
	}
	err := m.Validate()
	require.NoError(t, err)
}

func TestLoadManifest_RefField(t *testing.T) {
	yamlStr := `registries:
  - name: pinned
    url: https://example.com/repo
    ref: v1.2.0
skills:
  - name: s
targets:
  - opencode
`
	m, err := LoadManifestFromBytes([]byte(yamlStr))
	require.NoError(t, err)
	assert.Equal(t, "v1.2.0", m.Registries[0].Ref)
}

func TestSaveManifest_RefFieldRoundTrip(t *testing.T) {
	m := &Manifest{
		Registries: []RegistryRef{{Name: "r", URL: "https://r", Ref: "abc123"}},
		Skills:     []SkillRef{{Name: "s"}},
		Targets:    []string{"opencode"},
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "vibes.yml")

	require.NoError(t, SaveManifest(m, p))

	m2, err := LoadManifest(p)
	require.NoError(t, err)
	assert.Equal(t, "abc123", m2.Registries[0].Ref)
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
		Registries:   []RegistryRef{{Name: "r", URL: "https://r", Ref: "latest"}},
		Skills:       []SkillRef{{Name: "s", Version: "v1", Path: "./p"}},
		Instructions: []InstructionRef{{Name: "inst", Content: "do the thing"}},
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
	assert.Equal(t, m.Instructions[0].Name, m2.Instructions[0].Name)
	assert.Equal(t, m.Instructions[0].Content, m2.Instructions[0].Content)
	assert.Equal(t, m.Targets[0], m2.Targets[0])

	// ensure file exists
	_, err = os.Stat(p)
	require.NoError(t, err)
}

// --- LoadManifestFromProject tests ---

func TestManifestFilenames_YamlIsPreferred(t *testing.T) {
	// vibes.yaml should be the first (preferred) filename; vibes.yml is legacy
	require.Len(t, ManifestFilenames, 2)
	assert.Equal(t, "vibes.yaml", ManifestFilenames[0], "vibes.yaml should be preferred")
	assert.Equal(t, "vibes.yml", ManifestFilenames[1], "vibes.yml should be legacy fallback")
}

func TestLoadManifestFromProject_PrefersVibesYaml(t *testing.T) {
	dir := t.TempDir()

	// Create both vibes.yaml and vibes.yml with different content
	yamlContent := `skills:
  - name: from-yaml
targets:
  - cursor
`
	ymlContent := `skills:
  - name: from-yml
targets:
  - opencode
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "vibes.yaml"), []byte(yamlContent), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "vibes.yml"), []byte(ymlContent), 0o644))

	m, path, err := LoadManifestFromProject(dir)
	require.NoError(t, err)
	require.NotNil(t, m)

	// Should prefer vibes.yaml (canonical); vibes.yml is legacy fallback
	assert.Equal(t, "from-yaml", m.Skills[0].Name)
	assert.Equal(t, filepath.Join(dir, "vibes.yaml"), path)
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
    ref: latest
skills:
  - name: global-skill
targets:
  - cursor
instructions:
  - name: global-inst
    content: "global instruction"
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
	require.Len(t, m.Instructions, 1)
	assert.Equal(t, "global-inst", m.Instructions[0].Name)
	assert.Equal(t, "global instruction", m.Instructions[0].Content)
}

func TestLoadMergedManifest_MergesRegistriesByName(t *testing.T) {
	projectDir := t.TempDir()
	globalDir := t.TempDir()

	globalContent := `registries:
  - name: shared-reg
    url: https://global.example.com
    ref: latest
  - name: global-only
    url: https://global-only.example.com
    ref: latest
skills:
  - name: placeholder
targets:
  - opencode
`
	projectContent := `registries:
  - name: shared-reg
    url: https://project.example.com
    ref: latest
  - name: project-only
    url: https://project-only.example.com
    ref: latest
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

func TestLoadMergedManifest_InstructionsMergedByName(t *testing.T) {
	projectDir := t.TempDir()
	globalDir := t.TempDir()

	globalContent := `skills:
  - name: s
targets:
  - opencode
instructions:
  - name: global-first
    content: "global first content"
  - name: shared-inst
    content: "global version"
`
	projectContent := `skills:
  - name: s
targets:
  - opencode
instructions:
  - name: shared-inst
    content: "project version"
  - name: project-only
    content: "project only content"
`
	globalPath := filepath.Join(globalDir, "vibes.yml")
	require.NoError(t, os.WriteFile(globalPath, []byte(globalContent), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "vibes.yml"), []byte(projectContent), 0o644))

	m, err := LoadMergedManifest(projectDir, globalPath)
	require.NoError(t, err)

	// Should have 3 instructions: global-first, shared-inst (project wins), project-only
	require.Len(t, m.Instructions, 3)

	instMap := map[string]InstructionRef{}
	for _, inst := range m.Instructions {
		instMap[inst.Name] = inst
	}
	// shared-inst should use project content (project overrides)
	assert.Equal(t, "project version", instMap["shared-inst"].Content)
	assert.Equal(t, "global first content", instMap["global-first"].Content)
	assert.Equal(t, "project only content", instMap["project-only"].Content)

	// Order: global-first, shared-inst, project-only
	assert.Equal(t, "global-first", m.Instructions[0].Name)
	assert.Equal(t, "shared-inst", m.Instructions[1].Name)
	assert.Equal(t, "project-only", m.Instructions[2].Name)
}

func TestLoadMergedManifest_AgentsMergedByName(t *testing.T) {
	projectDir := t.TempDir()
	globalDir := t.TempDir()

	globalContent := `skills:
  - name: s
targets:
  - opencode
agents:
  - name: shared-agent
    registry: global-reg
  - name: global-only-agent
    registry: global-reg
`
	projectContent := `skills:
  - name: s
targets:
  - opencode
agents:
  - name: shared-agent
    path: ./local-agent.md
  - name: project-only-agent
    path: ./project-agent.md
`
	globalPath := filepath.Join(globalDir, "vibes.yml")
	require.NoError(t, os.WriteFile(globalPath, []byte(globalContent), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "vibes.yml"), []byte(projectContent), 0o644))

	m, err := LoadMergedManifest(projectDir, globalPath)
	require.NoError(t, err)

	// Should have 3 agents: shared-agent (project wins), global-only-agent, project-only-agent
	require.Len(t, m.Agents, 3)

	agentMap := map[string]AgentRef{}
	for _, a := range m.Agents {
		agentMap[a.Name] = a
	}
	// shared-agent should use project (path overrides registry)
	assert.Equal(t, "./local-agent.md", agentMap["shared-agent"].Path)
	assert.Empty(t, agentMap["shared-agent"].Registry)
	assert.Equal(t, "global-reg", agentMap["global-only-agent"].Registry)
	assert.Equal(t, "./project-agent.md", agentMap["project-only-agent"].Path)
}

func TestLoadMergedManifest_NeitherExists(t *testing.T) {
	projectDir := t.TempDir()
	globalPath := filepath.Join(t.TempDir(), "vibes.yml")

	_, err := LoadMergedManifest(projectDir, globalPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no manifest found")
}

// --- InstructionRef validation tests ---

func TestValidate_InstructionRef_Valid(t *testing.T) {
	m := &Manifest{
		Skills:       []SkillRef{{Name: "x"}},
		Targets:      []string{"opencode"},
		Instructions: []InstructionRef{{Name: "inst", Content: "do stuff"}},
	}
	require.NoError(t, m.Validate())
}

func TestValidate_InstructionRef_ValidWithPath(t *testing.T) {
	m := &Manifest{
		Skills:       []SkillRef{{Name: "x"}},
		Targets:      []string{"opencode"},
		Instructions: []InstructionRef{{Name: "inst", Path: "./instructions/foo.md"}},
	}
	require.NoError(t, m.Validate())
}

func TestValidate_InstructionRef_MissingName(t *testing.T) {
	m := &Manifest{
		Skills:       []SkillRef{{Name: "x"}},
		Targets:      []string{"opencode"},
		Instructions: []InstructionRef{{Content: "do stuff"}},
	}
	err := m.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name")
}

func TestValidate_InstructionRef_ContentAndPath(t *testing.T) {
	m := &Manifest{
		Skills:       []SkillRef{{Name: "x"}},
		Targets:      []string{"opencode"},
		Instructions: []InstructionRef{{Name: "inst", Content: "stuff", Path: "./foo.md"}},
	}
	err := m.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content")
}

func TestValidate_InstructionRef_NeitherContentNorPath(t *testing.T) {
	m := &Manifest{
		Skills:       []SkillRef{{Name: "x"}},
		Targets:      []string{"opencode"},
		Instructions: []InstructionRef{{Name: "inst"}},
	}
	err := m.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content")
}

// --- AgentRef validation tests ---

func TestValidate_AgentRef_ValidWithRegistry(t *testing.T) {
	m := &Manifest{
		Skills:  []SkillRef{{Name: "x"}},
		Targets: []string{"opencode"},
		Agents:  []AgentRef{{Name: "agent", Registry: "my-reg"}},
	}
	require.NoError(t, m.Validate())
}

func TestValidate_AgentRef_ValidWithPath(t *testing.T) {
	m := &Manifest{
		Skills:  []SkillRef{{Name: "x"}},
		Targets: []string{"opencode"},
		Agents:  []AgentRef{{Name: "agent", Path: "./agents/foo.md"}},
	}
	require.NoError(t, m.Validate())
}

func TestValidate_AgentRef_MissingName(t *testing.T) {
	m := &Manifest{
		Skills:  []SkillRef{{Name: "x"}},
		Targets: []string{"opencode"},
		Agents:  []AgentRef{{Registry: "my-reg"}},
	}
	err := m.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name")
}

func TestValidate_AgentRef_PathAndRegistry(t *testing.T) {
	m := &Manifest{
		Skills:  []SkillRef{{Name: "x"}},
		Targets: []string{"opencode"},
		Agents:  []AgentRef{{Name: "agent", Path: "./foo.md", Registry: "reg"}},
	}
	err := m.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "path")
}

func TestValidate_AgentRef_NeitherPathNorRegistry(t *testing.T) {
	m := &Manifest{
		Skills:  []SkillRef{{Name: "x"}},
		Targets: []string{"opencode"},
		Agents:  []AgentRef{{Name: "agent"}},
	}
	err := m.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "path")
}

// --- InstructionRef/AgentRef YAML parsing tests ---

func TestLoadManifest_InstructionRef(t *testing.T) {
	yamlStr := `skills:
  - name: s
targets:
  - opencode
instructions:
  - name: ts-pref
    content: "Use TypeScript"
    apply_to: "**/*.ts"
  - name: from-file
    path: ./instructions/rules.md
`
	m, err := LoadManifestFromBytes([]byte(yamlStr))
	require.NoError(t, err)

	require.Len(t, m.Instructions, 2)
	assert.Equal(t, "ts-pref", m.Instructions[0].Name)
	assert.Equal(t, "Use TypeScript", m.Instructions[0].Content)
	assert.Equal(t, "**/*.ts", m.Instructions[0].ApplyTo)
	assert.Equal(t, "from-file", m.Instructions[1].Name)
	assert.Equal(t, "./instructions/rules.md", m.Instructions[1].Path)
	assert.Empty(t, m.Instructions[1].Content)
}

func TestLoadManifest_AgentRef(t *testing.T) {
	yamlStr := `skills:
  - name: s
targets:
  - opencode
agents:
  - name: reviewer
    registry: awesome-copilot
  - name: local-bot
    path: ./agents/bot.md
`
	m, err := LoadManifestFromBytes([]byte(yamlStr))
	require.NoError(t, err)

	require.Len(t, m.Agents, 2)
	assert.Equal(t, "reviewer", m.Agents[0].Name)
	assert.Equal(t, "awesome-copilot", m.Agents[0].Registry)
	assert.Empty(t, m.Agents[0].Path)
	assert.Equal(t, "local-bot", m.Agents[1].Name)
	assert.Equal(t, "./agents/bot.md", m.Agents[1].Path)
	assert.Empty(t, m.Agents[1].Registry)
}
