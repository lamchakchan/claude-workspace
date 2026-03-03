package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/lamchakchan/claude-workspace/internal/mcpregistry"
)

// McpAddModel is the interactive MCP server add form screen.
type McpAddModel struct {
	theme     *Theme
	form      *FormModel
	transport mcpregistry.Transport
	title     string // banner title
}

// NewMcpAdd creates a new blank stdio MCP add form screen.
func NewMcpAdd(theme *Theme) *McpAddModel {
	fields := []FormField{
		{Label: "Server name", Placeholder: "e.g. postgres, brave-search", Required: true},
		{Label: "API key env var", Placeholder: "e.g. DATABASE_URL (leave blank if not needed)"},
		{Label: "Command", Placeholder: "e.g. npx -y @bytebase/dbhub", Required: true},
		{Label: "Scope", Choices: []string{"local", "user", "project"}},
	}

	return &McpAddModel{
		theme:     theme,
		form:      NewForm("Add MCP Server (stdio)", fields, theme),
		transport: mcpregistry.TransportStdio,
		title:     "Add MCP Server",
	}
}

// NewMcpAddHTTP creates a new blank HTTP/SSE MCP add form screen.
func NewMcpAddHTTP(theme *Theme) *McpAddModel {
	fields := []FormField{
		{Label: "Server name", Placeholder: "e.g. sentry, github (optional, derived from URL if blank)"},
		{Label: "URL", Placeholder: "e.g. https://mcp.sentry.dev/mcp", Required: true},
		{Label: "Scope", Choices: []string{"user", "local", "project"}},
	}

	return &McpAddModel{
		theme:     theme,
		form:      NewForm("Add MCP Server (http)", fields, theme),
		transport: mcpregistry.TransportHTTP,
		title:     "Add MCP Server",
	}
}

// NewMcpAddFromRecipe creates a pre-filled MCP add form from a recipe.
func NewMcpAddFromRecipe(recipe *mcpregistry.Recipe, theme *Theme) *McpAddModel {
	if recipe.Transport == mcpregistry.TransportHTTP {
		m := NewMcpAddHTTP(theme)
		m.title = "Add " + recipe.Key
		m.form.SetValue(0, recipe.Key)
		m.form.SetValue(1, recipe.URL)
		m.form.SetChoice(2, recipe.Scope)
		return m
	}

	m := NewMcpAdd(theme)
	m.title = "Add " + recipe.Key
	m.form.SetValue(0, recipe.Key)
	m.form.SetValue(1, recipe.FirstEnvVar())
	m.form.SetValue(2, recipe.CommandString())
	m.form.SetChoice(3, recipe.Scope)
	return m
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
	if m.transport == mcpregistry.TransportHTTP {
		return m.runHTTPAdd(values)
	}
	return m.runStdioAdd(values)
}

func (m *McpAddModel) runStdioAdd(values []string) tea.Cmd {
	name := strings.TrimSpace(values[0])
	apiKey := strings.TrimSpace(values[1])
	cmdStr := strings.TrimSpace(values[2])
	scope := values[3]

	args := []string{"mcp", "add", name, "--scope", scope}
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

func (m *McpAddModel) runHTTPAdd(values []string) tea.Cmd {
	name := strings.TrimSpace(values[0])
	url := strings.TrimSpace(values[1])
	scope := values[2]

	args := []string{"mcp", "remote", url, "--scope", scope}
	if name != "" {
		args = append(args, "--name", name)
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
		m.theme.SectionBanner(m.title),
		m.form.View(),
	))
}
