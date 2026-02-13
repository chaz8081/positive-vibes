package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type railTab int

const (
	railSkills railTab = iota
	railInstructions
	railAgents
)

type model struct {
	activeRail railTab
	cursor     int
	showHelp   bool
	items      []string
	keys       keyMap
}

func newModel() model {
	return model{
		activeRail: railSkills,
		cursor:     0,
		showHelp:   false,
		items:      []string{"placeholder-1", "placeholder-2", "placeholder-3"},
		keys:       defaultKeyMap(),
	}
}

func (model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, m.keys.Help) {
			m.showHelp = true
			return m, nil
		}

		if m.showHelp {
			if key.Matches(msg, m.keys.CloseHelp) {
				m.showHelp = false
			}
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keys.LeftRail):
			m.activeRail = (m.activeRail + 2) % 3
		case key.Matches(msg, m.keys.RightRail):
			m.activeRail = (m.activeRail + 1) % 3
		case key.Matches(msg, m.keys.CursorDown):
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case key.Matches(msg, m.keys.CursorUp):
			if m.cursor > 0 {
				m.cursor--
			}
		}
	}

	return m, nil
}

func (m model) railTabs() []string {
	return []string{"skills", "instructions", "agents"}
}
