package tui

import "charm.land/lipgloss/v2"

// BadgeOK renders a green [OK] badge.
func (t Theme) BadgeOKStr() string {
	return t.BadgeOK.Render("[OK]")
}

// BadgeWarnStr renders a yellow [WARN] badge.
func (t Theme) BadgeWarnStr() string {
	return t.BadgeWarn.Render("[WARN]")
}

// BadgeFailStr renders a red [FAIL] badge.
func (t Theme) BadgeFailStr() string {
	return t.BadgeFail.Render("[FAIL]")
}

// StatusIcon renders a colored Unicode check or cross.
func (t Theme) StatusIcon(ok bool) string {
	if ok {
		return lipgloss.NewStyle().Foreground(t.Success).Bold(true).Render("✓")
	}
	return lipgloss.NewStyle().Foreground(t.Error).Bold(true).Render("✗")
}
