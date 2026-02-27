package tui

import (
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// AttachModel is the interactive attach project form screen.
type AttachModel struct {
	theme *Theme
	form  *FormModel
}

// NewAttach creates a new attach form screen.
func NewAttach(theme *Theme) *AttachModel {
	fields := []FormField{
		{Label: "Project path", Placeholder: "e.g. ./my-project or /abs/path", Required: true},
	}

	return &AttachModel{
		theme: theme,
		form:  NewForm("Attach Platform Config", fields, theme),
	}
}

func (m *AttachModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m *AttachModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.form, cmd = formViewUpdate(m.form, msg, m.runAttach)
	return m, cmd
}

func (m *AttachModel) runAttach(values []string) tea.Cmd {
	projectPath := strings.TrimSpace(values[0])

	exe, _ := os.Executable()
	cmd := exec.Command(exe, "attach", projectPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return tea.ExecProcess(cmd, func(_ error) tea.Msg {
		return PopViewMsg{}
	})
}

func (m *AttachModel) View() tea.View {
	return tea.NewView(m.theme.SectionBanner("Attach Platform Config") + "\n" + m.form.View())
}
