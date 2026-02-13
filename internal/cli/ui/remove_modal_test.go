package ui

import (
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRemoveModal_Flow(t *testing.T) {
	state := []ResourceRow{
		{Name: "alpha", Installed: true},
		{Name: "bravo", Installed: false},
		{Name: "charlie", Installed: true},
	}

	var listCalls int
	var removeCalls int
	var gotRemoveKind string
	var gotRemoveNames []string

	m := newModel()
	m.listResources = func(kind string) ([]ResourceRow, error) {
		listCalls++
		rows := make([]ResourceRow, len(state))
		copy(rows, state)
		return rows, nil
	}
	m.removeResources = func(kind string, names []string) error {
		removeCalls++
		gotRemoveKind = kind
		gotRemoveNames = append([]string(nil), names...)

		toRemove := make(map[string]bool, len(names))
		for _, name := range names {
			toRemove[name] = true
		}
		for i := range state {
			if toRemove[state[i].Name] {
				state[i].Installed = false
			}
		}
		return nil
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if !m.showRemoveModal {
		t.Fatal("expected remove modal to open with r")
	}
	if len(m.removeChoices) != 2 {
		t.Fatalf("expected 2 installed choices, got %d", len(m.removeChoices))
	}
	if m.removeChoices[0].Name != "alpha" || m.removeChoices[1].Name != "charlie" {
		t.Fatalf("unexpected remove choices: %#v", m.removeChoices)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeySpace})
	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeySpace})

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.showRemoveModal {
		t.Fatal("expected remove modal to close after enter")
	}
	if removeCalls != 1 {
		t.Fatalf("expected one remove call, got %d", removeCalls)
	}
	if gotRemoveKind != "skills" {
		t.Fatalf("expected remove kind skills, got %q", gotRemoveKind)
	}
	if !reflect.DeepEqual(gotRemoveNames, []string{"alpha", "charlie"}) {
		t.Fatalf("expected selected names [alpha charlie], got %#v", gotRemoveNames)
	}
	if listCalls != 2 {
		t.Fatalf("expected list refresh before modal and after confirm, got %d calls", listCalls)
	}

	for _, row := range m.rows {
		if row.Name == "alpha" || row.Name == "charlie" {
			if row.Installed {
				t.Fatalf("expected removed rows to be uninstalled after refresh, got %#v", m.rows)
			}
		}
	}

	beforeCancelRemovals := removeCalls
	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.showRemoveModal {
		t.Fatal("expected remove modal to close with esc")
	}
	if removeCalls != beforeCancelRemovals {
		t.Fatalf("expected esc cancel to avoid remove, got %d calls", removeCalls)
	}
}
