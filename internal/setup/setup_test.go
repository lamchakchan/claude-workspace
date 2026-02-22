package setup

import (
	"reflect"
	"sort"
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
