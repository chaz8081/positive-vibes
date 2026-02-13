package cli

import (
	"encoding/json"
	"testing"

	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/chaz8081/positive-vibes/pkg/schema"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ParseResourceType tests ---

func TestParseResourceType_Valid(t *testing.T) {
	tests := []struct {
		input    string
		expected ResourceType
	}{
		{"skills", ResourceSkills},
		{"agents", ResourceAgents},
		{"instructions", ResourceInstructions},
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
		{Name: "code-review", Installed: false},
		{Name: "tdd", Installed: false},
	}
	installed := []ResourceItem{
		{Name: "code-review", Installed: true},
	}

	merged := MergeResourceItems(available, installed)

	byName := make(map[string]ResourceItem, len(merged))
	for _, item := range merged {
		byName[item.Name] = item
	}
	assert.Len(t, byName, 2)
	assert.True(t, byName["code-review"].Installed)
	assert.False(t, byName["tdd"].Installed)
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
