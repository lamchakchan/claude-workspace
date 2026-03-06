package tui //nolint:dupl // picker views share identical keyboard handling by design

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lamchakchan/claude-workspace/internal/mcpregistry"
	"github.com/lamchakchan/claude-workspace/internal/platform"
)

// mcpPickerEntry is a flattened row in the picker: a section header, custom entry, or recipe.
type mcpPickerEntry struct {
	isHeader  bool
	header    string
	isCustom  bool
	transport mcpregistry.Transport
	recipe    *mcpregistry.Recipe
}

// mcpPickerLoadedMsg carries loaded recipes to the picker.
type mcpPickerLoadedMsg struct {
	categories []mcpregistry.Category
}

// mcpPickerErrorMsg carries an error from the async loader.
type mcpPickerErrorMsg struct {
	err string
}

// McpPickerModel displays a grouped list of MCP server recipes with custom
// entry options at the top. Selecting an entry pushes the appropriate form.
type McpPickerModel struct {
	theme   *Theme
	entries []mcpPickerEntry
	cursor  int
	scroll  int
	width   int
	height  int
	loading bool
	err     string
}

// NewMcpPicker creates a new MCP recipe picker.
func NewMcpPicker(theme *Theme) *McpPickerModel {
	return &McpPickerModel{
		theme:   theme,
		loading: true,
	}
}

func (m *McpPickerModel) Init() tea.Cmd {
	return func() tea.Msg {
		categories, err := mcpregistry.LoadAll(platform.McpConfigFS)
		if err != nil {
			return mcpPickerErrorMsg{err: err.Error()}
		}
		return mcpPickerLoadedMsg{categories: categories}
	}
}

func (m *McpPickerModel) buildEntries(categories []mcpregistry.Category) {
	// Count total entries for pre-allocation
	count := 3 // "Custom" header + 2 custom entries
	for _, c := range categories {
		count += 1 + len(c.Recipes) // header + recipes
	}

	m.entries = make([]mcpPickerEntry, 0, count)

	// Custom entries at the top
	m.entries = append(m.entries,
		mcpPickerEntry{isHeader: true, header: "Custom"},
		mcpPickerEntry{isCustom: true, transport: mcpregistry.TransportStdio},
		mcpPickerEntry{isCustom: true, transport: mcpregistry.TransportHTTP},
	)

	// Recipe entries by category
	for _, cat := range categories {
		m.entries = append(m.entries, mcpPickerEntry{isHeader: true, header: formatCategoryName(cat.Name)})
		for i := range cat.Recipes {
			m.entries = append(m.entries, mcpPickerEntry{recipe: &cat.Recipes[i]})
		}
	}

	// Set cursor to first selectable entry
	m.cursor = m.nextSelectable(0, 1)
}

func (m *McpPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case mcpPickerLoadedMsg:
		m.loading = false
		m.buildEntries(msg.categories)
		return m, nil

	case mcpPickerErrorMsg:
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

func (m *McpPickerModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) { //nolint:dupl // picker views share identical keyboard handling
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

func (m *McpPickerModel) activateEntry() tea.Cmd {
	if m.cursor < 0 || m.cursor >= len(m.entries) {
		return nil
	}
	e := m.entries[m.cursor]

	if e.isCustom {
		if e.transport == mcpregistry.TransportHTTP {
			return pushView(NewMcpAddHTTP(m.theme))
		}
		return pushView(NewMcpAdd(m.theme))
	}

	if e.recipe != nil {
		return pushView(NewMcpAddFromRecipe(e.recipe, m.theme))
	}

	return nil
}

// visibleLines returns how many content lines fit in the viewport.
func (m *McpPickerModel) visibleLines() int {
	v := m.height - bannerOverhead - footerOverhead
	if v < 1 {
		return 1
	}
	return v
}

// clampScroll ensures the scroll offset keeps the cursor visible.
func (m *McpPickerModel) clampScroll() {
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

// movePage moves the cursor by approximately one page in the given direction.
func (m *McpPickerModel) movePage(dir int) {
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

// nextSelectable finds the next selectable entry from start in the given direction.
func (m *McpPickerModel) nextSelectable(start, dir int) int {
	for i := start; i >= 0 && i < len(m.entries); i += dir {
		if !m.entries[i].isHeader {
			return i
		}
	}
	return m.cursor
}

// selectableCount returns the total number of selectable (non-header) entries.
func (m *McpPickerModel) selectableCount() int {
	n := 0
	for _, e := range m.entries {
		if !e.isHeader {
			n++
		}
	}
	return n
}

// selectableIndex returns the 1-based position of the cursor among selectable items.
func (m *McpPickerModel) selectableIndex() int {
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

func (m *McpPickerModel) View() tea.View {
	var b strings.Builder
	b.WriteString(m.theme.SectionBanner("Add MCP Server"))

	if m.loading {
		b.WriteString("\n  Loading recipes...")
		return tea.NewView(b.String())
	}

	if m.err != "" {
		b.WriteString("\n  ")
		b.WriteString(lipgloss.NewStyle().Foreground(m.theme.Error).Render(m.err))
		b.WriteString("\n\n  Press q to go back.\n")
		return tea.NewView(b.String())
	}

	if len(m.entries) == 0 {
		b.WriteString("\n  No recipes found.\n")
		return tea.NewView(b.String())
	}

	// Build rendered lines
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#06B6D4"))
	transportStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)

	lines := make([]string, 0, len(m.entries))
	for i, e := range m.entries {
		if e.isHeader {
			lines = append(lines, "  "+sectionStyle.Render(e.header))
			continue
		}

		label := m.entryLabel(e)
		selected := i == m.cursor

		if selected {
			cursor := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render("> ")
			styledLabel := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render(label)
			lines = append(lines, "  "+cursor+styledLabel)
		} else {
			transport := ""
			if e.recipe != nil {
				transport = transportStyle.Render(" [" + string(e.recipe.Transport) + "]")
			}
			lines = append(lines, "    "+label+transport)
		}
	}

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

// entryLabel returns the display label for a picker entry.
func (m *McpPickerModel) entryLabel(e mcpPickerEntry) string {
	if e.isCustom {
		if e.transport == mcpregistry.TransportHTTP {
			return "Custom remote server (http)"
		}
		return "Custom local server (stdio)"
	}
	if e.recipe != nil {
		desc := e.recipe.NotesFirstLine()
		if desc != "" {
			return e.recipe.Key + "  " + lipgloss.NewStyle().Foreground(m.theme.Muted).Render(desc)
		}
		return e.recipe.Key
	}
	return ""
}

// formatCategoryName capitalizes the first letter of a category name.
func formatCategoryName(name string) string {
	if name == "" {
		return ""
	}
	return strings.ToUpper(name[:1]) + name[1:]
}
