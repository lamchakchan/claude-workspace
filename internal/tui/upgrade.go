package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lamchakchan/claude-workspace/internal/upgrade"
)

// releaseInfo holds upgrade release data fetched in the background.
type releaseInfo struct {
	release *upgrade.Release
	err     error
}

// UpgradeModel is the interactive upgrade confirmation screen.
type UpgradeModel struct {
	theme   Theme
	version string
	release *upgrade.Release
	loading bool
	err     string
	confirm *ConfirmModel
}

// NewUpgrade creates a new upgrade screen. It starts loading release info on Init.
func NewUpgrade(version string, theme Theme) UpgradeModel {
	return UpgradeModel{
		theme:   theme,
		version: version,
		loading: true,
	}
}

func (m UpgradeModel) Init() tea.Cmd {
	return fetchRelease
}

// fetchRelease is a Cmd that fetches the latest release in the background.
func fetchRelease() tea.Msg {
	r, err := upgrade.FetchLatest()
	return releaseInfo{release: r, err: err}
}

func (m UpgradeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case releaseInfo:
		m.loading = false
		if msg.err != nil {
			m.err = fmt.Sprintf("Could not fetch release info: %v", msg.err)
			return m, nil
		}
		m.release = msg.release
		body := m.buildConfirmBody()
		confirm := NewConfirm("Upgrade claude-workspace?", body, true, m.theme)
		m.confirm = &confirm
		return m, nil

	case ConfirmResult:
		if msg.Confirmed {
			return m, m.runUpgrade()
		}
		return m, func() tea.Msg { return PopViewMsg{} }

	case tea.KeyPressMsg:
		if IsQuit(msg) {
			return m, func() tea.Msg { return PopViewMsg{} }
		}
	}

	if m.confirm != nil {
		updated, cmd := m.confirm.Update(msg)
		m.confirm = &updated
		return m, cmd
	}

	return m, nil
}

func (m UpgradeModel) buildConfirmBody() string {
	var b strings.Builder

	current := m.version
	latest := m.release.TagName

	b.WriteString(lipgloss.NewStyle().Foreground(m.theme.Muted).Render("Current: ") + current + "\n")
	b.WriteString(lipgloss.NewStyle().Foreground(m.theme.Success).Render("Latest:  ") + latest)

	if m.release.PublishedAt != "" {
		date := m.release.PublishedAt
		if idx := strings.Index(date, "T"); idx > 0 {
			date = date[:idx]
		}
		b.WriteString(lipgloss.NewStyle().Foreground(m.theme.Muted).Render("  (" + date + ")"))
	}
	b.WriteString("\n")

	if m.release.Body != "" {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(m.theme.Secondary).Render("Changelog:"))
		b.WriteString("\n")
		for _, line := range strings.Split(m.release.Body, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				b.WriteString("  " + lipgloss.NewStyle().Foreground(m.theme.Muted).Render(line) + "\n")
			}
		}
	}

	return b.String()
}

func (m UpgradeModel) runUpgrade() tea.Cmd {
	exe, _ := os.Executable()
	cmd := exec.Command(exe, "upgrade", "--yes")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return PopViewMsg{}
	})
}

func (m UpgradeModel) View() tea.View {
	var b strings.Builder

	b.WriteString(m.theme.SectionBanner("Upgrade"))

	if m.loading {
		b.WriteString("\n  Checking for updates...")
		return tea.NewView(b.String())
	}

	if m.err != "" {
		b.WriteString("\n  ")
		b.WriteString(lipgloss.NewStyle().Foreground(m.theme.Error).Render(m.err))
		b.WriteString("\n\n  Press q to go back.\n")
		return tea.NewView(b.String())
	}

	if m.confirm != nil {
		b.WriteString("\n")
		b.WriteString(m.confirm.View())
	}

	return tea.NewView(b.String())
}
