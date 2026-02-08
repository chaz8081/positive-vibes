package cli

import (
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

func TestFormatPaths_LegacyYamlFound(t *testing.T) {
	dir := t.TempDir()
	localDir := filepath.Join(dir, "project")
	require.NoError(t, os.MkdirAll(localDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(localDir, "vibes.yaml"), []byte("skills: []"), 0o644))

	globalPath := filepath.Join(dir, "nope", "vibes.yml")
	out := formatPaths(globalPath, localDir, filepath.Join(dir, "cache"))
	assert.Contains(t, out, "vibes.yaml")
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
		Skills:       []manifest.SkillRef{{Name: "s"}},
		Targets:      []string{"opencode"},
		Instructions: []string{"Use Go modules"},
	}
	out := renderMergedYAML(m)
	assert.Contains(t, out, "instructions:")
	assert.Contains(t, out, "Use Go modules")
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

func TestAnnotateManifest_InstructionSources(t *testing.T) {
	global := &manifest.Manifest{
		Skills:       []manifest.SkillRef{{Name: "s"}},
		Targets:      []string{"opencode"},
		Instructions: []string{"global instruction"},
	}
	local := &manifest.Manifest{
		Skills:       []manifest.SkillRef{{Name: "s"}},
		Targets:      []string{"opencode"},
		Instructions: []string{"local instruction"},
	}
	merged := &manifest.Manifest{
		Skills:       []manifest.SkillRef{{Name: "s"}},
		Targets:      []string{"opencode"},
		Instructions: []string{"global instruction", "local instruction"},
	}

	out := annotateManifest(global, local, merged)
	assert.Contains(t, out, "global instruction")
	assert.Contains(t, out, "local instruction")
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
