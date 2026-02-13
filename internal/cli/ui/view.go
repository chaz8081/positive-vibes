package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	width := m.width
	if width < 1 {
		width = 96
	}

	railWidth, listWidth, previewWidth, stacked := m.layoutWidths(width)

	var body string
	if stacked {
		body = lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderRail(railWidth),
			m.renderList(listWidth),
			m.renderPreview(previewWidth),
		)
	} else {
		body = lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.renderRail(railWidth),
			m.renderList(listWidth),
			m.renderPreview(previewWidth),
		)
	}

	footerWidth := contentWidthForStyle(width, footerStyle)
	footer := footerStyle.Width(footerWidth).Render(m.footerText())

	if m.showInstallModal {
		installWidth := contentWidthForStyle(width, helpStyle)
		install := m.renderInstallModal(installWidth)
		return lipgloss.JoinVertical(lipgloss.Left, body, footer, "", install)
	}

	if m.showRemoveModal {
		removeWidth := contentWidthForStyle(width, helpStyle)
		remove := m.renderRemoveModal(removeWidth)
		return lipgloss.JoinVertical(lipgloss.Left, body, footer, "", remove)
	}

	if m.showHelp {
		helpWidth := contentWidthForStyle(width, helpStyle)
		help := helpStyle.Width(helpWidth).Render("Help\n- left/right: switch rail\n- up/down: move cursor\n- esc: close help")
		return lipgloss.JoinVertical(lipgloss.Left, body, footer, "", help)
	}

	return lipgloss.JoinVertical(lipgloss.Left, body, footer)
}

func (m model) renderRail(width int) string {
	tabs := m.railTabs()
	lines := make([]string, 0, len(tabs))
	for i, tab := range tabs {
		line := "  " + tab
		if i == int(m.activeRail) {
			line = highlightStyle.Render("> " + tab)
		}
		lines = append(lines, line)
	}
	return panelStyle.Width(width).Render(strings.Join(lines, "\n"))
}

func (m model) renderList(width int) string {
	lines := make([]string, 0, len(m.items))
	for i, item := range m.items {
		line := "  " + item
		if i == m.cursor {
			line = highlightStyle.Render("> " + item)
		}
		lines = append(lines, line)
	}
	return panelStyle.Width(width).Render(strings.Join(lines, "\n"))
}

func (model) renderPreview(width int) string {
	return panelStyle.Width(width).Render(mutedStyle.Render("preview panel"))
}

func (m model) renderInstallModal(width int) string {
	lines := []string{"Install resources", "- space: toggle  enter: confirm  esc: cancel", ""}
	if len(m.installChoices) == 0 {
		lines = append(lines, mutedStyle.Render("No resources available to install."))
		return helpStyle.Width(width).Render(strings.Join(lines, "\n"))
	}

	for i, row := range m.installChoices {
		marker := "[ ]"
		if m.installSelected[row.Name] {
			marker = "[x]"
		}
		line := marker + " " + row.Name
		if i == m.installCursor {
			line = highlightStyle.Render("> " + line)
		} else {
			line = "  " + line
		}
		lines = append(lines, line)
	}

	return helpStyle.Width(width).Render(strings.Join(lines, "\n"))
}

func (m model) renderRemoveModal(width int) string {
	lines := []string{"Remove resources", "- space: toggle  enter: confirm  esc: cancel", ""}
	if len(m.removeChoices) == 0 {
		lines = append(lines, mutedStyle.Render("No resources available to remove."))
		return helpStyle.Width(width).Render(strings.Join(lines, "\n"))
	}

	for i, row := range m.removeChoices {
		marker := "[ ]"
		if m.removeSelected[row.Name] {
			marker = "[x]"
		}
		line := marker + " " + row.Name
		if i == m.removeCursor {
			line = highlightStyle.Render("> " + line)
		} else {
			line = "  " + line
		}
		lines = append(lines, line)
	}

	return helpStyle.Width(width).Render(strings.Join(lines, "\n"))
}

func (m model) layoutWidths(totalWidth int) (rail int, list int, preview int, stacked bool) {
	frame := styleFrameWidth(panelStyle)
	if totalWidth < (frame*3)+30 {
		content := totalWidth - frame
		if content < 1 {
			content = 1
		}
		return content, content, content, true
	}

	available := totalWidth - (frame * 3)
	rail = available / 5
	if rail < 12 {
		rail = 12
	}

	remaining := available - rail
	list = remaining / 2
	preview = remaining - list

	if list < 10 || preview < 10 {
		content := totalWidth - frame
		if content < 1 {
			content = 1
		}
		return content, content, content, true
	}

	return rail, list, preview, false
}

func contentWidthForStyle(totalWidth int, style lipgloss.Style) int {
	content := totalWidth - styleFrameWidth(style)
	if content < 1 {
		return 1
	}
	return content
}

func (m model) footerText() string {
	installKey := "i"
	if keys := m.keys.Install.Keys(); len(keys) > 0 {
		installKey = keys[0]
	}

	removeKey := "r"
	if keys := m.keys.Remove.Keys(); len(keys) > 0 {
		removeKey = keys[0]
	}

	text := "left/right: rail  up/down: move  " + installKey + ": install  " + removeKey + ": remove  ?: help"
	if m.statusMessage == "" {
		return text
	}
	return text + "  |  " + m.statusMessage
}

func styleFrameWidth(style lipgloss.Style) int {
	return lipgloss.Width(style.Width(1).Render("x")) - 1
}
