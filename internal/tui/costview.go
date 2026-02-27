package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/lamchakchan/claude-workspace/internal/cost"
)

// CostModel displays usage and cost output in a scrollable viewer.
type CostModel struct {
	viewer *ViewerModel
}

// NewCost creates a new cost output viewer.
func NewCost(theme *Theme) *CostModel {
	return &CostModel{
		viewer: NewLoadingViewer("Usage & Costs", loadCostOutput, theme),
	}
}

func (m *CostModel) Init() tea.Cmd  { return m.viewer.Init() }
func (m *CostModel) View() tea.View { return m.viewer.View() }
func (m *CostModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := m.viewer.Update(msg)
	return m, cmd
}

func loadCostOutput() (string, error) {
	out, err := cost.RunCapture(nil)
	if err != nil {
		return "", err
	}
	return out, nil
}
