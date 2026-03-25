package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsClaudeAuthenticated_EnvVar(t *testing.T) {
	orig := os.Getenv("ANTHROPIC_API_KEY")
	t.Cleanup(func() { os.Setenv("ANTHROPIC_API_KEY", orig) })

	os.Setenv("ANTHROPIC_API_KEY", "sk-test-123")
	if !IsClaudeAuthenticated() {
		t.Error("expected true when ANTHROPIC_API_KEY is set")
	}
}

func TestIsClaudeAuthenticated_NoEnvVar(t *testing.T) {
	orig := os.Getenv("ANTHROPIC_API_KEY")
	t.Cleanup(func() { os.Setenv("ANTHROPIC_API_KEY", orig) })

	os.Unsetenv("ANTHROPIC_API_KEY")
	// Result depends on whether ~/.claude.json has auth on this machine,
	// so we only verify it doesn't panic. Config-path logic is tested via
	// hasClaudeConfigAuth below.
	_ = IsClaudeAuthenticated()
}

func TestHasClaudeConfigAuth(t *testing.T) {
	tests := []struct {
		name    string
		content string // empty means file not created
		want    bool
	}{
		{
			name:    "oauth account",
			content: `{"oauthAccount": {"token": "abc"}}`,
			want:    true,
		},
		{
			name:    "primary api key",
			content: `{"primaryApiKey": "sk-123"}`,
			want:    true,
		},
		{
			name:    "both keys",
			content: `{"oauthAccount": {"token": "abc"}, "primaryApiKey": "sk-123"}`,
			want:    true,
		},
		{
			name:    "no auth keys",
			content: `{"someOtherKey": "value"}`,
			want:    false,
		},
		{
			name:    "empty object",
			content: `{}`,
			want:    false,
		},
		{
			name:    "invalid json",
			content: `not json`,
			want:    false,
		},
		{
			name:    "missing file",
			content: "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			configPath := filepath.Join(dir, ".claude.json")

			if tt.content != "" {
				if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
					t.Fatalf("writing test config: %v", err)
				}
			}

			got := hasClaudeConfigAuth(configPath)
			if got != tt.want {
				t.Errorf("hasClaudeConfigAuth() = %v, want %v", got, tt.want)
			}
		})
	}
}
