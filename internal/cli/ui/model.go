package ui

import (
	"fmt"
	"strings"

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
	activeRail       railTab
	cursor           int
	showHelp         bool
	showInstallModal bool
	showRemoveModal  bool
	statusMessage    string
	width            int
	rows             []ResourceRow
	items            []string
	keys             keyMap

	installCursor   int
	installChoices  []ResourceRow
	installSelected map[string]bool
	removeCursor    int
	removeChoices   []ResourceRow
	removeSelected  map[string]bool

	listResources    func(kind string) ([]ResourceRow, error)
	installResources func(kind string, names []string) error
	removeResources  func(kind string, names []string) error
}

func newModel() model {
	rows := []ResourceRow{
		{Name: "placeholder-1"},
		{Name: "placeholder-2"},
		{Name: "placeholder-3"},
	}

	return model{
		activeRail:       railSkills,
		cursor:           0,
		showHelp:         false,
		showInstallModal: false,
		showRemoveModal:  false,
		width:            96,
		rows:             rows,
		items:            []string{"placeholder-1", "placeholder-2", "placeholder-3"},
		keys:             defaultKeyMap(),
		installSelected:  make(map[string]bool),
		removeSelected:   make(map[string]bool),
	}
}

func (model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tea.KeyMsg:
		if m.showRemoveModal {
			switch {
			case key.Matches(msg, m.keys.CloseHelp):
				m.closeRemoveModal()
			case key.Matches(msg, m.keys.CursorDown):
				if m.removeCursor < len(m.removeChoices)-1 {
					m.removeCursor++
				}
			case key.Matches(msg, m.keys.CursorUp):
				if m.removeCursor > 0 {
					m.removeCursor--
				}
			case msg.Type == tea.KeySpace:
				m.toggleRemoveSelection()
			case msg.Type == tea.KeyEnter:
				m.confirmRemoveSelection()
			}
			return m, nil
		}

		if m.showInstallModal {
			switch {
			case key.Matches(msg, m.keys.CloseHelp):
				m.closeInstallModal()
			case key.Matches(msg, m.keys.CursorDown):
				if m.installCursor < len(m.installChoices)-1 {
					m.installCursor++
				}
			case key.Matches(msg, m.keys.CursorUp):
				if m.installCursor > 0 {
					m.installCursor--
				}
			case msg.Type == tea.KeySpace:
				m.toggleInstallSelection()
			case msg.Type == tea.KeyEnter:
				m.confirmInstallSelection()
			}
			return m, nil
		}

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

		if key.Matches(msg, m.keys.Install) {
			m.openInstallModal()
			return m, nil
		}

		if key.Matches(msg, m.keys.Remove) {
			m.openRemoveModal()
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keys.LeftRail):
			m.activeRail = m.wrapRail(-1)
		case key.Matches(msg, m.keys.RightRail):
			m.activeRail = m.wrapRail(1)
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

func (m *model) openInstallModal() {
	if !m.refreshRowsForActiveRail() {
		m.closeInstallModal()
		return
	}

	m.installChoices = m.installChoices[:0]
	for _, row := range m.rows {
		if !row.Installed {
			m.installChoices = append(m.installChoices, row)
		}
	}

	if len(m.installChoices) == 0 {
		m.statusMessage = fmt.Sprintf("no installable resources in %s", m.activeKind())
		m.closeInstallModal()
		return
	}

	m.installCursor = 0
	m.installSelected = make(map[string]bool)
	m.showInstallModal = true
}

func (m *model) closeInstallModal() {
	m.showInstallModal = false
	m.installCursor = 0
	m.installChoices = nil
	m.installSelected = make(map[string]bool)
}

func (m *model) openRemoveModal() {
	if !m.refreshRowsForActiveRail() {
		m.closeRemoveModal()
		return
	}

	m.removeChoices = m.removeChoices[:0]
	for _, row := range m.rows {
		if row.Installed {
			m.removeChoices = append(m.removeChoices, row)
		}
	}

	if len(m.removeChoices) == 0 {
		m.statusMessage = fmt.Sprintf("no removable resources in %s", m.activeKind())
		m.closeRemoveModal()
		return
	}

	m.removeCursor = 0
	m.removeSelected = make(map[string]bool)
	m.showRemoveModal = true
}

func (m *model) closeRemoveModal() {
	m.showRemoveModal = false
	m.removeCursor = 0
	m.removeChoices = nil
	m.removeSelected = make(map[string]bool)
}

func (m *model) toggleRemoveSelection() {
	if len(m.removeChoices) == 0 || m.removeCursor < 0 || m.removeCursor >= len(m.removeChoices) {
		return
	}

	name := m.removeChoices[m.removeCursor].Name
	if m.removeSelected[name] {
		delete(m.removeSelected, name)
		return
	}
	m.removeSelected[name] = true
}

func (m *model) confirmRemoveSelection() {
	selected := m.selectedRemoveNames()
	if len(selected) == 0 {
		m.statusMessage = "select at least one resource to remove"
		return
	}

	if m.removeResources != nil {
		if err := m.removeResources(m.activeKind(), selected); err != nil {
			m.statusMessage = fmt.Sprintf("remove failed: %v", err)
			return
		}
	}

	if !m.refreshRowsForActiveRail() {
		m.closeRemoveModal()
		return
	}

	m.statusMessage = fmt.Sprintf("removed: %s", strings.Join(selected, ", "))
	m.closeRemoveModal()
}

func (m *model) toggleInstallSelection() {
	if len(m.installChoices) == 0 || m.installCursor < 0 || m.installCursor >= len(m.installChoices) {
		return
	}

	name := m.installChoices[m.installCursor].Name
	if m.installSelected[name] {
		delete(m.installSelected, name)
		return
	}
	m.installSelected[name] = true
}

func (m *model) confirmInstallSelection() {
	selected := m.selectedInstallNames()
	if len(selected) == 0 {
		m.statusMessage = "select at least one resource to install"
		return
	}

	if m.installResources != nil {
		if err := m.installResources(m.activeKind(), selected); err != nil {
			m.statusMessage = fmt.Sprintf("install failed: %v", err)
			return
		}
	}

	if !m.refreshRowsForActiveRail() {
		m.closeInstallModal()
		return
	}

	m.statusMessage = fmt.Sprintf("installed: %s", strings.Join(selected, ", "))
	m.closeInstallModal()
}

func (m *model) refreshRowsForActiveRail() bool {
	if m.listResources == nil {
		return true
	}
	rows, err := m.listResources(m.activeKind())
	if err != nil {
		m.statusMessage = fmt.Sprintf("list failed: %v", err)
		return false
	}
	m.setRows(rows)
	return true
}

func (m *model) setRows(rows []ResourceRow) {
	m.rows = append([]ResourceRow(nil), rows...)
	m.items = make([]string, 0, len(m.rows))
	for _, row := range m.rows {
		m.items = append(m.items, row.Name)
	}
	if len(m.items) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor >= len(m.items) {
		m.cursor = len(m.items) - 1
	}
}

func (m model) selectedInstallNames() []string {
	names := make([]string, 0, len(m.installSelected))
	for _, row := range m.installChoices {
		if m.installSelected[row.Name] {
			names = append(names, row.Name)
		}
	}
	return names
}

func (m model) selectedRemoveNames() []string {
	names := make([]string, 0, len(m.removeSelected))
	for _, row := range m.removeChoices {
		if m.removeSelected[row.Name] {
			names = append(names, row.Name)
		}
	}
	return names
}

func (m model) activeKind() string {
	switch m.activeRail {
	case railSkills:
		return resourceKindSkills
	case railInstructions:
		return resourceKindInstructions
	case railAgents:
		return resourceKindAgents
	default:
		return resourceKindSkills
	}
}

func (m model) railTabs() []string {
	return []string{"skills", "instructions", "agents"}
}

func (m model) wrapRail(delta int) railTab {
	tabCount := len(m.railTabs())
	if tabCount == 0 {
		return 0
	}

	next := (int(m.activeRail) + delta + tabCount) % tabCount
	return railTab(next)
}
