package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ListItem is the interface that items in an expandable list must satisfy.
type ListItem interface {
	Title() string  // one-line summary shown in the list
	Detail() string // multi-line content shown in the detail view
}

// ListSection groups items under a section header.
type ListSection struct {
	Title string
	Items []ListItem
}

// listEntry is a flattened row: either a section header or a selectable item.
type listEntry struct {
	isHeader bool
	header   string
	item     ListItem
}

// expandListLoadedMsg carries loaded sections to the expand list.
type expandListLoadedMsg struct {
	sections []ListSection
}

// expandListErrorMsg carries an error from the async loader.
type expandListErrorMsg struct {
	err string
}

// ExpandListModel provides a reusable expandable-list widget with
// section headers, cursor navigation, detail view with word wrap, and scrollbar.
type ExpandListModel struct {
	theme      *Theme
	title      string
	footer     string
	entries    []listEntry
	cursor     int // index into entries (only lands on non-header entries)
	scroll     int // first visible line in the rendered line buffer
	width      int
	height     int
	loading    bool
	err        string
	loader     func() ([]ListSection, error)
	detailMode bool
	detailVP   viewport.Model
}

// bannerOverhead is the number of lines consumed by the SectionBanner.
// SectionBanner renders: \n + rule + \n + heading + \n = 4 lines
const bannerOverhead = 4

// footerOverhead is the number of lines consumed by the footer.
// Footer renders: \n + footer text + help line = 3 lines
const footerOverhead = 3

// detailExtraOverhead is overhead in detail mode beyond banner + footer:
// 1 for the title line + 1 for the separator line.
const detailExtraOverhead = 2

// NewExpandList creates an expand list that loads data asynchronously.
func NewExpandList(title string, loader func() ([]ListSection, error), footer string, theme *Theme) *ExpandListModel {
	return &ExpandListModel{
		theme:   theme,
		title:   title,
		footer:  footer,
		loading: true,
		loader:  loader,
	}
}

// flatten converts sections into a flat entries slice.
func (m *ExpandListModel) flatten(sections []ListSection) {
	var count int
	for _, s := range sections {
		count += 1 + len(s.Items) // header + items
	}
	m.entries = make([]listEntry, 0, count)
	for _, s := range sections {
		m.entries = append(m.entries, listEntry{isHeader: true, header: s.Title})
		for _, item := range s.Items {
			m.entries = append(m.entries, listEntry{item: item})
		}
	}
	// Set cursor to first selectable entry
	m.cursor = m.nextSelectable(0, 1)
}

// entryHeight returns the number of rendered lines for an entry (always 1 in list mode).
func (m *ExpandListModel) entryHeight(_ int) int {
	return 1
}

// totalLines returns the total number of rendered lines across all entries.
func (m *ExpandListModel) totalLines() int {
	return len(m.entries)
}

// lineOf returns the starting line number for entry i.
func (m *ExpandListModel) lineOf(i int) int {
	return i
}

// visibleLines returns how many content lines fit in the viewport.
func (m *ExpandListModel) visibleLines() int {
	v := m.height - bannerOverhead - footerOverhead
	if v < 1 {
		return 1
	}
	return v
}

// clampScroll ensures the scroll offset keeps the cursor visible.
func (m *ExpandListModel) clampScroll() {
	visible := m.visibleLines()
	cursorLine := m.lineOf(m.cursor)

	if cursorLine < m.scroll {
		m.scroll = cursorLine
	}
	if cursorLine >= m.scroll+visible {
		m.scroll = cursorLine - visible + 1
	}

	maxScroll := m.totalLines() - visible
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

// nextSelectable finds the next selectable entry index from start in the given direction.
// dir should be 1 (forward) or -1 (backward).
// Returns m.cursor if no selectable entry is found, keeping the cursor at its current valid position.
func (m *ExpandListModel) nextSelectable(start, dir int) int {
	for i := start; i >= 0 && i < len(m.entries); i += dir {
		if !m.entries[i].isHeader {
			return i
		}
	}
	return m.cursor
}

// selectableCount returns the total number of selectable (non-header) entries.
func (m *ExpandListModel) selectableCount() int {
	n := 0
	for _, e := range m.entries {
		if !e.isHeader {
			n++
		}
	}
	return n
}

// selectableIndex returns the 1-based position of the cursor among selectable items.
func (m *ExpandListModel) selectableIndex() int {
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

// enterDetail switches to detail mode for the current cursor item.
func (m *ExpandListModel) enterDetail() {
	if m.cursor < 0 || m.cursor >= len(m.entries) || m.entries[m.cursor].isHeader {
		return
	}
	m.detailMode = true

	vpHeight := m.detailVPHeight()
	vpWidth := m.detailVPWidth()

	m.detailVP = viewport.New(viewport.WithWidth(vpWidth), viewport.WithHeight(vpHeight))
	m.detailVP.SoftWrap = true

	detail := m.entries[m.cursor].item.Detail()
	m.detailVP.SetContent(detail)
}

// exitDetail returns to list mode.
func (m *ExpandListModel) exitDetail() {
	m.detailMode = false
}

// detailVPHeight returns the viewport height for detail mode.
func (m *ExpandListModel) detailVPHeight() int {
	h := m.height - bannerOverhead - footerOverhead - detailExtraOverhead
	if h < 1 {
		return 1
	}
	return h
}

// detailVPWidth returns the viewport width for detail mode.
func (m *ExpandListModel) detailVPWidth() int {
	// 6 chars indent (matching detail content indent) + 2 for scrollbar
	w := m.width - 6 - scrollbarWidth
	if w < 10 {
		return 10
	}
	return w
}

func (m *ExpandListModel) Init() tea.Cmd {
	if m.loader == nil {
		return nil
	}
	loader := m.loader
	return func() tea.Msg {
		sections, err := loader()
		if err != nil {
			return expandListErrorMsg{err: err.Error()}
		}
		return expandListLoadedMsg{sections: sections}
	}
}

func (m *ExpandListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.detailMode {
			m.detailVP.SetWidth(m.detailVPWidth())
			m.detailVP.SetHeight(m.detailVPHeight())
		}
		return m, nil

	case expandListLoadedMsg:
		m.loading = false
		m.flatten(msg.sections)
		return m, nil

	case expandListErrorMsg:
		m.loading = false
		m.err = msg.err
		return m, nil

	case tea.KeyPressMsg:
		if m.detailMode {
			return m.handleDetailKey(msg)
		}
		if IsQuit(msg) || IsBack(msg) {
			return m, func() tea.Msg { return PopViewMsg{} }
		}
		m.handleListKey(msg)
	}
	return m, nil
}

func (m *ExpandListModel) handleListKey(msg tea.KeyPressMsg) {
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
		target := m.cursor
		linesLeft := m.visibleLines()
		for linesLeft > 0 && target > 0 {
			prev := m.nextSelectable(target-1, -1)
			if prev >= target {
				break
			}
			target = prev
			linesLeft--
		}
		m.cursor = target
		m.clampScroll()

	case keyPgDown, "f":
		target := m.cursor
		linesLeft := m.visibleLines()
		for linesLeft > 0 && target < len(m.entries)-1 {
			next := m.nextSelectable(target+1, 1)
			if next <= target {
				break
			}
			target = next
			linesLeft--
		}
		m.cursor = target
		m.clampScroll()

	case "g":
		m.cursor = m.nextSelectable(0, 1)
		m.clampScroll()

	case "G":
		m.cursor = m.nextSelectable(len(m.entries)-1, -1)
		m.clampScroll()

	case keyEnter, keySpace:
		m.enterDetail()
	}
}

func (m *ExpandListModel) handleDetailKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case IsQuit(msg):
		return m, func() tea.Msg { return PopViewMsg{} }
	case IsBack(msg):
		m.exitDetail()
		return m, nil
	}

	switch msg.String() {
	case keyEnter:
		m.exitDetail()
		return m, nil
	case "g":
		m.detailVP.GotoTop()
		return m, nil
	case "G":
		m.detailVP.GotoBottom()
		return m, nil
	}

	// Delegate to viewport for j/k, pgup/pgdn, etc.
	var cmd tea.Cmd
	m.detailVP, cmd = m.detailVP.Update(msg)
	return m, cmd
}

func (m *ExpandListModel) View() tea.View {
	var b strings.Builder
	b.WriteString(m.theme.SectionBanner(m.title))

	if m.loading {
		b.WriteString("\n  Loading...")
		return tea.NewView(b.String())
	}

	if m.err != "" {
		b.WriteString("\n  ")
		b.WriteString(lipgloss.NewStyle().Foreground(m.theme.Error).Render(m.err))
		b.WriteString("\n\n  Press q to go back.\n")
		return tea.NewView(b.String())
	}

	if len(m.entries) == 0 {
		b.WriteString("\n  No items found.\n")
		if m.footer != "" {
			b.WriteString("\n  " + m.footer + "\n")
		}
		return tea.NewView(b.String())
	}

	if m.detailMode {
		return m.viewDetail(&b)
	}
	return m.viewList(&b)
}

func (m *ExpandListModel) viewList(b *strings.Builder) tea.View {
	// Build rendered lines
	lines := make([]string, 0, m.totalLines())
	for i, e := range m.entries {
		if e.isHeader {
			sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#06B6D4"))
			lines = append(lines, "  "+sectionStyle.Render(e.header))
			continue
		}

		title := e.item.Title()
		selected := i == m.cursor
		indicator := "▸"

		if selected {
			cursor := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render("> ")
			styledIndicator := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render(indicator)
			styledTitle := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render(title)
			lines = append(lines, "  "+cursor+styledIndicator+" "+styledTitle)
		} else {
			lines = append(lines, "    "+indicator+" "+title)
		}
	}

	// Slice visible lines from the scroll offset
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
	if m.footer != "" {
		footerStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)
		b.WriteString("  " + footerStyle.Render(m.footer) + "\n")
	}
	help := fmt.Sprintf(
		"%s navigate  %s page  %s/%s top/bottom  %s detail  %s back  %d/%d",
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

func (m *ExpandListModel) viewDetail(b *strings.Builder) tea.View {
	e := m.entries[m.cursor]

	// Title line with ▾ indicator
	title := e.item.Title()
	cursor := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render("> ")
	styledIndicator := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render("▾")
	styledTitle := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render(title)
	b.WriteString("  " + cursor + styledIndicator + " " + styledTitle + "\n")

	// Separator line
	ruleWidth := m.width - 4
	if ruleWidth < 10 {
		ruleWidth = 10
	}
	ruleStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)
	b.WriteString("  " + ruleStyle.Render(strings.Repeat("─", ruleWidth)) + "\n")

	// Viewport content with indent
	vpContent := m.detailVP.View()
	indentedLines := make([]string, 0)
	for _, line := range strings.Split(vpContent, "\n") {
		indentedLines = append(indentedLines, "      "+line)
	}
	vpRendered := strings.Join(indentedLines, "\n")

	// Scrollbar
	totalLines := strings.Count(m.detailVP.GetContent(), "\n") + 1
	vpHeight := m.detailVP.Height()
	bar := renderScrollbar(vpHeight, totalLines, vpHeight, m.detailVP.ScrollPercent(), m.theme)
	if bar != "" {
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, vpRendered, " ", bar))
	} else {
		b.WriteString(vpRendered)
	}
	b.WriteString("\n")

	// Footer
	pct := int(m.detailVP.ScrollPercent() * 100)
	trail := m.theme.HelpKey.Render(fmt.Sprintf("%d", pct)) + "%"
	help := fmt.Sprintf(
		"%s scroll  %s page  %s/%s top/bottom  %s back  %s",
		m.theme.HelpKey.Render("j/k"),
		m.theme.HelpKey.Render("pgup/pgdn/space"),
		m.theme.HelpKey.Render("g"),
		m.theme.HelpKey.Render("G"),
		m.theme.HelpKey.Render("enter/esc"),
		trail,
	)
	b.WriteString(help)

	return tea.NewView(b.String())
}
