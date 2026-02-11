package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- formatPaths tests ---

func TestFormatPaths_BothExist(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "global", "vibes.yml")
	localDir := filepath.Join(dir, "project")

	require.NoError(t, os.MkdirAll(filepath.Dir(globalPath), 0o755))
	require.NoError(t, os.WriteFile(globalPath, []byte("skills: []"), 0o644))
	require.NoError(t, os.MkdirAll(localDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(localDir, "vibes.yml"), []byte("skills: []"), 0o644))

	out := formatPaths(globalPath, localDir, filepath.Join(dir, "cache"))
	assert.Contains(t, out, globalPath)
	assert.Contains(t, out, "[found]")
	assert.Contains(t, out, localDir)
	assert.Contains(t, out, filepath.Join(dir, "cache"))
}

func TestFormatPaths_NeitherExists(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "nope", "vibes.yml")
	localDir := filepath.Join(dir, "noproject")
	require.NoError(t, os.MkdirAll(localDir, 0o755))

	out := formatPaths(globalPath, localDir, filepath.Join(dir, "cache"))
	assert.Contains(t, out, "[not found]")
}

func TestFormatPaths_LegacyYmlFound(t *testing.T) {
	dir := t.TempDir()
	localDir := filepath.Join(dir, "project")
	require.NoError(t, os.MkdirAll(localDir, 0o755))
	// vibes.yml is now the legacy name
	require.NoError(t, os.WriteFile(filepath.Join(localDir, "vibes.yml"), []byte("skills: []"), 0o644))

	globalPath := filepath.Join(dir, "nope", "vibes.yaml")
	out := formatPaths(globalPath, localDir, filepath.Join(dir, "cache"))
	assert.Contains(t, out, "vibes.yml")
	assert.Contains(t, out, "legacy")
}

// --- renderMergedYAML tests ---

func TestRenderMergedYAML_ContainsAllSections(t *testing.T) {
	m := &manifest.Manifest{
		Registries: []manifest.RegistryRef{
			{Name: "test-reg", URL: "https://example.com/reg"},
		},
		Skills:  []manifest.SkillRef{{Name: "skill-a"}, {Name: "skill-b"}},
		Targets: []string{"opencode", "cursor"},
	}
	out := renderMergedYAML(m)
	assert.Contains(t, out, "registries:")
	assert.Contains(t, out, "test-reg")
	assert.Contains(t, out, "skills:")
	assert.Contains(t, out, "skill-a")
	assert.Contains(t, out, "skill-b")
	assert.Contains(t, out, "targets:")
	assert.Contains(t, out, "opencode")
	assert.Contains(t, out, "cursor")
}

func TestRenderMergedYAML_IncludesInstructions(t *testing.T) {
	m := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "s"}},
		Targets: []string{"opencode"},
		Instructions: []manifest.InstructionRef{
			{Name: "go-modules", Content: "Use Go modules"},
		},
	}
	out := renderMergedYAML(m)
	assert.Contains(t, out, "instructions:")
	assert.Contains(t, out, "go-modules")
	assert.Contains(t, out, "Use Go modules")
}

func TestRenderMergedYAML_IncludesAgents(t *testing.T) {
	m := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "s"}},
		Targets: []string{"opencode"},
		Agents: []manifest.AgentRef{
			{Name: "reviewer", Path: "./agents/reviewer.md"},
		},
	}
	out := renderMergedYAML(m)
	assert.Contains(t, out, "agents:")
	assert.Contains(t, out, "reviewer")
}

// --- annotateManifest tests ---

func TestAnnotateManifest_SkillSources(t *testing.T) {
	global := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "global-skill"}},
		Targets: []string{"opencode"},
	}
	local := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "local-skill"}},
		Targets: []string{"cursor"},
	}
	merged := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "global-skill"}, {Name: "local-skill"}},
		Targets: []string{"cursor"},
	}

	out := annotateManifest(global, local, merged)
	assert.Contains(t, out, "global-skill")
	assert.Contains(t, out, "[global]")
	assert.Contains(t, out, "local-skill")
	assert.Contains(t, out, "[local]")
}

func TestAnnotateManifest_OverriddenSkill(t *testing.T) {
	global := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "shared", Path: "/old"}},
		Targets: []string{"opencode"},
	}
	local := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "shared", Path: "/new"}},
		Targets: []string{"opencode"},
	}
	merged := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "shared", Path: "/new"}},
		Targets: []string{"opencode"},
	}

	out := annotateManifest(global, local, merged)
	assert.Contains(t, out, "shared")
	assert.Contains(t, out, "[local, overrides global]")
}

func TestAnnotateManifest_TargetSource(t *testing.T) {
	global := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "s"}},
		Targets: []string{"opencode"},
	}
	local := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "s"}},
		Targets: []string{"cursor", "opencode"},
	}
	merged := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "s"}},
		Targets: []string{"cursor", "opencode"},
	}

	out := annotateManifest(global, local, merged)
	assert.Contains(t, out, "targets:")
	assert.Contains(t, out, "[local]")
}

func TestAnnotateManifest_GlobalOnly(t *testing.T) {
	global := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "s"}},
		Targets: []string{"opencode"},
	}
	out := annotateManifest(global, nil, global)
	assert.Contains(t, out, "[global]")
	assert.NotContains(t, out, "[local]")
}

func TestAnnotateManifest_LocalOnly(t *testing.T) {
	local := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "s"}},
		Targets: []string{"opencode"},
	}
	out := annotateManifest(nil, local, local)
	assert.Contains(t, out, "[local]")
	assert.NotContains(t, out, "[global]")
}

func TestAnnotateManifestWithOptions_RelativePathsForLocal(t *testing.T) {
	projectDir := t.TempDir()
	local := &manifest.Manifest{
		Skills: []manifest.SkillRef{{
			Name: "local-skill",
			Path: filepath.Join(projectDir, "skills", "local-skill"),
		}},
		Targets: []string{"opencode"},
	}

	out := annotateManifestWithOptions(nil, local, local, annotateRenderOptions{
		RelativePaths: true,
		ProjectDir:    projectDir,
	})

	assert.Contains(t, out, "path: ./skills/local-skill")
}

func TestAnnotateManifestWithOptions_RelativePathsForGlobal(t *testing.T) {
	globalDir := t.TempDir()
	globalPath := filepath.Join(globalDir, "vibes.yaml")
	global := &manifest.Manifest{
		Skills: []manifest.SkillRef{{
			Name: "global-skill",
			Path: filepath.Join(globalDir, "skills", "global-skill"),
		}},
		Targets: []string{"opencode"},
	}

	out := annotateManifestWithOptions(global, nil, global, annotateRenderOptions{
		RelativePaths: true,
		GlobalPath:    globalPath,
	})

	assert.Contains(t, out, "path: ./skills/global-skill")
}

func TestAnnotateManifest_InstructionSources(t *testing.T) {
	global := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "s"}},
		Targets: []string{"opencode"},
		Instructions: []manifest.InstructionRef{
			{Name: "global-inst", Content: "global instruction"},
		},
	}
	local := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "s"}},
		Targets: []string{"opencode"},
		Instructions: []manifest.InstructionRef{
			{Name: "local-inst", Content: "local instruction"},
		},
	}
	merged := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "s"}},
		Targets: []string{"opencode"},
		Instructions: []manifest.InstructionRef{
			{Name: "global-inst", Content: "global instruction"},
			{Name: "local-inst", Content: "local instruction"},
		},
	}

	out := annotateManifest(global, local, merged)
	assert.Contains(t, out, "global-inst")
	assert.Contains(t, out, "[global]")
	assert.Contains(t, out, "local-inst")
	assert.Contains(t, out, "[local]")
	assert.Contains(t, out, "content:")
}

func TestAnnotateManifest_InstructionOverride(t *testing.T) {
	global := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "s"}},
		Targets: []string{"opencode"},
		Instructions: []manifest.InstructionRef{
			{Name: "shared-inst", Content: "old content"},
		},
	}
	local := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "s"}},
		Targets: []string{"opencode"},
		Instructions: []manifest.InstructionRef{
			{Name: "shared-inst", Content: "new content"},
		},
	}
	merged := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "s"}},
		Targets: []string{"opencode"},
		Instructions: []manifest.InstructionRef{
			{Name: "shared-inst", Content: "new content"},
		},
	}

	out := annotateManifest(global, local, merged)
	assert.Contains(t, out, "shared-inst")
	assert.Contains(t, out, "[local, overrides global]")
}

func TestAnnotateManifest_InstructionWithPath(t *testing.T) {
	local := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "s"}},
		Targets: []string{"opencode"},
		Instructions: []manifest.InstructionRef{
			{Name: "path-inst", Path: "./instructions/coding.md"},
		},
	}

	out := annotateManifest(nil, local, local)
	assert.Contains(t, out, "path-inst")
	assert.Contains(t, out, "path:")
	assert.Contains(t, out, "./instructions/coding.md")
}

func TestAnnotateManifest_InstructionWithApplyTo(t *testing.T) {
	local := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "s"}},
		Targets: []string{"opencode"},
		Instructions: []manifest.InstructionRef{
			{Name: "ts-inst", Content: "Use TypeScript", ApplyTo: "opencode"},
		},
	}

	out := annotateManifest(nil, local, local)
	assert.Contains(t, out, "ts-inst")
	assert.Contains(t, out, "apply_to:")
}

func TestAnnotateManifest_AgentSources(t *testing.T) {
	global := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "s"}},
		Targets: []string{"opencode"},
		Agents: []manifest.AgentRef{
			{Name: "global-agent", Path: "./agents/global.md"},
		},
	}
	local := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "s"}},
		Targets: []string{"opencode"},
		Agents: []manifest.AgentRef{
			{Name: "local-agent", Registry: "my-registry/agent"},
		},
	}
	merged := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "s"}},
		Targets: []string{"opencode"},
		Agents: []manifest.AgentRef{
			{Name: "global-agent", Path: "./agents/global.md"},
			{Name: "local-agent", Registry: "my-registry/agent"},
		},
	}

	out := annotateManifest(global, local, merged)
	assert.Contains(t, out, "agents:")
	assert.Contains(t, out, "global-agent")
	assert.Contains(t, out, "[global]")
	assert.Contains(t, out, "local-agent")
	assert.Contains(t, out, "[local]")
}

func TestAnnotateManifest_AgentOverride(t *testing.T) {
	global := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "s"}},
		Targets: []string{"opencode"},
		Agents: []manifest.AgentRef{
			{Name: "shared-agent", Path: "/old/path"},
		},
	}
	local := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "s"}},
		Targets: []string{"opencode"},
		Agents: []manifest.AgentRef{
			{Name: "shared-agent", Path: "/new/path"},
		},
	}
	merged := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "s"}},
		Targets: []string{"opencode"},
		Agents: []manifest.AgentRef{
			{Name: "shared-agent", Path: "/new/path"},
		},
	}

	out := annotateManifest(global, local, merged)
	assert.Contains(t, out, "shared-agent")
	assert.Contains(t, out, "[local, overrides global]")
}

// --- validateConfig tests ---

func TestValidateConfig_AllValid(t *testing.T) {
	m := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "conventional-commits"}, {Name: "code-review"}},
		Targets: []string{"opencode", "cursor"},
	}
	embeddedSkills := []string{"conventional-commits", "code-review"}

	result := validateConfig(m, embeddedSkills)
	assert.True(t, result.ok())
	assert.Empty(t, result.problems)
}

func TestValidateConfig_InvalidTarget(t *testing.T) {
	m := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "conventional-commits"}},
		Targets: []string{"opencode", "vim-copilot"},
	}
	embeddedSkills := []string{"conventional-commits"}

	result := validateConfig(m, embeddedSkills)
	assert.False(t, result.ok())
	found := false
	for _, p := range result.problems {
		if p.field == "vim-copilot" {
			found = true
			assert.Contains(t, p.message, "invalid target")
		}
	}
	assert.True(t, found, "should report invalid target")
}

func TestValidateConfig_UnresolvableSkill(t *testing.T) {
	m := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "nonexistent-skill"}},
		Targets: []string{"opencode"},
	}
	embeddedSkills := []string{"conventional-commits", "code-review"}

	result := validateConfig(m, embeddedSkills)
	assert.False(t, result.ok())
	found := false
	for _, p := range result.problems {
		if p.field == "nonexistent-skill" {
			found = true
			assert.Contains(t, p.message, "not found")
		}
	}
	assert.True(t, found, "should report unresolvable skill")
}

func TestValidateConfig_LocalPathSkill_Exists(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "my-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))

	m := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "my-skill", Path: skillDir}},
		Targets: []string{"opencode"},
	}

	result := validateConfig(m, nil)
	assert.True(t, result.ok())
}

func TestValidateConfig_LocalPathSkill_Missing(t *testing.T) {
	m := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "my-skill", Path: "/nonexistent/path"}},
		Targets: []string{"opencode"},
	}

	result := validateConfig(m, nil)
	assert.False(t, result.ok())
	found := false
	for _, p := range result.problems {
		if p.field == "my-skill" {
			found = true
			assert.Contains(t, p.message, "path not found")
		}
	}
	assert.True(t, found, "should report missing local path")
}

func TestValidateConfig_NoSkills(t *testing.T) {
	m := &manifest.Manifest{
		Skills:  nil,
		Targets: []string{"opencode"},
	}

	result := validateConfig(m, nil)
	assert.False(t, result.ok())
}

func TestValidateConfig_NoTargets(t *testing.T) {
	m := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "conventional-commits"}},
		Targets: nil,
	}
	embeddedSkills := []string{"conventional-commits"}

	result := validateConfig(m, embeddedSkills)
	assert.False(t, result.ok())
}

func TestValidateConfig_InstructionPathExists(t *testing.T) {
	dir := t.TempDir()
	instFile := filepath.Join(dir, "instructions.md")
	require.NoError(t, os.WriteFile(instFile, []byte("# Instructions"), 0o644))

	m := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "s"}},
		Targets: []string{"opencode"},
		Instructions: []manifest.InstructionRef{
			{Name: "my-inst", Path: instFile},
		},
	}

	result := validateConfig(m, []string{"s"})
	// Should not report a problem for instruction path
	for _, p := range result.problems {
		assert.NotEqual(t, "my-inst", p.field, "should not report problem for existing instruction path")
	}
}

func TestValidateConfig_InstructionPathMissing(t *testing.T) {
	m := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "s"}},
		Targets: []string{"opencode"},
		Instructions: []manifest.InstructionRef{
			{Name: "bad-inst", Path: "/nonexistent/instructions.md"},
		},
	}

	result := validateConfig(m, []string{"s"})
	assert.False(t, result.ok())
	found := false
	for _, p := range result.problems {
		if p.field == "bad-inst" {
			found = true
			assert.Contains(t, p.message, "path not found")
		}
	}
	assert.True(t, found, "should report missing instruction path")
}

func TestValidateConfig_AgentPathExists(t *testing.T) {
	dir := t.TempDir()
	agentFile := filepath.Join(dir, "agent.md")
	require.NoError(t, os.WriteFile(agentFile, []byte("# Agent"), 0o644))

	m := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "s"}},
		Targets: []string{"opencode"},
		Agents: []manifest.AgentRef{
			{Name: "my-agent", Path: agentFile},
		},
	}

	result := validateConfig(m, []string{"s"})
	for _, p := range result.problems {
		assert.NotEqual(t, "my-agent", p.field, "should not report problem for existing agent path")
	}
}

func TestValidateConfig_AgentPathMissing(t *testing.T) {
	m := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "s"}},
		Targets: []string{"opencode"},
		Agents: []manifest.AgentRef{
			{Name: "bad-agent", Path: "/nonexistent/agent.md"},
		},
	}

	result := validateConfig(m, []string{"s"})
	assert.False(t, result.ok())
	found := false
	for _, p := range result.problems {
		if p.field == "bad-agent" {
			found = true
			assert.Contains(t, p.message, "path not found")
		}
	}
	assert.True(t, found, "should report missing agent path")
}

// --- Context-aware validation: global-only should not flag missing skills/targets ---

func TestValidateConfig_GlobalOnly_NoSkillsOrTargets_IsOK(t *testing.T) {
	// A global-only config with just registries and no skills/targets is valid.
	m := &manifest.Manifest{
		Registries: []manifest.RegistryRef{
			{Name: "awesome-copilot", URL: "https://github.com/github/awesome-copilot"},
		},
		Skills:  nil,
		Targets: nil,
	}

	result := validateConfig(m, nil, false) // hasLocalConfig=false
	assert.True(t, result.ok(), "global-only config with no skills/targets should not report problems")
	assert.Empty(t, result.problems)
}

func TestValidateConfig_WithLocalConfig_NoSkills_IsError(t *testing.T) {
	// When local config exists, missing all resources IS a problem.
	m := &manifest.Manifest{
		Skills:  nil,
		Targets: []string{"opencode"},
	}

	result := validateConfig(m, nil, true) // hasLocalConfig=true
	assert.False(t, result.ok())
	found := false
	for _, p := range result.problems {
		if p.field == "resources" {
			found = true
		}
	}
	assert.True(t, found, "should report missing resources when local config present")
}

func TestValidateConfig_WithLocalConfig_NoTargets_IsError(t *testing.T) {
	// When local config exists, missing targets IS a problem.
	m := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "conventional-commits"}},
		Targets: nil,
	}

	result := validateConfig(m, []string{"conventional-commits"}, true) // hasLocalConfig=true
	assert.False(t, result.ok())
	found := false
	for _, p := range result.problems {
		if p.field == "targets" {
			found = true
		}
	}
	assert.True(t, found, "should report missing targets when local config present")
}

func TestValidateConfig_GlobalOnly_StillValidatesOtherChecks(t *testing.T) {
	// Even in global-only mode, invalid targets and bad paths should still be caught.
	m := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "bad-skill", Path: "/nonexistent/path"}},
		Targets: []string{"vim-copilot"},
	}

	result := validateConfig(m, nil, false) // hasLocalConfig=false
	assert.False(t, result.ok(), "global-only should still catch invalid targets and bad paths")
	// Should have problems for invalid target and missing path, but NOT for
	// "no resources" or "no targets"
	for _, p := range result.problems {
		assert.NotEqual(t, "resources", p.field, "should not flag 'no resources' in global-only")
		assert.NotEqual(t, "targets", p.field, "should not flag 'no targets' in global-only")
	}
}

func TestValidateConfig_WithLocalConfig_InstructionOnly_IsOK(t *testing.T) {
	m := &manifest.Manifest{
		Instructions: []manifest.InstructionRef{{Name: "inst", Content: "be kind"}},
		Targets:      []string{"opencode"},
	}

	result := validateConfig(m, nil, true)
	assert.True(t, result.ok(), "instruction-only config should be valid when targets are set")
}

func TestValidateConfig_WithContext_WarnsOnOverrides(t *testing.T) {
	localPath := filepath.Join(t.TempDir(), "shared")
	require.NoError(t, os.MkdirAll(localPath, 0o755))

	global := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "shared"}},
		Targets: []string{"opencode"},
	}
	local := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "shared", Path: localPath}},
		Targets: []string{"opencode"},
	}
	merged := &manifest.Manifest{
		Skills:  []manifest.SkillRef{{Name: "shared", Path: localPath}},
		Targets: []string{"opencode"},
	}

	result := validateConfigWithContext(merged, nil, true, global, local)
	assert.True(t, result.ok(), "override warning should not fail validation")
	assert.NotEmpty(t, result.warnings)
	assert.Equal(t, "shared", result.warnings[0].field)
}

func TestFormatConfigDiff_IncludesGlobalLocalAndOverrides(t *testing.T) {
	global := &manifest.Manifest{
		Skills:       []manifest.SkillRef{{Name: "shared"}, {Name: "global-only"}},
		Instructions: []manifest.InstructionRef{{Name: "inst-shared", Content: "global"}},
		Targets:      []string{"opencode"},
	}
	local := &manifest.Manifest{
		Skills:       []manifest.SkillRef{{Name: "shared"}, {Name: "local-only"}},
		Instructions: []manifest.InstructionRef{{Name: "inst-shared", Content: "local"}},
		Targets:      []string{"cursor"},
	}
	merged := &manifest.Manifest{
		Skills:       []manifest.SkillRef{{Name: "shared"}, {Name: "global-only"}, {Name: "local-only"}},
		Instructions: []manifest.InstructionRef{{Name: "inst-shared", Content: "local"}},
		Targets:      []string{"cursor"},
	}

	out := formatConfigDiff(global, local, merged)
	assert.Contains(t, out, "Global-only:")
	assert.Contains(t, out, "global-only")
	assert.Contains(t, out, "Local-only:")
	assert.Contains(t, out, "local-only")
	assert.Contains(t, out, "Overrides:")
	assert.Contains(t, out, "shared")
	assert.Contains(t, out, "inst-shared")
	assert.Contains(t, out, "Effective config summary:")
}

func TestFormatConfigDiffJSON_ParsesAndContainsKeys(t *testing.T) {
	global := &manifest.Manifest{Skills: []manifest.SkillRef{{Name: "global-only"}}}
	local := &manifest.Manifest{Skills: []manifest.SkillRef{{Name: "local-only"}}}
	merged := &manifest.Manifest{Skills: []manifest.SkillRef{{Name: "global-only"}, {Name: "local-only"}}}

	jsonOut, err := formatConfigDiffJSON(global, local, merged)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal([]byte(jsonOut), &decoded))
	assert.Contains(t, decoded, "global_only")
	assert.Contains(t, decoded, "local_only")
	assert.Contains(t, decoded, "overrides")
	assert.Contains(t, decoded, "effective_summary")
}

func TestBuildConfigDiffOutput_TextAndJSON(t *testing.T) {
	projectDir := t.TempDir()
	globalDir := t.TempDir()

	globalPath := filepath.Join(globalDir, "vibes.yaml")
	require.NoError(t, os.WriteFile(globalPath, []byte("skills:\n  - name: global-only\ntargets:\n  - opencode\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "vibes.yaml"), []byte("skills:\n  - name: local-only\ntargets:\n  - cursor\n"), 0o644))

	textOut, err := buildConfigDiffOutput(projectDir, globalPath, false)
	require.NoError(t, err)
	assert.Contains(t, textOut, "Global-only:")

	jsonOut, err := buildConfigDiffOutput(projectDir, globalPath, true)
	require.NoError(t, err)
	assert.Contains(t, jsonOut, "\"effective_summary\"")
}

func TestConfigDiffCommand_HasJSONFlag(t *testing.T) {
	f := configDiffCmd.Flags().Lookup("json")
	require.NotNil(t, f)
	assert.Equal(t, "bool", f.Value.Type())
}
