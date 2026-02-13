package ui

import (
	"testing"
)

func TestService_ShowResource_UsesBridge(t *testing.T) {
	called := false
	svc, err := NewServiceWithBridge(".", ResourceServiceBridge{
		ListAvailableRows: func(projectDir, globalPath, kind string) ([]ResourceRow, error) { return nil, nil },
		ListInstalledRows: func(projectDir, globalPath, kind string) ([]ResourceRow, error) { return nil, nil },
		ShowResource: func(projectDir, globalPath, kind, name string) (ResourceDetail, error) {
			called = true
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
	if !called {
		t.Fatalf("ShowResource() did not call bridge")
	}
	if detail.Name != "code-review" || detail.Kind != "skills" || !detail.Installed {
		t.Fatalf("ShowResource() detail = %+v", detail)
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
