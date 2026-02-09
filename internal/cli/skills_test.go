package cli

import (
	"encoding/json"
	"testing"

	"github.com/chaz8081/positive-vibes/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- registrySkillSet tests ---

func TestRegistrySkillSet_FromSources(t *testing.T) {
	// Using the embedded registry as a real source -- no mocks needed.
	// The embedded registry should have at least some skills.
	sets := collectSkillSets(nil) // nil manifest = embedded only
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
	// The installed skill should be marked differently from non-installed
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
	// Skill with only name - no version, author, tags, or description
	skill := &schema.Skill{
		Name: "bare-skill",
	}

	out := formatSkillShow(skill, "embedded", "", false)
	assert.Contains(t, out, "bare-skill")
	// Should not crash or show empty labeled lines
	assert.NotContains(t, out, "Version:")
	assert.NotContains(t, out, "Author:")
	assert.NotContains(t, out, "Tags:")
}

// --- resolveSkillFromSources tests ---

func TestResolveSkillFromSources_FindsEmbedded(t *testing.T) {
	// Using the real embedded registry to find a known skill
	sources := buildAllSources(nil) // nil manifest = embedded only
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
	assert.NotContains(t, out, "code-review")          // wrong registry
	assert.NotContains(t, out, "agent-testing")        // not installed
	assert.NotContains(t, out, "conventional-commits") // wrong registry
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

	// Check installed flag
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
