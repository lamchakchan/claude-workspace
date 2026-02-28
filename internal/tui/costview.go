package tui

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lamchakchan/claude-workspace/internal/cost"
)

// costTab identifies a time-window tab in the cost view.
type costTab int

const (
	tabDaily costTab = iota
	tabWeekly
	tabMonthly
	tabSession
	tabBlocks
	costTabCount // sentinel for modular arithmetic
)

var costTabLabels = []string{"Daily", "Weekly", "Monthly", "Session", "Blocks"}
var costTabArgs = []string{"daily", "weekly", "monthly", "session", "blocks"}

// costTabLoadedMsg carries the loaded content for a specific tab.
type costTabLoadedMsg struct {
	tab     costTab
	gen     int // load generation — used to discard stale results
	content string
	err     string
}

// CostModel displays usage and cost output with tab-based time window switching.
type CostModel struct {
	theme      *Theme
	activeTab  costTab
	viewport   viewport.Model
	loading    bool
	err        string
	ready      bool
	width      int
	height     int
	loadGen    int                // incremented on each tab switch
	cancelLoad context.CancelFunc // cancels the in-flight subprocess
}

// costHeaderLines is the number of lines used by the tab bar header.
const costHeaderLines = 4 // banner + tab bar + separator + blank

// costFooterLines is the number of lines used by the footer.
const costFooterLines = 2 // blank + help text

// NewCost creates a new cost output viewer with tab support.
func NewCost(theme *Theme) *CostModel {
	return &CostModel{
		theme:     theme,
		activeTab: tabDaily,
		loading:   true,
	}
}

func (m *CostModel) Init() tea.Cmd {
	return m.loadTab(tabDaily)
}

func (m *CostModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeViewport()
		return m, nil

	case costTabLoadedMsg:
		cmd := m.handleTabLoaded(msg)
		return m, cmd

	case tea.KeyPressMsg:
		if IsQuit(msg) || IsBack(msg) {
			if m.cancelLoad != nil {
				m.cancelLoad()
			}
			return m, func() tea.Msg { return PopViewMsg{} }
		}
		if cmd := m.handleCostKey(msg); cmd != nil {
			return m, cmd
		}
	}

	// Forward scroll keys to viewport
	if m.ready && !m.loading {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleTabLoaded processes a loaded tab result.
func (m *CostModel) handleTabLoaded(msg costTabLoadedMsg) tea.Cmd {
	// Ignore stale loads from a previously active tab or generation
	if msg.tab != m.activeTab || msg.gen != m.loadGen {
		return nil
	}
	m.loading = false
	if msg.err != "" {
		m.err = msg.err
		return nil
	}
	m.err = ""
	m.viewport = viewport.New(
		viewport.WithWidth(max(1, m.width-scrollbarWidth)),
		viewport.WithHeight(max(1, m.height-costHeaderLines-costFooterLines)),
	)
	m.viewport.SoftWrap = true
	m.viewport.SetContent(msg.content)
	m.ready = true
	return nil
}

// handleCostKey handles tab switching keys.
func (m *CostModel) handleCostKey(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "1":
		return m.switchTab(tabDaily)
	case "2":
		return m.switchTab(tabWeekly)
	case "3":
		return m.switchTab(tabMonthly)
	case "4":
		return m.switchTab(tabSession)
	case "5":
		return m.switchTab(tabBlocks)
	case keyTab, "right", "l":
		next := (m.activeTab + 1) % costTabCount
		return m.switchTab(next)
	case keyShiftTab, "left", "h":
		next := (m.activeTab + costTabCount - 1) % costTabCount
		return m.switchTab(next)
	}
	return nil
}

func (m *CostModel) View() tea.View {
	var b strings.Builder

	// Title banner
	b.WriteString(m.theme.SectionBanner("Usage & Costs"))

	// Tab bar
	b.WriteString(m.renderTabs())
	b.WriteString("\n")

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

	if m.ready {
		vpContent := m.viewport.View()
		totalLines := strings.Count(m.viewport.GetContent(), "\n") + 1
		vpHeight := m.viewport.Height()
		bar := renderScrollbar(vpHeight, totalLines, vpHeight, m.viewport.ScrollPercent(), m.theme)
		if bar != "" {
			b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, vpContent, " ", bar))
		} else {
			b.WriteString(vpContent)
		}
		b.WriteString("\n")
	}

	// Footer
	pct := int(m.viewport.ScrollPercent() * 100)
	help := fmt.Sprintf(
		"%s switch  %s/%s cycle  %s scroll  %s page  %s/%s top/bottom  %s back  %s",
		m.theme.HelpKey.Render("1-5"),
		m.theme.HelpKey.Render(keyTab),
		m.theme.HelpKey.Render(keyShiftTab),
		m.theme.HelpKey.Render("j/k"),
		m.theme.HelpKey.Render("pgup/pgdn"),
		m.theme.HelpKey.Render("g"),
		m.theme.HelpKey.Render("G"),
		m.theme.HelpKey.Render(keyEsc),
		m.theme.HelpKey.Render(fmt.Sprintf("%d", pct))+"%",
	)
	b.WriteString(help)

	return tea.NewView(b.String())
}

// renderTabs renders the tab bar with the active tab highlighted.
func (m *CostModel) renderTabs() string {
	selected := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Primary)
	unselected := lipgloss.NewStyle().Foreground(m.theme.Muted)
	separator := lipgloss.NewStyle().Foreground(m.theme.Muted)

	tabs := make([]string, len(costTabLabels))
	for i, label := range costTabLabels {
		key := fmt.Sprintf("[%d]", i+1)
		if costTab(i) == m.activeTab {
			tabs[i] = selected.Render(key+" "+label) + "  "
		} else {
			tabs[i] = unselected.Render(key+" "+label) + "  "
		}
	}

	line := "  " + strings.Join(tabs, "")
	rule := separator.Render("  " + strings.Repeat("─", max(1, m.width-4)))
	return line + "\n" + rule
}

// switchTab changes the active tab and triggers a reload.
func (m *CostModel) switchTab(tab costTab) tea.Cmd {
	if tab == m.activeTab && !m.loading {
		return nil // already on this tab
	}
	// Cancel any in-flight subprocess from the previous tab
	if m.cancelLoad != nil {
		m.cancelLoad()
	}
	m.activeTab = tab
	m.loading = true
	m.err = ""
	m.loadGen++
	return m.loadTab(tab)
}

// loadTab returns a tea.Cmd that loads the content for the given tab.
func (m *CostModel) loadTab(tab costTab) tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelLoad = cancel
	width := m.width
	theme := m.theme
	gen := m.loadGen
	subcommand := costTabArgs[tab]
	return func() tea.Msg {
		defer cancel()
		return loadTabContent(ctx, tab, gen, subcommand, width, theme)
	}
}

// loadTabContent loads chart data and table output for any tab sequentially.
func loadTabContent(ctx context.Context, tab costTab, gen int, subcommand string, width int, theme *Theme) costTabLoadedMsg {
	// First: get JSON for chart (sequential to avoid concurrent bun processes)
	var chartData string
	jsonOut, err := cost.RunCaptureContext(ctx, []string{subcommand, "--json"})
	if ctx.Err() != nil {
		return costTabLoadedMsg{tab: tab, gen: gen}
	}
	if err == nil {
		entries, parseErr := cost.ParseCostJSON(subcommand, jsonOut)
		if parseErr == nil && len(entries) > 0 && theme != nil {
			chartData = renderBarChart(entries, max(40, width), theme)
		}
	}

	// Second: get table output
	tableOut, err := cost.RunCaptureContext(ctx, []string{subcommand})
	if ctx.Err() != nil {
		return costTabLoadedMsg{tab: tab, gen: gen}
	}
	if err != nil {
		return costTabLoadedMsg{tab: tab, gen: gen, err: err.Error()}
	}

	var content strings.Builder
	if chartData != "" {
		content.WriteString(chartData)
		content.WriteString("\n")
	}
	content.WriteString(tableOut)
	return costTabLoadedMsg{tab: tab, gen: gen, content: content.String()}
}

// resizeViewport updates the viewport dimensions when the window size changes.
func (m *CostModel) resizeViewport() {
	if !m.ready {
		return
	}
	m.viewport.SetWidth(max(1, m.width-scrollbarWidth))
	m.viewport.SetHeight(max(1, m.height-costHeaderLines-costFooterLines))
}
