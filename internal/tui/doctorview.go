package tui

import (
	"bytes"

	tea "charm.land/bubbletea/v2"

	"github.com/lamchakchan/claude-workspace/internal/doctor"
)

// DoctorModel displays doctor health check results in a scrollable viewer.
type DoctorModel struct {
	viewer *ViewerModel
}

// NewDoctor creates a new doctor output viewer.
func NewDoctor(theme *Theme) *DoctorModel {
	return &DoctorModel{
		viewer: NewLoadingViewer("Doctor", loadDoctorOutput, theme),
	}
}

func (m *DoctorModel) Init() tea.Cmd  { return m.viewer.Init() }
func (m *DoctorModel) View() tea.View { return m.viewer.View() }
func (m *DoctorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := m.viewer.Update(msg)
	return m, cmd
}

func loadDoctorOutput() (string, error) {
	var buf bytes.Buffer
	if err := doctor.RunTo(&buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}
