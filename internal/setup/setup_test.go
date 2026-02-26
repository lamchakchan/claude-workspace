package setup

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

func TestMain(m *testing.M) {
	// Wire up GlobalFS so GetDefaultGlobalSettings() can read the embedded template
	platform.GlobalFS = os.DirFS(filepath.Join("..", "..", "_template", "global"))
	os.Exit(m.Run())
}

// toStringSlice converts []interface{} (from JSON) to []string for test assertions.
func toStringSlice(v interface{}) []string {
	switch s := v.(type) {
	case []string:
		return s
	case []interface{}:
		out := make([]string, 0, len(s))
		for _, item := range s {
			if str, ok := item.(string); ok {
				out = append(out, str)
			}
		}
		return out
	}
	return nil
}

func TestMergeSettings_EmptyExisting(t *testing.T) {
	defaults := map[string]interface{}{
		"env": map[string]interface{}{
			"KEY1": "val1",
		},
		"alwaysThinkingEnabled": true,
	}
	existing := map[string]interface{}{}

	merged := MergeSettings(existing, defaults)

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

	merged := MergeSettings(existing, defaults)

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

	merged := MergeSettings(existing, defaults)

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

	merged := MergeSettings(existing, defaults)

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

	merged := MergeSettings(existing, defaults)

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

	merged := MergeSettings(existing, defaults)

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

	merged := MergeSettings(existing, defaults)

	if merged["customKey"] != "customValue" {
		t.Errorf("should preserve extra keys from existing: got %v", merged["customKey"])
	}
}

func TestMergeSettings_PreservesExtraPermissions(t *testing.T) {
	defaults := map[string]interface{}{
		"permissions": map[string]interface{}{
			"deny":  []string{"rule-a"},
			"allow": []string{"Read", "Write"},
		},
	}
	existing := map[string]interface{}{
		"permissions": map[string]interface{}{
			"deny":                  []string{"rule-b"},
			"allow":                 []string{"Bash(npm test)"},
			"additionalDirectories": []string{"/tmp"},
		},
	}

	merged := MergeSettings(existing, defaults)

	perms := merged["permissions"].(map[string]interface{})

	// allow should be unioned
	allow := perms["allow"].([]string)
	allowSet := make(map[string]bool)
	for _, a := range allow {
		allowSet[a] = true
	}
	for _, want := range []string{"Bash(npm test)", "Read", "Write"} {
		if !allowSet[want] {
			t.Errorf("expected %q in allow list, got %v", want, allow)
		}
	}

	// additionalDirectories should be preserved
	if _, ok := perms["additionalDirectories"]; !ok {
		t.Error("should preserve 'additionalDirectories' key from existing permissions")
	}
}

func TestGetDefaultGlobalSettings(t *testing.T) {
	settings := GetDefaultGlobalSettings()

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
	deny := toStringSlice(perms["deny"])
	if len(deny) == 0 {
		t.Error("deny list should not be empty")
	}

	// Verify allow list exists and has entries
	allow := toStringSlice(perms["allow"])
	if len(allow) == 0 {
		t.Error("allow list should not be empty")
	}

	// Verify boolean flags
	if settings["alwaysThinkingEnabled"] != true {
		t.Error("alwaysThinkingEnabled should be true")
	}
	if settings["showTurnDuration"] != true {
		t.Error("showTurnDuration should be true")
	}
}

func TestGetDefaultGlobalSettings_HasAllow(t *testing.T) {
	settings := GetDefaultGlobalSettings()
	perms := settings["permissions"].(map[string]interface{})
	allow := toStringSlice(perms["allow"])

	allowSet := make(map[string]bool)
	for _, a := range allow {
		allowSet[a] = true
	}

	// Spot-check key entries
	for _, want := range []string{"Read", "Write", "Edit", "mcp__*", "WebSearch", "Bash(go *)", "Bash(git push *)"} {
		if !allowSet[want] {
			t.Errorf("expected %q in allow list", want)
		}
	}
}

func TestGetDefaultGlobalSettings_DenyReconciled(t *testing.T) {
	settings := GetDefaultGlobalSettings()
	perms := settings["permissions"].(map[string]interface{})
	deny := toStringSlice(perms["deny"])

	denySet := make(map[string]bool)
	for _, d := range deny {
		denySet[d] = true
	}

	// Wildcards should be present
	if !denySet["Bash(git push --force *)"] {
		t.Error("expected wildcard force-push deny rule")
	}
	if !denySet["Read(./.env.*)"] {
		t.Error("expected Read(./.env.*) deny rule")
	}

	// Branch-scoped rules should NOT be present (superseded by wildcards)
	if denySet["Bash(git push --force * main)"] {
		t.Error("branch-scoped force-push rule should not be present")
	}
	if denySet["Bash(git push -f * master)"] {
		t.Error("branch-scoped force-push rule should not be present")
	}
}

func TestMergeSettings_AllowListUnion(t *testing.T) {
	defaults := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []string{"Read", "Write", "Edit"},
		},
	}
	existing := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []string{"Write", "Bash(go *)"},
		},
	}

	merged := MergeSettings(existing, defaults)

	perms := merged["permissions"].(map[string]interface{})
	allow := perms["allow"].([]string)

	sort.Strings(allow)
	want := []string{"Bash(go *)", "Edit", "Read", "Write"}
	sort.Strings(want)

	if !reflect.DeepEqual(allow, want) {
		t.Errorf("allow list union: got %v, want %v", allow, want)
	}
}

func TestMergeSettings_AllowListFromJSON(t *testing.T) {
	// When loaded from JSON, allow is []interface{} not []string
	defaults := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []string{"Read", "Write"},
		},
	}
	existing := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []interface{}{"Write", "Edit"},
		},
	}

	merged := MergeSettings(existing, defaults)

	perms := merged["permissions"].(map[string]interface{})
	allow := perms["allow"].([]string)

	sort.Strings(allow)
	want := []string{"Edit", "Read", "Write"}
	sort.Strings(want)

	if !reflect.DeepEqual(allow, want) {
		t.Errorf("allow list union with []interface{}: got %v, want %v", allow, want)
	}
}

func TestMergeSettings_NoAllowInExisting(t *testing.T) {
	defaults := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []string{"Read", "Write"},
			"deny":  []string{"rule-a"},
		},
	}
	existing := map[string]interface{}{}

	merged := MergeSettings(existing, defaults)

	perms := merged["permissions"].(map[string]interface{})
	allow := perms["allow"].([]string)

	if len(allow) != 2 || allow[0] != "Read" || allow[1] != "Write" {
		t.Errorf("expected default allow list, got %v", allow)
	}
}

func TestMergeSettings_ForcePermissionsReplace(t *testing.T) {
	defaults := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []string{"Read", "Write"},
			"deny":  []string{"rule-a"},
		},
		"env": map[string]interface{}{
			"KEY1": "default",
		},
	}
	existing := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow":                 []string{"Bash(custom *)"},
			"deny":                  []string{"rule-b"},
			"additionalDirectories": []string{"/tmp"},
		},
		"env": map[string]interface{}{
			"KEY1": "custom",
			"KEY2": "user",
		},
		"customSetting": true,
	}

	merged := MergeSettingsForce(existing, defaults)

	// Permissions should be replaced wholesale from defaults
	perms := merged["permissions"].(map[string]interface{})
	allow := perms["allow"].([]string)
	if len(allow) != 2 || allow[0] != "Read" || allow[1] != "Write" {
		t.Errorf("force should replace allow list: got %v", allow)
	}
	deny := perms["deny"].([]string)
	if len(deny) != 1 || deny[0] != "rule-a" {
		t.Errorf("force should replace deny list: got %v", deny)
	}
	// additionalDirectories should NOT be present (replaced wholesale)
	if _, ok := perms["additionalDirectories"]; ok {
		t.Error("force should replace all permissions, removing additionalDirectories")
	}

	// Env should still be merged (existing takes precedence)
	env := merged["env"].(map[string]interface{})
	if env["KEY1"] != "custom" {
		t.Errorf("env should preserve existing values: got KEY1=%v", env["KEY1"])
	}
	if env["KEY2"] != "user" {
		t.Errorf("env should preserve user-only keys: got KEY2=%v", env["KEY2"])
	}

	// Other settings should be preserved
	if merged["customSetting"] != true {
		t.Error("force should preserve non-permission settings")
	}
}

func TestMergeUserMCPServers_AddsWhenAbsent(t *testing.T) {
	config := map[string]interface{}{}
	servers := map[string]interface{}{
		"memory": map[string]interface{}{"command": "npx", "args": []string{"-y", "memory-server"}},
		"git":    map[string]interface{}{"command": "npx", "args": []string{"-y", "git-server"}},
	}

	merged := MergeUserMCPServers(config, servers)

	mcp, ok := merged["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatal("expected mcpServers to be a map")
	}
	if _, ok := mcp["memory"]; !ok {
		t.Error("expected memory server to be added")
	}
	if _, ok := mcp["git"]; !ok {
		t.Error("expected git server to be added")
	}
}

func TestMergeUserMCPServers_DoesNotOverwriteExisting(t *testing.T) {
	customCfg := map[string]interface{}{"command": "custom", "args": []string{"--custom"}}
	config := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"memory": customCfg,
		},
	}
	servers := map[string]interface{}{
		"memory": map[string]interface{}{"command": "npx", "args": []string{"-y", "memory-server"}},
		"git":    map[string]interface{}{"command": "npx", "args": []string{"-y", "git-server"}},
	}

	merged := MergeUserMCPServers(config, servers)

	mcp := merged["mcpServers"].(map[string]interface{})
	memory := mcp["memory"].(map[string]interface{})
	if memory["command"] != "custom" {
		t.Errorf("existing memory server should not be overwritten, got command=%v", memory["command"])
	}
	if _, ok := mcp["git"]; !ok {
		t.Error("expected git server to be added alongside existing memory")
	}
}

func TestMergeUserMCPServers_PreservesOtherConfigKeys(t *testing.T) {
	config := map[string]interface{}{
		"primaryApiKey": "sk-test",
		"mcpServers":    map[string]interface{}{},
	}
	servers := map[string]interface{}{
		"git": map[string]interface{}{"command": "npx"},
	}

	merged := MergeUserMCPServers(config, servers)

	if merged["primaryApiKey"] != "sk-test" {
		t.Error("expected primaryApiKey to be preserved")
	}
}

func TestMergeUserMCPServers_NilMcpServers(t *testing.T) {
	config := map[string]interface{}{
		"primaryApiKey": "sk-test",
		// no mcpServers key
	}
	servers := map[string]interface{}{
		"git": map[string]interface{}{"command": "npx"},
	}

	merged := MergeUserMCPServers(config, servers)

	mcp, ok := merged["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatal("expected mcpServers map to be created")
	}
	if _, ok := mcp["git"]; !ok {
		t.Error("expected git server to be present")
	}
}

func TestEnsureLocalBinClaude_CreatesSymlink(t *testing.T) {
	home := t.TempDir()

	// Create a mock "claude" binary to symlink to
	mockBin := filepath.Join(home, "usr", "bin", "claude")
	if err := os.MkdirAll(filepath.Dir(mockBin), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mockBin, []byte("#!/bin/bash\necho stub"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := ensureLocalBinClaude(home, mockBin); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	symlinkPath := filepath.Join(home, ".local", "bin", "claude")
	info, err := os.Lstat(symlinkPath)
	if err != nil {
		t.Fatalf("symlink not created: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("expected a symlink at %s, got regular file", symlinkPath)
	}
	target, err := os.Readlink(symlinkPath)
	if err != nil {
		t.Fatal(err)
	}
	if target != mockBin {
		t.Errorf("symlink target = %q, want %q", target, mockBin)
	}
}

func TestEnsureLocalBinClaude_NoopWhenExists(t *testing.T) {
	home := t.TempDir()

	// Pre-create ~/.local/bin/claude as a regular file
	localBinClaude := filepath.Join(home, ".local", "bin", "claude")
	if err := os.MkdirAll(filepath.Dir(localBinClaude), 0755); err != nil {
		t.Fatal(err)
	}
	originalContent := []byte("original")
	if err := os.WriteFile(localBinClaude, originalContent, 0755); err != nil {
		t.Fatal(err)
	}

	if err := ensureLocalBinClaude(home, "/some/other/claude"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(localBinClaude)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(originalContent) {
		t.Errorf("existing file was modified; want %q, got %q", originalContent, got)
	}
}

func TestEnsureLocalBinClaude_AppendPathToRC(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SHELL", "/bin/zsh")

	// Create a mock binary target
	mockBin := filepath.Join(home, "usr", "bin", "claude")
	if err := os.MkdirAll(filepath.Dir(mockBin), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mockBin, []byte("#!/bin/bash"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create a .zshrc without any .local/bin entry
	zshrc := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(zshrc, []byte("# existing zshrc\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := ensureLocalBinClaude(home, mockBin); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(zshrc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), ".local/bin") {
		t.Errorf(".zshrc was not updated with .local/bin PATH entry; got:\n%s", content)
	}
}

func TestPlatformMCPServers_ContainsLibsql(t *testing.T) {
	home := t.TempDir()
	servers := platformMCPServers(home)

	if _, ok := servers["mcp-memory-libsql"]; !ok {
		t.Error("expected mcp-memory-libsql in platform MCP servers")
	}
	if _, ok := servers["engram"]; ok {
		t.Error("engram should not be in default platform MCP servers")
	}

	cfg, ok := servers["mcp-memory-libsql"].(map[string]interface{})
	if !ok {
		t.Fatal("expected mcp-memory-libsql config to be a map")
	}
	if cfg["command"] != "npx" {
		t.Errorf("expected command=npx, got %v", cfg["command"])
	}
	env, ok := cfg["env"].(map[string]interface{})
	if !ok {
		t.Fatal("expected env to be a map")
	}
	libsqlURL, ok := env["LIBSQL_URL"].(string)
	if !ok || libsqlURL == "" {
		t.Error("expected non-empty LIBSQL_URL in env")
	}
	if !strings.HasPrefix(libsqlURL, "file:") {
		t.Errorf("expected LIBSQL_URL to start with 'file:', got %q", libsqlURL)
	}
}

func TestRemoveUserMCPServers(t *testing.T) {
	config := map[string]interface{}{
		"primaryApiKey": "sk-test",
		"mcpServers": map[string]interface{}{
			"engram":            map[string]interface{}{"command": "engram"},
			"mcp-memory-libsql": map[string]interface{}{"command": "npx"},
			"other":             map[string]interface{}{"command": "other"},
		},
	}

	result := RemoveUserMCPServers(config, []string{"engram", "mcp-memory-libsql", "memory"})

	mcp, ok := result["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatal("expected mcpServers map")
	}
	if _, found := mcp["engram"]; found {
		t.Error("engram should have been removed")
	}
	if _, found := mcp["mcp-memory-libsql"]; found {
		t.Error("mcp-memory-libsql should have been removed")
	}
	if _, found := mcp["other"]; !found {
		t.Error("other server should have been preserved")
	}
	if result["primaryApiKey"] != "sk-test" {
		t.Error("other config keys should be preserved")
	}
}

func TestRemoveUserMCPServers_NoMcpServers(t *testing.T) {
	config := map[string]interface{}{"primaryApiKey": "sk-test"}
	result := RemoveUserMCPServers(config, []string{"engram"})
	if result["primaryApiKey"] != "sk-test" {
		t.Error("config keys should be preserved when mcpServers is absent")
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
