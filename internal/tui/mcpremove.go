package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lamchakchan/claude-workspace/internal/mcp"
)

// mcpRemoveState represents the current phase of the remove flow.
type mcpRemoveState int

const (
	mcpRemovePicking    mcpRemoveState = iota // selecting a server
	mcpRemoveConfirming                       // confirming removal
	mcpRemoveRemoving                         // executing removal
)

// mcpRemoveLoadedMsg carries discovered servers to the view.
type mcpRemoveLoadedMsg struct {
	servers []mcp.Server
}

// mcpRemoveErrorMsg carries an error from discovery.
type mcpRemoveErrorMsg struct {
	err string
}

// McpRemoveModel is the interactive MCP server removal screen.
type McpRemoveModel struct {
	theme    *Theme
	state    mcpRemoveState
	servers  []mcp.Server // removable servers (excludes managed)
	cursor   int
	scroll   int
	width    int
	height   int
	loading  bool
	err      string
	confirm  *ConfirmModel
	selected mcp.Server
}

// NewMcpRemove creates a new MCP server removal screen.
func NewMcpRemove(theme *Theme) *McpRemoveModel {
	return &McpRemoveModel{
		theme:   theme,
		loading: true,
	}
}

func (m *McpRemoveModel) Init() tea.Cmd {
	return func() tea.Msg {
		servers, err := mcp.DiscoverServers()
		if err != nil {
			return mcpRemoveErrorMsg{err: err.Error()}
		}
		return mcpRemoveLoadedMsg{servers: servers}
	}
}

func (m *McpRemoveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case mcpRemoveLoadedMsg:
		m.loading = false
		m.servers = filterRemovable(msg.servers)
		return m, nil

	case mcpRemoveErrorMsg:
		m.loading = false
		m.err = msg.err
		return m, nil

	case ConfirmResult:
		if msg.Confirmed {
			cmd := m.executeRemove()
			return m, cmd
		}
		// Cancelled — go back to picker
		m.state = mcpRemovePicking
		m.confirm = nil
		return m, nil

	case tea.KeyPressMsg:
		switch m.state {
		case mcpRemovePicking:
			return m.updatePicking(msg)
		case mcpRemoveConfirming:
			return m.updateConfirming(msg)
		}
	}
	return m, nil
}

func (m *McpRemoveModel) updatePicking(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
		if m.cursor < len(m.servers)-1 {
			m.cursor++
			m.clampScroll()
		}
	case keyEnter:
		if len(m.servers) > 0 {
			m.selected = m.servers[m.cursor]
			m.confirm = NewConfirm(
				"Remove Server",
				fmt.Sprintf("Remove '%s' from %s config?", m.selected.Name, m.selected.Scope),
				false,
				m.theme,
			)
			m.state = mcpRemoveConfirming
		}
	}
	return m, nil
}

func (m *McpRemoveModel) updateConfirming(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.confirm, cmd = m.confirm.Update(msg)
	return m, cmd
}

func (m *McpRemoveModel) executeRemove() tea.Cmd {
	m.state = mcpRemoveRemoving
	exe, _ := os.Executable()
	cmd := exec.Command(exe, "mcp", "remove", m.selected.Name, "--scope", m.selected.Scope)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return tea.ExecProcess(cmd, func(_ error) tea.Msg {
		return PopViewMsg{}
	})
}

// visibleLines returns how many content lines fit in the viewport.
func (m *McpRemoveModel) visibleLines() int {
	v := m.height - bannerOverhead - footerOverhead
	if v < 1 {
		return 1
	}
	return v
}

// clampScroll ensures the scroll offset keeps the cursor visible.
func (m *McpRemoveModel) clampScroll() {
	visible := m.visibleLines()
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+visible {
		m.scroll = m.cursor - visible + 1
	}
	maxScroll := len(m.servers) - visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scroll > maxScroll {
		m.scroll = maxScroll
	}
}

func (m *McpRemoveModel) View() tea.View {
	var b strings.Builder
	b.WriteString(m.theme.SectionBanner("Remove MCP Server"))

	if m.loading {
		b.WriteString("\n  Loading servers...")
		return tea.NewView(b.String())
	}

	if m.err != "" {
		b.WriteString("\n  ")
		b.WriteString(lipgloss.NewStyle().Foreground(m.theme.Error).Render(m.err))
		b.WriteString("\n\n  Press esc to go back.\n")
		return tea.NewView(b.String())
	}

	if m.state == mcpRemoveConfirming && m.confirm != nil {
		b.WriteString("\n")
		b.WriteString(m.confirm.View())
		return tea.NewView(b.String())
	}

	if len(m.servers) == 0 {
		b.WriteString("\n  No servers configured.\n")
		b.WriteString("\n  Press esc to go back.\n")
		return tea.NewView(b.String())
	}

	// Build rendered lines
	scopeStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)
	lines := make([]string, 0, len(m.servers))
	for i, srv := range m.servers {
		label := fmt.Sprintf("%s %s", srv.Name, scopeStyle.Render("("+srv.Scope+")"))
		if i == m.cursor {
			cursor := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render("> ")
			name := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render(srv.Name)
			scope := lipgloss.NewStyle().Foreground(m.theme.Primary).Render("(" + srv.Scope + ")")
			label = cursor + name + " " + scope
			lines = append(lines, "  "+label)
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
		m.cursor+1, len(m.servers),
	)
	b.WriteString(help)

	return tea.NewView(b.String())
}

// filterRemovable returns servers that are not managed (enterprise-controlled).
func filterRemovable(servers []mcp.Server) []mcp.Server {
	result := make([]mcp.Server, 0, len(servers))
	for _, s := range servers {
		if s.Scope != "managed" {
			result = append(result, s)
		}
	}
	return result
}
