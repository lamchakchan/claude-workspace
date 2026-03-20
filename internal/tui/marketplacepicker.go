package tui //nolint:dupl // picker views share identical structure and keyboard handling by design

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lamchakchan/claude-workspace/internal/platform"
	"github.com/lamchakchan/claude-workspace/internal/plugins"
)

// marketplacePickerEntry is a flattened row in the picker: a section header, a marketplace, or an action.
type marketplacePickerEntry struct {
	isHeader     bool
	header       string
	recipe       *plugins.MarketplaceRecipe
	isConfigured bool
	isAction     bool
	actionLabel  string
}

// marketplacePickerLoadedMsg carries discovered curated and configured marketplaces.
type marketplacePickerLoadedMsg struct {
	curated    []plugins.MarketplaceRecipe
	configured map[string]bool
}

// marketplacePickerErrorMsg carries an error from the async loader.
type marketplacePickerErrorMsg struct {
	err string
}

// MarketplacePickerModel displays curated marketplaces with add action.
type MarketplacePickerModel struct {
	theme   *Theme
	entries []marketplacePickerEntry
	cursor  int
	scroll  int
	width   int
	height  int
	loading bool
	err     string
}

// NewMarketplacePicker creates a new marketplace picker.
func NewMarketplacePicker(theme *Theme) *MarketplacePickerModel {
	return &MarketplacePickerModel{
		theme:   theme,
		loading: true,
	}
}

func (m *MarketplacePickerModel) Init() tea.Cmd {
	return func() tea.Msg {
		curated, err := plugins.LoadMarketplaces(platform.MarketplaceRegistryFS)
		if err != nil {
			return marketplacePickerErrorMsg{err: err.Error()}
		}
		configured, _ := plugins.DiscoverMarketplaces()
		configuredSet := make(map[string]bool, len(configured))
		for _, mp := range configured {
			configuredSet[mp.Name] = true
		}
		return marketplacePickerLoadedMsg{curated: curated, configured: configuredSet}
	}
}

func (m *MarketplacePickerModel) buildEntries(curated []plugins.MarketplaceRecipe, configured map[string]bool) {
	// Count total entries: action + headers + recipes
	count := 1 // "Add custom marketplace..." action
	if len(curated) > 0 {
		count += 1 + len(curated) // header + recipes
	}

	m.entries = make([]marketplacePickerEntry, 0, count)

	// First entry: add custom marketplace
	m.entries = append(m.entries, marketplacePickerEntry{
		isAction:    true,
		actionLabel: "Add custom marketplace...",
	})

	// Curated marketplaces
	if len(curated) > 0 {
		m.entries = append(m.entries, marketplacePickerEntry{isHeader: true, header: "CURATED MARKETPLACES"})
		for i := range curated {
			m.entries = append(m.entries, marketplacePickerEntry{
				recipe:       &curated[i],
				isConfigured: configured[curated[i].Key],
			})
		}
	}

	m.cursor = m.nextSelectable(0, 1)
}

func (m *MarketplacePickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { //nolint:dupl // picker views share identical update structure by design
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case marketplacePickerLoadedMsg:
		m.loading = false
		m.buildEntries(msg.curated, msg.configured)
		return m, nil

	case marketplacePickerErrorMsg:
		m.loading = false
		m.err = msg.err
		return m, nil

	case tea.KeyPressMsg:
		if IsQuit(msg) || IsBack(msg) {
			return m, func() tea.Msg { return PopViewMsg{} }
		}
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *MarketplacePickerModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) { //nolint:dupl // picker views share identical keyboard handling by design
	switch msg.String() {
	case keyUp, "k":
		next := m.nextSelectable(m.cursor-1, -1)
		if next < m.cursor {
			m.cursor = next
		}
		m.clampScroll()

	case keyDown, "j":
		next := m.nextSelectable(m.cursor+1, 1)
		if next > m.cursor {
			m.cursor = next
		}
		m.clampScroll()

	case keyPgUp, "b":
		m.movePage(-1)

	case keyPgDown, "f":
		m.movePage(1)

	case "g":
		m.cursor = m.nextSelectable(0, 1)
		m.clampScroll()

	case "G":
		m.cursor = m.nextSelectable(len(m.entries)-1, -1)
		m.clampScroll()

	case keyEnter:
		cmd := m.activateEntry()
		return m, cmd
	}
	return m, nil
}

func (m *MarketplacePickerModel) activateEntry() tea.Cmd {
	if m.cursor < 0 || m.cursor >= len(m.entries) {
		return nil
	}
	e := m.entries[m.cursor]

	// "Add custom marketplace..." action -> push the add form
	if e.isAction {
		return pushView(NewMarketplaceAdd(m.theme))
	}

	// Skip already-configured or header entries
	if e.recipe == nil || e.isConfigured {
		return nil
	}

	exe, _ := os.Executable()
	cmd := exec.Command(exe, "plugins", "marketplace", "add", e.recipe.Repo)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return tea.ExecProcess(cmd, func(_ error) tea.Msg {
		return PopViewMsg{}
	})
}

func (m *MarketplacePickerModel) visibleLines() int {
	v := m.height - bannerOverhead - footerOverhead
	if v < 1 {
		return 1
	}
	return v
}

func (m *MarketplacePickerModel) clampScroll() {
	visible := m.visibleLines()
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+visible {
		m.scroll = m.cursor - visible + 1
	}
	maxScroll := len(m.entries) - visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scroll > maxScroll {
		m.scroll = maxScroll
	}
	if m.scroll < 0 {
		m.scroll = 0
	}
}

func (m *MarketplacePickerModel) movePage(dir int) {
	target := m.cursor
	linesLeft := m.visibleLines()
	for linesLeft > 0 {
		next := m.nextSelectable(target+dir, dir)
		if dir < 0 && next >= target {
			break
		}
		if dir > 0 && next <= target {
			break
		}
		target = next
		linesLeft--
	}
	m.cursor = target
	m.clampScroll()
}

func (m *MarketplacePickerModel) nextSelectable(start, dir int) int {
	for i := start; i >= 0 && i < len(m.entries); i += dir {
		if !m.entries[i].isHeader {
			return i
		}
	}
	return m.cursor
}

func (m *MarketplacePickerModel) selectableCount() int {
	n := 0
	for _, e := range m.entries {
		if !e.isHeader {
			n++
		}
	}
	return n
}

func (m *MarketplacePickerModel) selectableIndex() int {
	n := 0
	for i, e := range m.entries {
		if !e.isHeader {
			n++
		}
		if i == m.cursor {
			return n
		}
	}
	return 0
}

func (m *MarketplacePickerModel) View() tea.View {
	var b strings.Builder
	b.WriteString(m.theme.SectionBanner("Add Marketplace"))

	if m.loading {
		b.WriteString("\n  Loading marketplaces...")
		return tea.NewView(b.String())
	}

	if m.err != "" {
		b.WriteString("\n  ")
		b.WriteString(lipgloss.NewStyle().Foreground(m.theme.Error).Render(m.err))
		b.WriteString("\n\n  Press esc to go back.\n")
		return tea.NewView(b.String())
	}

	if len(m.entries) == 0 {
		b.WriteString("\n  No curated marketplaces available.\n")
		b.WriteString("\n  Press esc to go back.\n")
		return tea.NewView(b.String())
	}

	lines := m.buildLines()

	// Slice visible lines
	visible := m.visibleLines()
	start := m.scroll
	if start > len(lines) {
		start = len(lines)
	}
	end := start + visible
	if end > len(lines) {
		end = len(lines)
	}

	var rows strings.Builder
	for i := start; i < end; i++ {
		rows.WriteString(lines[i])
		if i < end-1 {
			rows.WriteString("\n")
		}
	}

	// Scrollbar
	total := len(lines)
	var scrollPct float64
	if total > visible {
		scrollPct = float64(m.scroll) / float64(total-visible)
	}
	bar := renderScrollbar(end-start, total, visible, scrollPct, m.theme)
	if bar != "" {
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, rows.String(), " ", bar))
	} else {
		b.WriteString(rows.String())
	}
	b.WriteString("\n")

	// Footer
	b.WriteString("\n")
	help := fmt.Sprintf(
		"%s navigate  %s page  %s/%s top/bottom  %s select  %s back  %d/%d",
		m.theme.HelpKey.Render("j/k"),
		m.theme.HelpKey.Render("pgup/pgdn"),
		m.theme.HelpKey.Render("g"),
		m.theme.HelpKey.Render("G"),
		m.theme.HelpKey.Render(keyEnter),
		m.theme.HelpKey.Render("esc"),
		m.selectableIndex(), m.selectableCount(),
	)
	b.WriteString(help)

	return tea.NewView(b.String())
}

// buildLines renders all picker entries into display lines.
func (m *MarketplacePickerModel) buildLines() []string {
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#06B6D4"))
	configuredStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)

	lines := make([]string, 0, len(m.entries))
	for i, e := range m.entries {
		if e.isHeader {
			lines = append(lines, "  "+sectionStyle.Render(e.header))
			continue
		}

		selected := i == m.cursor

		// Action entry (e.g., "Add custom marketplace...")
		if e.isAction {
			if selected {
				cursor := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render("> ")
				label := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render(e.actionLabel)
				lines = append(lines, "  "+cursor+label)
			} else {
				lines = append(lines, "    "+e.actionLabel)
			}
			continue
		}

		// Marketplace recipe entry
		switch { //nolint:dupl // picker views share identical styling logic by design
		case e.isConfigured:
			lines = append(lines, configuredStyle.Render("  \u2713 "+e.recipe.Key+" (configured)"))
		case selected:
			cursor := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render("> ")
			styledLabel := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render(e.recipe.Key)
			desc := ""
			if e.recipe.Description != "" {
				desc = "  " + lipgloss.NewStyle().Foreground(m.theme.Primary).Render(e.recipe.Description)
			}
			lines = append(lines, "  "+cursor+styledLabel+desc)
		default:
			label := e.recipe.Key
			if e.recipe.Description != "" {
				label += "  " + lipgloss.NewStyle().Foreground(m.theme.Muted).Render(e.recipe.Description)
			}
			lines = append(lines, "    "+label)
		}
	}
	return lines
}
