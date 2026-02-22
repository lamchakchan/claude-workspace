package setup

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestMergeSettings_EmptyExisting(t *testing.T) {
	defaults := map[string]interface{}{
		"env": map[string]interface{}{
			"KEY1": "val1",
		},
		"alwaysThinkingEnabled": true,
	}
	existing := map[string]interface{}{}

	merged := mergeSettings(existing, defaults)

	env := merged["env"].(map[string]interface{})
	if env["KEY1"] != "val1" {
		t.Errorf("expected KEY1=val1, got %v", env["KEY1"])
	}
	if merged["alwaysThinkingEnabled"] != true {
		t.Errorf("expected alwaysThinkingEnabled=true, got %v", merged["alwaysThinkingEnabled"])
	}
}

func TestMergeSettings_ExistingEnvTakesPrecedence(t *testing.T) {
	defaults := map[string]interface{}{
		"env": map[string]interface{}{
			"KEY1": "default",
			"KEY2": "default",
		},
	}
	existing := map[string]interface{}{
		"env": map[string]interface{}{
			"KEY1": "custom",
		},
	}

	merged := mergeSettings(existing, defaults)

	env := merged["env"].(map[string]interface{})
	if env["KEY1"] != "custom" {
		t.Errorf("existing env should take precedence: got KEY1=%v", env["KEY1"])
	}
	if env["KEY2"] != "default" {
		t.Errorf("missing key should use default: got KEY2=%v", env["KEY2"])
	}
}

func TestMergeSettings_DenyListUnion(t *testing.T) {
	defaults := map[string]interface{}{
		"permissions": map[string]interface{}{
			"deny": []string{"rule-a", "rule-b", "rule-c"},
		},
	}
	existing := map[string]interface{}{
		"permissions": map[string]interface{}{
			"deny": []string{"rule-b", "rule-d"},
		},
	}

	merged := mergeSettings(existing, defaults)

	perms := merged["permissions"].(map[string]interface{})
	deny := perms["deny"].([]string)

	sort.Strings(deny)
	want := []string{"rule-a", "rule-b", "rule-c", "rule-d"}
	sort.Strings(want)

	if !reflect.DeepEqual(deny, want) {
		t.Errorf("deny list union: got %v, want %v", deny, want)
	}
}

func TestMergeSettings_DenyListFromJSON(t *testing.T) {
	// When loaded from JSON, deny is []interface{} not []string
	defaults := map[string]interface{}{
		"permissions": map[string]interface{}{
			"deny": []string{"rule-a", "rule-b"},
		},
	}
	existing := map[string]interface{}{
		"permissions": map[string]interface{}{
			"deny": []interface{}{"rule-b", "rule-c"},
		},
	}

	merged := mergeSettings(existing, defaults)

	perms := merged["permissions"].(map[string]interface{})
	deny := perms["deny"].([]string)

	sort.Strings(deny)
	want := []string{"rule-a", "rule-b", "rule-c"}
	sort.Strings(want)

	if !reflect.DeepEqual(deny, want) {
		t.Errorf("deny list union with []interface{}: got %v, want %v", deny, want)
	}
}

func TestMergeSettings_NoDenyInExisting(t *testing.T) {
	defaults := map[string]interface{}{
		"permissions": map[string]interface{}{
			"deny": []string{"rule-a"},
		},
	}
	existing := map[string]interface{}{}

	merged := mergeSettings(existing, defaults)

	perms := merged["permissions"].(map[string]interface{})
	deny := perms["deny"].([]string)

	if len(deny) != 1 || deny[0] != "rule-a" {
		t.Errorf("expected default deny list, got %v", deny)
	}
}

func TestMergeSettings_BoolFlagsNotOverwritten(t *testing.T) {
	defaults := map[string]interface{}{
		"alwaysThinkingEnabled": true,
		"showTurnDuration":      true,
	}
	existing := map[string]interface{}{
		"alwaysThinkingEnabled": false,
	}

	merged := mergeSettings(existing, defaults)

	if merged["alwaysThinkingEnabled"] != false {
		t.Errorf("should not overwrite existing alwaysThinkingEnabled: got %v", merged["alwaysThinkingEnabled"])
	}
	if merged["showTurnDuration"] != true {
		t.Errorf("should set missing showTurnDuration: got %v", merged["showTurnDuration"])
	}
}

func TestMergeSettings_PreservesExtraKeys(t *testing.T) {
	defaults := map[string]interface{}{}
	existing := map[string]interface{}{
		"customKey": "customValue",
	}

	merged := mergeSettings(existing, defaults)

	if merged["customKey"] != "customValue" {
		t.Errorf("should preserve extra keys from existing: got %v", merged["customKey"])
	}
}

func TestMergeSettings_PreservesExtraPermissions(t *testing.T) {
	defaults := map[string]interface{}{
		"permissions": map[string]interface{}{
			"deny": []string{"rule-a"},
		},
	}
	existing := map[string]interface{}{
		"permissions": map[string]interface{}{
			"deny":  []string{"rule-b"},
			"allow": []string{"Bash(npm test)"},
		},
	}

	merged := mergeSettings(existing, defaults)

	perms := merged["permissions"].(map[string]interface{})
	if allow, ok := perms["allow"]; !ok {
		t.Error("should preserve 'allow' key from existing permissions")
	} else {
		allowSlice := allow.([]string)
		if len(allowSlice) != 1 || allowSlice[0] != "Bash(npm test)" {
			t.Errorf("unexpected allow value: %v", allowSlice)
		}
	}
}

func TestGetDefaultGlobalSettings(t *testing.T) {
	settings := getDefaultGlobalSettings()

	// Verify env keys exist
	env, ok := settings["env"].(map[string]interface{})
	if !ok {
		t.Fatal("expected env to be map[string]interface{}")
	}
	expectedEnvKeys := []string{
		"CLAUDE_CODE_ENABLE_TELEMETRY",
		"CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS",
		"CLAUDE_CODE_ENABLE_TASKS",
		"CLAUDE_CODE_SUBAGENT_MODEL",
		"CLAUDE_AUTOCOMPACT_PCT_OVERRIDE",
	}
	for _, key := range expectedEnvKeys {
		if _, ok := env[key]; !ok {
			t.Errorf("missing env key %q", key)
		}
	}

	// Verify deny list exists and has entries
	perms, ok := settings["permissions"].(map[string]interface{})
	if !ok {
		t.Fatal("expected permissions to be map[string]interface{}")
	}
	deny, ok := perms["deny"].([]string)
	if !ok {
		t.Fatal("expected permissions.deny to be []string")
	}
	if len(deny) == 0 {
		t.Error("deny list should not be empty")
	}

	// Verify boolean flags
	if settings["alwaysThinkingEnabled"] != true {
		t.Error("alwaysThinkingEnabled should be true")
	}
	if settings["showTurnDuration"] != true {
		t.Error("showTurnDuration should be true")
	}
}

func TestDetectShellRC_ZshFromEnv(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")
	home := t.TempDir()

	rcPath, shellName := detectShellRC(home)

	if shellName != "zsh" {
		t.Errorf("expected shellName=zsh, got %s", shellName)
	}
	if filepath.Base(rcPath) != ".zshrc" {
		t.Errorf("expected .zshrc, got %s", rcPath)
	}
}

func TestDetectShellRC_BashFromEnv(t *testing.T) {
	t.Setenv("SHELL", "/usr/bin/bash")
	home := t.TempDir()

	rcPath, shellName := detectShellRC(home)

	if shellName != "bash" {
		t.Errorf("expected shellName=bash, got %s", shellName)
	}
	if filepath.Base(rcPath) != ".bashrc" {
		t.Errorf("expected .bashrc, got %s", rcPath)
	}
}

func TestDetectShellRC_FishFromEnv(t *testing.T) {
	t.Setenv("SHELL", "/usr/bin/fish")
	home := t.TempDir()

	rcPath, shellName := detectShellRC(home)

	if shellName != "fish" {
		t.Errorf("expected shellName=fish, got %s", shellName)
	}
	if !strings.HasSuffix(rcPath, filepath.Join(".config", "fish", "config.fish")) {
		t.Errorf("expected config.fish path, got %s", rcPath)
	}
}

func TestDetectShellRC_FallbackFileExists(t *testing.T) {
	t.Setenv("SHELL", "")
	home := t.TempDir()

	// Create .zshrc so file-existence fallback triggers
	os.WriteFile(filepath.Join(home, ".zshrc"), []byte(""), 0644)

	rcPath, shellName := detectShellRC(home)

	if shellName != "zsh" {
		t.Errorf("expected shellName=zsh from fallback, got %s", shellName)
	}
	if filepath.Base(rcPath) != ".zshrc" {
		t.Errorf("expected .zshrc, got %s", rcPath)
	}
}

func TestDetectShellRC_FallbackDefault(t *testing.T) {
	t.Setenv("SHELL", "")
	home := t.TempDir()

	rcPath, shellName := detectShellRC(home)

	if shellName != "bash" {
		t.Errorf("expected shellName=bash as default, got %s", shellName)
	}
	if filepath.Base(rcPath) != ".bashrc" {
		t.Errorf("expected .bashrc, got %s", rcPath)
	}
}

func TestAppendPathToRC_AddsWhenAbsent(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".bashrc")
	os.WriteFile(rcPath, []byte("# existing content\n"), 0644)

	modified, err := appendPathToRC(home, "bash", rcPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !modified {
		t.Error("expected modified=true")
	}

	content, _ := os.ReadFile(rcPath)
	if !strings.Contains(string(content), ".local/bin") {
		t.Error("expected .local/bin in RC file content")
	}
	if !strings.Contains(string(content), "# Added by claude-workspace setup") {
		t.Error("expected comment marker in RC file content")
	}
	// Verify original content preserved
	if !strings.Contains(string(content), "# existing content") {
		t.Error("expected original content to be preserved")
	}
}

func TestAppendPathToRC_Idempotent(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".bashrc")
	os.WriteFile(rcPath, []byte("export PATH=\"$HOME/.local/bin:$PATH\"\n"), 0644)

	modified, err := appendPathToRC(home, "bash", rcPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if modified {
		t.Error("expected modified=false when .local/bin already present")
	}

	content, _ := os.ReadFile(rcPath)
	count := strings.Count(string(content), ".local/bin")
	if count != 1 {
		t.Errorf("expected 1 occurrence of .local/bin, got %d", count)
	}
}

func TestAppendPathToRC_CreatesFile(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".bashrc")
	// Don't create the file — appendPathToRC should create it

	modified, err := appendPathToRC(home, "bash", rcPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !modified {
		t.Error("expected modified=true for new file")
	}

	content, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}
	if !strings.Contains(string(content), ".local/bin") {
		t.Error("expected .local/bin in newly created RC file")
	}
}

func TestJoinStrings(t *testing.T) {
	tests := []struct {
		name string
		ss   []string
		sep  string
		want string
	}{
		{"empty", []string{}, ", ", ""},
		{"single", []string{"a"}, ", ", "a"},
		{"multiple", []string{"a", "b", "c"}, ", ", "a, b, c"},
		{"custom sep", []string{"x", "y"}, " | ", "x | y"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinStrings(tt.ss, tt.sep)
			if got != tt.want {
				t.Errorf("joinStrings(%v, %q) = %q, want %q", tt.ss, tt.sep, got, tt.want)
			}
		})
	}
}
