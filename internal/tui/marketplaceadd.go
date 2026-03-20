package tui

import (
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// MarketplaceAddModel is the interactive form for adding a marketplace by path or owner/repo.
type MarketplaceAddModel struct {
	theme *Theme
	form  *FormModel
}

// NewMarketplaceAdd creates a new marketplace add form with path autocomplete.
func NewMarketplaceAdd(theme *Theme) *MarketplaceAddModel {
	fields := []FormField{
		{Label: "Marketplace (owner/repo or local path)", Placeholder: "e.g. anthropics/claude-plugins-official or ~/git/org/repo", Required: true, PathType: PathDir},
	}

	return &MarketplaceAddModel{
		theme: theme,
		form:  NewForm("Add Marketplace", fields, theme),
	}
}

func (m *MarketplaceAddModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m *MarketplaceAddModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.form, cmd = formViewUpdate(m.form, msg, m.runAdd)
	return m, cmd
}

func (m *MarketplaceAddModel) runAdd(values []string) tea.Cmd {
	repo := strings.TrimSpace(values[0])

	exe, _ := os.Executable()
	cmd := exec.Command(exe, "plugins", "marketplace", "add", repo)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return tea.ExecProcess(cmd, func(_ error) tea.Msg {
		return PopViewMsg{}
	})
}

func (m *MarketplaceAddModel) View() tea.View {
	return tea.NewView(m.theme.SectionBanner("Add Marketplace") + "\n" + m.form.View())
}
