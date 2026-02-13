package ui

import tea "github.com/charmbracelet/bubbletea"

type model struct{}

func (model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (model) View() string {
	return "positive-vibes TUI coming soon\n"
}

func Run() error {
	p := tea.NewProgram(model{})
	_, err := p.Run()
	return err
}
