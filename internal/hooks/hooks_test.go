package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestParseScriptDescriptionBytes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "standard script with comment",
			input: `#!/bin/bash
set -euo pipefail

# Auto-formats files after write operations
INPUT=$(cat)
`,
			want: "Auto-formats files after write operations",
		},
		{
			name: "no comment",
			input: `#!/bin/bash
set -euo pipefail

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command')
`,
			want: "",
		},
		{
			name:  "shebang only",
			input: "#!/bin/bash\n",
			want:  "",
		},
		{
			name: "blank lines before comment",
			input: `#!/bin/bash
set -euo pipefail


# Validates secrets in file content
INPUT=$(cat)
`,
			want: "Validates secrets in file content",
		},
		{
			name: "multiple comments returns first",
			input: `#!/bin/bash
set -euo pipefail

# First description line
# Second description line
`,
			want: "First description line",
		},
		{
			name:  "empty file",
			input: "",
			want:  "",
		},
		{
			name: "comment without shebang",
			input: `# Just a comment at the top
echo hello
`,
			want: "Just a comment at the top",
		},
		{
			name: "no description in first 10 lines",
			input: `#!/bin/bash
set -euo pipefail

export PATH="$HOME/.local/bin:$PATH"
export FOO=bar
export BAZ=qux
VAR1=one
VAR2=two
VAR3=three
VAR4=four
# This comment is on line 11
`,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseScriptDescriptionBytes([]byte(tt.input))
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDiscoverHookScripts(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, root string)
		want  []HookScript
	}{
		{
			name: "multiple scripts",
			setup: func(t *testing.T, root string) {
				mkHookScript(t, root, "auto-format.sh", "Auto-formats code after write operations")
				mkHookScript(t, root, "validate-secrets.sh", "Scans file content for potential secrets")
			},
			want: []HookScript{
				{Name: "auto-format.sh", Description: "Auto-formats code after write operations"},
				{Name: "validate-secrets.sh", Description: "Scans file content for potential secrets"},
			},
		},
		{
			name:  "empty directory",
			setup: func(_ *testing.T, _ string) {},
			want:  nil,
		},
		{
			name: "non-sh files ignored",
			setup: func(t *testing.T, root string) {
				mkHookScript(t, root, "valid-hook.sh", "A valid hook")
				if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Hooks"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, "config.json"), []byte("{}"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			want: []HookScript{
				{Name: "valid-hook.sh", Description: "A valid hook"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			tt.setup(t, root)
			got := DiscoverHookScripts(root)
			assertHookScripts(t, got, tt.want)
		})
	}
}

func TestDiscoverHookConfig(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, dir string) string
		want  []HookConfig
	}{
		{
			name: "valid settings with all events",
			setup: func(t *testing.T, dir string) string {
				return mkSettingsJSON(t, dir, map[string]interface{}{
					"hooks": map[string]interface{}{
						"PreToolUse": []map[string]interface{}{
							{
								"matcher": "Bash",
								"hooks": []map[string]interface{}{
									{"type": "command", "command": "block.sh", "statusMessage": "Checking..."},
								},
							},
							{
								"matcher": "Write|Edit",
								"hooks": []map[string]interface{}{
									{"type": "command", "command": "secrets.sh", "statusMessage": "Scanning..."},
								},
							},
						},
						"PostToolUse": []map[string]interface{}{
							{
								"matcher": "Write|Edit",
								"hooks": []map[string]interface{}{
									{"type": "command", "command": "format.sh", "statusMessage": "Formatting..."},
								},
							},
						},
						"TaskCompleted": []map[string]interface{}{
							{
								"hooks": []map[string]interface{}{
									{"type": "command", "command": "verify.sh", "statusMessage": "Verifying..."},
								},
							},
						},
					},
				})
			},
			want: []HookConfig{
				{Event: "PreToolUse", Matcher: "Bash", Command: "block.sh", StatusMessage: "Checking..."},
				{Event: "PreToolUse", Matcher: "Write|Edit", Command: "secrets.sh", StatusMessage: "Scanning..."},
				{Event: "PostToolUse", Matcher: "Write|Edit", Command: "format.sh", StatusMessage: "Formatting..."},
				{Event: "TaskCompleted", Matcher: "(any)", Command: "verify.sh", StatusMessage: "Verifying..."},
			},
		},
		{
			name: "missing hooks key",
			setup: func(t *testing.T, dir string) string {
				return mkSettingsJSON(t, dir, map[string]interface{}{
					"model": "sonnet",
				})
			},
			want: nil,
		},
		{
			name: "empty hooks",
			setup: func(t *testing.T, dir string) string {
				return mkSettingsJSON(t, dir, map[string]interface{}{
					"hooks": map[string]interface{}{},
				})
			},
			want: nil,
		},
		{
			name: "malformed JSON",
			setup: func(t *testing.T, dir string) string {
				path := filepath.Join(dir, "settings.json")
				if err := os.WriteFile(path, []byte("{not valid json"), 0644); err != nil {
					t.Fatal(err)
				}
				return path
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := tt.setup(t, dir)
			got := DiscoverHookConfig(path)
			assertHookConfigs(t, got, tt.want)
		})
	}
}

// mkHookScript creates a .sh file with a shebang and description comment.
func mkHookScript(t *testing.T, root, name, description string) {
	t.Helper()
	content := fmt.Sprintf("#!/bin/bash\nset -euo pipefail\n\n# %s\nINPUT=$(cat)\n", description)
	if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
}

// mkSettingsJSON creates a settings.json file and returns its path.
func mkSettingsJSON(t *testing.T, dir string, data map[string]interface{}) string {
	t.Helper()
	path := filepath.Join(dir, "settings.json")
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// assertHookScripts compares two HookScript slices in an order-independent manner.
func assertHookScripts(t *testing.T, got, want []HookScript) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d items, want %d", len(got), len(want))
	}
	wantMap := make(map[string]string, len(want))
	for _, s := range want {
		wantMap[s.Name] = s.Description
	}
	for _, s := range got {
		wantDesc, ok := wantMap[s.Name]
		if !ok {
			t.Errorf("unexpected item: %q", s.Name)
			continue
		}
		if s.Description != wantDesc {
			t.Errorf("item %q description = %q, want %q", s.Name, s.Description, wantDesc)
		}
	}
}

// assertHookConfigs compares two HookConfig slices in an order-independent manner.
func assertHookConfigs(t *testing.T, got, want []HookConfig) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d items, want %d\ngot:  %+v\nwant: %+v", len(got), len(want), got, want)
	}
	type key struct {
		Event   string
		Matcher string
		Command string
	}
	wantMap := make(map[key]string, len(want))
	for _, c := range want {
		wantMap[key{c.Event, c.Matcher, c.Command}] = c.StatusMessage
	}
	for _, c := range got {
		k := key{c.Event, c.Matcher, c.Command}
		wantMsg, ok := wantMap[k]
		if !ok {
			t.Errorf("unexpected config: %+v", c)
			continue
		}
		if c.StatusMessage != wantMsg {
			t.Errorf("config %+v statusMessage = %q, want %q", k, c.StatusMessage, wantMsg)
		}
	}
}
