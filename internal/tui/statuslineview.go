package tui //nolint:dupl // viewer wrappers share identical structure by design

import (
	"bytes"

	tea "charm.land/bubbletea/v2"

	"github.com/lamchakchan/claude-workspace/internal/statusline"
)

// StatuslineModel displays statusline configuration output in a scrollable viewer.
type StatuslineModel struct {
	viewer *ViewerModel
}

// NewStatusline creates a new statusline output viewer.
func NewStatusline(theme *Theme) *StatuslineModel {
	return &StatuslineModel{
		viewer: NewLoadingViewer("Statusline", loadStatuslineOutput, theme),
	}
}

func (m *StatuslineModel) Init() tea.Cmd  { return m.viewer.Init() }
func (m *StatuslineModel) View() tea.View { return m.viewer.View() }
func (m *StatuslineModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := m.viewer.Update(msg)
	return m, cmd
}

func loadStatuslineOutput() (string, error) {
	var buf bytes.Buffer
	if err := statusline.RunTo(&buf, nil); err != nil {
		return "", err
	}
	return buf.String(), nil
}
