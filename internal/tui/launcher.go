package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// commandItem represents a single command in the launcher menu.
type commandItem struct {
	name    string
	desc    string
	icon    string
	command string   // CLI command to execute (e.g., "setup", "attach")
	args    []string // default args
}

// commandGroup represents a section of related commands.
type commandGroup struct {
	title string
	items []commandItem
}

// launcherModel is the interactive command launcher shown when no args are given.
type launcherModel struct {
	groups   []commandGroup
	cursor   int // flat index across all items
	total    int // total number of items
	width    int
	height   int
	theme    *Theme
	version  string
	quitting bool
}

func newLauncher(version string, theme *Theme) *launcherModel {
	groups := []commandGroup{
		{
			title: "Getting Started",
			items: []commandItem{
				{name: "Setup", desc: "First-time setup & API key provisioning", icon: "⚙ ", command: "setup"},
				{name: "Attach", desc: "Overlay platform config onto a project", icon: "📎", command: "attach"},
				{name: "Enrich", desc: "Re-generate CLAUDE.md with AI analysis", icon: "✨", command: "enrich"},
				{name: "Sandbox", desc: "Create a sandboxed branch worktree", icon: "🔀", command: "sandbox"},
			},
		},
		{
			title: "MCP Servers",
			items: []commandItem{
				{name: "Add Server", desc: "Add a local or remote MCP server", icon: "➕", command: "mcp", args: []string{"add"}},
				{name: "List Servers", desc: "Show all configured servers", icon: "📋", command: "mcp", args: []string{"list"}},
			},
		},
		{
			title: "Inspect & Manage",
			items: []commandItem{
				{name: "Doctor", desc: "Check platform configuration health", icon: "🩺", command: "doctor"},
				{name: "Skills", desc: "List available skills and personal commands", icon: "🛠 ", command: "skills"},
				{name: "Sessions", desc: "Browse and review session prompts", icon: "💬", command: "sessions"},
				{name: "Memory", desc: "Inspect and manage memory layers", icon: "🧠", command: "memory"},
				{name: "Cost", desc: "View usage and costs", icon: "💰", command: "cost"},
			},
		},
		{
			title: "Maintenance",
			items: []commandItem{
				{name: "Upgrade", desc: "Upgrade claude-workspace and CLI", icon: "⬆ ", command: "upgrade"},
				{name: "Statusline", desc: "Configure Claude Code statusline", icon: "📊", command: "statusline"},
			},
		},
	}

	total := 0
	for _, g := range groups {
		total += len(g.items)
	}

	return &launcherModel{
		groups:  groups,
		total:   total,
		theme:   theme,
		version: version,
	}
}

// selectedItem returns the currently selected command item.
func (m *launcherModel) selectedItem() commandItem {
	idx := 0
	for _, g := range m.groups {
		for _, item := range g.items {
			if idx == m.cursor {
				return item
			}
			idx++
		}
	}
	return commandItem{}
}

func (m *launcherModel) Init() tea.Cmd {
	return nil
}

func (m *launcherModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		if IsQuit(msg) {
			m.quitting = true
			return m, tea.Quit
		}

		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < m.total-1 {
				m.cursor++
			}
		case keyEnter:
			item := m.selectedItem()
			cmd := m.activate(&item)
			return m, cmd
		case "?":
			return m, pushView(NewHelp(m.theme))
		}
	}
	return m, nil
}

// activate returns the appropriate command for the selected item.
// Commands with dedicated TUI views push them; data-display commands run
// inline so users return to the launcher; others exit the TUI to run.
func (m *launcherModel) activate(item *commandItem) tea.Cmd {
	switch item.command {
	// TUI form screens
	case "attach":
		return pushView(NewAttach(m.theme))
	case "enrich":
		return pushView(NewEnrich(m.theme))
	case "sandbox":
		return pushView(NewSandbox(m.theme))
	case "upgrade":
		return pushView(NewUpgrade(m.version, m.theme))
	case "mcp":
		if len(item.args) > 0 && item.args[0] == "add" {
			return pushView(NewMcpAdd(m.theme))
		}
		return pushView(NewMcpList(m.theme))

	// In-app viewers for data display commands
	case "doctor":
		return pushView(NewDoctor(m.theme))
	case "skills":
		return pushView(NewSkills(m.theme))
	case "sessions":
		return pushView(NewSessions(m.theme))
	case "memory":
		return pushView(NewMemory(m.theme))
	case "cost":
		return pushView(NewCost(m.theme))

	// Setup and statusline suspend the TUI, run their interactive flow,
	// then resume back to the launcher when done.
	default:
		return execAndReturn(item.command, item.args)
	}
}

func (m *launcherModel) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}

	var b strings.Builder

	// Banner
	title := m.theme.Title.Render(fmt.Sprintf("claude-workspace  %s", m.version))
	subtitle := m.theme.Subtitle.Render("Claude Code Platform Engineering Kit")
	banner := m.theme.Banner.Render(title + "\n" + subtitle)
	b.WriteString(banner)
	b.WriteString("\n")

	// Compute max name width for column alignment
	maxName := 0
	for _, group := range m.groups {
		for _, item := range group.items {
			if len(item.name) > maxName {
				maxName = len(item.name)
			}
		}
	}

	// Command groups
	flatIdx := 0
	for _, group := range m.groups {
		b.WriteString(m.theme.SectionHead.Render(group.title))
		b.WriteString("\n")

		for _, item := range group.items {
			selected := flatIdx == m.cursor

			// Pad name to fixed width before styling
			paddedName := fmt.Sprintf("%-*s", maxName, item.name)

			cursor := "  "
			icon := lipgloss.NewStyle().Foreground(m.theme.Muted).Render(item.icon)
			name := paddedName
			desc := lipgloss.NewStyle().Foreground(m.theme.Muted).Render(item.desc)

			if selected {
				cursor = lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render("> ")
				name = lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render(paddedName)
				icon = lipgloss.NewStyle().Foreground(m.theme.Primary).Render(item.icon)
			}

			fmt.Fprintf(&b, "%s%s %s  %s\n", cursor, icon, name, desc)
			flatIdx++
		}
	}

	// Footer
	b.WriteString("\n")
	help := fmt.Sprintf(
		"%s navigate  %s select  %s help  %s quit",
		m.theme.HelpKey.Render("↑/↓"),
		m.theme.HelpKey.Render(keyEnter),
		m.theme.HelpKey.Render("?"),
		m.theme.HelpKey.Render("q"),
	)
	b.WriteString(help)
	b.WriteString("\n")

	return tea.NewView(b.String())
}

func pushView(model tea.Model) tea.Cmd {
	return func() tea.Msg {
		return PushViewMsg{Model: model}
	}
}

func execAndReturn(command string, args []string) tea.Cmd {
	return func() tea.Msg {
		return ExecAndReturnMsg{Command: command, Args: args}
	}
}
