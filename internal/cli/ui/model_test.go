package ui

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	xansi "github.com/charmbracelet/x/ansi"
)

func TestModel_NavigationAndHelpModal(t *testing.T) {
	m := newModel()
	m.screen = screenHome

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
	if m.homeCursor != 1 {
		t.Fatalf("expected home cursor on instructions after right, got %d", m.homeCursor)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRight})
	if m.homeCursor != 2 {
		t.Fatalf("expected home cursor on agents after right, got %d", m.homeCursor)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRight})
	if m.homeCursor != 3 {
		t.Fatalf("expected home cursor to stay at last item after right, got %d", m.homeCursor)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != screenBrowser {
		t.Fatal("expected enter on home to open browser")
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
	m.screen = screenBrowser
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
	m.screen = screenHome

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if m.homeCursor != 1 {
		t.Fatalf("expected l to move home cursor right, got %d", m.homeCursor)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if m.homeCursor != 0 {
		t.Fatalf("expected h to move home cursor left, got %d", m.homeCursor)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.homeCursor != 1 {
		t.Fatalf("expected j to move home cursor down, got %d", m.homeCursor)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.homeCursor != 0 {
		t.Fatalf("expected k to move home cursor up, got %d", m.homeCursor)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != screenBrowser {
		t.Fatal("expected enter on home to open browser")
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.cursor != 1 {
		t.Fatalf("expected j to move cursor down in browser, got %d", m.cursor)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.cursor != 0 {
		t.Fatalf("expected k to move cursor up, got %d", m.cursor)
	}
}

func TestModel_RailSwitchRefreshesRowsImmediately(t *testing.T) {
	m := newModel()
	m.screen = screenHome

	var gotKinds []string
	m.listResources = func(kind string) ([]ResourceRow, error) {
		gotKinds = append(gotKinds, kind)
		switch kind {
		case resourceKindSkills:
			return []ResourceRow{{Name: "skill-1", Installed: true}}, nil
		case resourceKindInstructions:
			return []ResourceRow{{Name: "instruction-1", Installed: false}}, nil
		default:
			return nil, nil
		}
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRight})
	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.activeRail != railInstructions {
		t.Fatalf("expected active rail instructions after selecting home category, got %v", m.activeRail)
	}
	if len(m.rows) != 1 || m.rows[0].Name != "instruction-1" {
		t.Fatalf("expected instruction rows after right, got %#v", m.rows)
	}
	if !reflect.DeepEqual(m.items, []string{"instruction-1"}) {
		t.Fatalf("expected instruction items after right, got %#v", m.items)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.screen != screenHome {
		t.Fatalf("expected back to home screen, got %v", m.screen)
	}
	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.activeRail != railInstructions {
		t.Fatalf("expected active rail to stay selected category on re-enter, got %v", m.activeRail)
	}

	if len(gotKinds) < 2 {
		t.Fatalf("expected at least two refresh calls, got %#v", gotKinds)
	}
	if gotKinds[0] != resourceKindInstructions {
		t.Fatalf("expected first refresh for instructions, got %#v", gotKinds)
	}
}

func TestModel_BrowserLeftRightDoesNotSwitchCategory(t *testing.T) {
	m := newModel()
	m.screen = screenBrowser
	m.activeRail = railInstructions

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyLeft})
	if m.activeRail != railInstructions {
		t.Fatalf("expected left to not switch category in browser, got %v", m.activeRail)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRight})
	if m.activeRail != railInstructions {
		t.Fatalf("expected right to not switch category in browser, got %v", m.activeRail)
	}
}

func TestView_FooterPinnedToBottom(t *testing.T) {
	m := newModel()
	m.screen = screenHome
	m.height = 20

	view := m.View()
	if lipgloss.Height(view) < m.height {
		t.Fatalf("expected rendered view height >= %d, got %d", m.height, lipgloss.Height(view))
	}
	if !strings.Contains(view, m.footerText()) {
		t.Fatal("expected footer text present in rendered view")
	}
}

func TestModel_PreviewScrollWithVimMotions(t *testing.T) {
	m := newModel()
	m.screen = screenBrowser
	m.height = 8
	m.previewFocused = true
	m.previewDetail = ResourceDetail{Payload: map[string]any{"description": "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10"}}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'J'}})
	if m.previewOffset == 0 {
		t.Fatal("expected preview offset to increase with J")
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if m.previewOffset != 0 {
		t.Fatalf("expected preview offset reset at top with g, got %d", m.previewOffset)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if m.previewOffset == 0 {
		t.Fatal("expected preview offset moved to bottom with G")
	}
}

func TestView_ResponsiveLayoutFitsTerminalWidth(t *testing.T) {
	m := newModel()
	m.width = 72

	view := m.View()
	if lipgloss.Width(xansi.Strip(view)) > m.width {
		t.Fatalf("expected view width <= %d, got %d", m.width, lipgloss.Width(xansi.Strip(view)))
	}

	m.width = 48
	view = m.View()
	if lipgloss.Width(xansi.Strip(view)) > m.width {
		t.Fatalf("expected narrow view width <= %d, got %d", m.width, lipgloss.Width(xansi.Strip(view)))
	}
}

func TestModel_QuitKeyReturnsQuitCmd(t *testing.T) {
	m := newModel()

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected quit command for q key")
	}

	if _, ok := updated.(model); !ok {
		t.Fatalf("expected updated model type %T, got %T", m, updated)
	}

	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestModel_HomeScreenShowsInstalledNamesAndEnterOpensBrowser(t *testing.T) {
	m := newModel()
	m.screen = screenHome

	m.listResources = func(kind string) ([]ResourceRow, error) {
		switch kind {
		case resourceKindSkills:
			return []ResourceRow{{Name: "go-lint", Installed: true}, {Name: "fmt-fix", Installed: false}}, nil
		case resourceKindInstructions:
			return []ResourceRow{{Name: "ship-fast", Installed: true}}, nil
		case resourceKindAgents:
			return []ResourceRow{{Name: "code-reviewer", Installed: true}}, nil
		default:
			return nil, nil
		}
	}

	m.refreshHomeInstalled()
	if !reflect.DeepEqual(m.homeInstalled[resourceKindSkills], []string{"go-lint"}) {
		t.Fatalf("expected installed-only skills in home card, got %#v", m.homeInstalled[resourceKindSkills])
	}

	m.homeCursor = 2
	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != screenBrowser {
		t.Fatalf("expected browser screen after enter, got %v", m.screen)
	}
	if m.activeKind() != resourceKindAgents {
		t.Fatalf("expected agents browser after selecting agents card, got %s", m.activeKind())
	}
}

func TestModel_BrowserSearchFiltersByNameAndDescription(t *testing.T) {
	m := newModel()
	m.screen = screenBrowser
	m.activeRail = railSkills
	m.rows = []ResourceRow{
		{Name: "go-lint", Description: "lint your golang package", Installed: false},
		{Name: "fmt-fix", Description: "format and tidy code", Installed: false},
	}
	m.setRows(m.rows)

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !m.showSearch {
		t.Fatal("expected search modal to open with /")
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})

	if len(m.filteredRows) != 1 || m.filteredRows[0].Name != "go-lint" {
		t.Fatalf("expected fuzzy search to match description/name, got %#v", m.filteredRows)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.showSearch {
		t.Fatal("expected search modal to close with esc")
	}
}

func TestModel_SearchModeCapturesQWithoutQuitting(t *testing.T) {
	m := newModel()
	m.screen = screenBrowser
	m.setRows([]ResourceRow{{Name: "sql-helper", Description: "query sql"}})

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !m.showSearch {
		t.Fatal("expected search to open")
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	next, ok := updated.(model)
	if !ok {
		t.Fatalf("expected model type after update, got %T", updated)
	}
	if cmd != nil {
		t.Fatal("expected no quit command while typing in search mode")
	}
	if next.searchQuery != "q" {
		t.Fatalf("expected search query to capture q, got %q", next.searchQuery)
	}
	if !next.showSearch {
		t.Fatal("expected search to remain open")
	}
}

func TestView_SearchModalVisibleImmediatelyOnSlash(t *testing.T) {
	m := newModel()
	m.screen = screenBrowser
	m.width = 60
	m.height = 10
	rows := make([]ResourceRow, 0, 30)
	for i := 0; i < 30; i++ {
		rows = append(rows, ResourceRow{Name: fmt.Sprintf("item-%02d", i), Description: "desc"})
	}
	m.rows = rows
	m.setRows(rows)

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	view := xansi.Strip(m.View())
	if !strings.Contains(view, "Search") {
		t.Fatal("expected search modal to be visible immediately after pressing /")
	}
}

func TestModel_SearchBackspaceHandlesMultibyteRunes(t *testing.T) {
	m := newModel()
	m.screen = screenBrowser
	m.setRows([]ResourceRow{{Name: "alpha", Description: "desc"}})

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'é'}})
	if m.searchQuery != "é" {
		t.Fatalf("expected multibyte rune captured in query, got %q", m.searchQuery)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyBackspace})
	if m.searchQuery != "" {
		t.Fatalf("expected backspace to remove full rune, got %q", m.searchQuery)
	}
}

func TestModel_EscNavigatesBackAndExits(t *testing.T) {
	m := newModel()
	m.screen = screenBrowser

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.screen != screenHome {
		t.Fatalf("expected esc in browser to return home, got %v", m.screen)
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected esc on home to quit")
	}
	if _, ok := updated.(model); !ok {
		t.Fatalf("expected updated model type %T, got %T", m, updated)
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg from esc on home")
	}
}

func TestModel_EscPriority_SearchDetailBrowserHome(t *testing.T) {
	m := newModel()
	m.screen = screenDetail
	m.showDetailModal = true
	m.showSearch = true

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.showSearch {
		t.Fatal("expected first esc to close search")
	}
	if m.screen != screenDetail {
		t.Fatalf("expected to stay on detail after closing search, got %v", m.screen)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.screen != screenBrowser {
		t.Fatalf("expected second esc to go detail -> browser, got %v", m.screen)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.screen != screenHome {
		t.Fatalf("expected third esc to go browser -> home, got %v", m.screen)
	}
}

func TestModel_EscOnHome_Quits(t *testing.T) {
	m := newModel()
	m.screen = screenHome

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected esc on home to quit")
	}
	if _, ok := updated.(model); !ok {
		t.Fatalf("expected updated model type %T, got %T", m, updated)
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg from esc on home")
	}
}

func TestModel_BKeyNoLongerNavigatesBack(t *testing.T) {
	m := newModel()
	m.screen = screenBrowser

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	if m.screen != screenBrowser {
		t.Fatalf("expected b to no-op in browser, got screen %v", m.screen)
	}
}

func TestModel_BrowserFirstEntryShowsHintOnce(t *testing.T) {
	m := newModel()
	m.screen = screenHome

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != screenBrowser {
		t.Fatalf("expected enter from home to open browser, got %v", m.screen)
	}
	if !strings.Contains(m.statusMessage, "space cycles install scope") || !strings.Contains(m.statusMessage, "esc") {
		t.Fatalf("expected one-time browser hint, got %q", m.statusMessage)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if strings.Contains(m.statusMessage, "space cycles install scope") {
		t.Fatalf("expected browser hint to show only once, got %q", m.statusMessage)
	}
}

func TestModel_EscInBrowserWithHelpOpen_GoesHomeInOneKeypress(t *testing.T) {
	m := newModel()
	m.screen = screenBrowser
	m.showHelp = true

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.screen != screenHome {
		t.Fatalf("expected esc to go browser -> home even when help is open, got %v", m.screen)
	}
	if m.showHelp {
		t.Fatal("expected help to close when leaving browser with esc")
	}
}

func TestModel_EscOnHomeWithHelpOpen_QuitsInOneKeypress(t *testing.T) {
	m := newModel()
	m.screen = screenHome
	m.showHelp = true

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected esc on home to quit even when help is open")
	}
	if _, ok := updated.(model); !ok {
		t.Fatalf("expected updated model type %T, got %T", m, updated)
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg from esc on home with help open")
	}
}

func TestModel_DetailSkills_JKNavigateFileListEvenWhenPreviewFocused(t *testing.T) {
	m := newModel()
	m.screen = screenDetail
	m.showDetailModal = true
	m.activeRail = railSkills
	m.previewFocused = true
	m.detailFiles = []detailFile{{Name: "a.md", Content: "a"}, {Name: "b.md", Content: "b"}, {Name: "c.md", Content: "c"}}
	m.detailCursor = 0

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.detailCursor != 1 {
		t.Fatalf("expected j to move file cursor down in detail skills, got %d", m.detailCursor)
	}
	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.detailCursor != 0 {
		t.Fatalf("expected k to move file cursor up in detail skills, got %d", m.detailCursor)
	}
}

func TestView_PreviewWindowStartsAtTop(t *testing.T) {
	m := newModel()
	m.screen = screenDetail
	m.height = 10
	m.activeRail = railSkills
	m.detailFiles = []detailFile{{
		Name:    "SKILL.md",
		Content: "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9",
	}}

	view := m.View()
	if !strings.Contains(view, "line1") {
		t.Fatal("expected preview to start from top line")
	}
}

func TestView_LongPreviewDoesNotOverflowScreenHeight(t *testing.T) {
	m := newModel()
	m.screen = screenDetail
	m.height = 14
	m.width = 50
	m.activeRail = railSkills
	m.detailFiles = []detailFile{{
		Name:    "SKILL.md",
		Content: strings.Repeat("abcdefghijklmnopqrstuvwxyz0123456789", 30),
	}}

	view := m.View()
	if lipgloss.Height(view) > m.height {
		t.Fatalf("expected long preview to stay within screen height %d, got %d", m.height, lipgloss.Height(view))
	}
}

func TestView_DetailMarkdownLikePreview_DoesNotBleedIntoFooter(t *testing.T) {
	m := newModel()
	m.screen = screenDetail
	m.height = 12
	m.width = 44
	m.activeRail = railSkills
	m.detailFiles = []detailFile{{
		Name: "SKILL.md",
		Content: strings.Join([]string{
			"# azure-role-selector",
			"",
			"| role | scope | notes |",
			"| --- | --- | --- |",
			"| owner | /subscriptions/00000000-0000-0000-0000-000000000000 | https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles |",
			"",
			"```bash",
			"az role assignment create --assignee user@contoso.com --role \"Owner\" --scope /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/very-long-resource-group-name-for-overflow-testing",
			"```",
		}, "\n"),
	}}

	view := m.View()
	if lipgloss.Height(view) != m.height {
		t.Fatalf("expected rendered height to equal terminal height %d, got %d", m.height, lipgloss.Height(view))
	}
	if lipgloss.Width(xansi.Strip(view)) > m.width {
		t.Fatalf("expected rendered width <= %d, got %d", m.width, lipgloss.Width(xansi.Strip(view)))
	}
	lines := strings.Split(view, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[len(lines)-1]) == "" {
		t.Fatal("expected footer line to remain visible")
	}
}

func TestView_FooterPinnedToBottom_NarrowTerminal(t *testing.T) {
	m := newModel()
	m.screen = screenDetail
	m.height = 12
	m.width = 34
	m.activeRail = railSkills
	m.detailFiles = []detailFile{{
		Name:    "SKILL.md",
		Content: strings.Repeat("abcdefghijklmnopqrstuvwxyz0123456789", 20),
	}}

	view := m.View()
	if lipgloss.Height(view) != m.height {
		t.Fatalf("expected rendered view height to match terminal height %d, got %d", m.height, lipgloss.Height(view))
	}
	if lipgloss.Width(xansi.Strip(view)) > m.width {
		t.Fatalf("expected rendered view width <= %d, got %d", m.width, lipgloss.Width(xansi.Strip(view)))
	}
	lines := strings.Split(view, "\n")
	if len(lines) == 0 {
		t.Fatal("expected non-empty render")
	}
	lastLine := strings.TrimSpace(lines[len(lines)-1])
	if lastLine == "" {
		t.Fatal("expected footer text on final line")
	}
	if !strings.Contains(lastLine, ":") {
		t.Fatalf("expected final line to look like footer hints, got %q", lines[len(lines)-1])
	}
}

func TestView_ResizeSmaller_KeepsFooterVisible(t *testing.T) {
	m := newModel()
	m.screen = screenDetail
	m.activeRail = railSkills
	m.detailFiles = []detailFile{{
		Name:    "SKILL.md",
		Content: strings.Repeat("line\n", 200),
	}}

	m.width = 80
	m.height = 24
	_ = m.View()

	m.width = 40
	m.height = 10
	view := m.View()
	if lipgloss.Height(view) != m.height {
		t.Fatalf("expected resized render height to match terminal height %d, got %d", m.height, lipgloss.Height(view))
	}
	lines := strings.Split(view, "\n")
	if len(lines) == 0 {
		t.Fatal("expected non-empty render")
	}
	foundFooterHint := false
	start := len(lines) - 2
	if start < 0 {
		start = 0
	}
	for i := start; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed != "" && strings.Contains(trimmed, ":") {
			foundFooterHint = true
			break
		}
	}
	if !foundFooterHint {
		t.Fatal("expected footer to remain visible on final row after resize")
	}
}

func TestView_BrowserLongNamesDoNotOverflowWidth(t *testing.T) {
	m := newModel()
	m.screen = screenBrowser
	m.width = 50
	m.rows = []ResourceRow{{Name: strings.Repeat("verylongtoken", 12), Description: "d"}}
	m.setRows(m.rows)

	view := m.View()
	if lipgloss.Width(xansi.Strip(view)) > m.width {
		t.Fatalf("expected long names to stay within width %d, got %d", m.width, lipgloss.Width(xansi.Strip(view)))
	}
}

func TestView_DetailFileListLongNamesDoNotOverflowWidth(t *testing.T) {
	m := newModel()
	m.screen = screenDetail
	m.width = 52
	m.height = 18
	m.activeRail = railSkills
	m.detailFiles = []detailFile{{Name: strings.Repeat("verylongfilepathsegment", 6), Content: "hello"}}

	view := m.View()
	if lipgloss.Width(xansi.Strip(view)) > m.width {
		t.Fatalf("expected detail file list to stay within width %d, got %d", m.width, lipgloss.Width(xansi.Strip(view)))
	}
}

func TestView_DetailPreviewLongUnbrokenLineDoesNotOverflowWidth(t *testing.T) {
	m := newModel()
	m.screen = screenDetail
	m.width = 60
	m.height = 16
	m.activeRail = railSkills
	m.detailFiles = []detailFile{{
		Name:    "SKILL.md",
		Content: strings.Repeat("abcdefghijklmnopqrstuvwxyz0123456789", 40),
	}}

	view := m.View()
	if lipgloss.Width(xansi.Strip(view)) > m.width {
		t.Fatalf("expected detail preview to stay within width %d, got %d", m.width, lipgloss.Width(xansi.Strip(view)))
	}
}

func TestHardClampPreviewContentWidth_TruncatesAnsiSafe(t *testing.T) {
	input := "\x1b[31m" + strings.Repeat("x", 200) + "\x1b[0m"
	clamped := hardClampPreviewContentWidth(input, 20)
	if xansi.StringWidth(clamped) > 20 {
		t.Fatalf("expected clamped width <= 20, got %d", xansi.StringWidth(clamped))
	}
}

func TestView_DetailFileListKeepsCursorVisibleWithManyFiles(t *testing.T) {
	m := newModel()
	m.screen = screenDetail
	m.width = 70
	m.height = 12
	m.activeRail = railSkills
	files := make([]detailFile, 0, 40)
	for i := 0; i < 40; i++ {
		files = append(files, detailFile{Name: fmt.Sprintf("file-%02d.md", i), Content: "x"})
	}
	m.detailFiles = files
	m.detailCursor = 35

	view := xansi.Strip(m.View())
	if !strings.Contains(view, "file-35.md") {
		t.Fatal("expected focused file to remain visible in constrained detail file list")
	}
}

func TestView_BrowserListKeepsCursorVisibleWithManyRows(t *testing.T) {
	m := newModel()
	m.screen = screenBrowser
	m.width = 70
	m.height = 12
	rows := make([]ResourceRow, 0, 40)
	for i := 0; i < 40; i++ {
		rows = append(rows, ResourceRow{Name: fmt.Sprintf("item-%02d", i), Description: "d"})
	}
	m.rows = rows
	m.setRows(rows)
	m.cursor = 34

	view := xansi.Strip(m.View())
	if !strings.Contains(view, "item-34") {
		t.Fatal("expected focused browser row to remain visible in constrained list")
	}
}

func TestView_PreviewHighlighting_CodeFile(t *testing.T) {
	m := newModel()
	source := "package main\n\nfunc main() {}\n"
	rendered := m.renderPreviewContent(source, "main.go", false, 80)

	if rendered == source {
		t.Fatal("expected code preview rendering to transform source content")
	}
	if !strings.Contains(rendered, "package") || !strings.Contains(rendered, "func") {
		t.Fatalf("expected highlighted code to retain source tokens, got %q", rendered)
	}
}

func TestView_PreviewHighlighting_Markdown(t *testing.T) {
	m := newModel()
	source := "# Title\n\n```go\nfmt.Println(\"x\")\n```\n"
	rendered := m.renderPreviewContent(source, "README.md", false, 80)

	if strings.Contains(rendered, "```") {
		t.Fatalf("expected markdown renderer to consume fenced code markers, got %q", rendered)
	}
	if !strings.Contains(rendered, "Title") {
		t.Fatalf("expected markdown heading content to remain visible, got %q", rendered)
	}
}

func TestView_PreviewBinaryFallback_Message(t *testing.T) {
	m := newModel()
	rendered := m.renderPreviewContent("\x00\x01", "artifact.bin", true, 80)
	if rendered != "binary/unpreviewable file" {
		t.Fatalf("expected binary fallback message, got %q", rendered)
	}
}

func TestModel_DetailSkill_FileListPopulatesBeyondSkillMd(t *testing.T) {
	detail := ResourceDetail{Payload: map[string]any{
		"files": []map[string]any{
			{"name": "zeta.md", "content": "z", "binary": false},
			{"name": "SKILL.md", "content": "# skill", "binary": false},
			{"name": "examples/sample.md", "content": "sample", "binary": false},
		},
	}}

	files := detailFilesFromDetail(detail)
	if len(files) != 3 {
		t.Fatalf("expected 3 detail files, got %d", len(files))
	}
	if files[0].Name != "SKILL.md" || files[1].Name != "examples/sample.md" || files[2].Name != "zeta.md" {
		t.Fatalf("expected deterministic sorted file list, got %#v", files)
	}
}

func TestModel_DetailPreviewScroll_WorksWhileDetailModalOpen(t *testing.T) {
	m := newModel()
	m.screen = screenDetail
	m.showDetailModal = true
	m.previewFocused = true
	m.height = 10
	m.width = 60
	m.activeRail = railSkills
	m.detailFiles = []detailFile{{
		Name:    "SKILL.md",
		Content: strings.Repeat("line\n", 60),
	}}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'J'}})
	if m.previewOffset == 0 {
		t.Fatal("expected preview to scroll while detail modal is open")
	}
}

func TestView_BrowserUsesHomePaneRatio(t *testing.T) {
	m := newModel()
	m.screen = screenBrowser
	m.width = 96

	browserLeft, browserRight := m.browserWidths(m.width)
	homeLeft, homeRight := m.twoPaneContentWidths(m.width, 3)
	if browserLeft != homeLeft || browserRight != homeRight {
		t.Fatalf("expected browser pane widths to match home ratio, got browser=(%d,%d) home=(%d,%d)", browserLeft, browserRight, homeLeft, homeRight)
	}
}

func TestModel_HomeIncludesTargetsCategory(t *testing.T) {
	m := newModel()
	kinds := m.homeKinds()
	foundTargets := false
	foundRegistries := false
	for _, kind := range kinds {
		if kind == "targets" {
			foundTargets = true
		}
		if kind == "registries" {
			foundRegistries = true
		}
	}
	if !foundTargets || !foundRegistries {
		t.Fatalf("expected home categories to include targets, got %#v", kinds)
	}
}

func TestModel_BrowserSpaceTogglesSelectedItemInstallState(t *testing.T) {
	m := newModel()
	m.screen = screenBrowser
	m.activeRail = railSkills
	baseRows := []ResourceRow{{Name: "alpha", Installed: false, InstallScope: "none"}}
	m.setRows(baseRows)

	globalInstalled := false
	localInstalled := false
	m.listResources = func(kind string) ([]ResourceRow, error) {
		if kind != resourceKindSkills {
			t.Fatalf("expected skills kind, got %s", kind)
		}
		scope := "none"
		installed := false
		if globalInstalled && localInstalled {
			scope = "both"
			installed = true
		} else if globalInstalled {
			scope = "global"
			installed = true
		} else if localInstalled {
			scope = "local"
			installed = true
		}
		return []ResourceRow{{Name: "alpha", Installed: installed, InstallScope: scope}}, nil
	}
	m.installResources = func(kind string, names []string) error {
		if kind != resourceKindSkills {
			t.Fatalf("expected skills kind, got %s", kind)
		}
		if len(names) != 1 || names[0] != "alpha" {
			t.Fatalf("unexpected install names: %#v", names)
		}
		localInstalled = true
		return nil
	}
	m.installResourcesGlobal = func(kind string, names []string) error {
		if len(names) != 1 || names[0] != "alpha" {
			t.Fatalf("unexpected global install names: %#v", names)
		}
		globalInstalled = true
		return nil
	}
	m.removeResources = func(kind string, names []string) error {
		if kind != resourceKindSkills {
			t.Fatalf("expected skills kind, got %s", kind)
		}
		if len(names) != 1 || names[0] != "alpha" {
			t.Fatalf("unexpected remove names: %#v", names)
		}
		localInstalled = false
		return nil
	}
	m.removeResourcesGlobal = func(kind string, names []string) error {
		if len(names) != 1 || names[0] != "alpha" {
			t.Fatalf("unexpected global remove names: %#v", names)
		}
		globalInstalled = false
		return nil
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeySpace})
	if len(m.filteredRows) == 0 || m.filteredRows[0].InstallScope != "local" {
		t.Fatalf("expected none -> local on first space, got %#v", m.filteredRows)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeySpace})
	if len(m.filteredRows) == 0 || m.filteredRows[0].InstallScope != "global" {
		t.Fatalf("expected local -> global on second space, got %#v", m.filteredRows)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeySpace})
	if len(m.filteredRows) == 0 || m.filteredRows[0].InstallScope != "both" {
		t.Fatalf("expected global -> both on third space, got %#v", m.filteredRows)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeySpace})
	if len(m.filteredRows) == 0 || m.filteredRows[0].InstallScope != "none" {
		t.Fatalf("expected both -> none on fourth space, got %#v", m.filteredRows)
	}
}

func TestModel_BrowserSpaceDoesNotRemoveGlobalOnlyItem(t *testing.T) {
	m := newModel()
	m.screen = screenBrowser
	m.activeRail = railSkills
	m.setRows([]ResourceRow{{Name: "alpha", Installed: true, InstallScope: "global"}})

	removeCalls := 0
	m.removeResources = func(kind string, names []string) error {
		removeCalls++
		return nil
	}
	installCalls := 0
	m.installResources = func(kind string, names []string) error {
		installCalls++
		return nil
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeySpace})
	if removeCalls != 0 {
		t.Fatalf("expected no remove call for global-only item, got %d", removeCalls)
	}
	if installCalls != 1 {
		t.Fatalf("expected install call for global-only item, got %d", installCalls)
	}
}

func TestModel_BrowserSpaceOnGlobalOnlySkill_TogglesBetweenGlobalAndBoth(t *testing.T) {
	m := newModel()
	m.screen = screenBrowser
	m.activeRail = railSkills

	globalInstalled := true
	localInstalled := false
	rowsForState := func() []ResourceRow {
		scope := "none"
		installed := false
		if globalInstalled && localInstalled {
			scope = "both"
			installed = true
		} else if globalInstalled {
			scope = "global"
			installed = true
		} else if localInstalled {
			scope = "local"
			installed = true
		}
		return []ResourceRow{{Name: "alpha", Installed: installed, InstallScope: scope}}
	}
	m.setRows(rowsForState())

	m.listResources = func(kind string) ([]ResourceRow, error) {
		return rowsForState(), nil
	}
	m.installResources = func(kind string, names []string) error {
		localInstalled = true
		return nil
	}
	m.removeResources = func(kind string, names []string) error {
		localInstalled = false
		return nil
	}
	m.removeResourcesGlobal = func(kind string, names []string) error {
		globalInstalled = false
		return nil
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeySpace})
	if len(m.filteredRows) == 0 || m.filteredRows[0].InstallScope != "both" {
		t.Fatalf("expected global-only space toggle to install locally (both), got %#v", m.filteredRows)
	}

	m = updateWithKey(t, m, tea.KeyMsg{Type: tea.KeySpace})
	if len(m.filteredRows) == 0 || m.filteredRows[0].InstallScope != "none" {
		t.Fatalf("expected second space toggle to remove both layers back to none, got %#v", m.filteredRows)
	}
}

func TestColorizeScopePrefix_PreservesScopedMarkers(t *testing.T) {
	local := colorizeScopePrefix("L alpha", "local")
	if !strings.Contains(xansi.Strip(local), "L alpha") {
		t.Fatalf("expected local scope prefix to preserve token, got %q", local)
	}
	global := colorizeScopePrefix("G beta", "global")
	if !strings.Contains(xansi.Strip(global), "G beta") {
		t.Fatalf("expected global scope prefix to preserve token, got %q", global)
	}
	both := colorizeScopePrefix("B gamma", "both")
	if !strings.Contains(xansi.Strip(both), "B gamma") {
		t.Fatalf("expected both scope prefix to preserve token, got %q", both)
	}
	none := colorizeScopePrefix("  delta", "none")
	if none != "  delta" {
		t.Fatalf("expected none scope prefix to remain unchanged, got %q", none)
	}
}

func TestFooterText_IncludesScopeLegend(t *testing.T) {
	m := newModel()
	m.screen = screenBrowser
	text := m.footerText()
	if !strings.Contains(text, "L") || !strings.Contains(text, "G") || !strings.Contains(text, "B") {
		t.Fatalf("expected footer to include scope legend tokens, got %q", text)
	}
	if !strings.Contains(text, "local") || !strings.Contains(text, "global") || !strings.Contains(text, "both") {
		t.Fatalf("expected footer legend labels, got %q", text)
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
