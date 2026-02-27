package tui

import (
	"bytes"

	tea "charm.land/bubbletea/v2"

	"github.com/lamchakchan/claude-workspace/internal/mcp"
)

// McpListModel displays MCP server list in a scrollable viewer.
type McpListModel struct {
	viewer *ViewerModel
}

// NewMcpList creates a new MCP list viewer.
func NewMcpList(theme *Theme) *McpListModel {
	return &McpListModel{
		viewer: NewLoadingViewer("MCP Servers", loadMcpList, theme),
	}
}

func (m *McpListModel) Init() tea.Cmd  { return m.viewer.Init() }
func (m *McpListModel) View() tea.View { return m.viewer.View() }
func (m *McpListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := m.viewer.Update(msg)
	return m, cmd
}

func loadMcpList() (string, error) {
	var buf bytes.Buffer
	if err := mcp.ListTo(&buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}
