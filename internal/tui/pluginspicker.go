package tui //nolint:dupl // picker views share identical keyboard handling by design

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lamchakchan/claude-workspace/internal/plugins"
)

// pluginPickerEntry is a flattened row in the picker: a section header or a plugin.
type pluginPickerEntry struct {
	isHeader    bool
	header      string
	plugin      *plugins.Plugin
	isInstalled bool
}

// pluginPickerLoadedMsg carries discovered available plugins.
type pluginPickerLoadedMsg struct {
	available []plugins.Plugin
	installed map[string]bool
}

// pluginPickerErrorMsg carries an error from the async loader.
type pluginPickerErrorMsg struct {
	err string
}

// PluginsPickerModel displays available plugins grouped by marketplace with install action.
type PluginsPickerModel struct {
	theme   *Theme
	entries []pluginPickerEntry
	cursor  int
	scroll  int
	width   int
	height  int
	loading bool
	err     string
}

// NewPluginsPicker creates a new available plugins picker.
func NewPluginsPicker(theme *Theme) *PluginsPickerModel {
	return &PluginsPickerModel{
		theme:   theme,
		loading: true,
	}
}

func (m *PluginsPickerModel) Init() tea.Cmd {
	return func() tea.Msg {
		available, err := plugins.DiscoverAvailable()
		if err != nil {
			return pluginPickerErrorMsg{err: err.Error()}
		}
		installed, _ := plugins.DiscoverInstalled()
		installedSet := make(map[string]bool, len(installed))
		for _, p := range installed {
			key := p.Name
			if p.Marketplace != "" {
				key += "@" + p.Marketplace
			}
			installedSet[key] = true
		}
		return pluginPickerLoadedMsg{available: available, installed: installedSet}
	}
}

func (m *PluginsPickerModel) buildEntries(available []plugins.Plugin, installed map[string]bool) {
	// Group by marketplace
	type group struct {
		name    string
		plugins []plugins.Plugin
	}
	var groups []group
	seen := make(map[string]int)
	for _, p := range available {
		mp := p.Marketplace
		if mp == "" {
			mp = "unknown"
		}
		if idx, ok := seen[mp]; ok {
			groups[idx].plugins = append(groups[idx].plugins, p)
		} else {
			seen[mp] = len(groups)
			groups = append(groups, group{name: mp, plugins: []plugins.Plugin{p}})
		}
	}

	// Count total entries for pre-allocation
	count := 0
	for _, g := range groups {
		count += 1 + len(g.plugins)
	}

	m.entries = make([]pluginPickerEntry, 0, count)
	for _, g := range groups {
		m.entries = append(m.entries, pluginPickerEntry{isHeader: true, header: strings.ToUpper(g.name)})
		for i := range g.plugins {
			key := g.plugins[i].Name + "@" + g.name
			m.entries = append(m.entries, pluginPickerEntry{
				plugin:      &g.plugins[i],
				isInstalled: installed[key],
			})
		}
	}

	m.cursor = m.nextSelectable(0, 1)
}

func (m *PluginsPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { //nolint:dupl // picker views share identical update structure by design
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case pluginPickerLoadedMsg:
		m.loading = false
		m.buildEntries(msg.available, msg.installed)
		return m, nil

	case pluginPickerErrorMsg:
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

func (m *PluginsPickerModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) { //nolint:dupl // picker views share identical keyboard handling by design
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

func (m *PluginsPickerModel) activateEntry() tea.Cmd {
	if m.cursor < 0 || m.cursor >= len(m.entries) {
		return nil
	}
	e := m.entries[m.cursor]
	if e.plugin == nil || e.isInstalled {
		return nil
	}

	name := e.plugin.Name
	if e.plugin.Marketplace != "" {
		name += "@" + e.plugin.Marketplace
	}

	exe, _ := os.Executable()
	cmd := exec.Command(exe, "plugins", "add", name)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return tea.ExecProcess(cmd, func(_ error) tea.Msg {
		return PopViewMsg{}
	})
}

func (m *PluginsPickerModel) visibleLines() int {
	v := m.height - bannerOverhead - footerOverhead
	if v < 1 {
		return 1
	}
	return v
}

func (m *PluginsPickerModel) clampScroll() {
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

func (m *PluginsPickerModel) movePage(dir int) {
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

func (m *PluginsPickerModel) nextSelectable(start, dir int) int {
	for i := start; i >= 0 && i < len(m.entries); i += dir {
		if !m.entries[i].isHeader {
			return i
		}
	}
	return m.cursor
}

func (m *PluginsPickerModel) selectableCount() int {
	n := 0
	for _, e := range m.entries {
		if !e.isHeader {
			n++
		}
	}
	return n
}

func (m *PluginsPickerModel) selectableIndex() int {
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

func (m *PluginsPickerModel) View() tea.View {
	var b strings.Builder
	b.WriteString(m.theme.SectionBanner("Install Plugin"))

	if m.loading {
		b.WriteString("\n  Loading available plugins...")
		return tea.NewView(b.String())
	}

	if m.err != "" {
		b.WriteString("\n  ")
		b.WriteString(lipgloss.NewStyle().Foreground(m.theme.Error).Render(m.err))
		b.WriteString("\n\n  Press esc to go back.\n")
		return tea.NewView(b.String())
	}

	if len(m.entries) == 0 {
		b.WriteString("\n  No marketplace plugins found.\n")
		b.WriteString("\n  Add a marketplace first:")
		b.WriteString("\n    claude plugin marketplace add anthropics/claude-plugins-official\n")
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
		"%s navigate  %s page  %s/%s top/bottom  %s install  %s back  %d/%d",
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
func (m *PluginsPickerModel) buildLines() []string {
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#06B6D4"))
	installedStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)

	lines := make([]string, 0, len(m.entries))
	for i, e := range m.entries {
		if e.isHeader {
			lines = append(lines, "  "+sectionStyle.Render(e.header))
			continue
		}

		selected := i == m.cursor

		switch { //nolint:dupl // picker views share identical styling logic by design
		case e.isInstalled:
			lines = append(lines, installedStyle.Render("  ✓ "+e.plugin.Name+" (installed)"))
		case selected:
			cursor := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render("> ")
			styledLabel := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render(e.plugin.Name)
			desc := ""
			if e.plugin.Description != "" {
				desc = "  " + lipgloss.NewStyle().Foreground(m.theme.Primary).Render(e.plugin.Description)
			}
			lines = append(lines, "  "+cursor+styledLabel+desc)
		default:
			label := e.plugin.Name
			if e.plugin.Description != "" {
				label += "  " + lipgloss.NewStyle().Foreground(m.theme.Muted).Render(e.plugin.Description)
			}
			lines = append(lines, "    "+label)
		}
	}
	return lines
}
