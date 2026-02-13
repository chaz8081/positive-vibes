package ui

import tea "github.com/charmbracelet/bubbletea"

func Run() error {
	p := tea.NewProgram(newModel())
	_, err := p.Run()
	return err
}
