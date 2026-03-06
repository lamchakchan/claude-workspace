package plugins

import (
	"bytes"
	"strings"
	"testing"
)

func TestSplitPluginKey(t *testing.T) {
	tests := []struct {
		key        string
		wantName   string
		wantMarket string
	}{
		{"skill-creator@claude-plugins-official", "skill-creator", "claude-plugins-official"},
		{"my-plugin@my-marketplace", "my-plugin", "my-marketplace"},
		{"plain-name", "plain-name", ""},
		{"with@multiple@ats", "with@multiple", "ats"},
		{"", "", ""},
	}

	for _, tt := range tests {
		name, marketplace := splitPluginKey(tt.key)
		if name != tt.wantName || marketplace != tt.wantMarket {
			t.Errorf("splitPluginKey(%q) = (%q, %q), want (%q, %q)",
				tt.key, name, marketplace, tt.wantName, tt.wantMarket)
		}
	}
}

func TestListTo_Empty(t *testing.T) {
	var buf bytes.Buffer
	// ListTo calls DiscoverInstalled which reads from filesystem.
	// When no plugins file exists, it should show the "no plugins" message.
	err := ListTo(&buf)
	if err != nil {
		t.Fatalf("ListTo() error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "No plugins installed") && !strings.Contains(out, "Installed Plugins") {
		t.Errorf("ListTo() output should mention plugins, got: %s", out)
	}
}

func TestAvailableTo_NoMarketplace(t *testing.T) {
	var buf bytes.Buffer
	err := AvailableTo(&buf)
	if err != nil {
		t.Fatalf("AvailableTo() error = %v", err)
	}
	out := buf.String()
	// Should either show available plugins or say none found
	if !strings.Contains(out, "marketplace") && !strings.Contains(out, "Available Plugins") {
		t.Errorf("AvailableTo() output should reference marketplaces or available plugins, got: %s", out)
	}
}

func TestRun_UnknownSubcommand(t *testing.T) {
	err := Run([]string{"bogus"})
	if err == nil {
		t.Fatal("Run(bogus) should return error")
	}
	if !strings.Contains(err.Error(), "unknown plugins subcommand") {
		t.Errorf("Run(bogus) error = %v, want 'unknown plugins subcommand'", err)
	}
}

func TestRun_AddNoArgs(t *testing.T) {
	err := Run([]string{"add"})
	if err == nil {
		t.Fatal("Run(add) with no plugin name should return error")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("Run(add) error = %v, want usage message", err)
	}
}

func TestRun_RemoveNoArgs(t *testing.T) {
	err := Run([]string{"remove"})
	if err == nil {
		t.Fatal("Run(remove) with no plugin name should return error")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("Run(remove) error = %v, want usage message", err)
	}
}
