package tui

import (
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// SandboxListModel prompts for a project path, then displays sandboxes in a viewer.
type SandboxListModel struct {
	theme *Theme
	form  *FormModel
}

// NewSandboxList creates a new sandbox list form screen.
func NewSandboxList(theme *Theme) *SandboxListModel {
	fields := []FormField{
		{Label: "Project path", Placeholder: "e.g. ./my-project or /abs/path", Required: true, IsPath: true},
	}
	return &SandboxListModel{
		theme: theme,
		form:  NewForm("List Sandboxes", fields, theme),
	}
}

func (m *SandboxListModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m *SandboxListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.form, cmd = formViewUpdate(m.form, msg, m.runList)
	return m, cmd
}

func (m *SandboxListModel) runList(values []string) tea.Cmd {
	projectPath := strings.TrimSpace(values[0])

	exe, _ := os.Executable()
	cmd := exec.Command(exe, "sandbox", "list", projectPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return tea.ExecProcess(cmd, func(_ error) tea.Msg {
		return PopViewMsg{}
	})
}

func (m *SandboxListModel) View() tea.View {
	return tea.NewView(m.theme.SectionBanner("List Sandboxes") + "\n" + m.form.View())
}
