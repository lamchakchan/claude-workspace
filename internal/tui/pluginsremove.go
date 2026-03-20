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

const scopeDefault = "user"

// pluginRemoveState represents the current phase of the remove flow.
type pluginRemoveState int

const (
	pluginRemovePicking    pluginRemoveState = iota // selecting a plugin
	pluginRemoveConfirming                          // confirming removal
	pluginRemoveRemoving                            // executing removal
)

// pluginRemoveLoadedMsg carries discovered plugins to the view.
type pluginRemoveLoadedMsg struct {
	plugins []plugins.Plugin
}

// pluginRemoveErrorMsg carries an error from discovery.
type pluginRemoveErrorMsg struct {
	err string
}

// PluginsRemoveModel is the interactive plugin removal screen.
type PluginsRemoveModel struct {
	theme    *Theme
	state    pluginRemoveState
	plugins  []plugins.Plugin
	cursor   int
	scroll   int
	width    int
	height   int
	loading  bool
	err      string
	confirm  *ConfirmModel
	selected plugins.Plugin
}

// NewPluginsRemove creates a new plugin removal screen.
func NewPluginsRemove(theme *Theme) *PluginsRemoveModel {
	return &PluginsRemoveModel{
		theme:   theme,
		loading: true,
	}
}

func (m *PluginsRemoveModel) Init() tea.Cmd {
	return func() tea.Msg {
		installed, err := plugins.DiscoverInstalled()
		if err != nil {
			return pluginRemoveErrorMsg{err: err.Error()}
		}
		return pluginRemoveLoadedMsg{plugins: installed}
	}
}

func (m *PluginsRemoveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { //nolint:dupl // remove views share identical update structure by design
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case pluginRemoveLoadedMsg:
		m.loading = false
		m.plugins = msg.plugins
		return m, nil

	case pluginRemoveErrorMsg:
		m.loading = false
		m.err = msg.err
		return m, nil

	case ConfirmResult:
		if msg.Confirmed {
			cmd := m.executeRemove()
			return m, cmd
		}
		m.state = pluginRemovePicking
		m.confirm = nil
		return m, nil

	case tea.KeyPressMsg:
		switch m.state {
		case pluginRemovePicking:
			return m.updatePicking(msg)
		case pluginRemoveConfirming:
			return m.updateConfirming(msg)
		}
	}
	return m, nil
}

func (m *PluginsRemoveModel) updatePicking(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
		if m.cursor < len(m.plugins)-1 {
			m.cursor++
			m.clampScroll()
		}
	case keyEnter:
		if len(m.plugins) > 0 {
			m.selected = m.plugins[m.cursor]
			name := m.selected.Name
			if m.selected.Marketplace != "" {
				name += "@" + m.selected.Marketplace
			}
			scope := m.selected.Scope
			if scope == "" {
				scope = scopeDefault
			}
			m.confirm = NewConfirm(
				"Remove Plugin",
				fmt.Sprintf("Remove '%s' (%s scope)?", name, scope),
				false,
				m.theme,
			)
			m.state = pluginRemoveConfirming
		}
	}
	return m, nil
}

func (m *PluginsRemoveModel) updateConfirming(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.confirm, cmd = m.confirm.Update(msg)
	return m, cmd
}

func (m *PluginsRemoveModel) executeRemove() tea.Cmd {
	m.state = pluginRemoveRemoving
	name := m.selected.Name
	if m.selected.Marketplace != "" {
		name += "@" + m.selected.Marketplace
	}
	scope := m.selected.Scope
	if scope == "" {
		scope = scopeDefault
	}
	exe, _ := os.Executable()
	cmd := exec.Command(exe, "plugins", "remove", name, "--scope", scope)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return tea.ExecProcess(cmd, func(_ error) tea.Msg {
		return PopViewMsg{}
	})
}

func (m *PluginsRemoveModel) visibleLines() int {
	v := m.height - bannerOverhead - footerOverhead
	if v < 1 {
		return 1
	}
	return v
}

func (m *PluginsRemoveModel) clampScroll() {
	visible := m.visibleLines()
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+visible {
		m.scroll = m.cursor - visible + 1
	}
	maxScroll := len(m.plugins) - visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scroll > maxScroll {
		m.scroll = maxScroll
	}
}

func (m *PluginsRemoveModel) View() tea.View {
	var b strings.Builder
	b.WriteString(m.theme.SectionBanner("Remove Plugin"))

	if m.loading {
		b.WriteString("\n  Loading plugins...")
		return tea.NewView(b.String())
	}

	if m.err != "" {
		b.WriteString("\n  ")
		b.WriteString(lipgloss.NewStyle().Foreground(m.theme.Error).Render(m.err))
		b.WriteString("\n\n  Press esc to go back.\n")
		return tea.NewView(b.String())
	}

	if m.state == pluginRemoveConfirming && m.confirm != nil {
		b.WriteString("\n")
		b.WriteString(m.confirm.View())
		return tea.NewView(b.String())
	}

	if len(m.plugins) == 0 {
		b.WriteString("\n  No plugins installed.\n")
		b.WriteString("\n  Press esc to go back.\n")
		return tea.NewView(b.String())
	}

	// Build rendered lines
	scopeStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)
	lines := make([]string, 0, len(m.plugins))
	for i, p := range m.plugins {
		scope := p.Scope
		if scope == "" {
			scope = scopeDefault
		}
		name := p.Name
		if p.Marketplace != "" {
			name += "@" + p.Marketplace
		}
		label := fmt.Sprintf("%s %s", name, scopeStyle.Render("("+scope+")"))
		if i == m.cursor {
			cursor := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render("> ")
			styledName := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render(name)
			styledScope := lipgloss.NewStyle().Foreground(m.theme.Primary).Render("(" + scope + ")")
			label = cursor + styledName + " " + styledScope
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
		m.cursor+1, len(m.plugins),
	)
	b.WriteString(help)

	return tea.NewView(b.String())
}
