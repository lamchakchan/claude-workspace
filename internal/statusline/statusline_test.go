package statusline

import (
	"strings"
	"testing"
)

func TestRuntimeLabel(t *testing.T) {
	tests := []struct {
		cmd  string
		want string
	}{
		{"bun x ccusage statusline", "bun (ccusage)"},
		{"npx -y ccusage statusline", "npx (ccusage)"},
		{"jq -r '...'", "inline jq (fallback)"},
	}
	for _, tt := range tests {
		got := runtimeLabel(tt.cmd)
		if got != tt.want {
			t.Errorf("runtimeLabel(%q) = %q, want %q", tt.cmd, got, tt.want)
		}
	}
}

func TestDetectStatusLineCommand_ReturnsKnownForm(t *testing.T) {
	cmd := detectStatusLineCommand()
	if cmd == "" {
		t.Error("detectStatusLineCommand should never return empty string")
	}
	validPrefixes := []string{"bun x ccusage", "npx -y ccusage", "jq -r"}
	found := false
	for _, p := range validPrefixes {
		if strings.HasPrefix(cmd, p) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("unexpected command: %q", cmd)
	}
}
