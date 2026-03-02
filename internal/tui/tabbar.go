package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// TabItem represents a single tab in a tab bar.
type TabItem struct {
	Label string
}

// renderTabBar renders a horizontally-scrolling tab bar that keeps the active
// tab visible within the given width. When tabs overflow, < and > indicators
// appear at the edges to signal hidden tabs in that direction.
func renderTabBar(tabs []TabItem, activeIdx, width int, theme *Theme) string {
	separator := lipgloss.NewStyle().Foreground(theme.Muted)
	rule := separator.Render("  " + strings.Repeat("─", max(1, width-4)))

	if len(tabs) == 0 {
		return "  \n" + rule
	}

	// Clamp activeIdx to valid range.
	if activeIdx < 0 {
		activeIdx = 0
	}
	if activeIdx >= len(tabs) {
		activeIdx = len(tabs) - 1
	}

	selected := lipgloss.NewStyle().Bold(true).Foreground(theme.Primary)
	unselected := lipgloss.NewStyle().Foreground(theme.Muted)

	// Pre-render each tab and measure its visible width.
	rendered := make([]string, len(tabs))
	widths := make([]int, len(tabs))
	for i, tab := range tabs {
		key := fmt.Sprintf("[%d]", i+1)
		if i == activeIdx {
			rendered[i] = selected.Render(key+" "+tab.Label) + "  "
		} else {
			rendered[i] = unselected.Render(key+" "+tab.Label) + "  "
		}
		widths[i] = lipgloss.Width(rendered[i])
	}

	// If all tabs fit (with left padding), render them all — no indicators.
	const leftPad = 2
	totalWidth := 0
	for _, w := range widths {
		totalWidth += w
	}
	if totalWidth <= width-leftPad {
		line := "  " + strings.Join(rendered, "")
		return line + "\n" + rule
	}

	// Compute the visible window of tabs.
	lo, hi := tabWindow(widths, activeIdx, width-leftPad)

	// Build the output line with indicators where needed.
	indicatorStyle := lipgloss.NewStyle().Foreground(theme.Muted).Bold(true)
	var line strings.Builder
	line.WriteString("  ")

	if lo > 0 {
		line.WriteString(indicatorStyle.Render("< "))
	}
	for i := lo; i <= hi; i++ {
		line.WriteString(rendered[i])
	}
	if hi < len(tabs)-1 {
		line.WriteString(indicatorStyle.Render(" >"))
	}

	return line.String() + "\n" + rule
}

// tabWindow computes the range [lo, hi] of tab indices visible in the
// scrolling window, keeping activeIdx centered and reserving indicator space.
func tabWindow(widths []int, activeIdx, availableWidth int) (int, int) {
	const indicatorWidth = 2
	n := len(widths)
	lo, hi := activeIdx, activeIdx
	usedWidth := widths[activeIdx]

	for {
		expanded := false

		// Try expanding left.
		if lo > 0 {
			extra := indicatorReserve(lo-1, hi, n, indicatorWidth)
			if usedWidth+widths[lo-1]+extra <= availableWidth {
				lo--
				usedWidth += widths[lo]
				expanded = true
			}
		}

		// Try expanding right.
		if hi < n-1 {
			extra := indicatorReserve(lo, hi+1, n, indicatorWidth)
			if usedWidth+widths[hi+1]+extra <= availableWidth {
				hi++
				usedWidth += widths[hi]
				expanded = true
			}
		}

		if !expanded {
			break
		}
	}

	return lo, hi
}

// indicatorReserve returns the total space needed for scroll indicators
// given the proposed visible window [lo, hi] across n total tabs.
func indicatorReserve(lo, hi, n, indicatorWidth int) int {
	reserve := 0
	if lo > 0 {
		reserve += indicatorWidth
	}
	if hi < n-1 {
		reserve += indicatorWidth
	}
	return reserve
}
