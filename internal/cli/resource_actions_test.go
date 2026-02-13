package cli

import (
	"path/filepath"
	"reflect"
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
