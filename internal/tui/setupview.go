package tui

import (
	"bytes"

	tea "charm.land/bubbletea/v2"

	"github.com/lamchakchan/claude-workspace/internal/setup"
)

// SetupModel displays setup output in a scrollable viewer.
type SetupModel struct {
	viewer *ViewerModel
}

// NewSetup creates a new setup output viewer.
func NewSetup(theme *Theme) *SetupModel {
	return &SetupModel{
		viewer: NewLoadingViewer("Setup", loadSetupOutput, theme),
	}
}

func (m *SetupModel) Init() tea.Cmd  { return m.viewer.Init() }
func (m *SetupModel) View() tea.View { return m.viewer.View() }
func (m *SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := m.viewer.Update(msg)
	return m, cmd
}

func loadSetupOutput() (string, error) {
	var buf bytes.Buffer
	if err := setup.RunTo(&buf, nil); err != nil {
		return "", err
	}
	return buf.String(), nil
}
