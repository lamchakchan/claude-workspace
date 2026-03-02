package tui

import (
	"fmt"
	"strings"
	"testing"
)

func TestRenderTabBar(t *testing.T) {
	theme := DefaultTheme()

	tests := []struct {
		name         string
		tabs         []TabItem
		activeIdx    int
		width        int
		wantLabels   []string
		wantLeftInd  bool
		wantRightInd bool
	}{
		{
			name: "all_fit",
			tabs: []TabItem{
				{Label: "Daily"},
				{Label: "Weekly"},
				{Label: "Monthly"},
			},
			activeIdx:    0,
			width:        80,
			wantLabels:   []string{"Daily", "Weekly", "Monthly"},
			wantLeftInd:  false,
			wantRightInd: false,
		},
		{
			name:      "empty_tabs",
			tabs:      nil,
			activeIdx: 0,
			width:     80,
		},
		{
			name:         "single_tab",
			tabs:         []TabItem{{Label: "Only"}},
			activeIdx:    0,
			width:        80,
			wantLabels:   []string{"Only"},
			wantLeftInd:  false,
			wantRightInd: false,
		},
		{
			name:         "overflow_right",
			tabs:         manyTabs(10),
			activeIdx:    0,
			width:        40,
			wantLabels:   []string{"Tab-1"},
			wantLeftInd:  false,
			wantRightInd: true,
		},
		{
			name:         "overflow_left",
			tabs:         manyTabs(10),
			activeIdx:    9,
			width:        40,
			wantLabels:   []string{"Tab-10"},
			wantLeftInd:  true,
			wantRightInd: false,
		},
		{
			name:         "overflow_both",
			tabs:         manyTabs(10),
			activeIdx:    5,
			width:        40,
			wantLabels:   []string{"Tab-6"},
			wantLeftInd:  true,
			wantRightInd: true,
		},
		{
			name:         "narrow_terminal",
			tabs:         manyTabs(5),
			activeIdx:    2,
			width:        15,
			wantLabels:   []string{"Tab-3"},
			wantLeftInd:  true,
			wantRightInd: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := renderTabBar(tt.tabs, tt.activeIdx, tt.width, &theme)

			// Every output should contain a horizontal rule.
			if !strings.Contains(out, "─") {
				t.Error("expected horizontal rule (─)")
			}

			for _, label := range tt.wantLabels {
				if !strings.Contains(out, label) {
					t.Errorf("expected label %q in output", label)
				}
			}

			// Only check indicator presence/absence when there are tabs.
			if len(tt.tabs) > 0 {
				if tt.wantLeftInd {
					if !strings.Contains(out, "<") {
						t.Error("expected left scroll indicator <")
					}
				} else {
					if strings.Contains(out, "<") {
						t.Error("unexpected left scroll indicator <")
					}
				}

				if tt.wantRightInd {
					if !strings.Contains(out, ">") {
						t.Error("expected right scroll indicator >")
					}
				} else {
					if strings.Contains(out, ">") {
						t.Error("unexpected right scroll indicator >")
					}
				}
			}
		})
	}
}

func TestRenderTabBar_ActiveAlwaysVisible(t *testing.T) {
	theme := DefaultTheme()
	tabs := manyTabs(15)

	for i := range tabs {
		out := renderTabBar(tabs, i, 50, &theme)
		if !strings.Contains(out, tabs[i].Label) {
			t.Errorf("activeIdx=%d: label %q not visible in output", i, tabs[i].Label)
		}
	}
}

// manyTabs creates n TabItems with labels "Tab-1" through "Tab-N".
func manyTabs(n int) []TabItem {
	tabs := make([]TabItem, n)
	for i := range tabs {
		tabs[i] = TabItem{Label: fmt.Sprintf("Tab-%d", i+1)}
	}
	return tabs
}
