package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	xansi "github.com/charmbracelet/x/ansi"
	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/chaz8081/positive-vibes/pkg/schema"
)

func (m model) View() string {
	// Safe to call on value receiver: mutations stay local to this View() call
	// and do not escape to bubbletea's stored model.
	m.syncPreviewViewport()

	width := m.width
	if width < 1 {
		width = 96
	}

	var body string
	switch m.screen {
	case screenHome:
		body = m.renderHome(width)
	case screenDetail:
		body = m.renderDetailScreen(width)
	default:
		body = m.renderBrowser(width)
	}

	footerWidth := contentWidthForStyle(width, footerStyle)
	footer := footerStyle.Width(footerWidth).Render(singleLine(m.footerText(), footerWidth))
	modal := ""
	if m.showSearch {
		searchWidth := contentWidthForStyle(width, helpStyle)
		modal = m.renderSearchModal(searchWidth)
	}
	if m.showHelp {
		helpWidth := contentWidthForStyle(width, helpStyle)
		modal = m.renderHelp(helpWidth)
	}
	if m.showRegistryFix {
		fixWidth := contentWidthForStyle(width, helpStyle)
		modal = m.renderRegistryFixModal(fixWidth)
	}

	content := body

	if m.height > 1 {
		maxBody := m.height - 1
		if maxBody < 0 {
			maxBody = 0
		}
		if modal == "" {
			content = clampToHeight(content, maxBody)
		} else {
			modalHeight := lipgloss.Height(modal)
			bodyBudget := maxBody - modalHeight
			if bodyBudget < 0 {
				bodyBudget = 0
			}
			bodyClamped := clampToHeight(body, bodyBudget)
			if bodyClamped == "" {
				content = clampToHeight(modal, maxBody)
			} else {
				content = lipgloss.JoinVertical(lipgloss.Left, bodyClamped, "", modal)
			}
		}
		result := content + "\n" + footer
		for lipgloss.Height(result) > m.height {
			lines := strings.Split(content, "\n")
			if len(lines) == 0 {
				content = ""
				result = footer
				break
			}
			if len(lines) == 1 {
				content = ""
				result = footer
				break
			}
			content = strings.Join(lines[:len(lines)-1], "\n")
			result = content + "\n" + footer
		}
		if lipgloss.Height(result) < m.height {
			pad := m.height - lipgloss.Height(result)
			if content == "" {
				content = strings.Repeat("\n", pad)
			} else {
				content += strings.Repeat("\n", pad)
			}
			result = content + "\n" + footer
		}
		return result
	}

	return content + "\n" + footer
}

func (m model) renderHome(totalWidth int) string {
	leftWidth, rightWidth := m.twoPaneContentWidths(totalWidth, 3)

	leftLines := []string{cardTitleStyle.Render("CATEGORIES"), ""}
	kinds := m.homeKinds()
	for i, kind := range kinds {
		line := "  " + kind
		if i == m.homeCursor {
			line = highlightStyle.Render("> " + kind)
		}
		leftLines = append(leftLines, line)
	}

	activeKind := kinds[m.homeCursor]
	rightLines := []string{cardTitleStyle.Render("INSTALLED: " + strings.ToUpper(activeKind)), ""}
	installed := m.homeInstalled[activeKind]
	if len(installed) == 0 {
		rightLines = append(rightLines, mutedStyle.Render("(none installed)"))
	} else {
		limit := 10
		if len(installed) < limit {
			limit = len(installed)
		}
		for _, name := range installed[:limit] {
			rightLines = append(rightLines, "- "+name)
		}
		if len(installed) > limit {
			rightLines = append(rightLines, mutedStyle.Render(fmt.Sprintf("+%d more", len(installed)-limit)))
		}
	}

	header := highlightStyle.Render("positive-vibes") + "  " + mutedStyle.Render("Choose a category, then press Enter")
	header = singleLine(header, totalWidth)
	left := cardActiveStyle.Width(leftWidth).Render(strings.Join(leftLines, "\n"))
	right := panelStyle.Width(rightWidth).Render(strings.Join(rightLines, "\n"))
	return lipgloss.JoinVertical(lipgloss.Left, header, "", lipgloss.JoinHorizontal(lipgloss.Top, left, right))
}

func (m model) renderBrowser(totalWidth int) string {
	listWidth, previewWidth := m.browserWidths(totalWidth)
	focus := "LIST"
	if m.previewFocused {
		focus = "PREVIEW"
	}
	header := highlightStyle.Render(strings.ToUpper(m.activeKind())) + "  " + mutedStyle.Render("/ search  tab focus  ["+focus+"]")
	header = singleLine(header, totalWidth)
	left := m.renderList(listWidth, !m.previewFocused)
	right := m.renderDescriptionPane(previewWidth, m.previewFocused)
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	return lipgloss.JoinVertical(lipgloss.Left, header, "", body)
}

func (m model) renderDetailScreen(totalWidth int) string {
	header := highlightStyle.Render("DETAIL") + "  " + mutedStyle.Render("esc to go back")
	header = singleLine(header, totalWidth)
	if m.activeKind() != resourceKindSkills || len(m.detailFiles) == 0 {
		contentWidth := contentWidthForStyle(totalWidth, panelStyle)
		content := contentFromDetail(m.showDetail)
		if content == "" {
			content = mutedStyle.Render("(no content)")
		} else {
			content = m.previewContentView()
		}
		panel := panelStyle.Width(contentWidth).Render(content)
		return lipgloss.JoinVertical(lipgloss.Left, header, "", panel)
	}

	leftWidth, rightWidth := m.twoPaneContentWidths(totalWidth, 3)

	fileLines := []string{cardTitleStyle.Render("FILES"), ""}
	maxFileRows := m.previewViewportHeight() - 2
	if maxFileRows < 1 {
		maxFileRows = 1
	}
	start, end := centeredWindow(len(m.detailFiles), m.detailCursor, maxFileRows)
	nameWidth := leftWidth - 2
	if nameWidth < 1 {
		nameWidth = 1
	}
	for i := start; i < end; i++ {
		name := singleLine(m.detailFiles[i].Name, nameWidth)
		line := "  " + name
		if i == m.detailCursor {
			line = highlightStyle.Render("> " + name)
		}
		fileLines = append(fileLines, line)
	}

	content := ""
	if m.detailCursor >= 0 && m.detailCursor < len(m.detailFiles) {
		content = m.previewContentView()
	}
	if content == "" {
		content = mutedStyle.Render("(no preview)")
	}

	leftStyle := panelStyle
	rightStyle := panelStyle
	if m.previewFocused {
		rightStyle = cardActiveStyle
	} else {
		leftStyle = cardActiveStyle
	}
	left := leftStyle.Width(leftWidth).Render(strings.Join(fileLines, "\n"))
	right := rightStyle.Width(rightWidth).Render(content)
	return lipgloss.JoinVertical(lipgloss.Left, header, "", lipgloss.JoinHorizontal(lipgloss.Top, left, right))
}

func (m model) renderHelp(width int) string {
	text := []string{
		"Help",
		"- up/down or j/k: move/scroll vertically",
		"- left/right or h/l: switch list/preview focus",
		"- space: cycle scope (none/local/global/both)",
		"- registries L!: guided fix dialog",
		"- targets r: reset local override",
		"- J/K: scroll preview",
		"- ctrl+d/ctrl+u: half-page preview scroll",
		"- g/G: top/bottom preview",
		"- enter: multi-file skills detail (single-file stays inline)",
		"- /: fuzzy search",
		"- esc: back",
		"- " + m.quitKeyText() + ": quit",
		"- " + m.closeKeyText() + ": close help",
	}
	return helpStyle.Width(width).Render(strings.Join(text, "\n"))
}

func (m model) renderList(width int, focused bool) string {
	if len(m.filteredRows) == 0 {
		return panelStyle.Width(width).Render(mutedStyle.Render("No matches"))
	}

	lines := make([]string, 0, len(m.filteredRows))
	maxRows := m.previewViewportHeight()
	if maxRows < 1 {
		maxRows = 1
	}
	start, end := centeredWindow(len(m.filteredRows), m.cursor, maxRows)
	nameWidth := width - 2
	if nameWidth < 1 {
		nameWidth = 1
	}
	for i := start; i < end; i++ {
		row := m.filteredRows[i]
		status := " "
		if m.activeKind() == resourceKindRegistries && row.State == "registry_attention" {
			status = "L!"
		}
		switch row.InstallScope {
		case "local":
			if status == " " {
				status = "L"
			}
		case "global":
			status = "G"
		case "both":
			status = "B"
		}
		name := singleLine(fmt.Sprintf("%s %s", status, row.Name), nameWidth)
		name = colorizeScopePrefix(name, row.InstallScope)
		line := "  " + name
		if i == m.cursor {
			line = highlightStyle.Render("> " + name)
		} else {
			line = "  " + name
		}
		lines = append(lines, line)
	}

	style := panelStyle
	if focused {
		style = cardActiveStyle
	}
	return style.Width(width).Render(strings.Join(lines, "\n"))
}

func (m model) renderDescriptionPane(width int, focused bool) string {
	d := m.previewDetail
	if d.Name == "" {
		style := panelStyle
		if focused {
			style = cardActiveStyle
		}
		return style.Width(width).Render(mutedStyle.Render("Select a resource to view description"))
	}
	if m.screen == screenBrowser && (m.activeKind() == resourceKindInstructions || m.activeKind() == resourceKindPrompts || m.activeKind() == resourceKindAgents) {
		content := m.previewContentView()
		if strings.TrimSpace(xansi.Strip(content)) == "" {
			content = mutedStyle.Render("(no content)")
		}
		style := panelStyle
		if focused {
			style = cardActiveStyle
		}
		return style.Width(width).Render(content)
	}
	if m.screen == screenBrowser && m.activeKind() == resourceKindSkills {
		files := detailFilesFromDetail(d)
		if len(files) == 1 {
			content := m.previewContentView()
			if strings.TrimSpace(xansi.Strip(content)) == "" {
				content = mutedStyle.Render("(no content)")
			}
			style := panelStyle
			if focused {
				style = cardActiveStyle
			}
			return style.Width(width).Render(content)
		}
	}

	desc := detailDescription(d)
	if desc == "" {
		desc = mutedStyle.Render("(no description)")
	} else {
		desc = m.previewContentView()
	}
	lines := []string{
		cardTitleStyle.Render(d.Name),
		mutedStyle.Render("kind: " + d.Kind),
		"",
		desc,
	}
	style := panelStyle
	if focused {
		style = cardActiveStyle
	}
	return style.Width(width).Render(strings.Join(lines, "\n"))
}

func contentFromDetail(detail ResourceDetail) string {
	switch p := detail.Payload.(type) {
	case *schema.Skill:
		if p == nil {
			return ""
		}
		if p.Instructions != "" {
			return p.Instructions
		}
		return p.Description
	case map[string]any:
		if content, ok := p["content"].(string); ok {
			return strings.TrimSpace(content)
		}
		return fmt.Sprintf("%v", p)
	case manifest.InstructionRef:
		return p.Content
	}
	return ""
}

func (m model) renderSearchModal(width int) string {
	text := []string{
		"Search",
		"- type to filter name + description",
		"- enter: apply, " + m.closeKeyText() + ": cancel",
		"",
		highlightStyle.Render("> " + m.searchQuery),
		"",
		mutedStyle.Render("Search captures typing; q will not quit until you exit search."),
	}
	return helpStyle.Width(width).Render(strings.Join(text, "\n"))
}

func (m model) browserWidths(totalWidth int) (list int, preview int) {
	return m.twoPaneContentWidths(totalWidth, 3)
}

func (m model) twoPaneContentWidths(totalWidth int, leftDivisor int) (int, int) {
	if leftDivisor < 2 {
		leftDivisor = 2
	}
	const minLeft = 18
	frame := styleFrameWidth(panelStyle)
	available := totalWidth - (frame * 2)
	if available < 2 {
		available = 2
	}
	left := available / leftDivisor
	if left < minLeft {
		left = minLeft
	}
	if left > available-1 {
		left = available - 1
	}
	right := available - left
	if left < 1 {
		left = 1
		right = available - left
	}
	if right < 1 {
		right = 1
		left = available - right
		if left < 1 {
			left = 1
		}
	}
	return left, right
}

func (m model) previewContentView() string {
	return m.previewViewport.View()
}

func contentWidthForStyle(totalWidth int, style lipgloss.Style) int {
	content := totalWidth - styleFrameWidth(style)
	if content < 1 {
		return 1
	}
	return content
}

func (m model) footerText() string {
	searchKey := "/"
	if keys := m.keys.Search.Keys(); len(keys) > 0 {
		searchKey = keys[0]
	}

	backKey := m.closeKeyText()

	quitKey := m.quitKeyText()

	var text string
	if m.screen == screenHome {
		text = "up/down: choose category  enter: open browser  " + quitKey + ": quit  ?: help"
	} else if m.screen == screenDetail {
		text = backKey + ": back  " + quitKey + ": quit"
		if m.activeKind() == resourceKindSkills {
			text = "left/right or h/l: focus panes  up/down or j/k: move/scroll  tab: toggle focus  ctrl+d/u: half page  g/G: top/bottom  " + backKey + ": back"
		}
	} else {
		text = "left/right or h/l: focus panes  up/down or j/k: move/scroll  space: cycle scope  r(targets): reset  " + scopeLegendInline() + "  " + searchKey + ": search  " + backKey + ": home"
	}
	if m.statusMessage == "" {
		return text
	}
	return text + "  |  " + m.statusMessage
}

func (m model) closeKeyText() string {
	closeKey := "esc"
	if keys := m.keys.CloseHelp.Keys(); len(keys) > 0 {
		closeKey = keys[0]
	}
	return closeKey
}

func (m model) quitKeyText() string {
	quitKey := "q"
	if keys := m.keys.Quit.Keys(); len(keys) > 0 {
		quitKey = keys[0]
	}
	return quitKey
}

func styleFrameWidth(style lipgloss.Style) int {
	return lipgloss.Width(style.Width(1).Render("x")) - 1
}

func styleFrameHeight(style lipgloss.Style) int {
	return lipgloss.Height(style.Height(1).Render("x")) - 1
}

func clampToHeight(content string, maxHeight int) string {
	if maxHeight < 1 {
		return ""
	}
	lines := strings.Split(content, "\n")
	if len(lines) <= maxHeight {
		return content
	}
	return strings.Join(lines[:maxHeight], "\n")
}

func singleLine(text string, width int) string {
	text = strings.ReplaceAll(text, "\n", " ")
	if width < 1 {
		return ""
	}
	r := []rune(text)
	if len(r) <= width {
		return text
	}
	if width <= 3 {
		return string(r[:width])
	}
	return string(r[:width-3]) + "..."
}

func centeredWindow(total, cursor, maxRows int) (start, end int) {
	if total <= 0 {
		return 0, 0
	}
	if maxRows < 1 {
		maxRows = 1
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= total {
		cursor = total - 1
	}
	if total <= maxRows {
		return 0, total
	}
	start = cursor - (maxRows / 2)
	if start < 0 {
		start = 0
	}
	end = start + maxRows
	if end > total {
		end = total
		start = end - maxRows
	}
	return start, end
}

func colorizeScopePrefix(text, scope string) string {
	if text == "" {
		return text
	}
	runes := []rune(text)
	if len(runes) == 0 {
		return text
	}
	first := string(runes[0])
	rest := ""
	if len(runes) > 1 {
		rest = string(runes[1:])
	}
	switch scope {
	case "local":
		return "\x1b[1;38;5;78m" + first + "\x1b[0m" + rest
	case "global":
		return "\x1b[1;38;5;214m" + first + "\x1b[0m" + rest
	case "both":
		return "\x1b[1;38;5;159m" + first + "\x1b[0m" + rest
	default:
		return text
	}
}

func scopeLegendInline() string {
	return "[" + scopeLocalStyle.Render("L") + " local " + scopeGlobalStyle.Render("G") + " global " + scopeBothStyle.Render("B") + " both]"
}

func (m model) renderRegistryFixModal(width int) string {
	lines := []string{
		"Registry Fix",
		"- name: " + m.registryFixName,
		"- issue: " + m.registryFixReason,
		"",
	}
	for i, opt := range m.registryFixOptions {
		line := "  " + opt
		if i == m.registryFixCursor {
			line = highlightStyle.Render("> " + opt)
		}
		lines = append(lines, line)
	}
	lines = append(lines, "", mutedStyle.Render("enter: select  esc: cancel"))
	return helpStyle.Width(width).Render(strings.Join(lines, "\n"))
}

func colorizePreview(content string) string {
	lines := strings.Split(content, "\n")
	inCode := false
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trim, "```"):
			inCode = !inCode
			lines[i] = mutedStyle.Render(line)
		case inCode:
			lines[i] = codeStyle.Render(line)
		case strings.HasPrefix(trim, "#"):
			lines[i] = headingStyle.Render(line)
		case strings.HasPrefix(trim, "- ") || strings.HasPrefix(trim, "* "):
			lines[i] = bulletStyle.Render(line)
		case strings.Contains(trim, ":") && !strings.HasPrefix(trim, "http"):
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				lines[i] = keyStyle.Render(parts[0]+":") + parts[1]
			}
		}
	}
	return strings.Join(lines, "\n")
}
