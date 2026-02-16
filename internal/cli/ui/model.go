package ui

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	xansi "github.com/charmbracelet/x/ansi"
	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/chaz8081/positive-vibes/pkg/schema"
)

type railTab int

const (
	railSkills railTab = iota
	railInstructions
	railPrompts
	railAgents
	railTargets
	railRegistries
)

type uiScreen int

const (
	screenHome uiScreen = iota
	screenBrowser
	screenDetail
)

type model struct {
	screen              uiScreen
	homeCursor          int
	activeRail          railTab
	cursor              int
	showHelp            bool
	showSearch          bool
	searchQuery         string
	showDetailModal     bool
	showRegistryFix     bool
	registryFixName     string
	registryFixReason   string
	registryFixOptions  []string
	registryFixCursor   int
	targetOverrideArmed bool
	targetOverrideName  string
	statusMessage       string
	browserHintShown    bool
	width               int
	height              int
	rows                []ResourceRow
	filteredRows        []ResourceRow
	items               []string
	keys                keyMap
	showDetail          ResourceDetail
	previewDetail       ResourceDetail
	previewOffset       int
	previewViewport     viewport.Model
	previewFocused      bool
	detailCursor        int
	detailFiles         []detailFile
	homeInstalled       map[string][]string
	detailCache         map[string]ResourceDetail
	previewCache        map[previewRenderCacheKey]string

	listResources          func(kind string) ([]ResourceRow, error)
	showResource           func(kind, name string) (ResourceDetail, error)
	installResources       func(kind string, names []string) error
	removeResources        func(kind string, names []string) error
	installResourcesGlobal func(kind string, names []string) error
	removeResourcesGlobal  func(kind string, names []string) error
	promoteLocalRegistries func() (RegistryPromotionResult, error)
}

type detailFile struct {
	Name    string
	Content string
	Binary  bool
}

type previewRenderCacheKey struct {
	ContentHash string
	Width       int
	Theme       string
	Filename    string
	Binary      bool
}

func newModel() model {
	rows := []ResourceRow{{Name: "placeholder-1"}, {Name: "placeholder-2"}, {Name: "placeholder-3"}}
	items := []string{"placeholder-1", "placeholder-2", "placeholder-3"}
	m := model{
		screen:          screenBrowser,
		homeCursor:      0,
		activeRail:      railSkills,
		cursor:          0,
		showHelp:        false,
		showSearch:      false,
		showDetailModal: false,
		showRegistryFix: false,
		width:           96,
		height:          24,
		rows:            rows,
		filteredRows:    rows,
		items:           items,
		keys:            defaultKeyMap(),
		homeInstalled: map[string][]string{
			resourceKindSkills:       {},
			resourceKindInstructions: {},
			resourceKindPrompts:      {},
			resourceKindAgents:       {},
			resourceKindTargets:      {},
			resourceKindRegistries:   {},
		},
		detailCache:  make(map[string]ResourceDetail),
		previewCache: make(map[previewRenderCacheKey]string),
	}
	m.previewViewport = viewport.New(m.previewContentWidth(), m.previewViewportHeight())
	m.syncPreviewViewport()
	return m
}

func (model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	defer m.syncPreviewViewport()
	m.syncPreviewViewport()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		if m.showRegistryFix {
			switch {
			case msg.Type == tea.KeyEsc:
				m.showRegistryFix = false
				return m, nil
			case key.Matches(msg, m.keys.CursorDown):
				if m.registryFixCursor < len(m.registryFixOptions)-1 {
					m.registryFixCursor++
				}
				return m, nil
			case key.Matches(msg, m.keys.CursorUp):
				if m.registryFixCursor > 0 {
					m.registryFixCursor--
				}
				return m, nil
			case msg.Type == tea.KeyEnter:
				choice := ""
				if m.registryFixCursor >= 0 && m.registryFixCursor < len(m.registryFixOptions) {
					choice = m.registryFixOptions[m.registryFixCursor]
				}
				switch choice {
				case "Promote now":
					if m.promoteLocalRegistries != nil {
						report, err := m.promoteLocalRegistries()
						if err != nil {
							m.statusMessage = fmt.Sprintf("registry sync failed: %v", err)
						} else {
							m.statusMessage = fmt.Sprintf("registry sync: promoted %d, skipped %d", len(report.PromotedNames), len(report.Skipped))
						}
						m.refreshRowsForActiveRail()
						m.applyFilter()
					}
				case "Show fix hint":
					m.statusMessage = fmt.Sprintf("fix %s: %s (edit local vibes.yaml)", m.registryFixName, m.registryFixReason)
				case "Rename local registry":
					m.statusMessage = fmt.Sprintf("rename %s in local vibes.yaml to resolve conflict", m.registryFixName)
				}
				m.showRegistryFix = false
				return m, nil
			}
		}

		if m.showSearch {
			switch msg.Type {
			case tea.KeyEsc:
				m.showSearch = false
				return m, nil
			case tea.KeyBackspace:
				runes := []rune(m.searchQuery)
				if len(runes) > 0 {
					m.searchQuery = string(runes[:len(runes)-1])
					m.applyFilter()
				}
				return m, nil
			case tea.KeyEnter:
				m.showSearch = false
				return m, nil
			case tea.KeyRunes:
				m.searchQuery += string(msg.Runes)
				m.applyFilter()
				return m, nil
			case tea.KeySpace:
				m.searchQuery += " "
				m.applyFilter()
				return m, nil
			}
			return m, nil
		}

		if key.Matches(msg, m.keys.Quit) {
			return m, tea.Quit
		}

		if m.showDetailModal && m.screen != screenDetail {
			if key.Matches(msg, m.keys.CloseHelp) {
				m.closeShowModal()
			}
			return m, nil
		}

		if m.screen == screenDetail {
			if m.activeKind() == resourceKindSkills && !m.previewFocused && msg.Type == tea.KeyRunes && len(msg.Runes) == 1 {
				switch msg.Runes[0] {
				case 'j':
					if m.detailCursor < len(m.detailFiles)-1 {
						m.detailCursor++
						m.previewOffset = 0
					}
					return m, nil
				case 'k':
					if m.detailCursor > 0 {
						m.detailCursor--
						m.previewOffset = 0
					}
					return m, nil
				}
			}

			switch {
			case key.Matches(msg, m.keys.CloseHelp):
				m.closeShowModal()
				return m, nil
			case key.Matches(msg, m.keys.LeftRail):
				if m.activeKind() == resourceKindSkills {
					m.previewFocused = false
				}
				return m, nil
			case key.Matches(msg, m.keys.RightRail):
				if m.activeKind() == resourceKindSkills {
					m.previewFocused = true
				}
				return m, nil
			case key.Matches(msg, m.keys.ToggleFocus):
				if m.activeKind() == resourceKindSkills {
					m.previewFocused = !m.previewFocused
				}
				return m, nil
			case key.Matches(msg, m.keys.PreviewDown):
				if m.previewFocused || m.activeKind() != resourceKindSkills {
					m.scrollPreview(1)
				} else if m.detailCursor < len(m.detailFiles)-1 {
					m.detailCursor++
					m.previewOffset = 0
				}
				return m, nil
			case key.Matches(msg, m.keys.PreviewUp):
				if m.previewFocused || m.activeKind() != resourceKindSkills {
					m.scrollPreview(-1)
				} else if m.detailCursor > 0 {
					m.detailCursor--
					m.previewOffset = 0
				}
				return m, nil
			case key.Matches(msg, m.keys.PreviewHalfDown):
				m.scrollPreview(m.previewViewportHeight() / 2)
				return m, nil
			case key.Matches(msg, m.keys.PreviewHalfUp):
				m.scrollPreview(-(m.previewViewportHeight() / 2))
				return m, nil
			case key.Matches(msg, m.keys.PreviewTop):
				m.scrollPreviewToTop()
				return m, nil
			case key.Matches(msg, m.keys.PreviewBottom):
				m.scrollPreviewToBottom()
				return m, nil
			case key.Matches(msg, m.keys.CursorDown):
				if m.activeKind() == resourceKindSkills && !m.previewFocused {
					if m.detailCursor < len(m.detailFiles)-1 {
						m.detailCursor++
						m.previewOffset = 0
					}
				} else {
					m.scrollPreview(1)
				}
				return m, nil
			case key.Matches(msg, m.keys.CursorUp):
				if m.activeKind() == resourceKindSkills && !m.previewFocused {
					if m.detailCursor > 0 {
						m.detailCursor--
						m.previewOffset = 0
					}
				} else {
					m.scrollPreview(-1)
				}
				return m, nil
			}
		}

		if key.Matches(msg, m.keys.Help) {
			m.showHelp = true
			return m, nil
		}

		if key.Matches(msg, m.keys.CloseHelp) {
			m.showHelp = false
			switch m.screen {
			case screenDetail:
				m.closeShowModal()
			case screenBrowser:
				m.closeBrowserToHome()
			case screenHome:
				return m, tea.Quit
			}
			return m, nil
		}

		if m.showHelp {
			return m, nil
		}

		if m.screen == screenHome {
			switch {
			case key.Matches(msg, m.keys.CursorDown), key.Matches(msg, m.keys.RightRail):
				if m.homeCursor < len(m.homeKinds())-1 {
					m.homeCursor++
				}
			case key.Matches(msg, m.keys.CursorUp), key.Matches(msg, m.keys.LeftRail):
				if m.homeCursor > 0 {
					m.homeCursor--
				}
			case msg.Type == tea.KeyEnter:
				m.openBrowserFromHome()
			}
			return m, nil
		}

		if m.screen == screenBrowser {
			if key.Matches(msg, m.keys.LeftRail) {
				m.previewFocused = false
				return m, nil
			}
			if key.Matches(msg, m.keys.RightRail) {
				m.previewFocused = true
				return m, nil
			}
		}

		if key.Matches(msg, m.keys.Search) {
			m.showSearch = true
			return m, nil
		}

		if msg.Type == tea.KeySpace {
			if m.activeKind() == resourceKindRegistries && len(m.filteredRows) > 0 && m.cursor >= 0 && m.cursor < len(m.filteredRows) {
				row := m.filteredRows[m.cursor]
				if row.State == "registry_attention" {
					m.showRegistryFix = true
					m.registryFixName = row.Name
					m.registryFixReason = row.Description
					m.registryFixOptions = []string{"Promote now", "Show fix hint", "Rename local registry", "Cancel"}
					m.registryFixCursor = 0
					return m, nil
				}
			}
			m.toggleSelectedResourceInstalled()
			return m, nil
		}

		if key.Matches(msg, m.keys.ResetTargets) && m.activeKind() == resourceKindTargets && m.screen == screenBrowser && !m.previewFocused {
			m.resetLocalTargetsOverride()
			return m, nil
		}

		if key.Matches(msg, m.keys.ToggleFocus) {
			m.previewFocused = !m.previewFocused
			return m, nil
		}

		if msg.Type == tea.KeyEnter {
			kind := m.activeKind()
			if kind == resourceKindInstructions || kind == resourceKindPrompts || kind == resourceKindAgents {
				return m, nil
			}
			if kind == resourceKindSkills {
				if m.showResource != nil && len(m.filteredRows) > 0 && m.cursor >= 0 && m.cursor < len(m.filteredRows) {
					selected := m.filteredRows[m.cursor]
					detail, err := m.ensureDetail(kind, selected.Name)
					if err == nil {
						files := detailFilesFromDetail(detail)
						if len(files) == 1 {
							return m, nil
						}
					}
				}
			}
			m.openShowModal()
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keys.PreviewDown):
			if m.previewFocused {
				m.scrollPreview(1)
			} else if m.cursor < len(m.filteredRows)-1 {
				m.cursor++
				m.updatePreviewForSelection()
			}
		case key.Matches(msg, m.keys.PreviewUp):
			if m.previewFocused {
				m.scrollPreview(-1)
			} else if m.cursor > 0 {
				m.cursor--
				m.updatePreviewForSelection()
			}
		case key.Matches(msg, m.keys.PreviewHalfDown):
			m.scrollPreview(m.previewViewportHeight() / 2)
		case key.Matches(msg, m.keys.PreviewHalfUp):
			m.scrollPreview(-(m.previewViewportHeight() / 2))
		case key.Matches(msg, m.keys.PreviewTop):
			m.scrollPreviewToTop()
		case key.Matches(msg, m.keys.PreviewBottom):
			m.scrollPreviewToBottom()
		case key.Matches(msg, m.keys.CursorDown):
			if m.previewFocused {
				m.scrollPreview(1)
			} else if m.cursor < len(m.filteredRows)-1 {
				m.cursor++
				m.updatePreviewForSelection()
			}
		case key.Matches(msg, m.keys.CursorUp):
			if m.previewFocused {
				m.scrollPreview(-1)
			} else if m.cursor > 0 {
				m.cursor--
				m.updatePreviewForSelection()
			}
		}
	}

	return m, nil
}

func (m *model) openBrowserFromHome() {
	kind := m.homeKinds()[m.homeCursor]
	m.activeRail = kindToRail(kind)
	m.screen = screenBrowser
	m.previewFocused = false
	m.previewOffset = 0
	m.searchQuery = ""
	m.showSearch = false
	if kind == resourceKindRegistries && m.promoteLocalRegistries != nil {
		report, err := m.promoteLocalRegistries()
		if err != nil {
			m.statusMessage = fmt.Sprintf("registry sync failed: %v", err)
		} else {
			promoted := len(report.PromotedNames)
			skipped := len(report.Skipped)
			if promoted > 0 || skipped > 0 {
				m.statusMessage = fmt.Sprintf("registry sync: promoted %d, skipped %d", promoted, skipped)
			}
		}
	}
	m.refreshRowsForActiveRail()
	if !m.browserHintShown {
		if m.statusMessage == "" {
			m.statusMessage = "tip: space cycles install scope; esc goes back"
		}
		m.browserHintShown = true
	}
}

func (m *model) closeBrowserToHome() {
	m.screen = screenHome
	m.cursor = 0
	m.previewFocused = false
	m.previewOffset = 0
	m.showSearch = false
	m.searchQuery = ""
	if m.statusMessage == "tip: space cycles install scope; esc goes back" {
		m.statusMessage = ""
	}
	m.applyFilter()
	m.refreshHomeInstalled()
}

func (m *model) refreshHomeInstalled() {
	for _, kind := range m.homeKinds() {
		if m.listResources == nil {
			continue
		}
		rows, err := m.listResources(kind)
		if err != nil {
			continue
		}
		names := make([]string, 0, len(rows))
		for _, row := range rows {
			if row.Installed {
				names = append(names, row.Name)
			}
		}
		m.homeInstalled[kind] = names
	}
}

func (m model) homeKinds() []string {
	return []string{resourceKindSkills, resourceKindInstructions, resourceKindPrompts, resourceKindAgents, resourceKindTargets, resourceKindRegistries}
}

func kindToRail(kind string) railTab {
	switch kind {
	case resourceKindInstructions:
		return railInstructions
	case resourceKindPrompts:
		return railPrompts
	case resourceKindAgents:
		return railAgents
	case resourceKindTargets:
		return railTargets
	case resourceKindRegistries:
		return railRegistries
	default:
		return railSkills
	}
}

func (m *model) toggleSelectedResourceInstalled() {
	if m.screen != screenBrowser {
		return
	}
	if len(m.filteredRows) == 0 || m.cursor < 0 || m.cursor >= len(m.filteredRows) {
		return
	}
	selected := m.filteredRows[m.cursor]
	kind := m.activeKind()
	if kind == resourceKindTargets && m.needsTargetOverrideConfirmation(selected) {
		if !m.targetOverrideArmed || m.targetOverrideName != selected.Name {
			m.targetOverrideArmed = true
			m.targetOverrideName = selected.Name
			m.statusMessage = "press space again to create local targets override"
			return
		}
		m.targetOverrideArmed = false
		m.targetOverrideName = ""
	}
	scope := selected.InstallScope
	if scope == "" {
		scope = "none"
	}
	switch scope {
	case "none":
		if m.installResources == nil {
			m.statusMessage = "backend unavailable: install action requires resource service"
			return
		}
		if err := m.installResources(kind, []string{selected.Name}); err != nil {
			m.statusMessage = fmt.Sprintf("install failed: %v", err)
			return
		}
		m.statusMessage = fmt.Sprintf("set local: %s", selected.Name)
	case "local":
		if m.installResourcesGlobal == nil {
			m.statusMessage = "backend unavailable: global install action requires resource service"
			return
		}
		if err := m.installResourcesGlobal(kind, []string{selected.Name}); err != nil {
			m.statusMessage = fmt.Sprintf("global install failed: %v", err)
			return
		}
		if m.removeResources == nil {
			m.statusMessage = "backend unavailable: remove action requires resource service"
			return
		}
		if err := m.removeResources(kind, []string{selected.Name}); err != nil {
			m.statusMessage = fmt.Sprintf("remove failed: %v", err)
			return
		}
		m.statusMessage = fmt.Sprintf("set global: %s", selected.Name)
	case "global":
		if m.installResources == nil {
			m.statusMessage = "backend unavailable: install action requires resource service"
			return
		}
		if err := m.installResources(kind, []string{selected.Name}); err != nil {
			m.statusMessage = fmt.Sprintf("install failed: %v", err)
			return
		}
		m.statusMessage = fmt.Sprintf("set both: %s", selected.Name)
	case "both":
		if m.removeResources == nil {
			m.statusMessage = "backend unavailable: remove action requires resource service"
			return
		}
		if err := m.removeResources(kind, []string{selected.Name}); err != nil {
			m.statusMessage = fmt.Sprintf("remove failed: %v", err)
			return
		}
		if m.removeResourcesGlobal == nil {
			m.statusMessage = "backend unavailable: global remove action requires resource service"
			return
		}
		if err := m.removeResourcesGlobal(kind, []string{selected.Name}); err != nil {
			m.statusMessage = fmt.Sprintf("global remove failed: %v", err)
			return
		}
		m.statusMessage = fmt.Sprintf("set none: %s", selected.Name)
	default:
		m.statusMessage = fmt.Sprintf("unknown install scope: %s", selected.InstallScope)
		return
	}
	if !m.refreshRowsForActiveRail() {
		return
	}
	m.applyFilter()
	for i, row := range m.filteredRows {
		if row.Name == selected.Name {
			m.cursor = i
			break
		}
	}
	m.updatePreviewForSelection()
}

func (m *model) resetLocalTargetsOverride() {
	if m.activeKind() != resourceKindTargets {
		return
	}
	if m.removeResources == nil {
		m.statusMessage = "backend unavailable: remove action requires resource service"
		return
	}
	localNames := make([]string, 0)
	seen := map[string]bool{}
	for _, row := range m.rows {
		if row.InstallScope != "local" && row.InstallScope != "both" {
			continue
		}
		if seen[row.Name] {
			continue
		}
		seen[row.Name] = true
		localNames = append(localNames, row.Name)
	}
	if len(localNames) == 0 {
		m.statusMessage = "targets already inherited from global"
		return
	}
	if err := m.removeResources(resourceKindTargets, localNames); err != nil {
		m.statusMessage = fmt.Sprintf("reset targets failed: %v", err)
		return
	}
	if !m.refreshRowsForActiveRail() {
		return
	}
	m.applyFilter()
	m.statusMessage = "reset targets to inherited global defaults"
}

func (m *model) needsTargetOverrideConfirmation(selected ResourceRow) bool {
	if m.activeKind() != resourceKindTargets {
		return false
	}
	if selected.InstallScope == "local" || selected.InstallScope == "both" {
		return false
	}
	for _, row := range m.rows {
		if row.InstallScope == "local" || row.InstallScope == "both" {
			return false
		}
	}
	return true
}

func (m *model) openShowModal() {
	if m.showResource == nil {
		m.statusMessage = "backend unavailable: show action requires resource service"
		m.closeShowModal()
		return
	}

	if !m.refreshRowsForActiveRail() {
		m.closeShowModal()
		return
	}

	if len(m.filteredRows) == 0 || m.cursor < 0 || m.cursor >= len(m.filteredRows) {
		m.statusMessage = "no resource selected"
		m.closeShowModal()
		return
	}

	selected := m.filteredRows[m.cursor]
	detail, err := m.ensureDetail(m.activeKind(), selected.Name)
	if err != nil {
		m.statusMessage = fmt.Sprintf("show failed: %v", err)
		m.closeShowModal()
		return
	}

	m.showDetail = detail
	m.showDetailModal = true
	m.screen = screenDetail
	m.previewFocused = false
	m.detailFiles = detailFilesFromDetail(detail)
	m.detailCursor = 0
	m.previewOffset = 0
}

func (m *model) closeShowModal() {
	m.showDetailModal = false
	m.showDetail = ResourceDetail{}
	m.detailFiles = nil
	m.detailCursor = 0
	m.previewFocused = false
	m.screen = screenBrowser
}

func (m *model) refreshRowsForActiveRail() bool {
	if m.listResources == nil {
		m.applyFilter()
		return true
	}
	rows, err := m.listResources(m.activeKind())
	if err != nil {
		m.statusMessage = fmt.Sprintf("list failed: %v", err)
		return false
	}
	rows = m.enrichRows(rows)
	m.setRows(rows)
	return true
}

func (m *model) enrichRows(rows []ResourceRow) []ResourceRow {
	if m.showResource == nil {
		return rows
	}
	kind := m.activeKind()
	out := make([]ResourceRow, 0, len(rows))
	for _, row := range rows {
		if row.Description == "" {
			detail, err := m.ensureDetail(kind, row.Name)
			if err == nil {
				row.Description = detailDescription(detail)
			}
		}
		out = append(out, row)
	}
	return out
}

func detailDescription(detail ResourceDetail) string {
	switch p := detail.Payload.(type) {
	case *schema.Skill:
		return p.Description
	case map[string]any:
		if reason, ok := p["source_reason"].(string); ok && reason != "" {
			return reason
		}
		if desc, ok := p["description"].(string); ok {
			return desc
		}
	}
	return ""
}

func detailFilesFromDetail(detail ResourceDetail) []detailFile {
	switch p := detail.Payload.(type) {
	case map[string]any:
		if rawFiles, ok := p["files"].([]map[string]any); ok {
			files := make([]detailFile, 0, len(rawFiles))
			for _, rf := range rawFiles {
				name, _ := rf["name"].(string)
				content, _ := rf["content"].(string)
				binary, _ := rf["binary"].(bool)
				if name != "" {
					files = append(files, detailFile{Name: name, Content: content, Binary: binary})
				}
			}
			if len(files) > 0 {
				sort.Slice(files, func(i, j int) bool {
					return files[i].Name < files[j].Name
				})
				return files
			}
		}
		if content, ok := p["content"].(string); ok {
			return []detailFile{{Name: "content.md", Content: content}}
		}
	case *schema.Skill:
		if p != nil {
			return []detailFile{{Name: "SKILL.md", Content: p.Instructions}}
		}
	case manifest.InstructionRef:
		return []detailFile{{Name: "instruction.md", Content: p.Content}}
	}
	return nil
}

func (m *model) scrollPreview(delta int) {
	if delta == 0 {
		return
	}
	if delta > 0 {
		m.previewViewport.LineDown(delta)
	} else {
		m.previewViewport.LineUp(-delta)
	}
	m.previewOffset = m.previewViewport.YOffset
}

func (m *model) scrollPreviewToTop() {
	m.previewViewport.GotoTop()
	m.previewOffset = m.previewViewport.YOffset
}

func (m *model) scrollPreviewToBottom() {
	m.previewViewport.GotoBottom()
	m.previewOffset = m.previewViewport.YOffset
}

func (m model) previewViewportHeight() int {
	terminalHeight := m.height
	if terminalHeight <= 0 {
		terminalHeight = 24
	}

	footerRows := 1
	headerRows := 2
	panelFrameRows := styleFrameHeight(panelStyle)
	if panelFrameRows < 0 {
		panelFrameRows = 0
	}
	available := terminalHeight - headerRows - footerRows - panelFrameRows
	if m.screen == screenBrowser {
		available -= 3 // title, kind, spacer inside preview pane
	}
	if available < 1 {
		available = 1
	}
	return available
}

func (m model) currentPreviewText() string {
	if m.screen == screenBrowser {
		kind := m.activeKind()
		if kind == resourceKindInstructions || kind == resourceKindPrompts || kind == resourceKindAgents {
			content := contentFromDetail(m.previewDetail)
			if content != "" {
				return content
			}
		}
		if kind == resourceKindSkills {
			files := detailFilesFromDetail(m.previewDetail)
			if len(files) == 1 {
				if files[0].Binary {
					return "binary/unpreviewable file"
				}
				if files[0].Content != "" {
					return files[0].Content
				}
			}
		}
		desc := detailDescription(m.previewDetail)
		if desc == "" {
			return "(no description)"
		}
		return desc
	}
	if m.screen == screenDetail {
		if m.activeKind() == resourceKindSkills && len(m.detailFiles) > 0 && m.detailCursor >= 0 && m.detailCursor < len(m.detailFiles) {
			f := m.detailFiles[m.detailCursor]
			if f.Binary {
				return "binary/unpreviewable file"
			}
			if f.Content != "" {
				return f.Content
			}
			return "(no preview)"
		}
		content := contentFromDetail(m.showDetail)
		if content == "" {
			return "(no content)"
		}
		return content
	}
	return ""
}

func (m model) previewContentWidth() int {
	width := m.width
	if width <= 0 {
		width = 96
	}
	if m.screen == screenBrowser {
		_, previewWidth := m.browserWidths(width)
		return previewWidth
	}
	if m.screen == screenDetail {
		if m.activeKind() == resourceKindSkills && len(m.detailFiles) > 0 {
			_, rightWidth := m.twoPaneContentWidths(width, 3)
			return rightWidth
		}
		return contentWidthForStyle(width, panelStyle)
	}
	return contentWidthForStyle(width, panelStyle)
}

func (m model) currentPreviewFileMeta() (string, bool) {
	if m.screen == screenDetail && m.activeKind() == resourceKindSkills && len(m.detailFiles) > 0 && m.detailCursor >= 0 && m.detailCursor < len(m.detailFiles) {
		f := m.detailFiles[m.detailCursor]
		return f.Name, f.Binary
	}
	if m.screen == screenBrowser {
		kind := m.activeKind()
		if kind == resourceKindInstructions || kind == resourceKindPrompts || kind == resourceKindAgents {
			if p := strings.TrimSpace(m.previewDetail.Path); p != "" {
				return filepath.Base(p), false
			}
			switch kind {
			case resourceKindInstructions:
				return "instruction.md", false
			case resourceKindPrompts:
				return "prompt.md", false
			case resourceKindAgents:
				return "agent.md", false
			}
		}
		if kind == resourceKindSkills {
			files := detailFilesFromDetail(m.previewDetail)
			if len(files) == 1 {
				return files[0].Name, files[0].Binary
			}
		}
		return "description.txt", false
	}
	if m.screen == screenDetail {
		return "detail.md", false
	}
	return "preview.txt", false
}

func (m *model) syncPreviewViewport() {
	width := m.previewContentWidth()
	height := m.previewViewportHeight()
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	m.previewViewport.Width = width
	m.previewViewport.Height = height

	text := m.currentPreviewText()
	filename, binary := m.currentPreviewFileMeta()
	rendered := m.renderPreviewContent(text, filename, binary, width)
	rendered = hardClampPreviewContentWidth(rendered, width)
	m.previewViewport.SetContent(rendered)

	maxOffset := m.previewViewport.TotalLineCount() - m.previewViewport.Height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.previewOffset < 0 {
		m.previewOffset = 0
	}
	if m.previewOffset > maxOffset {
		m.previewOffset = maxOffset
	}
	m.previewViewport.SetYOffset(m.previewOffset)
	m.previewOffset = m.previewViewport.YOffset
}

func hardClampPreviewContentWidth(content string, width int) string {
	if width < 1 || content == "" {
		return content
	}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if xansi.StringWidth(line) > width {
			lines[i] = xansi.Truncate(line, width, "")
		}
	}
	return strings.Join(lines, "\n")
}

func (m *model) renderPreviewContent(content, filename string, binary bool, width int) string {
	if binary {
		return "binary/unpreviewable file"
	}
	if content == "" {
		return ""
	}
	if width < 1 {
		width = 1
	}
	theme := "dracula"
	h := sha1.Sum([]byte(content))
	key := previewRenderCacheKey{
		ContentHash: hex.EncodeToString(h[:]),
		Width:       width,
		Theme:       theme,
		Filename:    filename,
		Binary:      binary,
	}
	if rendered, ok := m.previewCache[key]; ok {
		return rendered
	}

	rendered := content
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == ".md" || ext == ".markdown" || ext == ".mdx" {
		renderer, err := glamour.NewTermRenderer(
			glamour.WithStandardStyle("dark"),
			glamour.WithPreservedNewLines(),
			glamour.WithWordWrap(width),
		)
		if err == nil {
			if out, renderErr := renderer.Render(content); renderErr == nil {
				rendered = strings.TrimSuffix(out, "\n")
			}
		}
	} else {
		lexer := lexers.Match(filename)
		if lexer == nil {
			lexer = lexers.Analyse(content)
		}
		if lexer != nil {
			iterator, err := lexer.Tokenise(nil, content)
			if err == nil {
				formatter := formatters.Get("terminal16m")
				if formatter == nil {
					formatter = formatters.Fallback
				}
				style := styles.Get(theme)
				if style == nil {
					style = styles.Fallback
				}
				var b strings.Builder
				if formatErr := formatter.Format(&b, style, iterator); formatErr == nil {
					rendered = strings.TrimSuffix(b.String(), "\n")
				}
			}
		}
	}

	if rendered == content {
		rendered = colorizePreview(content)
	}

	m.previewCache[key] = rendered
	return rendered
}

func (m *model) ensureDetail(kind, name string) (ResourceDetail, error) {
	cacheKey := kind + ":" + name
	if detail, ok := m.detailCache[cacheKey]; ok {
		return detail, nil
	}
	detail, err := m.showResource(kind, name)
	if err != nil {
		return ResourceDetail{}, err
	}
	if detail.Kind == "" {
		detail.Kind = kind
	}
	if detail.Name == "" {
		detail.Name = name
	}
	m.detailCache[cacheKey] = detail
	return detail, nil
}

func (m *model) updatePreviewForSelection() {
	m.previewOffset = 0
	if m.showResource == nil || len(m.filteredRows) == 0 || m.cursor < 0 || m.cursor >= len(m.filteredRows) {
		m.previewDetail = ResourceDetail{}
		return
	}
	selected := m.filteredRows[m.cursor]
	detail, err := m.ensureDetail(m.activeKind(), selected.Name)
	if err != nil {
		m.previewDetail = ResourceDetail{}
		return
	}
	m.previewDetail = detail
}

func (m *model) setRows(rows []ResourceRow) {
	m.rows = append([]ResourceRow(nil), rows...)
	m.applyFilter()
}

func (m *model) applyFilter() {
	query := strings.TrimSpace(strings.ToLower(m.searchQuery))
	m.filteredRows = m.filteredRows[:0]
	if query == "" {
		m.filteredRows = append(m.filteredRows, m.rows...)
	} else {
		for _, row := range m.rows {
			hayName := strings.ToLower(row.Name)
			hayDesc := strings.ToLower(row.Description)
			if fuzzyMatch(query, hayName) || fuzzyMatch(query, hayDesc) {
				m.filteredRows = append(m.filteredRows, row)
			}
		}
	}

	m.items = make([]string, 0, len(m.filteredRows))
	for _, row := range m.filteredRows {
		m.items = append(m.items, row.Name)
	}
	if len(m.items) == 0 {
		m.cursor = 0
		m.previewDetail = ResourceDetail{}
		return
	}
	if m.cursor >= len(m.items) {
		m.cursor = len(m.items) - 1
	}
	m.updatePreviewForSelection()
}

func fuzzyMatch(query, text string) bool {
	if query == "" {
		return true
	}
	if strings.Contains(text, query) {
		return true
	}
	qi := 0
	for i := 0; i < len(text) && qi < len(query); i++ {
		if text[i] == query[qi] {
			qi++
		}
	}
	return qi == len(query)
}

func (m model) activeKind() string {
	switch m.activeRail {
	case railSkills:
		return resourceKindSkills
	case railInstructions:
		return resourceKindInstructions
	case railPrompts:
		return resourceKindPrompts
	case railAgents:
		return resourceKindAgents
	case railTargets:
		return resourceKindTargets
	case railRegistries:
		return resourceKindRegistries
	default:
		return resourceKindSkills
	}
}

func (m model) railTabs() []string {
	return []string{"skills", "instructions", "agents", "targets", "registries"}
}
