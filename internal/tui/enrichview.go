package tui

import (
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// EnrichModel is the interactive enrich project form screen.
type EnrichModel struct {
	theme Theme
	form  FormModel
}

// NewEnrich creates a new enrich form screen.
func NewEnrich(theme Theme) EnrichModel {
	fields := []FormField{
		{Label: "Project path", Placeholder: "leave blank for current directory"},
	}

	return EnrichModel{
		theme: theme,
		form:  NewForm("Enrich CLAUDE.md with AI", fields, theme),
	}
}

func (m EnrichModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m EnrichModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case FormResult:
		if msg.Cancelled {
			return m, func() tea.Msg { return PopViewMsg{} }
		}
		return m, m.runEnrich(msg.Values)

	case tea.KeyPressMsg:
		if IsQuit(msg) || IsBack(msg) {
			return m, func() tea.Msg { return PopViewMsg{} }
		}
	}

	updated, cmd := m.form.Update(msg)
	m.form = updated
	return m, cmd
}

func (m EnrichModel) runEnrich(values []string) tea.Cmd {
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
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return PopViewMsg{}
	})
}

func (m EnrichModel) View() tea.View {
	return tea.NewView(m.theme.SectionBanner("Enrich CLAUDE.md with AI") + "\n" + m.form.View())
}
