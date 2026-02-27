package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// StepStatus represents the state of a single step.
type StepStatus int

const (
	StepPending StepStatus = iota // not started yet
	StepRunning                   // currently in progress (spinner)
	StepDone                      // completed successfully
	StepFailed                    // completed with error
	StepSkipped                   // intentionally skipped
)

// Step is a single item in a multi-step progress view.
type Step struct {
	Label  string
	Status StepStatus
	Detail string // optional additional info (e.g., file count)
}

// StepperModel is a reusable animated multi-step progress component.
// Steps animate: pending → spinner → checkmark (or ✗ on failure).
type StepperModel struct {
	Steps   []Step
	spinner spinner.Model
	theme   *Theme
}

// NewStepper creates a new stepper with the given step labels.
func NewStepper(labels []string, theme *Theme) *StepperModel {
	steps := make([]Step, len(labels))
	for i, l := range labels {
		steps[i] = Step{Label: l, Status: StepPending}
	}

	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = lipgloss.NewStyle().Foreground(theme.Primary)

	return &StepperModel{
		Steps:   steps,
		spinner: sp,
		theme:   theme,
	}
}

// Init starts the spinner tick.
func (m *StepperModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles spinner tick messages.
func (m *StepperModel) Update(msg tea.Msg) (*StepperModel, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

// SetStatus updates the status of a step by index.
func (m *StepperModel) SetStatus(idx int, status StepStatus) {
	if idx >= 0 && idx < len(m.Steps) {
		m.Steps[idx].Status = status
	}
}

// SetDetail sets the detail text for a step by index.
func (m *StepperModel) SetDetail(idx int, detail string) {
	if idx >= 0 && idx < len(m.Steps) {
		m.Steps[idx].Detail = detail
	}
}

// View renders the stepper as a string (used inside a parent model's View).
func (m *StepperModel) View() string {
	var b strings.Builder

	for _, step := range m.Steps {
		var icon, label string

		switch step.Status {
		case StepDone:
			icon = lipgloss.NewStyle().Foreground(m.theme.Success).Bold(true).Render("✓")
			label = step.Label
		case StepFailed:
			icon = lipgloss.NewStyle().Foreground(m.theme.Error).Bold(true).Render("✗")
			label = lipgloss.NewStyle().Foreground(m.theme.Error).Render(step.Label)
		case StepRunning:
			icon = m.spinner.View()
			label = lipgloss.NewStyle().Bold(true).Render(step.Label)
		case StepSkipped:
			icon = lipgloss.NewStyle().Foreground(m.theme.Muted).Render("–")
			label = lipgloss.NewStyle().Foreground(m.theme.Muted).Render(step.Label)
		default: // StepPending
			icon = lipgloss.NewStyle().Foreground(m.theme.Muted).Render("○")
			label = lipgloss.NewStyle().Foreground(m.theme.Muted).Render(step.Label)
		}

		line := fmt.Sprintf("  %s %s", icon, label)
		if step.Detail != "" {
			detail := lipgloss.NewStyle().Foreground(m.theme.Muted).Render(step.Detail)
			line += "  " + detail
		}
		b.WriteString(line + "\n")
	}

	return b.String()
}
