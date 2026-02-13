package ui

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
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

func TestRemoveModal_HelpKeyIgnoredWhileOpen(t *testing.T) {
	m := newModel()
	m.listResources = func(kind string) ([]ResourceRow, error) {
		return []ResourceRow{{Name: "alpha", Installed: true}}, nil
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if !m.showRemoveModal {
		t.Fatal("expected remove modal to open")
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if m.showHelp {
		t.Fatal("expected help to remain closed while remove modal is open")
	}
	if !m.showRemoveModal {
		t.Fatal("expected remove modal to remain open when ? is pressed")
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.showRemoveModal {
		t.Fatal("expected esc to close remove modal")
	}
	if m.showHelp {
		t.Fatal("expected help to stay closed after esc")
	}
}

func TestRemoveModal_ConfirmWithoutSelection(t *testing.T) {
	var removeCalls int

	m := newModel()
	m.listResources = func(kind string) ([]ResourceRow, error) {
		return []ResourceRow{{Name: "alpha", Installed: true}}, nil
	}
	m.removeResources = func(kind string, names []string) error {
		removeCalls++
		return nil
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if !m.showRemoveModal {
		t.Fatal("expected modal to remain open when confirming with no selections")
	}
	if removeCalls != 0 {
		t.Fatalf("expected zero remove calls, got %d", removeCalls)
	}
	if !strings.Contains(m.statusMessage, "select at least one") {
		t.Fatalf("expected status message for empty selection, got %q", m.statusMessage)
	}
}

func TestRemoveModal_ErrorHandling(t *testing.T) {
	t.Run("list error", func(t *testing.T) {
		m := newModel()
		m.listResources = func(kind string) ([]ResourceRow, error) {
			return nil, errors.New("list exploded")
		}

		m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

		if m.showRemoveModal {
			t.Fatal("expected modal to stay closed when list fails")
		}
		if !strings.Contains(m.statusMessage, "list failed") {
			t.Fatalf("expected list error status, got %q", m.statusMessage)
		}
	})

	t.Run("remove error", func(t *testing.T) {
		m := newModel()
		m.listResources = func(kind string) ([]ResourceRow, error) {
			return []ResourceRow{{Name: "alpha", Installed: true}}, nil
		}
		m.removeResources = func(kind string, names []string) error {
			return errors.New("remove exploded")
		}

		m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
		m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeySpace})
		m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

		if !m.showRemoveModal {
			t.Fatal("expected modal to remain open when remove fails")
		}
		if !strings.Contains(m.statusMessage, "remove failed") {
			t.Fatalf("expected remove error status, got %q", m.statusMessage)
		}
	})

	t.Run("refresh error after successful remove", func(t *testing.T) {
		m := newModel()
		listCalls := 0
		m.listResources = func(kind string) ([]ResourceRow, error) {
			listCalls++
			if listCalls == 1 {
				return []ResourceRow{{Name: "alpha", Installed: true}}, nil
			}
			return nil, errors.New("refresh exploded")
		}
		m.removeResources = func(kind string, names []string) error {
			return nil
		}

		m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
		m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeySpace})
		m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

		if m.showRemoveModal {
			t.Fatal("expected modal to close after successful remove")
		}
		if !strings.Contains(m.statusMessage, "list failed") {
			t.Fatalf("expected refresh error status, got %q", m.statusMessage)
		}
		if strings.Contains(m.statusMessage, "removed:") {
			t.Fatalf("expected refresh error to not be overwritten by success status, got %q", m.statusMessage)
		}
	})
}

func TestRemoveModal_NoRemovableResources(t *testing.T) {
	m := newModel()
	m.listResources = func(kind string) ([]ResourceRow, error) {
		return []ResourceRow{{Name: "alpha", Installed: false}}, nil
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	if m.showRemoveModal {
		t.Fatal("expected modal to stay closed when nothing is removable")
	}
	if !strings.Contains(m.statusMessage, "no removable resources") {
		t.Fatalf("expected no-removable-resources status, got %q", m.statusMessage)
	}
}

func TestRemoveModal_UsesRemoveKeyBinding(t *testing.T) {
	m := newModel()
	m.keys.Remove = key.NewBinding(key.WithKeys("x"))
	m.listResources = func(kind string) ([]ResourceRow, error) {
		return []ResourceRow{{Name: "alpha", Installed: true}}, nil
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if m.showRemoveModal {
		t.Fatal("expected old hardcoded key r to not open modal")
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if !m.showRemoveModal {
		t.Fatal("expected configured remove key to open modal")
	}
}
