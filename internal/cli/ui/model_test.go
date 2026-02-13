package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModel_NavigationAndHelpModal(t *testing.T) {
	m := newModel()

	if m.activeRail != railSkills {
		t.Fatalf("expected initial rail tab skills, got %v", m.activeRail)
	}
	if m.cursor != 0 {
		t.Fatalf("expected initial cursor 0, got %d", m.cursor)
	}
	if m.showHelp {
		t.Fatal("expected help modal to start closed")
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRight})
	if m.activeRail != railInstructions {
		t.Fatalf("expected rail tab instructions after right, got %v", m.activeRail)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRight})
	if m.activeRail != railAgents {
		t.Fatalf("expected rail tab agents after right, got %v", m.activeRail)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRight})
	if m.activeRail != railSkills {
		t.Fatalf("expected rail tab wrap to skills after right, got %v", m.activeRail)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 1 {
		t.Fatalf("expected cursor 1 after down, got %d", m.cursor)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 2 {
		t.Fatalf("expected cursor 2 after down, got %d", m.cursor)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 1 {
		t.Fatalf("expected cursor 1 after up, got %d", m.cursor)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if !m.showHelp {
		t.Fatal("expected help modal to open with ?")
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.showHelp {
		t.Fatal("expected help modal to close with esc")
	}
}

func updateWithKey(t *testing.T, m model, msg tea.KeyMsg) model {
	t.Helper()

	updated, _ := m.Update(msg)
	next, ok := updated.(model)
	if !ok {
		t.Fatalf("expected updated model type %T, got %T", m, updated)
	}

	return next
}
