package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// MarketplaceAddModel is a simple text input for adding a marketplace by owner/repo.
type MarketplaceAddModel struct {
	theme    *Theme
	input    string
	cursor   int
	err      string
	quitting bool
}

// NewMarketplaceAdd creates a new marketplace add form.
func NewMarketplaceAdd(theme *Theme) *MarketplaceAddModel {
	return &MarketplaceAddModel{theme: theme}
}

func (m *MarketplaceAddModel) Init() tea.Cmd { return nil }

func (m *MarketplaceAddModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, nil

	case tea.KeyPressMsg:
		if IsQuit(msg) || IsBack(msg) {
			return m, func() tea.Msg { return PopViewMsg{} }
		}

		switch msg.String() {
		case keyEnter:
			return m, m.submit()
		case keyBackspace:
			if m.cursor > 0 {
				m.input = m.input[:m.cursor-1] + m.input[m.cursor:]
				m.cursor--
				m.err = ""
			}
		case keyLeft:
			if m.cursor > 0 {
				m.cursor--
			}
		case keyRight:
			if m.cursor < len(m.input) {
				m.cursor++
			}
		default:
			// Insert printable characters
			r := msg.String()
			if len(r) == 1 && r[0] >= 32 && r[0] < 127 {
				m.input = m.input[:m.cursor] + r + m.input[m.cursor:]
				m.cursor++
				m.err = ""
			}
		}
	}
	return m, nil
}

func (m *MarketplaceAddModel) submit() tea.Cmd {
	repo := strings.TrimSpace(m.input)
	if repo == "" {
		m.err = "Repository cannot be empty"
		return nil
	}
	if !strings.Contains(repo, "/") || strings.Count(repo, "/") != 1 {
		m.err = "Invalid format: expected owner/repo"
		return nil
	}

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
	var b strings.Builder
	b.WriteString(m.theme.SectionBanner("Add Marketplace"))

	b.WriteString("\n  Enter the marketplace repository (owner/repo):\n\n")

	// Render input field with cursor
	prompt := "  > "
	b.WriteString(prompt)

	inputStyle := lipgloss.NewStyle().Foreground(m.theme.Primary)
	if m.cursor < len(m.input) {
		b.WriteString(inputStyle.Render(m.input[:m.cursor]))
		b.WriteString(lipgloss.NewStyle().Background(m.theme.Primary).Foreground(lipgloss.Color("#000000")).Render(string(m.input[m.cursor])))
		b.WriteString(inputStyle.Render(m.input[m.cursor+1:]))
	} else {
		b.WriteString(inputStyle.Render(m.input))
		b.WriteString(lipgloss.NewStyle().Background(m.theme.Primary).Render(" "))
	}
	b.WriteString("\n")

	if m.err != "" {
		b.WriteString("\n  ")
		b.WriteString(lipgloss.NewStyle().Foreground(m.theme.Error).Render(m.err))
		b.WriteString("\n")
	}

	b.WriteString("\n  ")
	b.WriteString(lipgloss.NewStyle().Foreground(m.theme.Muted).Render("Example: anthropics/claude-plugins-official"))
	b.WriteString("\n")

	// Footer
	b.WriteString("\n")
	help := fmt.Sprintf(
		"%s submit  %s back",
		m.theme.HelpKey.Render(keyEnter),
		m.theme.HelpKey.Render("esc"),
	)
	b.WriteString(help)

	return tea.NewView(b.String())
}
