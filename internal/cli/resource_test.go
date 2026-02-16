package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/chaz8081/positive-vibes/internal/registry"
	"github.com/chaz8081/positive-vibes/pkg/schema"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type cliCleanupTrackingSource struct {
	name                 string
	fetchWithCleanupUsed bool
	cleanupCalled        bool
	skillName            string
}

func (s *cliCleanupTrackingSource) Name() string { return s.name }

func (s *cliCleanupTrackingSource) Fetch(name string) (*schema.Skill, string, error) {
	return nil, "", fmt.Errorf("unexpected Fetch call for %s", name)
}

func (s *cliCleanupTrackingSource) List() ([]string, error) {
	return []string{s.skillName}, nil
}

func (s *cliCleanupTrackingSource) FetchWithCleanup(name string) (*schema.Skill, string, func(), error) {
	s.fetchWithCleanupUsed = true
	cleanup := func() {
		s.cleanupCalled = true
	}
	if name != s.skillName {
		return nil, "", cleanup, fmt.Errorf("skill not found: %s", name)
	}
	return &schema.Skill{Name: name}, "", cleanup, nil
}

// --- ParseResourceType tests ---

func TestParseResourceType_Valid(t *testing.T) {
	tests := []struct {
		input    string
		expected ResourceType
	}{
		{"skills", ResourceSkills},
		{"agents", ResourceAgents},
		{"instructions", ResourceInstructions},
		{"prompts", ResourcePrompts},
	}
	for _, tt := range tests {
		rt, err := ParseResourceType(tt.input)
		require.NoError(t, err)
		assert.Equal(t, tt.expected, rt)
	}
}

func TestParseResourceType_Invalid(t *testing.T) {
	_, err := ParseResourceType("widgets")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown resource type")
}

func TestMergeResourceItems_InstalledWins(t *testing.T) {
	available := []ResourceItem{
		{Name: "code-review", Installed: false, InstallScope: installScopeNone},
		{Name: "tdd", Installed: false, InstallScope: installScopeNone},
	}
	installed := []ResourceItem{
		{Name: "code-review", Installed: true, InstallScope: installScopeBoth},
	}

	merged := MergeResourceItems(available, installed)

	byName := make(map[string]ResourceItem, len(merged))
	for _, item := range merged {
		byName[item.Name] = item
	}
	assert.Len(t, byName, 2)
	assert.True(t, byName["code-review"].Installed)
	assert.Equal(t, installScopeBoth, byName["code-review"].InstallScope)
	assert.False(t, byName["tdd"].Installed)
	assert.Equal(t, installScopeNone, byName["tdd"].InstallScope)
}

// --- registrySkillSet tests ---

func TestRegistrySkillSet_FromSources(t *testing.T) {
	sets := collectSkillSets(nil)
	assert.NotEmpty(t, sets, "should have at least the embedded registry")
	assert.Equal(t, "embedded", sets[0].RegistryName)
	assert.NotEmpty(t, sets[0].Skills, "embedded registry should have skills")
}

// --- formatSkillsList tests ---

func TestFormatSkillsList_GroupsByRegistry(t *testing.T) {
	sets := []registrySkillSet{
		{RegistryName: "embedded", Skills: []string{"code-review", "conventional-commits"}},
		{RegistryName: "awesome-copilot", URL: "https://github.com/github/awesome-copilot", Skills: []string{"agentic-eval"}},
	}
	installed := map[string]bool{"code-review": true}

	out := formatSkillsList(sets, installed)
	assert.Contains(t, out, "Embedded:")
	assert.Contains(t, out, "code-review")
	assert.Contains(t, out, "conventional-commits")
	assert.Contains(t, out, "awesome-copilot")
	assert.Contains(t, out, "agentic-eval")
}

func TestFormatSkillsList_MarksInstalled(t *testing.T) {
	sets := []registrySkillSet{
		{RegistryName: "embedded", Skills: []string{"code-review", "conventional-commits"}},
	}
	installed := map[string]bool{"code-review": true}

	out := formatSkillsList(sets, installed)
	assert.Contains(t, out, "code-review")
	assert.Contains(t, out, "[installed]")
}

func TestFormatSkillsList_EmptyRegistrySkipped(t *testing.T) {
	sets := []registrySkillSet{
		{RegistryName: "embedded", Skills: []string{"code-review"}},
		{RegistryName: "empty-reg", URL: "https://example.com", Skills: nil},
	}

	out := formatSkillsList(sets, nil)
	assert.Contains(t, out, "Embedded:")
	assert.NotContains(t, out, "empty-reg")
}

func TestFormatSkillsList_ShowsSummary(t *testing.T) {
	sets := []registrySkillSet{
		{RegistryName: "embedded", Skills: []string{"a", "b", "c"}},
	}
	installed := map[string]bool{"a": true}

	out := formatSkillsList(sets, installed)
	assert.Contains(t, out, "1 installed")
	assert.Contains(t, out, "3 available")
}

func TestFormatSkillsList_NoSkillsAnywhere(t *testing.T) {
	sets := []registrySkillSet{}

	out := formatSkillsList(sets, nil)
	assert.Contains(t, out, "No skills found")
}

// --- formatSkillsList with filters tests ---

func TestFormatSkillsList_FilterByRegistry(t *testing.T) {
	sets := []registrySkillSet{
		{RegistryName: "embedded", Skills: []string{"code-review", "conventional-commits"}},
		{RegistryName: "awesome-copilot", URL: "https://example.com", Skills: []string{"agentic-eval"}},
	}

	opts := listFormatOptions{Registry: "embedded"}
	out := formatSkillsListFiltered(sets, nil, opts)
	assert.Contains(t, out, "code-review")
	assert.Contains(t, out, "conventional-commits")
	assert.NotContains(t, out, "agentic-eval")
	assert.NotContains(t, out, "awesome-copilot")
}

func TestFormatSkillsList_FilterInstalledOnly(t *testing.T) {
	sets := []registrySkillSet{
		{RegistryName: "embedded", Skills: []string{"code-review", "conventional-commits"}},
	}
	installed := map[string]bool{"code-review": true}

	opts := listFormatOptions{InstalledOnly: true}
	out := formatSkillsListFiltered(sets, installed, opts)
	assert.Contains(t, out, "code-review")
	assert.NotContains(t, out, "conventional-commits")
}

func TestFormatSkillsList_FilterRegistryNoMatch(t *testing.T) {
	sets := []registrySkillSet{
		{RegistryName: "embedded", Skills: []string{"code-review"}},
	}

	opts := listFormatOptions{Registry: "nonexistent"}
	out := formatSkillsListFiltered(sets, nil, opts)
	assert.Contains(t, out, "No skills found")
}

func TestFormatSkillsList_FilterInstalledOnlyNoneInstalled(t *testing.T) {
	sets := []registrySkillSet{
		{RegistryName: "embedded", Skills: []string{"code-review"}},
	}

	opts := listFormatOptions{InstalledOnly: true}
	out := formatSkillsListFiltered(sets, nil, opts)
	assert.Contains(t, out, "No skills found")
}

func TestFormatSkillsList_FilterBothRegistryAndInstalled(t *testing.T) {
	sets := []registrySkillSet{
		{RegistryName: "embedded", Skills: []string{"code-review", "conventional-commits"}},
		{RegistryName: "awesome-copilot", URL: "https://example.com", Skills: []string{"agentic-eval", "agent-testing"}},
	}
	installed := map[string]bool{"code-review": true, "agentic-eval": true}

	opts := listFormatOptions{Registry: "awesome-copilot", InstalledOnly: true}
	out := formatSkillsListFiltered(sets, installed, opts)
	assert.Contains(t, out, "agentic-eval")
	assert.NotContains(t, out, "code-review")
	assert.NotContains(t, out, "agent-testing")
	assert.NotContains(t, out, "conventional-commits")
}

// --- JSON output tests ---

func TestFormatSkillsListJSON_Structure(t *testing.T) {
	sets := []registrySkillSet{
		{RegistryName: "embedded", Skills: []string{"code-review", "conventional-commits"}},
		{RegistryName: "awesome-copilot", URL: "https://example.com", Skills: []string{"agentic-eval"}},
	}
	installed := map[string]bool{"code-review": true}

	out := formatSkillsListJSON(sets, installed)

	var result skillsListJSON
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err, "output should be valid JSON")
	assert.Equal(t, 2, len(result.Registries))
	assert.Equal(t, "embedded", result.Registries[0].Name)
	assert.Equal(t, 3, result.TotalAvailable)
	assert.Equal(t, 1, result.TotalInstalled)

	for _, s := range result.Registries[0].Skills {
		if s.Name == "code-review" {
			assert.True(t, s.Installed)
		} else {
			assert.False(t, s.Installed)
		}
	}
}

func TestFormatSkillsListJSON_Empty(t *testing.T) {
	out := formatSkillsListJSON(nil, nil)

	var result skillsListJSON
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err, "output should be valid JSON")
	assert.Equal(t, 0, len(result.Registries))
	assert.Equal(t, 0, result.TotalAvailable)
}

// --- formatSkillShow tests ---

func TestFormatSkillShow_BasicMetadata(t *testing.T) {
	skill := &schema.Skill{
		Name:        "code-review",
		Description: "Provides thorough code review feedback",
		Version:     "1.0",
		Author:      "positive-vibes",
		Tags:        []string{"review", "quality"},
	}

	out := formatSkillShow(skill, "embedded", "", true)
	assert.Contains(t, out, "code-review")
	assert.Contains(t, out, "Provides thorough code review feedback")
	assert.Contains(t, out, "1.0")
	assert.Contains(t, out, "positive-vibes")
	assert.Contains(t, out, "review")
	assert.Contains(t, out, "quality")
	assert.Contains(t, out, "embedded")
	assert.Contains(t, out, "installed")
}

func TestFormatSkillShow_NotInstalled(t *testing.T) {
	skill := &schema.Skill{
		Name:        "code-review",
		Description: "Provides thorough code review feedback",
	}

	out := formatSkillShow(skill, "embedded", "", false)
	assert.Contains(t, out, "not installed")
}

func TestFormatSkillShow_WithURL(t *testing.T) {
	skill := &schema.Skill{
		Name:        "agentic-eval",
		Description: "Evaluates agentic behavior",
	}

	out := formatSkillShow(skill, "awesome-copilot", "https://github.com/github/awesome-copilot", false)
	assert.Contains(t, out, "awesome-copilot")
	assert.Contains(t, out, "https://github.com/github/awesome-copilot")
}

func TestFormatSkillShow_WithInstructions(t *testing.T) {
	skill := &schema.Skill{
		Name:         "code-review",
		Description:  "Reviews code",
		Instructions: "# Code Review\n\nReview all pull requests carefully.",
	}

	out := formatSkillShow(skill, "embedded", "", false)
	assert.Contains(t, out, "# Code Review")
	assert.Contains(t, out, "Review all pull requests carefully")
}

func TestFormatSkillShow_MinimalFields(t *testing.T) {
	skill := &schema.Skill{
		Name: "bare-skill",
	}

	out := formatSkillShow(skill, "embedded", "", false)
	assert.Contains(t, out, "bare-skill")
	assert.NotContains(t, out, "Version:")
	assert.NotContains(t, out, "Author:")
	assert.NotContains(t, out, "Tags:")
}

// --- resolveSkillFromSources tests ---

func TestResolveSkillFromSources_FindsEmbedded(t *testing.T) {
	sources := buildAllSources(nil)
	skill, regName, err := resolveSkillFromSources("code-review", sources)
	require.NoError(t, err)
	assert.Equal(t, "code-review", skill.Name)
	assert.Equal(t, "embedded", regName)
}

func TestResolveSkillFromSources_NotFound(t *testing.T) {
	sources := buildAllSources(nil)
	_, _, err := resolveSkillFromSources("no-such-skill-xyz", sources)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestResolveSkillFromSources_UsesFetchWithCleanup(t *testing.T) {
	src := &cliCleanupTrackingSource{
		name:      "test-cleanable",
		skillName: "cleanup-skill",
	}

	skill, regName, err := resolveSkillFromSources("cleanup-skill", []registry.SkillSource{src})
	require.NoError(t, err)
	require.Equal(t, "cleanup-skill", skill.Name)
	require.Equal(t, "test-cleanable", regName)
	require.True(t, src.fetchWithCleanupUsed, "expected FetchWithCleanup to be used")
	require.True(t, src.cleanupCalled, "expected cleanup to be called")
}

// --- formatResourceList tests ---

func TestFormatResourceList_Agents(t *testing.T) {
	items := []ResourceItem{
		{Name: "reviewer", Installed: true},
		{Name: "planner", Installed: true},
	}
	out := formatResourceList(ResourceAgents, items)
	assert.Contains(t, out, "reviewer")
	assert.Contains(t, out, "planner")
	assert.Contains(t, out, "2 installed, 2 available")
}

func TestFormatResourceList_Instructions(t *testing.T) {
	items := []ResourceItem{
		{Name: "coding-standards", Installed: true},
	}
	out := formatResourceList(ResourceInstructions, items)
	assert.Contains(t, out, "coding-standards")
	assert.Contains(t, out, "1 installed, 1 available")
}

func TestFormatResourceList_Empty(t *testing.T) {
	out := formatResourceList(ResourceAgents, nil)
	assert.Contains(t, out, "No agents found")
}

// --- formatResourceListJSON tests ---

func TestFormatResourceListJSON_Structure(t *testing.T) {
	items := []ResourceItem{
		{Name: "reviewer", Installed: true},
		{Name: "planner", Installed: true},
	}
	out := formatResourceListJSON(ResourceAgents, items)

	var result resourceListJSON
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "agents", result.Type)
	assert.Equal(t, 2, result.Total)
	assert.Equal(t, 2, len(result.Items))
}

// --- formatAgentShow tests ---

func TestFormatAgentShow(t *testing.T) {
	agent := manifest.AgentRef{
		Name:     "reviewer",
		Path:     "./agents/reviewer.md",
		Registry: "",
	}
	out := formatAgentShow(agent, true)
	assert.Contains(t, out, "reviewer")
	assert.Contains(t, out, "./agents/reviewer.md")
	assert.Contains(t, out, "installed")
}

func TestFormatAgentShow_WithRegistry(t *testing.T) {
	agent := manifest.AgentRef{
		Name:     "reviewer",
		Registry: "awesome-copilot",
		Path:     "my-skill/agents/reviewer.md",
	}
	out := formatAgentShow(agent, false)
	assert.Contains(t, out, "reviewer")
	assert.Contains(t, out, "awesome-copilot")
	assert.Contains(t, out, "my-skill/agents/reviewer.md")
	assert.Contains(t, out, "available")
}

// --- formatInstructionShow tests ---

func TestFormatInstructionShow(t *testing.T) {
	inst := manifest.InstructionRef{
		Name:    "coding-standards",
		Content: "Always use gofmt.",
	}
	out := formatInstructionShow(inst, true)
	assert.Contains(t, out, "coding-standards")
	assert.Contains(t, out, "Always use gofmt.")
	assert.Contains(t, out, "installed")
}

func TestFormatInstructionShow_WithPath(t *testing.T) {
	inst := manifest.InstructionRef{
		Name:    "coding-standards",
		Path:    "./instructions/standards.md",
		ApplyTo: "opencode",
	}
	out := formatInstructionShow(inst, true)
	assert.Contains(t, out, "coding-standards")
	assert.Contains(t, out, "./instructions/standards.md")
	assert.Contains(t, out, "opencode")
}

// --- collectAvailableSkills tests ---

func TestCollectAvailableSkills_NoDuplicates(t *testing.T) {
	items := collectAvailableSkills(nil) // embedded only
	seen := make(map[string]bool)
	for _, item := range items {
		assert.False(t, seen[item.Name], "duplicate skill: %s", item.Name)
		seen[item.Name] = true
	}
	assert.NotEmpty(t, items)
}

func TestCollectInstalledSkills(t *testing.T) {
	merged := &manifest.Manifest{
		Skills: []manifest.SkillRef{
			{Name: "code-review"},
			{Name: "conventional-commits"},
		},
	}
	items := collectInstalledSkills(merged)
	assert.Len(t, items, 2)
	assert.True(t, items[0].Installed)
}

func TestCollectInstalledSkills_Nil(t *testing.T) {
	items := collectInstalledSkills(nil)
	assert.Nil(t, items)
}

func TestCollectAgents(t *testing.T) {
	merged := &manifest.Manifest{
		Agents: []manifest.AgentRef{
			{Name: "reviewer", Path: "./agents/reviewer.md"},
		},
	}
	items := collectAgents(merged)
	assert.Len(t, items, 1)
	assert.Equal(t, "reviewer", items[0].Name)
}

func TestCollectInstructions(t *testing.T) {
	merged := &manifest.Manifest{
		Instructions: []manifest.InstructionRef{
			{Name: "standards", Content: "Use gofmt."},
		},
	}
	items := collectInstructions(merged)
	assert.Len(t, items, 1)
	assert.Equal(t, "standards", items[0].Name)
}

// --- buildInstalledSkillsMap tests ---

func TestBuildInstalledSkillsMap(t *testing.T) {
	merged := &manifest.Manifest{
		Skills: []manifest.SkillRef{
			{Name: "code-review"},
		},
	}
	m := buildInstalledSkillsMap(merged)
	assert.True(t, m["code-review"])
	assert.False(t, m["nonexistent"])
}

func TestBuildInstalledSkillsMap_Nil(t *testing.T) {
	m := buildInstalledSkillsMap(nil)
	assert.NotNil(t, m)
	assert.Len(t, m, 0)
}

// --- dedup tests ---

func TestDedup_Empty(t *testing.T) {
	assert.Empty(t, dedup(nil))
	assert.Empty(t, dedup([]string{}))
}

func TestDedup_NoDuplicates(t *testing.T) {
	input := []string{"a", "b", "c"}
	assert.Equal(t, []string{"a", "b", "c"}, dedup(input))
}

func TestDedup_WithDuplicates(t *testing.T) {
	input := []string{"a", "b", "a", "c", "b", "a"}
	assert.Equal(t, []string{"a", "b", "c"}, dedup(input))
}

func TestDedup_AllSame(t *testing.T) {
	input := []string{"x", "x", "x"}
	assert.Equal(t, []string{"x"}, dedup(input))
}

func TestDedup_PreservesOrder(t *testing.T) {
	input := []string{"c", "a", "b", "a", "c"}
	assert.Equal(t, []string{"c", "a", "b"}, dedup(input))
}

// --- resourceNamesFromItems tests ---

func TestResourceNamesFromItems(t *testing.T) {
	items := []ResourceItem{
		{Name: "alpha", Installed: true},
		{Name: "beta", Installed: false},
		{Name: "gamma", Installed: true},
	}
	names := resourceNamesFromItems(items)
	assert.Equal(t, []string{"alpha", "beta", "gamma"}, names)
}

func TestResourceNamesFromItems_Empty(t *testing.T) {
	names := resourceNamesFromItems(nil)
	assert.Empty(t, names)
}

// --- resourceTypeCompletions tests ---

func TestResourceTypeCompletions(t *testing.T) {
	completions := resourceTypeCompletions()
	assert.Equal(t, ValidResourceTypes(), completions)
	assert.Contains(t, completions, "skills")
	assert.Contains(t, completions, "agents")
	assert.Contains(t, completions, "instructions")
}

// --- makeValidArgsFunction tests ---

func TestMakeValidArgsFunction_FirstArg_ReturnsResourceTypes(t *testing.T) {
	fn := makeValidArgsFunction("all")
	suggestions, directive := fn(rootCmd, []string{}, "")
	assert.Equal(t, ValidResourceTypes(), suggestions)
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}

func TestMakeValidArgsFunction_EmptyMode_NoNameSuggestions(t *testing.T) {
	// list command uses "" mode — no name completions after resource type
	fn := makeValidArgsFunction("")
	suggestions, directive := fn(rootCmd, []string{"skills"}, "")
	assert.Nil(t, suggestions)
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}

func TestMakeValidArgsFunction_InvalidResourceType(t *testing.T) {
	fn := makeValidArgsFunction("all")
	suggestions, directive := fn(rootCmd, []string{"widgets"}, "")
	assert.Nil(t, suggestions)
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}

func TestMakeValidArgsFunction_SkillNames_AllMode(t *testing.T) {
	// "all" mode should return available skill names (from embedded registry at minimum)
	fn := makeValidArgsFunction("all")
	suggestions, directive := fn(rootCmd, []string{"skills"}, "")
	assert.NotEmpty(t, suggestions, "should return skill names from embedded registry")
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}

func TestMakeValidArgsFunction_FiltersAlreadyProvided(t *testing.T) {
	fn := makeValidArgsFunction("all")
	// Get all suggestions first
	all, _ := fn(rootCmd, []string{"skills"}, "")
	require.NotEmpty(t, all, "need at least one skill for this test")

	// Now provide the first skill as already typed — it should be excluded
	filtered, _ := fn(rootCmd, []string{"skills", all[0]}, "")
	for _, name := range filtered {
		assert.NotEqual(t, all[0], name, "already-provided name should be excluded")
	}
}

func TestMakeValidArgsFunction_AgentsReturnsEmpty_NoManifest(t *testing.T) {
	// With no manifest, agents should return empty (no agents configured)
	fn := makeValidArgsFunction("installed")
	suggestions, directive := fn(rootCmd, []string{"agents"}, "")
	// We expect nil/empty since there's no manifest with agents in the test project
	assert.Empty(t, suggestions)
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}

func TestMakeValidArgsFunction_InstructionsReturnsEmpty_NoManifest(t *testing.T) {
	fn := makeValidArgsFunction("installed")
	suggestions, directive := fn(rootCmd, []string{"instructions"}, "")
	assert.Empty(t, suggestions)
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}

func TestResourceNameFromPath_AgentAndInstructionSuffixes(t *testing.T) {
	assert.Equal(t, "debug", resourceNameFromPath(ResourceAgents, "agents/debug.agent.md"))
	assert.Equal(t, "markdown", resourceNameFromPath(ResourceInstructions, "instructions/markdown.instructions.md"))
	assert.Equal(t, "release", resourceNameFromPath(ResourcePrompts, "prompts/release.prompt.md"))
	assert.Equal(t, "cleanup", resourceNameFromPath(ResourcePrompts, "prompts/cleanup.md"))
	assert.Equal(t, "", resourceNameFromPath(ResourceAgents, "agents/readme.md"))
	assert.Equal(t, "", resourceNameFromPath(ResourceInstructions, "instructions/readme.md"))
}

// --- ValidArgsFunction wiring tests ---

func TestInstallCmd_HasValidArgsFunction(t *testing.T) {
	assert.NotNil(t, installCmd.ValidArgsFunction, "install command should have ValidArgsFunction set")
}

func TestListCmd_HasValidArgsFunction(t *testing.T) {
	assert.NotNil(t, listCmd.ValidArgsFunction, "list command should have ValidArgsFunction set")
}

func TestShowCmd_HasValidArgsFunction(t *testing.T) {
	assert.NotNil(t, showCmd.ValidArgsFunction, "show command should have ValidArgsFunction set")
}

func TestRemoveCmd_HasValidArgsFunction(t *testing.T) {
	assert.NotNil(t, removeCmd.ValidArgsFunction, "remove command should have ValidArgsFunction set")
}

func TestResource_ShowSkillDetail_IncludesAdditionalFiles(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "examples"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "SKILL.md"), []byte("# Skill\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "examples", "sample.md"), []byte("sample\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "zeta.txt"), []byte("z\n"), 0o644))

	files := collectLocalFiles(root)
	require.Len(t, files, 3)
	assert.Equal(t, "SKILL.md", files[0]["name"])
	assert.Equal(t, "examples/sample.md", files[1]["name"])
	assert.Equal(t, "zeta.txt", files[2]["name"])
}

func TestResource_ShowSkillDetail_PathMayPointToSkillFile(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "examples"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "SKILL.md"), []byte("# Skill\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "examples", "sample.md"), []byte("sample\n"), 0o644))

	files := collectLocalFiles(filepath.Join(root, "SKILL.md"))
	require.Len(t, files, 2)
	assert.Equal(t, "SKILL.md", files[0]["name"])
	assert.Equal(t, "examples/sample.md", files[1]["name"])
}

func TestCollectSkillPreviewFilesFromRegistrySource_IncludesNestedFiles(t *testing.T) {
	src := fakeRegistrySource{
		name: "embedded",
		resourceFiles: []string{
			"excalidraw-diagram-generator/SKILL.md",
			"excalidraw-diagram-generator/references/README.md",
			"excalidraw-diagram-generator/scripts/render.sh",
			"other-skill/SKILL.md",
		},
		resourceContent: map[string][]byte{
			"excalidraw-diagram-generator/SKILL.md":             []byte("# Skill"),
			"excalidraw-diagram-generator/references/README.md": []byte("refs"),
			"excalidraw-diagram-generator/scripts/render.sh":    []byte("#!/bin/sh"),
			"other-skill/SKILL.md":                              []byte("ignore"),
		},
	}

	files := collectSkillPreviewFilesFromRegistrySource(src, "excalidraw-diagram-generator")
	require.Len(t, files, 3)
	assert.Equal(t, "SKILL.md", files[0]["name"])
	assert.Equal(t, "references/README.md", files[1]["name"])
	assert.Equal(t, "scripts/render.sh", files[2]["name"])
}

func TestListAvailableResourceItems_TargetsUsesValidTargets(t *testing.T) {
	items, err := ListAvailableResourceItems(t.TempDir(), filepath.Join(t.TempDir(), "missing.yaml"), "targets")
	require.NoError(t, err)
	require.Len(t, items, len(manifest.ValidTargets))

	got := make(map[string]bool, len(items))
	for _, item := range items {
		got[item.Name] = true
	}
	for _, targetName := range manifest.ValidTargets {
		if !got[targetName] {
			t.Fatalf("expected available targets to include %q, got %#v", targetName, items)
		}
	}
}

func TestInstallAndRemoveResourceItems_TargetsMutateProjectManifest(t *testing.T) {
	projectDir := t.TempDir()
	globalPath := filepath.Join(t.TempDir(), "global-vibes.yaml")

	installReport, err := InstallResourceItemsWithReport(projectDir, globalPath, "targets", []string{"cursor"})
	require.NoError(t, err)
	assert.Equal(t, []string{"cursor"}, installReport.MutatedNames)

	m, _, err := manifest.LoadManifestFromProject(projectDir)
	require.NoError(t, err)
	assert.Equal(t, []string{"cursor"}, m.Targets)

	removeReport, err := RemoveResourceItemsWithReport(projectDir, "targets", []string{"cursor"})
	require.NoError(t, err)
	assert.Equal(t, []string{"cursor"}, removeReport.MutatedNames)

	m, _, err = manifest.LoadManifestFromProject(projectDir)
	require.NoError(t, err)
	assert.Len(t, m.Targets, 0)
}

func TestListAvailableResourceItems_RegistriesIncludesMerged(t *testing.T) {
	projectDir := t.TempDir()
	globalPath := filepath.Join(t.TempDir(), "global-vibes.yaml")
	require.NoError(t, os.WriteFile(globalPath, []byte(`registries:
  - name: awesome-copilot
    url: https://github.com/github/awesome-copilot
    ref: latest
`), 0o644))

	items, err := ListAvailableResourceItems(projectDir, globalPath, "registries")
	require.NoError(t, err)
	assert.NotEmpty(t, items)
	assert.Equal(t, "awesome-copilot", items[0].Name)
}

func TestInstallAndRemoveResourceItems_RegistriesMutateProjectManifest(t *testing.T) {
	projectDir := t.TempDir()
	globalPath := filepath.Join(t.TempDir(), "global-vibes.yaml")
	require.NoError(t, os.WriteFile(globalPath, []byte(`registries:
  - name: awesome-copilot
    url: https://github.com/github/awesome-copilot
    ref: latest
`), 0o644))

	installReport, err := InstallResourceItemsWithReport(projectDir, globalPath, "registries", []string{"awesome-copilot"})
	require.NoError(t, err)
	assert.Equal(t, []string{"awesome-copilot"}, installReport.MutatedNames)

	m, _, err := manifest.LoadManifestFromProject(projectDir)
	require.NoError(t, err)
	require.Len(t, m.Registries, 1)
	assert.Equal(t, "awesome-copilot", m.Registries[0].Name)
	assert.Equal(t, "skills/", m.Registries[0].Paths["skills"])
	assert.Equal(t, "instructions/", m.Registries[0].Paths["instructions"])
	assert.Equal(t, "agents/", m.Registries[0].Paths["agents"])
	assert.Equal(t, "prompts/", m.Registries[0].Paths["prompts"])

	removeReport, err := RemoveResourceItemsWithReport(projectDir, "registries", []string{"awesome-copilot"})
	require.NoError(t, err)
	assert.Equal(t, []string{"awesome-copilot"}, removeReport.MutatedNames)

	m, _, err = manifest.LoadManifestFromProject(projectDir)
	require.NoError(t, err)
	assert.Len(t, m.Registries, 0)
}

func TestListInstalledResourceItems_ScopeLocalGlobalBoth(t *testing.T) {
	configDir := t.TempDir()
	globalPath := filepath.Join(configDir, "global-vibes.yaml")
	require.NoError(t, os.WriteFile(globalPath, []byte("skills:\n  - name: global-only\n  - name: both\n"), 0o644))

	projectDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "vibes.yaml"), []byte("skills:\n  - name: local-only\n  - name: both\n"), 0o644))

	items, err := ListInstalledResourceItems(projectDir, globalPath, "skills")
	require.NoError(t, err)

	byName := map[string]ResourceItem{}
	for _, item := range items {
		byName[item.Name] = item
	}
	assert.Equal(t, installScopeLocal, byName["local-only"].InstallScope)
	assert.Equal(t, installScopeGlobal, byName["global-only"].InstallScope)
	assert.Equal(t, installScopeBoth, byName["both"].InstallScope)
}

func TestShowResourceDetail_RegistryIncludesEffectivePaths(t *testing.T) {
	projectDir := t.TempDir()
	globalPath := filepath.Join(t.TempDir(), "global-vibes.yaml")
	require.NoError(t, os.WriteFile(globalPath, []byte(`registries:
  - name: awesome-copilot
    url: https://github.com/github/awesome-copilot
    ref: latest
    paths:
      skills: skills/
`), 0o644))

	detail, err := ShowResourceDetail(projectDir, globalPath, "registries", "awesome-copilot")
	require.NoError(t, err)
	payload, ok := detail.Payload.(map[string]any)
	require.True(t, ok)
	content, ok := payload["content"].(string)
	require.True(t, ok)
	assert.Contains(t, content, "skills: skills/")
	assert.Contains(t, content, "instructions: instructions/")
	assert.Contains(t, content, "agents: agents/")
	assert.Contains(t, content, "prompts: prompts/")
}

func TestShowResourceDetail_RegistrySourceState_LocalOverrideActive(t *testing.T) {
	projectDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "vibes.yaml"), []byte(`registries:
  - name: team-reg
    url: https://example.com/local.git
    ref: dev
`), 0o644))

	globalPath := filepath.Join(t.TempDir(), "global-vibes.yaml")
	require.NoError(t, os.WriteFile(globalPath, []byte(`registries:
  - name: team-reg
    url: https://example.com/global.git
    ref: main
`), 0o644))

	detail, err := ShowResourceDetail(projectDir, globalPath, "registries", "team-reg")
	require.NoError(t, err)
	payload, ok := detail.Payload.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "local_conflict", payload["source_state"])
	assert.Equal(t, "local registry conflicts with global URL", payload["source_reason"])
}

func TestShowResourceDetail_RegistrySourceState_LocalIssue(t *testing.T) {
	projectDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "vibes.yaml"), []byte(`registries:
  - name: broken-reg
    url: https://example.com/broken.git
`), 0o644))

	globalPath := filepath.Join(t.TempDir(), "global-vibes.yaml")

	detail, err := ShowResourceDetail(projectDir, globalPath, "registries", "broken-reg")
	require.NoError(t, err)
	payload, ok := detail.Payload.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "local_issue", payload["source_state"])
	assert.Equal(t, "local registry is invalid: missing ref", payload["source_reason"])
}

func TestListInstalledResourceItems_RegistriesMarksAttentionState(t *testing.T) {
	projectDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "vibes.yaml"), []byte(`registries:
  - name: team-reg
    url: https://example.com/local.git
    ref: dev
`), 0o644))

	globalPath := filepath.Join(t.TempDir(), "global-vibes.yaml")
	require.NoError(t, os.WriteFile(globalPath, []byte(`registries:
  - name: team-reg
    url: https://example.com/global.git
    ref: main
`), 0o644))

	items, err := ListInstalledResourceItems(projectDir, globalPath, "registries")
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "registry_attention", items[0].State)
}

type fakeRegistrySource struct {
	name            string
	resourceFiles   []string
	resourceContent map[string][]byte
}

func (f fakeRegistrySource) Name() string { return f.name }

func (f fakeRegistrySource) Fetch(name string) (*schema.Skill, string, error) {
	return &schema.Skill{Name: name}, name, nil
}

func (f fakeRegistrySource) List() ([]string, error) { return nil, nil }

func (f fakeRegistrySource) FetchFile(skillName, relPath string) ([]byte, error) {
	key := filepath.ToSlash(filepath.Join(skillName, relPath))
	return f.resourceContent[key], nil
}

func (f fakeRegistrySource) ListFiles(skillName, relDir string) ([]string, error) {
	_ = skillName
	_ = relDir
	return nil, nil
}

func (f fakeRegistrySource) FetchResourceFile(kind, relPath string) ([]byte, error) {
	if kind != "skills" {
		return nil, nil
	}
	return f.resourceContent[filepath.ToSlash(relPath)], nil
}

func (f fakeRegistrySource) ListResourceFiles(kind string) ([]string, error) {
	if kind != "skills" {
		return nil, nil
	}
	return f.resourceFiles, nil
}

var _ registry.ResourceSource = fakeRegistrySource{}
