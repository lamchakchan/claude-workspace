package tui

import (
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// SandboxModel is a sandbox form screen for create and remove actions.
type SandboxModel struct {
	theme  *Theme
	form   *FormModel
	title  string
	subcmd string
}

func newSandboxForm(title, subcmd string, theme *Theme) *SandboxModel {
	fields := []FormField{
		{Label: "Project path", Placeholder: "e.g. ./my-project or /abs/path", Required: true, IsPath: true},
		{Label: "Branch name", Placeholder: "e.g. feature-auth, bugfix-login", Required: true},
	}
	return &SandboxModel{
		theme:  theme,
		form:   NewForm(title, fields, theme),
		title:  title,
		subcmd: subcmd,
	}
}

// NewSandbox creates a new sandbox creation form screen.
func NewSandbox(theme *Theme) *SandboxModel {
	return newSandboxForm("Create Sandbox Worktree", "create", theme)
}

// NewSandboxRemove creates a new sandbox removal form screen.
func NewSandboxRemove(theme *Theme) *SandboxModel {
	return newSandboxForm("Remove Sandbox Worktree", "remove", theme)
}

func (m *SandboxModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m *SandboxModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.form, cmd = formViewUpdate(m.form, msg, m.run)
	return m, cmd
}

func (m *SandboxModel) run(values []string) tea.Cmd {
	projectPath := strings.TrimSpace(values[0])
	branchName := strings.TrimSpace(values[1])

	exe, _ := os.Executable()
	cmd := exec.Command(exe, "sandbox", m.subcmd, projectPath, branchName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return tea.ExecProcess(cmd, func(_ error) tea.Msg {
		return PopViewMsg{}
	})
}

func (m *SandboxModel) View() tea.View {
	return tea.NewView(m.theme.SectionBanner(m.title) + "\n" + m.form.View())
}
