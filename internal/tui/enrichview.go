package tui

import (
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// EnrichModel is the interactive enrich project form screen.
type EnrichModel struct {
	theme *Theme
	form  *FormModel
}

// NewEnrich creates a new enrich form screen.
func NewEnrich(theme *Theme) *EnrichModel {
	fields := []FormField{
		{Label: "Project path", Placeholder: "leave blank for current directory", IsPath: true},
	}

	return &EnrichModel{
		theme: theme,
		form:  NewForm("Enrich CLAUDE.md with AI", fields, theme),
	}
}

func (m *EnrichModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m *EnrichModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.form, cmd = formViewUpdate(m.form, msg, m.runEnrich)
	return m, cmd
}

func (m *EnrichModel) runEnrich(values []string) tea.Cmd {
	projectPath := strings.TrimSpace(values[0])

	args := []string{"enrich"}
	if projectPath != "" {
		args = append(args, projectPath)
	}

	exe, _ := os.Executable()
	cmd := exec.Command(exe, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return tea.ExecProcess(cmd, func(_ error) tea.Msg {
		return PopViewMsg{}
	})
}

func (m *EnrichModel) View() tea.View {
	return tea.NewView(m.theme.SectionBanner("Enrich CLAUDE.md with AI") + "\n" + m.form.View())
}
