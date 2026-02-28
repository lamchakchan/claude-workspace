package tui

import (
	"strings"
	"testing"

	"github.com/lamchakchan/claude-workspace/internal/cost"
)

func TestRenderBarChart_Empty(t *testing.T) {
	theme := DefaultTheme()
	out := renderBarChart(nil, 80, &theme)
	if out != "" {
		t.Errorf("expected empty string for nil entries, got %q", out)
	}
}

func TestRenderBarChart_TooNarrow(t *testing.T) {
	theme := DefaultTheme()
	entries := []cost.ChartEntry{{Label: "2026-02-27", Value: 5.0}}
	out := renderBarChart(entries, 5, &theme)
	if out != "" {
		t.Errorf("expected empty string for narrow width, got %q", out)
	}
}

func TestRenderBarChart_SingleEntry(t *testing.T) {
	theme := DefaultTheme()
	entries := []cost.ChartEntry{{Label: "2026-02-27", Value: 5.0}}
	out := renderBarChart(entries, 80, &theme)
	if out == "" {
		t.Fatal("expected non-empty chart for single entry")
	}
	if !strings.Contains(out, "2/27") {
		t.Error("chart should contain date label 2/27")
	}
	// Should contain bar characters
	hasBar := false
	for _, ch := range barChars {
		if strings.Contains(out, ch) {
			hasBar = true
			break
		}
	}
	if !hasBar && !strings.Contains(out, "█") {
		t.Error("chart should contain at least one bar character")
	}
}

func TestRenderBarChart_MultipleEntries(t *testing.T) {
	theme := DefaultTheme()
	entries := []cost.ChartEntry{
		{Label: "2026-02-25", Value: 3.0},
		{Label: "2026-02-26", Value: 7.0},
		{Label: "2026-02-27", Value: 1.5},
	}
	out := renderBarChart(entries, 80, &theme)
	if out == "" {
		t.Fatal("expected non-empty chart")
	}
	lines := strings.Split(out, "\n")
	// Should have: 1 blank + 8 rows + 1 x-axis + 1 labels + 1 trailing = ~12 lines
	if len(lines) < 10 {
		t.Errorf("chart has %d lines, expected at least 10", len(lines))
	}
}

func TestRenderBarChart_ZeroCost(t *testing.T) {
	theme := DefaultTheme()
	entries := []cost.ChartEntry{
		{Label: "2026-02-27", Value: 0},
	}
	// Should not panic
	out := renderBarChart(entries, 80, &theme)
	if out == "" {
		t.Fatal("expected non-empty chart even with zero cost")
	}
}

func TestRenderBarChart_MonthlyLabels(t *testing.T) {
	theme := DefaultTheme()
	entries := []cost.ChartEntry{
		{Label: "2026-01", Value: 10.0},
		{Label: "2026-02", Value: 20.0},
	}
	out := renderBarChart(entries, 80, &theme)
	if out == "" {
		t.Fatal("expected non-empty chart for monthly entries")
	}
	if !strings.Contains(out, "Jan") {
		t.Error("chart should contain month label Jan")
	}
}

func TestRenderBarChart_WeeklyLabels(t *testing.T) {
	theme := DefaultTheme()
	entries := []cost.ChartEntry{
		{Label: "2026-W08", Value: 5.0},
		{Label: "2026-W09", Value: 8.0},
	}
	out := renderBarChart(entries, 80, &theme)
	if out == "" {
		t.Fatal("expected non-empty chart for weekly entries")
	}
	if !strings.Contains(out, "W8") {
		t.Error("chart should contain week label W8")
	}
}

func TestNiceMax(t *testing.T) {
	tests := []struct {
		input float64
		want  float64
	}{
		{0, 1},
		{0.5, 0.5},
		{1.2, 2},
		{3.7, 5},
		{7.5, 10},
		{15, 20},
		{45, 50},
		{85, 100},
	}
	for _, tt := range tests {
		got := niceMax(tt.input)
		if got != tt.want {
			t.Errorf("niceMax(%v) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestShortLabel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Dates
		{"2026-02-27", "2/27"},
		{"2026-01-05", "1/5"},
		{"2026-12-31", "12/31"},
		// Months
		{"2026-01", "Jan"},
		{"2026-06", "Jun"},
		{"2026-12", "Dec"},
		// Weeks
		{"2026-W08", "W8"},
		{"2026-W01", "W1"},
		{"2026-W52", "W52"},
		// Generic short
		{"short", "short"},
		// Generic long (truncated)
		{"a-very-long-label", "a-very"},
	}
	for _, tt := range tests {
		got := shortLabel(tt.input)
		if got != tt.want {
			t.Errorf("shortLabel(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatDollar(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{0, "$0"},
		{5.5, "$5.5"},
		{10, "$10"},
		{150, "$150"},
	}
	for _, tt := range tests {
		got := formatDollar(tt.input)
		if got != tt.want {
			t.Errorf("formatDollar(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
