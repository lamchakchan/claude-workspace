package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ConfirmResult is sent when the user responds to a confirmation dialog.
type ConfirmResult struct {
	Confirmed bool
}

// ConfirmModel is a simple yes/no confirmation dialog component.
type ConfirmModel struct {
	Title  string
	Body   string
	Cursor bool // true = yes, false = no
	done   bool
	theme  *Theme
}

// NewConfirm creates a new confirmation dialog.
func NewConfirm(title, body string, defaultYes bool, theme *Theme) *ConfirmModel {
	return &ConfirmModel{
		Title:  title,
		Body:   body,
		Cursor: defaultYes,
		theme:  theme,
	}
}

func (m *ConfirmModel) Update(msg tea.Msg) (*ConfirmModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch msg.String() {
		case "y", "Y":
			m.Cursor = true
			m.done = true
			return m, func() tea.Msg { return ConfirmResult{Confirmed: true} }
		case "n", "N", "q", keyCtrlC:
			m.Cursor = false
			m.done = true
			return m, func() tea.Msg { return ConfirmResult{Confirmed: false} }
		case keyEnter:
			m.done = true
			confirmed := m.Cursor
			return m, func() tea.Msg { return ConfirmResult{Confirmed: confirmed} }
		case "left", "h", "tab":
			m.Cursor = !m.Cursor
		case "right", "l":
			m.Cursor = !m.Cursor
		}
	}
	return m, nil
}

func (m *ConfirmModel) View() string {
	var b strings.Builder

	b.WriteString(m.theme.Title.Render(m.Title))
	b.WriteString("\n\n")

	if m.Body != "" {
		b.WriteString(m.Body)
		b.WriteString("\n\n")
	}

	selectedStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.Primary).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Primary).
		Padding(0, 1)

	unselectedStyle := lipgloss.NewStyle().
		Foreground(m.theme.Muted).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Muted).
		Padding(0, 1)

	var yes, no string
	if m.Cursor {
		yes = selectedStyle.Render("Yes")
		no = unselectedStyle.Render("No")
	} else {
		yes = unselectedStyle.Render("Yes")
		no = selectedStyle.Render("No")
	}

	b.WriteString(yes + "  " + no)
	b.WriteString("\n\n")

	help := m.theme.HelpKey.Render("←/→") + " " + m.theme.HelpDesc.Render("switch") + "  " +
		m.theme.HelpKey.Render(keyEnter) + " " + m.theme.HelpDesc.Render("confirm") + "  " +
		m.theme.HelpKey.Render("q") + " " + m.theme.HelpDesc.Render("cancel")
	b.WriteString(help)

	return b.String()
}
