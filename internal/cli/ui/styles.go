package ui

import "github.com/charmbracelet/lipgloss"

var (
	panelStyle     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	highlightStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("36"))
	mutedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	footerStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	helpStyle      = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1)
)
