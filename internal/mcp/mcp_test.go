package mcp

import (
	"testing"
)

const (
	scopeProject = "project"
)

//nolint:gocyclo
func TestParseAddArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		check   func(t *testing.T, cfg *addConfig)
	}{
		{
			name:    "empty args returns error",
			args:    []string{},
			wantErr: true,
		},
		{
			name: "stdio command with double dash",
			args: []string{"my-server", "--", "npx", "-y", "@modelcontextprotocol/server"},
			check: func(t *testing.T, cfg *addConfig) {
				if cfg.Name != "my-server" {
					t.Errorf("Name = %q, want %q", cfg.Name, "my-server")
				}
				if cfg.Transport != transportStdio {
					t.Errorf("Transport = %q, want %q", cfg.Transport, transportStdio)
				}
				if len(cfg.CommandArgs) != 3 || cfg.CommandArgs[0] != "npx" {
					t.Errorf("CommandArgs = %v, want [npx -y @modelcontextprotocol/server]", cfg.CommandArgs)
				}
			},
		},
		{
			name: "http URL auto-detects transport",
			args: []string{"github", "https://api.githubcopilot.com/mcp/"},
			check: func(t *testing.T, cfg *addConfig) {
				if cfg.Name != "github" {
					t.Errorf("Name = %q, want %q", cfg.Name, "github")
				}
				if cfg.Transport != transportHTTP {
					t.Errorf("Transport = %q, want %q", cfg.Transport, transportHTTP)
				}
				if cfg.McpURL != "https://api.githubcopilot.com/mcp/" {
					t.Errorf("McpURL = %q, want %q", cfg.McpURL, "https://api.githubcopilot.com/mcp/")
				}
			},
		},
		{
			name: "scope flag",
			args: []string{"srv", "--scope", scopeProject, "--", "cmd"},
			check: func(t *testing.T, cfg *addConfig) {
				if cfg.Scope != scopeProject {
					t.Errorf("Scope = %q, want %q", cfg.Scope, scopeProject)
				}
			},
		},
		{
			name: "transport flag",
			args: []string{"srv", "--transport", "sse", "https://example.com/sse"},
			check: func(t *testing.T, cfg *addConfig) {
				if cfg.Transport != transportSSE {
					t.Errorf("Transport = %q, want %q", cfg.Transport, transportSSE)
				}
			},
		},
		{
			name: "env flag",
			args: []string{"srv", "--env", "FOO=bar", "--env", "BAZ=qux", "--", "cmd"},
			check: func(t *testing.T, cfg *addConfig) {
				if cfg.EnvVars["FOO"] != "bar" {
					t.Errorf("EnvVars[FOO] = %q, want %q", cfg.EnvVars["FOO"], "bar")
				}
				if cfg.EnvVars["BAZ"] != "qux" {
					t.Errorf("EnvVars[BAZ] = %q, want %q", cfg.EnvVars["BAZ"], "qux")
				}
			},
		},
		{
			name: "header flag",
			args: []string{"srv", "--header", "X-Custom: value", "--", "cmd"},
			check: func(t *testing.T, cfg *addConfig) {
				if len(cfg.Headers) != 1 || cfg.Headers[0] != "X-Custom: value" {
					t.Errorf("Headers = %v, want [X-Custom: value]", cfg.Headers)
				}
			},
		},
		{
			name: "oauth flag",
			args: []string{"srv", "--oauth", "https://example.com/mcp"},
			check: func(t *testing.T, cfg *addConfig) {
				if !cfg.UseOAuth {
					t.Error("UseOAuth = false, want true")
				}
			},
		},
		{
			name: "client-id flag",
			args: []string{"srv", "--client-id", "my-id", "https://example.com/mcp"},
			check: func(t *testing.T, cfg *addConfig) {
				if cfg.ClientID != "my-id" {
					t.Errorf("ClientID = %q, want %q", cfg.ClientID, "my-id")
				}
			},
		},
		{
			name: "client-secret flag",
			args: []string{"srv", "--client-secret", "--", "cmd"},
			check: func(t *testing.T, cfg *addConfig) {
				if !cfg.PromptClientSecret {
					t.Error("PromptClientSecret = false, want true")
				}
			},
		},
		{
			name: "bearer flag",
			args: []string{"srv", "--bearer", "https://example.com/mcp"},
			check: func(t *testing.T, cfg *addConfig) {
				if !cfg.PromptBearer {
					t.Error("PromptBearer = false, want true")
				}
			},
		},
		{
			name: "api-key flag",
			args: []string{"srv", "--api-key", "MY_KEY", "--", "cmd"},
			check: func(t *testing.T, cfg *addConfig) {
				if cfg.APIKeyEnvVar != "MY_KEY" {
					t.Errorf("APIKeyEnvVar = %q, want %q", cfg.APIKeyEnvVar, "MY_KEY")
				}
			},
		},
		{
			name: "command args without double dash",
			args: []string{"srv", "npx", "-y", "some-pkg"},
			check: func(t *testing.T, cfg *addConfig) {
				if cfg.Transport != transportStdio {
					t.Errorf("Transport = %q, want %q", cfg.Transport, transportStdio)
				}
				if len(cfg.CommandArgs) != 3 || cfg.CommandArgs[0] != "npx" {
					t.Errorf("CommandArgs = %v, want [npx -y some-pkg]", cfg.CommandArgs)
				}
			},
		},
		{
			name: "default scope is local",
			args: []string{"srv", "--", "cmd"},
			check: func(t *testing.T, cfg *addConfig) {
				if cfg.Scope != "local" {
					t.Errorf("Scope = %q, want %q", cfg.Scope, "local")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := parseAddArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

//nolint:gocyclo
func TestParseRemoteArgs(t *testing.T) {
	tests := []struct {
		name      string
		mcpURL    string
		extraArgs []string
		wantErr   bool
		check     func(t *testing.T, cfg *remoteConfig)
	}{
		{
			name:    "empty URL returns error",
			mcpURL:  "",
			wantErr: true,
		},
		{
			name:   "basic URL derives name",
			mcpURL: "https://mcp.sentry.dev/mcp",
			check: func(t *testing.T, cfg *remoteConfig) {
				if cfg.Name != "sentry-dev" {
					t.Errorf("Name = %q, want %q", cfg.Name, "sentry-dev")
				}
				if cfg.Transport != transportHTTP {
					t.Errorf("Transport = %q, want %q", cfg.Transport, transportHTTP)
				}
			},
		},
		{
			name:      "name flag overrides derived name",
			mcpURL:    "https://mcp.sentry.dev/mcp",
			extraArgs: []string{"--name", "sentry"},
			check: func(t *testing.T, cfg *remoteConfig) {
				if cfg.Name != "sentry" {
					t.Errorf("Name = %q, want %q", cfg.Name, "sentry")
				}
			},
		},
		{
			name:      "scope flag",
			mcpURL:    "https://example.com/mcp",
			extraArgs: []string{"--scope", scopeProject},
			check: func(t *testing.T, cfg *remoteConfig) {
				if cfg.Scope != scopeProject {
					t.Errorf("Scope = %q, want %q", cfg.Scope, scopeProject)
				}
			},
		},
		{
			name:      "header flag",
			mcpURL:    "https://example.com/mcp",
			extraArgs: []string{"--header", "X-API-Key: abc"},
			check: func(t *testing.T, cfg *remoteConfig) {
				if len(cfg.Headers) != 1 || cfg.Headers[0] != "X-API-Key: abc" {
					t.Errorf("Headers = %v, want [X-API-Key: abc]", cfg.Headers)
				}
			},
		},
		{
			name:      "bearer flag",
			mcpURL:    "https://example.com/mcp",
			extraArgs: []string{"--bearer"},
			check: func(t *testing.T, cfg *remoteConfig) {
				if !cfg.PromptBearer {
					t.Error("PromptBearer = false, want true")
				}
			},
		},
		{
			name:      "oauth flag",
			mcpURL:    "https://example.com/mcp",
			extraArgs: []string{"--oauth"},
			check: func(t *testing.T, cfg *remoteConfig) {
				if !cfg.UseOAuth {
					t.Error("UseOAuth = false, want true")
				}
			},
		},
		{
			name:      "client-id and client-secret flags",
			mcpURL:    "https://example.com/mcp",
			extraArgs: []string{"--client-id", "my-id", "--client-secret"},
			check: func(t *testing.T, cfg *remoteConfig) {
				if cfg.ClientID != "my-id" {
					t.Errorf("ClientID = %q, want %q", cfg.ClientID, "my-id")
				}
				if !cfg.PromptClientSecret {
					t.Error("PromptClientSecret = false, want true")
				}
			},
		},
		{
			name:   "sse transport detection",
			mcpURL: "https://example.com/sse",
			check: func(t *testing.T, cfg *remoteConfig) {
				if cfg.Transport != transportSSE {
					t.Errorf("Transport = %q, want %q", cfg.Transport, transportSSE)
				}
			},
		},
		{
			name:   "http transport for non-sse URL",
			mcpURL: "https://example.com/mcp",
			check: func(t *testing.T, cfg *remoteConfig) {
				if cfg.Transport != transportHTTP {
					t.Errorf("Transport = %q, want %q", cfg.Transport, transportHTTP)
				}
			},
		},
		{
			name:   "default scope is user",
			mcpURL: "https://example.com/mcp",
			check: func(t *testing.T, cfg *remoteConfig) {
				if cfg.Scope != "user" {
					t.Errorf("Scope = %q, want %q", cfg.Scope, "user")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := parseRemoteArgs(tt.mcpURL, tt.extraArgs)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func TestDeriveServerName(t *testing.T) {
	tests := []struct {
		name   string
		mcpURL string
		want   string
	}{
		{
			name:   "sentry mcp URL",
			mcpURL: "https://mcp.sentry.dev/mcp",
			want:   "sentry-dev",
		},
		{
			name:   "github copilot URL",
			mcpURL: "https://api.githubcopilot.com/mcp/",
			want:   "api-githubcopilot",
		},
		{
			name:   "mcp-gateway prefix",
			mcpURL: "https://mcp-gateway.company.com/mcp",
			want:   "gateway-company",
		},
		{
			name:   "invalid URL falls back",
			mcpURL: "://invalid",
			want:   "remote-gateway",
		},
		{
			name:   "simple domain",
			mcpURL: "https://example.com/mcp",
			want:   "example",
		},
		{
			name:   "mcp dot prefix stripped",
			mcpURL: "https://mcp.notion.com/mcp",
			want:   "notion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveServerName(tt.mcpURL)
			if got != tt.want {
				t.Errorf("deriveServerName(%q) = %q, want %q", tt.mcpURL, got, tt.want)
			}
		})
	}
}

func TestBuildAddClaudeArgs(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *addConfig
		wantErr bool
		check   func(t *testing.T, args []string)
	}{
		{
			name: "stdio with command args",
			cfg: &addConfig{
				Name:        "my-server",
				Scope:       "local",
				Transport:   transportStdio,
				EnvVars:     map[string]string{},
				CommandArgs: []string{"npx", "-y", "some-pkg"},
			},
			check: func(t *testing.T, args []string) {
				want := []string{"mcp", "add", "--transport", "stdio", "--scope", "local", "my-server", "--", "npx", "-y", "some-pkg"}
				assertSliceEqual(t, args, want)
			},
		},
		{
			name: "http with URL",
			cfg: &addConfig{
				Name:      "github",
				Scope:     "user",
				Transport: transportHTTP,
				EnvVars:   map[string]string{},
				McpURL:    "https://api.githubcopilot.com/mcp/",
			},
			check: func(t *testing.T, args []string) {
				want := []string{"mcp", "add", "--transport", "http", "--scope", "user", "github", "https://api.githubcopilot.com/mcp/"}
				assertSliceEqual(t, args, want)
			},
		},
		{
			name: "http without URL returns error",
			cfg: &addConfig{
				Name:      "broken",
				Scope:     "local",
				Transport: transportHTTP,
				EnvVars:   map[string]string{},
			},
			wantErr: true,
		},
		{
			name: "sse without URL returns error",
			cfg: &addConfig{
				Name:      "broken",
				Scope:     "local",
				Transport: transportSSE,
				EnvVars:   map[string]string{},
			},
			wantErr: true,
		},
		{
			name: "env vars after name but before command",
			cfg: &addConfig{
				Name:        "srv",
				Scope:       "local",
				Transport:   transportStdio,
				EnvVars:     map[string]string{"KEY": "val"},
				CommandArgs: []string{"npx", "-y", "pkg"},
			},
			check: func(t *testing.T, args []string) {
				// -e must come after <name> (variadic flag can't consume the server name)
				// and before -- <cmd> (so claude treats it as a subprocess env var, not a cmd arg).
				nameIdx := indexOf(args, "srv")
				eIdx := indexOf(args, "-e")
				dashIdx := indexOf(args, "--")
				if nameIdx < 0 || eIdx < 0 || dashIdx < 0 {
					t.Fatalf("expected 'srv', '-e', and '--' in args: %v", args)
				}
				if eIdx < nameIdx {
					t.Errorf("-e (idx %d) must come after <name> (idx %d): %v", eIdx, nameIdx, args)
				}
				if eIdx > dashIdx {
					t.Errorf("-e (idx %d) must come before -- (idx %d): %v", eIdx, dashIdx, args)
				}
				assertContains(t, args, "KEY=val")
			},
		},
		{
			name: "headers included after positional args",
			cfg: &addConfig{
				authOpts:  authOpts{Headers: []string{"X-Custom: value"}},
				Name:      "srv",
				Scope:     "local",
				Transport: transportHTTP,
				EnvVars:   map[string]string{},
				McpURL:    "https://example.com",
			},
			check: func(t *testing.T, args []string) {
				// --header must come after <name> and <url> to avoid the variadic
				// flag consuming positional args in the Claude CLI parser.
				nameIdx := indexOf(args, "srv")
				headerIdx := indexOf(args, "--header")
				if nameIdx < 0 || headerIdx < 0 {
					t.Fatalf("expected 'srv' and '--header' in args: %v", args)
				}
				if headerIdx < nameIdx {
					t.Errorf("--header (idx %d) must come after <name> (idx %d) in args: %v", headerIdx, nameIdx, args)
				}
				assertContains(t, args, "X-Custom: value")
			},
		},
		{
			name: "client-id and client-secret included",
			cfg: &addConfig{
				authOpts:  authOpts{ClientID: "my-id", ClientSecret: "tok"},
				Name:      "srv",
				Scope:     "local",
				Transport: transportHTTP,
				EnvVars:   map[string]string{},
				McpURL:    "https://example.com",
			},
			check: func(t *testing.T, args []string) {
				assertContains(t, args, "--client-id")
				assertContains(t, args, "my-id")
				assertContains(t, args, "--client-secret")
				assertContains(t, args, "tok")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := buildAddClaudeArgs(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, args)
			}
		})
	}
}

func TestBuildRemoteClaudeArgs(t *testing.T) {
	tests := []struct {
		name  string
		cfg   *remoteConfig
		check func(t *testing.T, args []string)
	}{
		{
			name: "basic URL",
			cfg: &remoteConfig{
				Name:      "sentry",
				McpURL:    "https://mcp.sentry.dev/mcp",
				Scope:     "user",
				Transport: transportHTTP,
			},
			check: func(t *testing.T, args []string) {
				want := []string{"mcp", "add", "--transport", "http", "--scope", "user", "sentry", "https://mcp.sentry.dev/mcp"}
				assertSliceEqual(t, args, want)
			},
		},
		{
			name: "with headers after positional args",
			cfg: &remoteConfig{
				authOpts:  authOpts{Headers: []string{"Authorization: Bearer token123"}},
				Name:      "srv",
				McpURL:    "https://example.com/mcp",
				Scope:     "user",
				Transport: transportHTTP,
			},
			check: func(t *testing.T, args []string) {
				// --header must come after <name> and <url> to avoid the variadic
				// flag consuming positional args in the Claude CLI parser.
				urlIdx := indexOf(args, "https://example.com/mcp")
				headerIdx := indexOf(args, "--header")
				if urlIdx < 0 || headerIdx < 0 {
					t.Fatalf("expected URL and '--header' in args: %v", args)
				}
				if headerIdx < urlIdx {
					t.Errorf("--header (idx %d) must come after URL (idx %d) in args: %v", headerIdx, urlIdx, args)
				}
				assertContains(t, args, "Authorization: Bearer token123")
			},
		},
		{
			name: "with client-id and client-secret",
			cfg: &remoteConfig{
				authOpts:  authOpts{ClientID: "my-id", ClientSecret: "tok"},
				Name:      "srv",
				McpURL:    "https://example.com/mcp",
				Scope:     "user",
				Transport: transportHTTP,
			},
			check: func(t *testing.T, args []string) {
				assertContains(t, args, "--client-id")
				assertContains(t, args, "my-id")
				assertContains(t, args, "--client-secret")
				assertContains(t, args, "tok")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildRemoteClaudeArgs(tt.cfg)
			if tt.check != nil {
				tt.check(t, args)
			}
		})
	}
}

func TestMaskSensitiveArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "env values masked",
			args: []string{"mcp", "add", "-e", "API_KEY=secret123", "my-server"},
			want: []string{"mcp", "add", "-e", "API_KEY=****", "my-server"},
		},
		{
			name: "bearer headers masked",
			args: []string{"mcp", "add", "--header", "Authorization: Bearer mysecrettoken", "my-server"},
			want: []string{"mcp", "add", "--header", "Authorization: Bearer ****", "my-server"},
		},
		{
			name: "non-sensitive args preserved",
			args: []string{"mcp", "add", "--transport", "http", "--scope", "user", "my-server"},
			want: []string{"mcp", "add", "--transport", "http", "--scope", "user", "my-server"},
		},
		{
			name: "multiple env vars all masked",
			args: []string{"mcp", "add", "-e", "KEY1=val1", "-e", "KEY2=val2", "srv"},
			want: []string{"mcp", "add", "-e", "KEY1=****", "-e", "KEY2=****", "srv"},
		},
		{
			name: "non-bearer header not masked",
			args: []string{"mcp", "add", "--header", "X-Custom: value", "srv"},
			want: []string{"mcp", "add", "--header", "X-Custom: value", "srv"},
		},
		{
			name: "client-secret value masked",
			args: []string{"mcp", "add", "--client-id", "my-id", "--client-secret", "supersensitivevalue", "srv"},
			want: []string{"mcp", "add", "--client-id", "my-id", "--client-secret", "****", "srv"},
		},
		{
			name: "empty args",
			args: []string{},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskSensitiveArgs(tt.args)
			assertSliceEqual(t, got, tt.want)
		})
	}
}

func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}

func assertSliceEqual(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("slice length mismatch: got %d (%v), want %d (%v)", len(got), got, len(want), want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("index %d: got %q, want %q\n  full got:  %v\n  full want: %v", i, got[i], want[i], got, want)
		}
	}
}

func assertContains(t *testing.T, slice []string, item string) {
	t.Helper()
	for _, s := range slice {
		if s == item {
			return
		}
	}
	t.Errorf("slice %v does not contain %q", slice, item)
}
