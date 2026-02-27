// Package tui provides the Bubble Tea terminal UI for claude-workspace.
// It implements an interactive launcher and animated views for all commands.
package tui

import (
	"image/color"
	"os"

	"charm.land/lipgloss/v2"
)

// IsAccessible returns true when the environment requests accessible (no-color) output.
// Respects the NO_COLOR standard (https://no-color.org) and ACCESSIBLE=1.
func IsAccessible() bool {
	return os.Getenv("NO_COLOR") != "" || os.Getenv("ACCESSIBLE") == "1"
}

// Theme holds the lipgloss styles used throughout the TUI.
type Theme struct {
	// Brand colors
	Primary   color.Color
	Secondary color.Color
	Accent    color.Color

	// Status colors
	Success color.Color
	Warning color.Color
	Error   color.Color
	Muted   color.Color

	// Component styles
	Title       lipgloss.Style
	Subtitle    lipgloss.Style
	SectionHead lipgloss.Style
	HelpKey     lipgloss.Style
	HelpDesc    lipgloss.Style

	// Status badges
	BadgeOK   lipgloss.Style
	BadgeWarn lipgloss.Style
	BadgeFail lipgloss.Style

	// Box styles
	Banner lipgloss.Style
}

// DefaultTheme returns the standard claude-workspace visual theme.
func DefaultTheme() Theme {
	primary := lipgloss.Color("#7C3AED")   // violet
	secondary := lipgloss.Color("#06B6D4") // cyan
	accent := lipgloss.Color("#F59E0B")    // amber

	success := lipgloss.Color("#10B981")  // emerald
	warning := lipgloss.Color("#F59E0B")  // amber
	errColor := lipgloss.Color("#EF4444") // red
	muted := lipgloss.Color("#6B7280")    // gray

	return Theme{
		Primary:   primary,
		Secondary: secondary,
		Accent:    accent,
		Success:   success,
		Warning:   warning,
		Error:     errColor,
		Muted:     muted,

		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(primary),

		Subtitle: lipgloss.NewStyle().
			Foreground(muted),

		SectionHead: lipgloss.NewStyle().
			Bold(true).
			Foreground(secondary),

		HelpKey: lipgloss.NewStyle().
			Bold(true).
			Foreground(muted),

		HelpDesc: lipgloss.NewStyle().
			Foreground(muted),

		BadgeOK: lipgloss.NewStyle().
			Bold(true).
			Foreground(success),

		BadgeWarn: lipgloss.NewStyle().
			Bold(true).
			Foreground(warning),

		BadgeFail: lipgloss.NewStyle().
			Bold(true).
			Foreground(errColor),

		Banner: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primary).
			Padding(0, 2),
	}
}
