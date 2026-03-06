package tui //nolint:dupl // viewer wrappers share identical structure by design

import (
	"bytes"

	tea "charm.land/bubbletea/v2"

	"github.com/lamchakchan/claude-workspace/internal/plugins"
)

// PluginsListModel displays installed plugins in a scrollable viewer.
type PluginsListModel struct {
	viewer *ViewerModel
}

// NewPluginsList creates a new installed plugins viewer.
func NewPluginsList(theme *Theme) *PluginsListModel {
	return &PluginsListModel{
		viewer: NewLoadingViewer("Installed Plugins", loadPluginsList, theme),
	}
}

func (m *PluginsListModel) Init() tea.Cmd  { return m.viewer.Init() }
func (m *PluginsListModel) View() tea.View { return m.viewer.View() }
func (m *PluginsListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := m.viewer.Update(msg)
	return m, cmd
}

func loadPluginsList() (string, error) {
	var buf bytes.Buffer
	if err := plugins.ListTo(&buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}
