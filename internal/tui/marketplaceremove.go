package tui //nolint:dupl // remove views share identical structure by design

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lamchakchan/claude-workspace/internal/plugins"
)

// marketplaceRemoveState represents the current phase of the remove flow.
type marketplaceRemoveState int

const (
	marketplaceRemovePicking    marketplaceRemoveState = iota // selecting a marketplace
	marketplaceRemoveConfirming                               // confirming removal
	marketplaceRemoveRemoving                                 // executing removal
)

// marketplaceRemoveLoadedMsg carries discovered marketplaces to the view.
type marketplaceRemoveLoadedMsg struct {
	marketplaces []plugins.Marketplace
}

// marketplaceRemoveErrorMsg carries an error from discovery.
type marketplaceRemoveErrorMsg struct {
	err string
}

// MarketplaceRemoveModel is the interactive marketplace removal screen.
type MarketplaceRemoveModel struct {
	theme        *Theme
	state        marketplaceRemoveState
	marketplaces []plugins.Marketplace
	cursor       int
	scroll       int
	width        int
	height       int
	loading      bool
	err          string
	confirm      *ConfirmModel
	selected     plugins.Marketplace
}

// NewMarketplaceRemove creates a new marketplace removal screen.
func NewMarketplaceRemove(theme *Theme) *MarketplaceRemoveModel {
	return &MarketplaceRemoveModel{
		theme:   theme,
		loading: true,
	}
}

func (m *MarketplaceRemoveModel) Init() tea.Cmd {
	return func() tea.Msg {
		marketplaces, err := plugins.DiscoverMarketplaces()
		if err != nil {
			return marketplaceRemoveErrorMsg{err: err.Error()}
		}
		return marketplaceRemoveLoadedMsg{marketplaces: marketplaces}
	}
}

func (m *MarketplaceRemoveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { //nolint:dupl // remove views share identical update structure by design
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case marketplaceRemoveLoadedMsg:
		m.loading = false
		m.marketplaces = msg.marketplaces
		return m, nil

	case marketplaceRemoveErrorMsg:
		m.loading = false
		m.err = msg.err
		return m, nil

	case ConfirmResult:
		if msg.Confirmed {
			cmd := m.executeRemove()
			return m, cmd
		}
		m.state = marketplaceRemovePicking
		m.confirm = nil
		return m, nil

	case tea.KeyPressMsg:
		switch m.state {
		case marketplaceRemovePicking:
			return m.updatePicking(msg)
		case marketplaceRemoveConfirming:
			return m.updateConfirming(msg)
		}
	}
	return m, nil
}

func (m *MarketplaceRemoveModel) updatePicking(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if IsQuit(msg) || IsBack(msg) {
		return m, func() tea.Msg { return PopViewMsg{} }
	}

	switch msg.String() {
	case keyUp, "k":
		if m.cursor > 0 {
			m.cursor--
			m.clampScroll()
		}
	case keyDown, "j":
		if m.cursor < len(m.marketplaces)-1 {
			m.cursor++
			m.clampScroll()
		}
	case keyEnter:
		if len(m.marketplaces) > 0 {
			m.selected = m.marketplaces[m.cursor]
			m.confirm = NewConfirm(
				"Remove Marketplace",
				fmt.Sprintf("Remove marketplace '%s'?", m.selected.Name),
				false,
				m.theme,
			)
			m.state = marketplaceRemoveConfirming
		}
	}
	return m, nil
}

func (m *MarketplaceRemoveModel) updateConfirming(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.confirm, cmd = m.confirm.Update(msg)
	return m, cmd
}

func (m *MarketplaceRemoveModel) executeRemove() tea.Cmd {
	m.state = marketplaceRemoveRemoving
	exe, _ := os.Executable()
	cmd := exec.Command(exe, "plugins", "marketplace", "remove", m.selected.Name)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return tea.ExecProcess(cmd, func(_ error) tea.Msg {
		return PopViewMsg{}
	})
}

func (m *MarketplaceRemoveModel) visibleLines() int {
	v := m.height - bannerOverhead - footerOverhead
	if v < 1 {
		return 1
	}
	return v
}

func (m *MarketplaceRemoveModel) clampScroll() {
	visible := m.visibleLines()
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+visible {
		m.scroll = m.cursor - visible + 1
	}
	maxScroll := len(m.marketplaces) - visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scroll > maxScroll {
		m.scroll = maxScroll
	}
}

func (m *MarketplaceRemoveModel) View() tea.View {
	var b strings.Builder
	b.WriteString(m.theme.SectionBanner("Remove Marketplace"))

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

	if m.state == marketplaceRemoveConfirming && m.confirm != nil {
		b.WriteString("\n")
		b.WriteString(m.confirm.View())
		return tea.NewView(b.String())
	}

	if len(m.marketplaces) == 0 {
		b.WriteString("\n  No marketplaces configured.\n")
		b.WriteString("\n  Press esc to go back.\n")
		return tea.NewView(b.String())
	}

	// Build rendered lines
	repoStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)
	lines := make([]string, 0, len(m.marketplaces))
	for i, mp := range m.marketplaces {
		label := mp.Name
		if mp.Repo != "" {
			label += " " + repoStyle.Render("("+mp.Repo+")")
		}
		if i == m.cursor {
			cursor := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render("> ")
			styledName := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render(mp.Name)
			if mp.Repo != "" {
				styledName += " " + lipgloss.NewStyle().Foreground(m.theme.Primary).Render("("+mp.Repo+")")
			}
			lines = append(lines, "  "+cursor+styledName)
		} else {
			lines = append(lines, "    "+label)
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

	for i := start; i < end; i++ {
		b.WriteString(lines[i])
		if i < end-1 {
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")

	// Footer
	b.WriteString("\n")
	help := fmt.Sprintf(
		"%s navigate  %s select  %s back  %d/%d",
		m.theme.HelpKey.Render("j/k"),
		m.theme.HelpKey.Render(keyEnter),
		m.theme.HelpKey.Render("esc"),
		m.cursor+1, len(m.marketplaces),
	)
	b.WriteString(help)

	return tea.NewView(b.String())
}
