package ui

import "github.com/charmbracelet/lipgloss"

var (
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)
	highlightStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("111"))
	mutedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	footerStyle    = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("238")).
			Padding(0, 1)
	helpStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("111")).
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)
	cardTitleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("117"))
	headingStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("183"))
	bulletStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("111"))
	keyStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("144"))
	codeStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("151"))
	scopeLocalStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("78"))
	scopeGlobalStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	scopeBothStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("159"))
	cardActiveStyle  = lipgloss.NewStyle().
				Border(lipgloss.ThickBorder()).
				BorderForeground(lipgloss.Color("180")).
				Background(lipgloss.Color("236")).
				Padding(0, 1)
)
