package tui //nolint:dupl // viewer wrappers share identical structure by design

import (
	"bytes"

	tea "charm.land/bubbletea/v2"

	"github.com/lamchakchan/claude-workspace/internal/plugins"
)

// MarketplaceListModel displays configured marketplaces in a scrollable viewer.
type MarketplaceListModel struct {
	viewer *ViewerModel
}

// NewMarketplaceList creates a new marketplace list viewer.
func NewMarketplaceList(theme *Theme) *MarketplaceListModel {
	return &MarketplaceListModel{
		viewer: NewLoadingViewer("Plugin Marketplaces", loadMarketplaceList, theme),
	}
}

func (m *MarketplaceListModel) Init() tea.Cmd  { return m.viewer.Init() }
func (m *MarketplaceListModel) View() tea.View { return m.viewer.View() }
func (m *MarketplaceListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := m.viewer.Update(msg)
	return m, cmd
}

func loadMarketplaceList() (string, error) {
	var buf bytes.Buffer
	if err := plugins.MarketplaceListTo(&buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}
