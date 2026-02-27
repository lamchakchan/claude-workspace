package tui

import (
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// SandboxModel is the interactive sandbox creation form screen.
type SandboxModel struct {
	theme Theme
	form  FormModel
}

// NewSandbox creates a new sandbox form screen.
func NewSandbox(theme Theme) SandboxModel {
	fields := []FormField{
		{Label: "Project path", Placeholder: "e.g. ./my-project or /abs/path", Required: true},
		{Label: "Branch name", Placeholder: "e.g. feature-auth, bugfix-login", Required: true},
	}

	return SandboxModel{
		theme: theme,
		form:  NewForm("Create Sandbox Worktree", fields, theme),
	}
}

func (m SandboxModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m SandboxModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case FormResult:
		if msg.Cancelled {
			return m, func() tea.Msg { return PopViewMsg{} }
		}
		return m, m.runSandbox(msg.Values)

	case tea.KeyPressMsg:
		if IsQuit(msg) || IsBack(msg) {
			return m, func() tea.Msg { return PopViewMsg{} }
		}
	}

	updated, cmd := m.form.Update(msg)
	m.form = updated
	return m, cmd
}

func (m SandboxModel) runSandbox(values []string) tea.Cmd {
	projectPath := strings.TrimSpace(values[0])
	branchName := strings.TrimSpace(values[1])

	exe, _ := os.Executable()
	cmd := exec.Command(exe, "sandbox", projectPath, branchName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return PopViewMsg{}
	})
}

func (m SandboxModel) View() tea.View {
	return tea.NewView(m.theme.SectionBanner("Create Sandbox Worktree") + "\n" + m.form.View())
}
