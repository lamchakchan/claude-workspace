package tui

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/lamchakchan/claude-workspace/internal/cost"
)

// barChars are Unicode block elements ordered by height (1/8 to 8/8).
var barChars = []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

// chartRows is the number of text rows for the bar area.
const chartRows = 8

// renderBarChart renders an ASCII bar chart from cost entries.
// maxWidth is the available terminal width. Returns empty string if entries
// is empty or maxWidth is too narrow.
func renderBarChart(entries []cost.ChartEntry, maxWidth int, theme *Theme) string {
	if len(entries) == 0 {
		return ""
	}

	// Y-axis label width: "$" + up to 4 digits + " ┤" = ~8 chars
	const yAxisWidth = 8
	// Each bar needs barWidth chars + 1 gap
	const barWidth = 2
	const barGap = 1
	const barSlot = barWidth + barGap

	available := maxWidth - yAxisWidth - 2 // padding
	if available < barSlot {
		return ""
	}

	maxBars := available / barSlot
	// Show only the most recent N entries if there are too many
	visible := entries
	if len(visible) > maxBars {
		visible = visible[len(visible)-maxBars:]
	}

	// Find max cost for scaling
	maxCost := 0.0
	for _, e := range visible {
		if e.Value > maxCost {
			maxCost = e.Value
		}
	}
	if maxCost == 0 {
		maxCost = 1 // avoid division by zero
	}

	// Round max up to a nice number for Y-axis labels
	maxLabel := niceMax(maxCost)

	// Build the chart rows top-down
	barStyle := lipgloss.NewStyle().Foreground(theme.Primary)
	axisStyle := lipgloss.NewStyle().Foreground(theme.Muted)

	var b strings.Builder
	b.WriteString("\n")

	for row := chartRows; row >= 1; row-- {
		renderYAxisLabel(&b, row, maxLabel, yAxisWidth, &axisStyle)
		renderBarRow(&b, row, visible, maxLabel, &barStyle, &axisStyle)
		b.WriteString("\n")
	}

	renderXAxis(&b, visible, yAxisWidth, barSlot, &axisStyle)
	return b.String()
}

// renderYAxisLabel writes the Y-axis label and separator for a given row.
func renderYAxisLabel(b *strings.Builder, row int, maxLabel float64, yAxisWidth int, axisStyle *lipgloss.Style) {
	var label string
	switch row {
	case chartRows:
		label = formatDollar(maxLabel)
	case chartRows / 2:
		label = formatDollar(maxLabel / 2)
	case 1:
		label = formatDollar(0)
	default:
		label = ""
	}
	fmt.Fprintf(b, "%*s", yAxisWidth-2, label)
	b.WriteString(axisStyle.Render(" ┤"))
}

// renderBarRow writes the bar characters for a single chart row.
func renderBarRow(b *strings.Builder, row int, visible []cost.ChartEntry, maxLabel float64, barStyle, axisStyle *lipgloss.Style) {
	for _, e := range visible {
		height := e.Value / maxLabel * float64(chartRows)
		b.WriteString(" ")
		switch {
		case height >= float64(row):
			b.WriteString(barStyle.Render("██"))
		case height > float64(row-1):
			frac := height - float64(row-1)
			idx := int(frac * float64(len(barChars)))
			if idx >= len(barChars) {
				idx = len(barChars) - 1
			}
			ch := barChars[idx]
			b.WriteString(barStyle.Render(ch + ch))
		case row == 1 && e.Value > 0:
			b.WriteString(barStyle.Render(barChars[0] + barChars[0]))
		case row == 1:
			b.WriteString(axisStyle.Render("──"))
		default:
			b.WriteString("  ")
		}
	}
}

// renderXAxis writes the X-axis line and date labels.
func renderXAxis(b *strings.Builder, visible []cost.ChartEntry, yAxisWidth, barSlot int, axisStyle *lipgloss.Style) {
	// X-axis line
	fmt.Fprintf(b, "%*s", yAxisWidth, "")
	for range visible {
		b.WriteString(axisStyle.Render("───"))
	}
	b.WriteString("\n")

	// X-axis date labels — show a label every few bars to avoid overlap
	labelEvery := 1
	if len(visible) > 15 {
		labelEvery = 3
	} else if len(visible) > 8 {
		labelEvery = 2
	}

	fmt.Fprintf(b, "%*s", yAxisWidth, "")
	col := 0
	for i, e := range visible {
		if i%labelEvery == 0 {
			label := shortLabel(e.Label)
			b.WriteString(label)
			col += len(label)
		}
		// Pad to the next bar slot boundary
		nextCol := (i + 1) * barSlot
		if col < nextCol {
			b.WriteString(strings.Repeat(" ", nextCol-col))
			col = nextCol
		}
	}
	b.WriteString("\n")
}

// niceMax rounds up to a "nice" number for Y-axis scaling.
func niceMax(v float64) float64 {
	if v <= 0 {
		return 1
	}
	magnitude := math.Pow(10, math.Floor(math.Log10(v)))
	normalized := v / magnitude
	switch {
	case normalized <= 1:
		return magnitude
	case normalized <= 2:
		return 2 * magnitude
	case normalized <= 5:
		return 5 * magnitude
	default:
		return 10 * magnitude
	}
}

// formatDollar formats a dollar amount for the Y-axis.
func formatDollar(v float64) string {
	if v == 0 {
		return "$0"
	}
	if v >= 10 {
		return fmt.Sprintf("$%.0f", v)
	}
	return fmt.Sprintf("$%.1f", v)
}

// shortLabel extracts a short display label from various ccusage label formats.
// Handles: dates (2026-02-27 → 2/27), months (2026-01 → Jan),
// weeks (2026-W08 → W8), and generic strings (truncated to 6 chars).
func shortLabel(label string) string {
	// Date format: 2026-02-27 → 2/27
	if len(label) == 10 && label[4] == '-' && label[7] == '-' {
		month := strings.TrimLeft(label[5:7], "0")
		day := strings.TrimLeft(label[8:10], "0")
		return month + "/" + day
	}
	// Month format: 2026-01 → Jan
	if len(label) == 7 && label[4] == '-' {
		months := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
		m, err := strconv.Atoi(label[5:7])
		if err == nil && m >= 1 && m <= 12 {
			return months[m-1]
		}
	}
	// Week format: 2026-W08 → W8
	if len(label) >= 7 && label[4] == '-' && label[5] == 'W' {
		w := strings.TrimLeft(label[6:], "0")
		if w == "" {
			w = "0"
		}
		return "W" + w
	}
	// Generic: truncate long labels
	if len(label) > 6 {
		return label[:6]
	}
	return label
}
