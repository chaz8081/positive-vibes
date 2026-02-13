package ui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewRuntimeModel_WiresServiceCallbacks(t *testing.T) {
	originalFactory := newResourceService
	t.Cleanup(func() { newResourceService = originalFactory })

	var gotProjectDir string
	var showCalls int

	newResourceService = func(projectDir string) (*Service, error) {
		gotProjectDir = projectDir
		return newServiceWithDeps(serviceDeps{
			listAvailable: func(kind string) ([]ResourceRow, error) {
				return []ResourceRow{{Name: "alpha", Installed: true}}, nil
			},
			listInstalled: func(kind string) ([]ResourceRow, error) {
				return []ResourceRow{{Name: "alpha", Installed: true}}, nil
			},
			showDetail: func(kind, name string) (ResourceDetail, error) {
				showCalls++
				return ResourceDetail{Kind: kind, Name: name, Installed: true, Path: "skills/alpha.yaml"}, nil
			},
			mergeRows: func(available, _ []ResourceRow) []ResourceRow {
				return append([]ResourceRow(nil), available...)
			},
			install: func(kind string, names []string) error { return nil },
			remove:  func(kind string, names []string) error { return nil },
		}), nil
	}

	m := newRuntimeModel(".")
	if gotProjectDir != "." {
		t.Fatalf("newRuntimeModel projectDir = %q, want .", gotProjectDir)
	}
	if m.showResource == nil || m.listResources == nil || m.installResources == nil || m.removeResources == nil {
		t.Fatal("expected runtime model callbacks to be wired")
	}

	m.cursor = 0
	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.showDetailModal {
		t.Fatal("expected show modal to open from wired service")
	}
	if showCalls != 1 {
		t.Fatalf("expected one show call from service, got %d", showCalls)
	}
	if m.showDetail.Path != "skills/alpha.yaml" {
		t.Fatalf("expected show detail path from service, got %q", m.showDetail.Path)
	}
}

func TestNewRuntimeModel_ServiceInitFailureFallsBackGracefully(t *testing.T) {
	originalFactory := newResourceService
	t.Cleanup(func() { newResourceService = originalFactory })

	boom := errors.New("bridge missing")
	newResourceService = func(projectDir string) (*Service, error) {
		return nil, boom
	}

	m := newRuntimeModel(".")
	if !strings.Contains(m.statusMessage, "resource service unavailable") {
		t.Fatalf("expected unavailable status, got %q", m.statusMessage)
	}
	if !strings.Contains(m.statusMessage, boom.Error()) {
		t.Fatalf("expected init error in status, got %q", m.statusMessage)
	}
	if m.showResource != nil || m.listResources != nil || m.installResources != nil || m.removeResources != nil {
		t.Fatal("expected callbacks to stay nil on init failure")
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.showDetailModal {
		t.Fatal("expected fallback model to keep show modal functional")
	}
	if m.showDetail.Name == "" {
		t.Fatal("expected fallback show modal to render selected placeholder")
	}
}
