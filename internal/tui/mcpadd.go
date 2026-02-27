package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// McpAddModel is the interactive MCP server add form screen.
type McpAddModel struct {
	theme *Theme
	form  *FormModel
}

// NewMcpAdd creates a new MCP add form screen.
func NewMcpAdd(theme *Theme) *McpAddModel {
	fields := []FormField{
		{Label: "Server name", Placeholder: "e.g. postgres, brave-search", Required: true},
		{Label: "API key env var", Placeholder: "e.g. DATABASE_URL (leave blank if not needed)"},
		{Label: "Command", Placeholder: "e.g. npx -y @bytebase/dbhub", Required: true},
	}

	return &McpAddModel{
		theme: theme,
		form:  NewForm("Add MCP Server", fields, theme),
	}
}

func (m *McpAddModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m *McpAddModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.form, cmd = formViewUpdate(m.form, msg, m.runAdd)
	return m, cmd
}

func (m *McpAddModel) runAdd(values []string) tea.Cmd {
	name := strings.TrimSpace(values[0])
	apiKey := strings.TrimSpace(values[1])
	cmdStr := strings.TrimSpace(values[2])

	args := []string{"mcp", "add", name}
	if apiKey != "" {
		args = append(args, "--api-key", apiKey)
	}

	cmdParts := strings.Fields(cmdStr)
	if len(cmdParts) > 0 {
		args = append(args, "--")
		args = append(args, cmdParts...)
	}

	exe, _ := os.Executable()
	cmd := exec.Command(exe, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return tea.ExecProcess(cmd, func(_ error) tea.Msg {
		return PopViewMsg{}
	})
}

func (m *McpAddModel) View() tea.View {
	return tea.NewView(fmt.Sprintf("%s\n%s",
		m.theme.SectionBanner("Add MCP Server"),
		m.form.View(),
	))
}
