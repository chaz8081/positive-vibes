package cli

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/chaz8081/positive-vibes/internal/manifest"
)

func TestInstallResourcesCommandAction_ReportsMutationsAndDuplicateSkips(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	globalPath := filepath.Join(configDir, "positive-vibes", "vibes.yaml")

	projectDir := t.TempDir()
	writeResourceActionManifest(t, projectDir, &manifest.Manifest{
		Agents: []manifest.AgentRef{{Name: "reviewer", Path: "./agents/reviewer.md"}},
	})

	report, err := InstallResourcesCommandAction(projectDir, globalPath, "agents", []string{"reviewer", "reviewer", "planner"})
	if err != nil {
		t.Fatalf("InstallResourcesCommandAction() error = %v", err)
	}

	if !reflect.DeepEqual(report.MutatedNames, []string{"planner"}) {
		t.Fatalf("mutated names = %#v, want %#v", report.MutatedNames, []string{"planner"})
	}
	if !reflect.DeepEqual(report.SkippedDuplicateNames, []string{"reviewer"}) {
		t.Fatalf("skipped duplicate names = %#v, want %#v", report.SkippedDuplicateNames, []string{"reviewer"})
	}
	if len(report.SkippedMissingNames) != 0 {
		t.Fatalf("skipped missing names = %#v, want empty", report.SkippedMissingNames)
	}

	m := readResourceActionManifest(t, projectDir)
	if len(m.Agents) != 2 || m.Agents[0].Name != "reviewer" || m.Agents[1].Name != "planner" {
		t.Fatalf("unexpected agents after install: %#v", m.Agents)
	}
}

func TestRemoveResourcesCommandAction_ReportsMutationsAndMissingSkips(t *testing.T) {
	projectDir := t.TempDir()
	writeResourceActionManifest(t, projectDir, &manifest.Manifest{
		Instructions: []manifest.InstructionRef{{Name: "standards", Path: "./instructions/standards.md"}},
	})

	report, err := RemoveResourcesCommandAction(projectDir, "instructions", []string{"ghost", "standards", "ghost"})
	if err != nil {
		t.Fatalf("RemoveResourcesCommandAction() error = %v", err)
	}

	if !reflect.DeepEqual(report.MutatedNames, []string{"standards"}) {
		t.Fatalf("mutated names = %#v, want %#v", report.MutatedNames, []string{"standards"})
	}
	if !reflect.DeepEqual(report.SkippedMissingNames, []string{"ghost"}) {
		t.Fatalf("skipped missing names = %#v, want %#v", report.SkippedMissingNames, []string{"ghost"})
	}
	if !reflect.DeepEqual(report.SkippedDuplicateNames, []string{"ghost"}) {
		t.Fatalf("skipped duplicate names = %#v, want %#v", report.SkippedDuplicateNames, []string{"ghost"})
	}

	m := readResourceActionManifest(t, projectDir)
	if len(m.Instructions) != 0 {
		t.Fatalf("expected instructions to be empty after remove, got %#v", m.Instructions)
	}
}

func TestInstallResourcesCommandAction_Prompts_DefaultLocalPath(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	globalPath := filepath.Join(configDir, "positive-vibes", "vibes.yaml")

	projectDir := t.TempDir()

	report, err := InstallResourcesCommandAction(projectDir, globalPath, "prompts", []string{"release"})
	if err != nil {
		t.Fatalf("InstallResourcesCommandAction() error = %v", err)
	}
	if !reflect.DeepEqual(report.MutatedNames, []string{"release"}) {
		t.Fatalf("mutated names = %#v, want %#v", report.MutatedNames, []string{"release"})
	}

	m := readResourceActionManifest(t, projectDir)
	if len(m.Prompts) != 1 || m.Prompts[0].Name != "release" || m.Prompts[0].Path != "./prompts/release.prompt.md" {
		t.Fatalf("unexpected prompts after install: %#v", m.Prompts)
	}
}

func TestRemoveResourcesCommandAction_Prompts(t *testing.T) {
	projectDir := t.TempDir()
	writeResourceActionManifest(t, projectDir, &manifest.Manifest{
		Prompts: []manifest.PromptRef{{Name: "release", Path: "./prompts/release.prompt.md"}},
	})

	report, err := RemoveResourcesCommandAction(projectDir, "prompts", []string{"release"})
	if err != nil {
		t.Fatalf("RemoveResourcesCommandAction() error = %v", err)
	}
	if !reflect.DeepEqual(report.MutatedNames, []string{"release"}) {
		t.Fatalf("mutated names = %#v, want %#v", report.MutatedNames, []string{"release"})
	}

	m := readResourceActionManifest(t, projectDir)
	if len(m.Prompts) != 0 {
		t.Fatalf("expected prompts to be empty after remove, got %#v", m.Prompts)
	}
}

func writeResourceActionManifest(t *testing.T, projectDir string, m *manifest.Manifest) {
	t.Helper()
	if err := manifest.SaveManifest(m, filepath.Join(projectDir, "vibes.yaml")); err != nil {
		t.Fatalf("SaveManifest() error = %v", err)
	}
}

func readResourceActionManifest(t *testing.T, projectDir string) *manifest.Manifest {
	t.Helper()
	m, err := manifest.LoadManifest(filepath.Join(projectDir, "vibes.yaml"))
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}
	return m
}

func TestEnsureRegistryRefInManifest_AddsMissingRegistryFromMerged(t *testing.T) {
	m := &manifest.Manifest{}
	merged := &manifest.Manifest{Registries: []manifest.RegistryRef{{
		Name:  "awesome-copilot",
		URL:   "https://github.com/github/awesome-copilot",
		Ref:   "latest",
		Paths: map[string]string{"skills": "skills/"},
	}}}

	ensureRegistryRefInManifest(m, merged, "awesome-copilot")
	if len(m.Registries) != 1 {
		t.Fatalf("expected one registry added, got %#v", m.Registries)
	}
	if m.Registries[0].Name != "awesome-copilot" {
		t.Fatalf("expected added registry awesome-copilot, got %#v", m.Registries[0])
	}
	if m.Registries[0].Paths["skills"] != "skills/" || m.Registries[0].Paths["instructions"] != "instructions/" || m.Registries[0].Paths["agents"] != "agents/" || m.Registries[0].Paths["prompts"] != "prompts/" {
		t.Fatalf("expected resource-specific default paths, got %#v", m.Registries[0].Paths)
	}
}

func TestPromoteLocalRegistriesToGlobalWithReport_PromotesValidLocalOnly(t *testing.T) {
	globalDir := t.TempDir()
	globalPath := filepath.Join(globalDir, "vibes.yaml")
	projectDir := t.TempDir()
	writeResourceActionManifest(t, projectDir, &manifest.Manifest{
		Registries: []manifest.RegistryRef{{
			Name:  "team-reg",
			URL:   "https://example.com/team.git",
			Ref:   "main",
			Paths: map[string]string{"skills": "skills/"},
		}},
	})

	report, err := PromoteLocalRegistriesToGlobalWithReport(projectDir, globalPath)
	if err != nil {
		t.Fatalf("PromoteLocalRegistriesToGlobalWithReport() error = %v", err)
	}
	if !reflect.DeepEqual(report.PromotedNames, []string{"team-reg"}) {
		t.Fatalf("promoted names = %#v, want %#v", report.PromotedNames, []string{"team-reg"})
	}

	g, err := manifest.LoadManifest(globalPath)
	if err != nil {
		t.Fatalf("LoadManifest(global) error = %v", err)
	}
	if len(g.Registries) != 1 || g.Registries[0].Name != "team-reg" {
		t.Fatalf("unexpected global registries after promote: %#v", g.Registries)
	}
}

func TestPromoteLocalRegistriesToGlobalWithReport_SkipsConflictingName(t *testing.T) {
	globalDir := t.TempDir()
	globalPath := filepath.Join(globalDir, "vibes.yaml")
	if err := manifest.SaveManifest(&manifest.Manifest{Registries: []manifest.RegistryRef{{
		Name: "team-reg", URL: "https://example.com/global.git", Ref: "main",
	}}}, globalPath); err != nil {
		t.Fatalf("SaveManifest(global) error = %v", err)
	}

	projectDir := t.TempDir()
	writeResourceActionManifest(t, projectDir, &manifest.Manifest{Registries: []manifest.RegistryRef{{
		Name: "team-reg", URL: "https://example.com/local.git", Ref: "dev",
	}}})

	report, err := PromoteLocalRegistriesToGlobalWithReport(projectDir, globalPath)
	if err != nil {
		t.Fatalf("PromoteLocalRegistriesToGlobalWithReport() error = %v", err)
	}
	if len(report.PromotedNames) != 0 {
		t.Fatalf("expected no promotions, got %#v", report.PromotedNames)
	}
	if len(report.Skipped) != 1 || report.Skipped[0].Reason != "name exists globally with different url" {
		t.Fatalf("unexpected skip report: %#v", report.Skipped)
	}
}

func TestPromoteLocalRegistriesToGlobalWithReport_SkipsInvalidLocalRegistry(t *testing.T) {
	globalDir := t.TempDir()
	globalPath := filepath.Join(globalDir, "vibes.yaml")
	projectDir := t.TempDir()
	writeResourceActionManifest(t, projectDir, &manifest.Manifest{Registries: []manifest.RegistryRef{{
		Name: "broken-reg", URL: "https://example.com/broken.git",
	}}})

	report, err := PromoteLocalRegistriesToGlobalWithReport(projectDir, globalPath)
	if err != nil {
		t.Fatalf("PromoteLocalRegistriesToGlobalWithReport() error = %v", err)
	}
	if len(report.PromotedNames) != 0 {
		t.Fatalf("expected no promotions, got %#v", report.PromotedNames)
	}
	if len(report.Skipped) != 1 || report.Skipped[0].Reason != "missing ref" {
		t.Fatalf("unexpected skip report: %#v", report.Skipped)
	}
}

func TestRemoveResourceItemsGlobalWithReport_RegistriesBlockedByLocalReferences(t *testing.T) {
	globalDir := t.TempDir()
	globalPath := filepath.Join(globalDir, "vibes.yaml")
	if err := manifest.SaveManifest(&manifest.Manifest{Registries: []manifest.RegistryRef{{
		Name: "team-reg", URL: "https://example.com/team.git", Ref: "main",
	}}}, globalPath); err != nil {
		t.Fatalf("SaveManifest(global) error = %v", err)
	}

	projectDir := t.TempDir()
	writeResourceActionManifest(t, projectDir, &manifest.Manifest{
		Instructions: []manifest.InstructionRef{{Name: "repo", Registry: "team-reg", Path: "instructions/repo.instructions.md"}},
	})

	_, err := RemoveResourceItemsGlobalWithReport(projectDir, globalPath, "registries", []string{"team-reg"})
	if err == nil {
		t.Fatal("expected delete to be blocked by local references")
	}
	if !strings.Contains(err.Error(), "referenced by local resources") {
		t.Fatalf("unexpected error: %v", err)
	}
}
