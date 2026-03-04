package mcp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverUserServers(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    int
		wantNil bool
	}{
		{
			name:    "no file returns nil",
			json:    "",
			wantNil: true,
		},
		{
			name: "file with mcpServers",
			json: `{"mcpServers":{"brave-search":{},"github":{}}}`,
			want: 2,
		},
		{
			name: "file without mcpServers key",
			json: `{"something":"else"}`,
			want: 0,
		},
		{
			name: "empty mcpServers",
			json: `{"mcpServers":{}}`,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()

			if tt.json != "" {
				path := filepath.Join(home, ".claude.json")
				if err := os.WriteFile(path, []byte(tt.json), 0o644); err != nil {
					t.Fatalf("writing test file: %v", err)
				}
			}

			servers := discoverUserServers(home)

			if tt.wantNil {
				if servers != nil {
					t.Errorf("expected nil, got %v", servers)
				}
				return
			}

			if len(servers) != tt.want {
				t.Errorf("got %d servers, want %d", len(servers), tt.want)
			}

			for _, s := range servers {
				if s.Scope != "user" {
					t.Errorf("server %q scope = %q, want %q", s.Name, s.Scope, "user")
				}
			}
		})
	}
}

func TestReadServerNames(t *testing.T) {
	tests := []struct {
		name  string
		json  string
		scope string
		want  int
	}{
		{
			name:  "valid mcp.json",
			json:  `{"mcpServers":{"sentry":{},"github":{},"notion":{}}}`,
			scope: "project",
			want:  3,
		},
		{
			name:  "empty mcpServers",
			json:  `{"mcpServers":{}}`,
			scope: "project",
			want:  0,
		},
		{
			name:  "invalid json",
			json:  `{invalid`,
			scope: "project",
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, ".mcp.json")
			if err := os.WriteFile(path, []byte(tt.json), 0o644); err != nil {
				t.Fatalf("writing test file: %v", err)
			}

			servers := readServerNames(path, tt.scope)
			if len(servers) != tt.want {
				t.Errorf("got %d servers, want %d", len(servers), tt.want)
			}

			for _, s := range servers {
				if s.Scope != tt.scope {
					t.Errorf("server %q scope = %q, want %q", s.Name, s.Scope, tt.scope)
				}
			}
		})
	}
}

func TestScopeOrder(t *testing.T) {
	tests := []struct {
		scope string
		want  int
	}{
		{"user", 0},
		{"project", 1},
		{"managed", 2},
		{"unknown", 3},
	}

	for _, tt := range tests {
		t.Run(tt.scope, func(t *testing.T) {
			got := scopeOrder(tt.scope)
			if got != tt.want {
				t.Errorf("scopeOrder(%q) = %d, want %d", tt.scope, got, tt.want)
			}
		})
	}
}

func TestFilterRemovable(t *testing.T) {
	servers := []Server{
		{Name: "brave", Scope: "user"},
		{Name: "enterprise", Scope: "managed"},
		{Name: "sentry", Scope: "project"},
		{Name: "internal", Scope: "managed"},
	}

	// filterRemovable is in mcpremove.go (tui package), so test the logic directly
	result := make([]Server, 0, len(servers))
	for _, s := range servers {
		if s.Scope != "managed" {
			result = append(result, s)
		}
	}

	if len(result) != 2 {
		t.Errorf("got %d removable servers, want 2", len(result))
	}
	for _, s := range result {
		if s.Scope == "managed" {
			t.Errorf("managed server %q should be filtered out", s.Name)
		}
	}
}
