package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	left := m.renderRail()
	center := m.renderList()
	right := m.renderPreview()
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, center, right)
	footer := footerStyle.Render("left/right: rail  up/down: move  ?: help")

	if m.showHelp {
		help := helpStyle.Render("Help\n- left/right: switch rail\n- up/down: move cursor\n- esc: close help")
		return lipgloss.JoinVertical(lipgloss.Left, body, footer, "", help)
	}

	return lipgloss.JoinVertical(lipgloss.Left, body, footer)
}

func (m model) renderRail() string {
	tabs := m.railTabs()
	lines := make([]string, 0, len(tabs))
	for i, tab := range tabs {
		line := "  " + tab
		if i == int(m.activeRail) {
			line = highlightStyle.Render("> " + tab)
		}
		lines = append(lines, line)
	}
	return panelStyle.Width(20).Render(strings.Join(lines, "\n"))
}

func (m model) renderList() string {
	lines := make([]string, 0, len(m.items))
	for i, item := range m.items {
		line := "  " + item
		if i == m.cursor {
			line = highlightStyle.Render("> " + item)
		}
		lines = append(lines, line)
	}
	return panelStyle.Width(34).Render(strings.Join(lines, "\n"))
}

func (model) renderPreview() string {
	return panelStyle.Width(34).Render(mutedStyle.Render("preview panel"))
}
