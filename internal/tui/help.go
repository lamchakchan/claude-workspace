package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// HelpModel shows a keybinding reference overlay.
type HelpModel struct {
	theme *Theme
}

// NewHelp creates a new help overlay.
func NewHelp(theme *Theme) *HelpModel {
	return &HelpModel{theme: theme}
}

func (m *HelpModel) Init() tea.Cmd { return nil }

func (m *HelpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		if IsQuit(msg) || IsBack(msg) || msg.String() == "?" {
			return m, func() tea.Msg { return PopViewMsg{} }
		}
	}
	return m, nil
}

func (m *HelpModel) View() tea.View {
	var b strings.Builder

	b.WriteString(m.theme.SectionBanner("Keyboard Shortcuts"))
	b.WriteString("\n\n")

	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Primary).Width(18)
	descStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)

	sections := []struct {
		title string
		binds [][2]string
	}{
		{
			title: "Navigation",
			binds: [][2]string{
				{"↑ / k", "Move up"},
				{"↓ / j", "Move down"},
				{keyEnter, "Select / confirm"},
				{"esc", "Go back"},
				{"q / ctrl+c", "Quit"},
			},
		},
		{
			title: "Forms",
			binds: [][2]string{
				{"tab / ↓", "Next field"},
				{"shift+tab / ↑", "Previous field"},
				{keyEnter, "Next field / submit"},
				{"esc", "Cancel"},
			},
		},
		{
			title: "Path autocomplete",
			binds: [][2]string{
				{"↑ / ↓", "Cycle suggestions"},
				{"tab", "Accept suggestion"},
			},
		},
		{
			title: "Lists",
			binds: [][2]string{
				{"j / k", "Move up / down"},
				{"pgup / pgdn", "Page up / down"},
				{"g / G", "Go to top / bottom"},
				{keyEnter, "Select item"},
				{"esc / q", "Go back"},
			},
		},
		{
			title: "Viewers",
			binds: [][2]string{
				{"j / k", "Scroll up / down"},
				{"pgup / pgdn", "Page up / down"},
				{"g / G", "Go to top / bottom"},
				{"y", "Copy to clipboard"},
				{"esc / q", "Close viewer"},
			},
		},
		{
			title: "Confirmation dialogs",
			binds: [][2]string{
				{"y / Y", "Confirm yes"},
				{"n / N", "Confirm no"},
				{"← / → / tab", "Switch selection"},
				{keyEnter, "Confirm selection"},
			},
		},
		{
			title: "Help",
			binds: [][2]string{
				{"?", "Show / hide this screen"},
			},
		},
	}

	headStyle := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Secondary)

	for _, section := range sections {
		b.WriteString(headStyle.Render(section.title))
		b.WriteString("\n")
		for _, bind := range section.binds {
			b.WriteString("  " + keyStyle.Render(bind[0]) + descStyle.Render(bind[1]) + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(m.theme.HelpKey.Render("esc / q / ?") + " " + m.theme.HelpDesc.Render("close"))

	return tea.NewView(b.String())
}
