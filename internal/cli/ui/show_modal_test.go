package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestShowModal_Flow(t *testing.T) {
	rows := []ResourceRow{
		{Name: "alpha", Installed: false},
		{Name: "bravo", Installed: true},
	}

	var showCalls int
	var gotKind string
	var gotName string

	m := newModel()
	m.setRows(rows)
	m.cursor = 1
	m.showResource = func(kind, name string) (ResourceDetail, error) {
		showCalls++
		gotKind = kind
		gotName = name
		return ResourceDetail{
			Kind:        kind,
			Name:        name,
			Installed:   true,
			Registry:    "community",
			RegistryURL: "https://example.test/registry",
			Path:        "skills/bravo.yaml",
			Payload: map[string]any{
				"description": "bravo detail payload",
			},
		}, nil
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.showDetailModal {
		t.Fatal("expected show modal to open with enter")
	}
	if showCalls != 1 {
		t.Fatalf("expected one show call, got %d", showCalls)
	}
	if gotKind != "skills" {
		t.Fatalf("expected show kind skills, got %q", gotKind)
	}
	if gotName != "bravo" {
		t.Fatalf("expected show name bravo, got %q", gotName)
	}

	view := m.View()
	for _, want := range []string{
		"Source metadata",
		"Path/registry",
		"Content preview",
		"skills/bravo.yaml",
		"community",
		"bravo detail payload",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected show modal view to contain %q", want)
		}
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.showDetailModal {
		t.Fatal("expected show modal to close with esc")
	}
}
