package ui

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestService_ListResources_ForwardsToBridge(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	expectedGlobalPath := filepath.Join(configDir, "positive-vibes", "vibes.yaml")

	var gotListAvailableProject string
	var gotListAvailableGlobal string
	var gotListAvailableKind string
	var gotListInstalledProject string
	var gotListInstalledGlobal string
	var gotListInstalledKind string

	svc, err := NewServiceWithBridge(".", ResourceServiceBridge{
		ListAvailableRows: func(projectDir, globalPath, kind string) ([]ResourceRow, error) {
			gotListAvailableProject = projectDir
			gotListAvailableGlobal = globalPath
			gotListAvailableKind = kind
			return []ResourceRow{{Name: "zeta"}}, nil
		},
		ListInstalledRows: func(projectDir, globalPath, kind string) ([]ResourceRow, error) {
			gotListInstalledProject = projectDir
			gotListInstalledGlobal = globalPath
			gotListInstalledKind = kind
			return []ResourceRow{{Name: "alpha", Installed: true}}, nil
		},
		ShowResource:     func(projectDir, globalPath, kind, name string) (ResourceDetail, error) { return ResourceDetail{}, nil },
		MergeRows:        func(available, installed []ResourceRow) []ResourceRow { return append(available, installed...) },
		InstallResources: func(projectDir, globalPath, kind string, names []string) error { return nil },
		RemoveResources:  func(projectDir, kind string, names []string) error { return nil },
	})
	if err != nil {
		t.Fatalf("NewServiceWithBridge() error = %v", err)
	}

	resources, err := svc.ListResources("skills")
	if err != nil {
		t.Fatalf("ListResources() error = %v", err)
	}

	if gotListAvailableProject != "." || gotListInstalledProject != "." {
		t.Fatalf("list forwarding projectDir mismatch: available=%q installed=%q", gotListAvailableProject, gotListInstalledProject)
	}
	if gotListAvailableGlobal != expectedGlobalPath || gotListInstalledGlobal != expectedGlobalPath {
		t.Fatalf("list forwarding globalPath mismatch: available=%q installed=%q want=%q", gotListAvailableGlobal, gotListInstalledGlobal, expectedGlobalPath)
	}
	if gotListAvailableKind != "skills" || gotListInstalledKind != "skills" {
		t.Fatalf("list forwarding kind mismatch: available=%q installed=%q", gotListAvailableKind, gotListInstalledKind)
	}

	if len(resources) != 2 {
		t.Fatalf("ListResources() len = %d, want 2", len(resources))
	}
	if resources[0].Name != "alpha" || !resources[0].Installed {
		t.Fatalf("ListResources()[0] = %+v", resources[0])
	}
	if resources[1].Name != "zeta" || resources[1].Installed {
		t.Fatalf("ListResources()[1] = %+v", resources[1])
	}
}

func TestService_ShowResource_ForwardsToBridge(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	expectedGlobalPath := filepath.Join(configDir, "positive-vibes", "vibes.yaml")

	var gotProjectDir string
	var gotGlobalPath string
	var gotKind string
	var gotName string

	svc, err := NewServiceWithBridge(".", ResourceServiceBridge{
		ListAvailableRows: func(projectDir, globalPath, kind string) ([]ResourceRow, error) { return nil, nil },
		ListInstalledRows: func(projectDir, globalPath, kind string) ([]ResourceRow, error) { return nil, nil },
		ShowResource: func(projectDir, globalPath, kind, name string) (ResourceDetail, error) {
			gotProjectDir = projectDir
			gotGlobalPath = globalPath
			gotKind = kind
			gotName = name
			return ResourceDetail{Kind: kind, Name: name, Installed: true}, nil
		},
		MergeRows:        func(available, installed []ResourceRow) []ResourceRow { return nil },
		InstallResources: func(projectDir, globalPath, kind string, names []string) error { return nil },
		RemoveResources:  func(projectDir, kind string, names []string) error { return nil },
	})
	if err != nil {
		t.Fatalf("NewServiceWithBridge() error = %v", err)
	}

	detail, err := svc.ShowResource("skills", "code-review")
	if err != nil {
		t.Fatalf("ShowResource() error = %v", err)
	}
	if gotProjectDir != "." || gotGlobalPath != expectedGlobalPath || gotKind != "skills" || gotName != "code-review" {
		t.Fatalf("show forwarding mismatch: project=%q global=%q kind=%q name=%q", gotProjectDir, gotGlobalPath, gotKind, gotName)
	}
	if detail.Name != "code-review" || detail.Kind != "skills" || !detail.Installed {
		t.Fatalf("ShowResource() detail = %+v", detail)
	}
}

func TestService_InstallResources_ForwardsToBridge(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	expectedGlobalPath := filepath.Join(configDir, "positive-vibes", "vibes.yaml")

	var gotProjectDir string
	var gotGlobalPath string
	var gotKind string
	var gotNames []string

	svc, err := NewServiceWithBridge(".", ResourceServiceBridge{
		ListAvailableRows: func(projectDir, globalPath, kind string) ([]ResourceRow, error) { return nil, nil },
		ListInstalledRows: func(projectDir, globalPath, kind string) ([]ResourceRow, error) { return nil, nil },
		ShowResource:      func(projectDir, globalPath, kind, name string) (ResourceDetail, error) { return ResourceDetail{}, nil },
		MergeRows:         func(available, installed []ResourceRow) []ResourceRow { return nil },
		InstallResources: func(projectDir, globalPath, kind string, names []string) error {
			gotProjectDir = projectDir
			gotGlobalPath = globalPath
			gotKind = kind
			gotNames = append([]string(nil), names...)
			return nil
		},
		RemoveResources: func(projectDir, kind string, names []string) error { return nil },
	})
	if err != nil {
		t.Fatalf("NewServiceWithBridge() error = %v", err)
	}

	if err := svc.InstallResources("skills", []string{"code-review", "", "code-review", "tdd"}); err != nil {
		t.Fatalf("InstallResources() error = %v", err)
	}

	if gotProjectDir != "." || gotGlobalPath != expectedGlobalPath || gotKind != "skills" {
		t.Fatalf("install forwarding mismatch: project=%q global=%q kind=%q", gotProjectDir, gotGlobalPath, gotKind)
	}
	if !reflect.DeepEqual(gotNames, []string{"code-review", "tdd"}) {
		t.Fatalf("install forwarded names = %#v, want %#v", gotNames, []string{"code-review", "tdd"})
	}
}

func TestService_RemoveResources_ForwardsToBridge(t *testing.T) {
	var gotProjectDir string
	var gotKind string
	var gotNames []string

	svc, err := NewServiceWithBridge(".", ResourceServiceBridge{
		ListAvailableRows: func(projectDir, globalPath, kind string) ([]ResourceRow, error) { return nil, nil },
		ListInstalledRows: func(projectDir, globalPath, kind string) ([]ResourceRow, error) { return nil, nil },
		ShowResource:      func(projectDir, globalPath, kind, name string) (ResourceDetail, error) { return ResourceDetail{}, nil },
		MergeRows:         func(available, installed []ResourceRow) []ResourceRow { return nil },
		InstallResources:  func(projectDir, globalPath, kind string, names []string) error { return nil },
		RemoveResources: func(projectDir, kind string, names []string) error {
			gotProjectDir = projectDir
			gotKind = kind
			gotNames = append([]string(nil), names...)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("NewServiceWithBridge() error = %v", err)
	}

	if err := svc.RemoveResources("skills", []string{"code-review", "", "code-review", "tdd"}); err != nil {
		t.Fatalf("RemoveResources() error = %v", err)
	}

	if gotProjectDir != "." || gotKind != "skills" {
		t.Fatalf("remove forwarding mismatch: project=%q kind=%q", gotProjectDir, gotKind)
	}
	if !reflect.DeepEqual(gotNames, []string{"code-review", "tdd"}) {
		t.Fatalf("remove forwarded names = %#v, want %#v", gotNames, []string{"code-review", "tdd"})
	}
}

func TestNewService_ErrWhenBridgeNotConfigured(t *testing.T) {
	resetResourceServiceBridgeForTesting()

	_, err := NewService(".")
	if err == nil {
		t.Fatalf("NewService() error = nil, want non-nil")
	}
	if err != ErrResourceServiceBridgeNotConfigured {
		t.Fatalf("NewService() error = %v, want %v", err, ErrResourceServiceBridgeNotConfigured)
	}
}

func TestNewService_ErrWhenBridgeNotConfigured_Deterministic(t *testing.T) {
	resetResourceServiceBridgeForTesting()

	_, err1 := NewService(".")
	_, err2 := NewService(".")

	if !errors.Is(err1, ErrResourceServiceBridgeNotConfigured) {
		t.Fatalf("first NewService() error = %v, want %v", err1, ErrResourceServiceBridgeNotConfigured)
	}
	if !errors.Is(err2, ErrResourceServiceBridgeNotConfigured) {
		t.Fatalf("second NewService() error = %v, want %v", err2, ErrResourceServiceBridgeNotConfigured)
	}
}
