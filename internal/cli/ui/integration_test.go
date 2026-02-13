package ui_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/chaz8081/positive-vibes/internal/cli"
	"github.com/chaz8081/positive-vibes/internal/cli/ui"
	"github.com/chaz8081/positive-vibes/internal/manifest"
)

func TestInstallParityHook_MatchesServiceMutation(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	globalPath := filepath.Join(configDir, "positive-vibes", "vibes.yaml")

	cliProject := t.TempDir()
	serviceProject := t.TempDir()

	writeManifest(t, cliProject, &manifest.Manifest{})
	writeManifest(t, serviceProject, &manifest.Manifest{})

	report, err := cli.InstallResourcesCommandAction(cliProject, globalPath, "agents", []string{"reviewer"})
	if err != nil {
		t.Fatalf("cli install action error = %v", err)
	}
	if !reflect.DeepEqual(report.MutatedNames, []string{"reviewer"}) {
		t.Fatalf("cli report mutated names = %#v, want %#v", report.MutatedNames, []string{"reviewer"})
	}

	svc, err := ui.NewServiceWithBridge(serviceProject, parityBridge())
	if err != nil {
		t.Fatalf("NewServiceWithBridge() error = %v", err)
	}
	if err := svc.InstallResources("agents", []string{"reviewer"}); err != nil {
		t.Fatalf("service InstallResources() error = %v", err)
	}

	cliManifest := readManifest(t, cliProject)
	serviceManifest := readManifest(t, serviceProject)
	if !reflect.DeepEqual(cliManifest, serviceManifest) {
		t.Fatalf("manifest mismatch\ncli: %#v\nsvc: %#v", cliManifest, serviceManifest)
	}
}

func TestRemoveParityHook_MatchesServiceMutation(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	initial := &manifest.Manifest{
		Agents: []manifest.AgentRef{
			{Name: "reviewer", Path: "./agents/reviewer.md"},
			{Name: "planner", Path: "./agents/planner.md"},
		},
	}

	cliProject := t.TempDir()
	serviceProject := t.TempDir()
	writeManifest(t, cliProject, initial)
	writeManifest(t, serviceProject, initial)

	report, err := cli.RemoveResourcesCommandAction(cliProject, "agents", []string{"planner"})
	if err != nil {
		t.Fatalf("cli remove action error = %v", err)
	}
	if !reflect.DeepEqual(report.MutatedNames, []string{"planner"}) {
		t.Fatalf("cli report mutated names = %#v, want %#v", report.MutatedNames, []string{"planner"})
	}

	svc, err := ui.NewServiceWithBridge(serviceProject, parityBridge())
	if err != nil {
		t.Fatalf("NewServiceWithBridge() error = %v", err)
	}
	if err := svc.RemoveResources("agents", []string{"planner"}); err != nil {
		t.Fatalf("service RemoveResources() error = %v", err)
	}

	cliManifest := readManifest(t, cliProject)
	serviceManifest := readManifest(t, serviceProject)
	if !reflect.DeepEqual(cliManifest, serviceManifest) {
		t.Fatalf("manifest mismatch\ncli: %#v\nsvc: %#v", cliManifest, serviceManifest)
	}
}

func TestShowParityHook_MatchesServiceDetail(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	globalPath := filepath.Join(configDir, "positive-vibes", "vibes.yaml")

	projectDir := t.TempDir()
	writeManifest(t, projectDir, &manifest.Manifest{
		Agents: []manifest.AgentRef{{Name: "reviewer", Path: "./agents/reviewer.md"}},
	})

	cliDetail, err := cli.ShowResourceCommandAction(projectDir, globalPath, "agents", "reviewer")
	if err != nil {
		t.Fatalf("cli show action error = %v", err)
	}

	svc, err := ui.NewServiceWithBridge(projectDir, parityBridge())
	if err != nil {
		t.Fatalf("NewServiceWithBridge() error = %v", err)
	}
	svcDetail, err := svc.ShowResource("agents", "reviewer")
	if err != nil {
		t.Fatalf("service ShowResource() error = %v", err)
	}

	if cliDetail.Kind != cli.ResourceType(svcDetail.Kind) ||
		cliDetail.Name != svcDetail.Name ||
		cliDetail.Installed != svcDetail.Installed ||
		cliDetail.Registry != svcDetail.Registry ||
		cliDetail.Path != svcDetail.Path {
		t.Fatalf("detail mismatch\ncli: %#v\nsvc: %#v", cliDetail, svcDetail)
	}
}

func TestInstallParityHook_InstructionsDuplicateInstallEdge(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	globalPath := filepath.Join(configDir, "positive-vibes", "vibes.yaml")

	initial := &manifest.Manifest{
		Instructions: []manifest.InstructionRef{{Name: "standards", Path: "./instructions/standards.md"}},
	}

	cliProject := t.TempDir()
	serviceProject := t.TempDir()
	writeManifest(t, cliProject, initial)
	writeManifest(t, serviceProject, initial)

	report, err := cli.InstallResourcesCommandAction(cliProject, globalPath, "instructions", []string{"standards", "standards", "onboarding"})
	if err != nil {
		t.Fatalf("cli install action error = %v", err)
	}
	if !reflect.DeepEqual(report.MutatedNames, []string{"onboarding"}) {
		t.Fatalf("cli report mutated names = %#v, want %#v", report.MutatedNames, []string{"onboarding"})
	}
	if !reflect.DeepEqual(report.SkippedDuplicateNames, []string{"standards"}) {
		t.Fatalf("cli report skipped duplicate names = %#v, want %#v", report.SkippedDuplicateNames, []string{"standards"})
	}

	svc, err := ui.NewServiceWithBridge(serviceProject, parityBridge())
	if err != nil {
		t.Fatalf("NewServiceWithBridge() error = %v", err)
	}
	if err := svc.InstallResources("instructions", []string{"standards", "onboarding"}); err != nil {
		t.Fatalf("service InstallResources() error = %v", err)
	}

	cliManifest := readManifest(t, cliProject)
	serviceManifest := readManifest(t, serviceProject)
	if !reflect.DeepEqual(cliManifest, serviceManifest) {
		t.Fatalf("manifest mismatch\ncli: %#v\nsvc: %#v", cliManifest, serviceManifest)
	}
	if len(cliManifest.Instructions) != 2 ||
		cliManifest.Instructions[0].Name != "standards" ||
		cliManifest.Instructions[1].Name != "onboarding" {
		t.Fatalf("unexpected instructions after duplicate install edge: %#v", cliManifest.Instructions)
	}
}

func TestRemoveParityHook_InstructionsNonexistentRemoveEdge(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	initial := &manifest.Manifest{
		Instructions: []manifest.InstructionRef{
			{Name: "standards", Path: "./instructions/standards.md"},
			{Name: "onboarding", Path: "./instructions/onboarding.md"},
		},
	}

	cliProject := t.TempDir()
	serviceProject := t.TempDir()
	writeManifest(t, cliProject, initial)
	writeManifest(t, serviceProject, initial)

	report, err := cli.RemoveResourcesCommandAction(cliProject, "instructions", []string{"ghost", "standards"})
	if err != nil {
		t.Fatalf("cli remove action error = %v", err)
	}
	if !reflect.DeepEqual(report.MutatedNames, []string{"standards"}) {
		t.Fatalf("cli report mutated names = %#v, want %#v", report.MutatedNames, []string{"standards"})
	}
	if !reflect.DeepEqual(report.SkippedMissingNames, []string{"ghost"}) {
		t.Fatalf("cli report skipped missing names = %#v, want %#v", report.SkippedMissingNames, []string{"ghost"})
	}

	svc, err := ui.NewServiceWithBridge(serviceProject, parityBridge())
	if err != nil {
		t.Fatalf("NewServiceWithBridge() error = %v", err)
	}
	if err := svc.RemoveResources("instructions", []string{"ghost", "standards"}); err != nil {
		t.Fatalf("service RemoveResources() error = %v", err)
	}

	cliManifest := readManifest(t, cliProject)
	serviceManifest := readManifest(t, serviceProject)
	if !reflect.DeepEqual(cliManifest, serviceManifest) {
		t.Fatalf("manifest mismatch\ncli: %#v\nsvc: %#v", cliManifest, serviceManifest)
	}
	if len(cliManifest.Instructions) != 1 || cliManifest.Instructions[0].Name != "onboarding" {
		t.Fatalf("unexpected instructions after nonexistent remove edge: %#v", cliManifest.Instructions)
	}
}

func parityBridge() ui.ResourceServiceBridge {
	return ui.ResourceServiceBridge{
		ListAvailableRows: func(projectDir, globalPath, kind string) ([]ui.ResourceRow, error) {
			items, err := cli.ListAvailableResourceItems(projectDir, globalPath, kind)
			if err != nil {
				return nil, err
			}
			rows := make([]ui.ResourceRow, 0, len(items))
			for _, item := range items {
				rows = append(rows, ui.ResourceRow{Name: item.Name, Installed: item.Installed})
			}
			return rows, nil
		},
		ListInstalledRows: func(projectDir, globalPath, kind string) ([]ui.ResourceRow, error) {
			items, err := cli.ListInstalledResourceItems(projectDir, globalPath, kind)
			if err != nil {
				return nil, err
			}
			rows := make([]ui.ResourceRow, 0, len(items))
			for _, item := range items {
				rows = append(rows, ui.ResourceRow{Name: item.Name, Installed: item.Installed})
			}
			return rows, nil
		},
		ShowResource: func(projectDir, globalPath, kind, name string) (ui.ResourceDetail, error) {
			detail, err := cli.ShowResourceDetail(projectDir, globalPath, kind, name)
			if err != nil {
				return ui.ResourceDetail{}, err
			}
			return ui.ResourceDetail{
				Kind:        string(detail.Kind),
				Name:        detail.Name,
				Installed:   detail.Installed,
				Registry:    detail.Registry,
				RegistryURL: detail.RegistryURL,
				Path:        detail.Path,
				Payload:     detail.Payload,
			}, nil
		},
		MergeRows: func(available, installed []ui.ResourceRow) []ui.ResourceRow {
			merged := cli.MergeResourceItems(toResourceItems(available), toResourceItems(installed))
			rows := make([]ui.ResourceRow, 0, len(merged))
			for _, item := range merged {
				rows = append(rows, ui.ResourceRow{Name: item.Name, Installed: item.Installed})
			}
			return rows
		},
		InstallResources: func(projectDir, globalPath, kind string, names []string) error {
			return cli.InstallResourceItems(projectDir, globalPath, kind, names)
		},
		RemoveResources: func(projectDir, kind string, names []string) error {
			return cli.RemoveResourceItems(projectDir, kind, names)
		},
	}
}

func toResourceItems(rows []ui.ResourceRow) []cli.ResourceItem {
	items := make([]cli.ResourceItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, cli.ResourceItem{Name: row.Name, Installed: row.Installed})
	}
	return items
}

func writeManifest(t *testing.T, projectDir string, m *manifest.Manifest) {
	t.Helper()
	if err := manifest.SaveManifest(m, filepath.Join(projectDir, "vibes.yaml")); err != nil {
		t.Fatalf("SaveManifest() error = %v", err)
	}
}

func readManifest(t *testing.T, projectDir string) *manifest.Manifest {
	t.Helper()
	m, err := manifest.LoadManifest(filepath.Join(projectDir, "vibes.yaml"))
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}
	return m
}
