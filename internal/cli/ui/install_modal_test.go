package ui

import (
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestInstallModal_Flow(t *testing.T) {
	state := []ResourceRow{
		{Name: "alpha", Installed: false},
		{Name: "bravo", Installed: true},
		{Name: "charlie", Installed: false},
	}

	var listCalls int
	var installCalls int
	var gotInstallKind string
	var gotInstallNames []string

	m := newModel()
	m.listResources = func(kind string) ([]ResourceRow, error) {
		listCalls++
		rows := make([]ResourceRow, len(state))
		copy(rows, state)
		return rows, nil
	}
	m.installResources = func(kind string, names []string) error {
		installCalls++
		gotInstallKind = kind
		gotInstallNames = append([]string(nil), names...)

		toInstall := make(map[string]bool, len(names))
		for _, name := range names {
			toInstall[name] = true
		}
		for i := range state {
			if toInstall[state[i].Name] {
				state[i].Installed = true
			}
		}
		return nil
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if !m.showInstallModal {
		t.Fatal("expected install modal to open with i")
	}
	if len(m.installChoices) != 2 {
		t.Fatalf("expected 2 not-installed choices, got %d", len(m.installChoices))
	}
	if m.installChoices[0].Name != "alpha" || m.installChoices[1].Name != "charlie" {
		t.Fatalf("unexpected install choices: %#v", m.installChoices)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeySpace})
	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeySpace})

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.showInstallModal {
		t.Fatal("expected install modal to close after enter")
	}
	if installCalls != 1 {
		t.Fatalf("expected one install call, got %d", installCalls)
	}
	if gotInstallKind != "skills" {
		t.Fatalf("expected install kind skills, got %q", gotInstallKind)
	}
	if !reflect.DeepEqual(gotInstallNames, []string{"alpha", "charlie"}) {
		t.Fatalf("expected selected names [alpha charlie], got %#v", gotInstallNames)
	}
	if listCalls != 2 {
		t.Fatalf("expected list refresh before modal and after confirm, got %d calls", listCalls)
	}

	for _, row := range m.rows {
		if !row.Installed {
			t.Fatalf("expected all rows installed after refresh, got %#v", m.rows)
		}
	}

	beforeCancelInstalls := installCalls
	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.showInstallModal {
		t.Fatal("expected install modal to close with esc")
	}
	if installCalls != beforeCancelInstalls {
		t.Fatalf("expected esc cancel to avoid install, got %d calls", installCalls)
	}
}
