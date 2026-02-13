package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

func TestModel_CursorBoundaryNoOps(t *testing.T) {
	m := newModel()

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 0 {
		t.Fatalf("expected cursor to remain at 0 on up boundary, got %d", m.cursor)
	}

	m.cursor = len(m.items) - 1
	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != len(m.items)-1 {
		t.Fatalf("expected cursor to remain at max on down boundary, got %d", m.cursor)
	}
}

func TestModel_HelpModalSuppressesNavigation(t *testing.T) {
	m := newModel()
	m.activeRail = railInstructions
	m.cursor = 1

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if !m.showHelp {
		t.Fatal("expected help modal to be open")
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRight})
	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if m.activeRail != railInstructions {
		t.Fatalf("expected active rail unchanged while help open, got %v", m.activeRail)
	}
	if m.cursor != 1 {
		t.Fatalf("expected cursor unchanged while help open, got %d", m.cursor)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.showHelp {
		t.Fatal("expected help modal to close with esc")
	}
}

func TestModel_VimNavigationBindings(t *testing.T) {
	m := newModel()

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if m.activeRail != railInstructions {
		t.Fatalf("expected l to move rail right, got %v", m.activeRail)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if m.activeRail != railSkills {
		t.Fatalf("expected h to move rail left, got %v", m.activeRail)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.cursor != 1 {
		t.Fatalf("expected j to move cursor down, got %d", m.cursor)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.cursor != 0 {
		t.Fatalf("expected k to move cursor up, got %d", m.cursor)
	}
}

func TestView_ResponsiveLayoutFitsTerminalWidth(t *testing.T) {
	m := newModel()
	m.width = 72

	view := m.View()
	if lipgloss.Width(view) > m.width {
		t.Fatalf("expected view width <= %d, got %d", m.width, lipgloss.Width(view))
	}

	m.width = 48
	view = m.View()
	if lipgloss.Width(view) > m.width {
		t.Fatalf("expected narrow view width <= %d, got %d", m.width, lipgloss.Width(view))
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
