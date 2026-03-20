package plugins

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRunMarketplace_UnknownSubcommand(t *testing.T) {
	err := RunMarketplace([]string{"bogus"})
	if err == nil {
		t.Fatal("expected error for unknown subcommand")
	}
	if !strings.Contains(err.Error(), "unknown marketplace subcommand") {
		t.Errorf("error = %q, want mention of unknown subcommand", err.Error())
	}
}

func TestMarketplaceAdd_NoArgs(t *testing.T) {
	err := MarketplaceAdd(nil)
	if err == nil {
		t.Fatal("expected error for no args")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("error = %q, want usage hint", err.Error())
	}
}

func TestMarketplaceRemove_NoArgs(t *testing.T) {
	err := MarketplaceRemove(nil)
	if err == nil {
		t.Fatal("expected error for no args")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("error = %q, want usage hint", err.Error())
	}
}

func TestMarketplaceAdd_InvalidFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"no slash", "noslash"},
		{"too many slashes", "a/b/c"},
		{"empty", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{tt.input}
			if tt.input == "" {
				args = nil
			}
			err := MarketplaceAdd(args)
			if err == nil {
				t.Fatal("expected error for invalid format")
			}
		})
	}
}

func TestMarketplaceAdd_LocalPath(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"absolute path", "/home/user/git/org/repo"},
		{"relative dot-slash", "./local-marketplace"},
		{"relative dot-dot", "../sibling/marketplace"},
		{"home-relative", "~/git/org/repo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// These should pass format validation (the actual `claude` CLI call
			// will fail in test, but we only check that format validation passes).
			err := MarketplaceAdd([]string{tt.input})
			// The error should be from the claude CLI execution, not format validation
			if err != nil && strings.Contains(err.Error(), "invalid marketplace format") {
				t.Errorf("local path %q was rejected as invalid format", tt.input)
			}
		})
	}
}

func TestIsLocalPath(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"/absolute/path", true},
		{"./relative", true},
		{"../parent", true},
		{"~/home", true},
		{".", true},
		{"..", true},
		{"~", true},
		{"owner/repo", false},
		{"noslash", false},
		{"a/b/c", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isLocalPath(tt.input)
			if got != tt.want {
				t.Errorf("isLocalPath(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestMarketplaceListTo_Empty(t *testing.T) {
	// Use a temp dir with no marketplaces to get the empty-state message
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", origHome) }()

	var buf bytes.Buffer
	err := MarketplaceListTo(&buf)
	if err != nil {
		t.Fatalf("MarketplaceListTo() error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "No plugin marketplaces configured") {
		t.Errorf("output = %q, want empty-state message", out)
	}
}
