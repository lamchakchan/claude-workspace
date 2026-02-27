package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// SectionBanner renders a bold section header with a horizontal rule.
//
//	──────────────────────────────
//	▶ Title
func (t *Theme) SectionBanner(title string) string {
	rule := lipgloss.NewStyle().Foreground(t.Secondary).Render(strings.Repeat("─", 40))
	heading := lipgloss.NewStyle().Bold(true).Foreground(t.Secondary).Render("▶ " + title)
	return fmt.Sprintf("\n%s\n  %s\n", rule, heading)
}

// SubBanner renders a smaller subsection label: "  --- title ---"
func (t *Theme) SubBanner(title string) string {
	return lipgloss.NewStyle().Foreground(t.Muted).Render("  --- "+title+" ---") + "\n"
}

// SuccessBox renders a success message inside a subtle green-bordered box.
func (t *Theme) SuccessBox(msg string) string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Success).
		Padding(0, 2).
		Render(lipgloss.NewStyle().Foreground(t.Success).Bold(true).Render("✓ " + msg))
}

// ErrorBox renders an error message inside a red-bordered box.
func (t *Theme) ErrorBox(msg string) string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Error).
		Padding(0, 2).
		Render(lipgloss.NewStyle().Foreground(t.Error).Bold(true).Render("✗ " + msg))
}
