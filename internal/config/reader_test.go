package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const (
	testOpusModel = "claude-opus-4-6"
	testNotFound  = "(not found)"
)

// writeJSON writes v as JSON to path, creating parent dirs as needed.
func writeJSON(t *testing.T, path string, v interface{}) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestFlattenJSON_Flat(t *testing.T) {
	raw := map[string]json.RawMessage{
		"model": json.RawMessage(`"claude-opus-4-6"`),
	}
	flat := flattenJSON("", raw)
	if v, ok := flat["model"]; !ok {
		t.Error("expected key 'model'")
	} else {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			t.Fatal(err)
		}
		if s != testOpusModel {
			t.Errorf("model = %q, want %q", s, testOpusModel)
		}
	}
}

func TestFlattenJSON_Nested(t *testing.T) {
	raw := map[string]json.RawMessage{
		"sandbox": json.RawMessage(`{"enabled": true, "autoAllowBashIfSandboxed": false}`),
	}
	flat := flattenJSON("", raw)

	if _, ok := flat["sandbox"]; !ok {
		t.Error("expected parent key 'sandbox'")
	}
	if _, ok := flat["sandbox.enabled"]; !ok {
		t.Error("expected flattened key 'sandbox.enabled'")
	}
	if _, ok := flat["sandbox.autoAllowBashIfSandboxed"]; !ok {
		t.Error("expected flattened key 'sandbox.autoAllowBashIfSandboxed'")
	}
}

func TestFlattenJSON_Array_NotDescended(t *testing.T) {
	raw := map[string]json.RawMessage{
		"permissions": json.RawMessage(`{"allow": ["Bash(*)", "Read(*)"]}`),
	}
	flat := flattenJSON("", raw)
	if _, ok := flat["permissions.allow"]; !ok {
		t.Error("expected key 'permissions.allow'")
	}
	// Array elements should NOT be further flattened
	if _, ok := flat["permissions.allow.0"]; ok {
		t.Error("array elements should not be flattened into indexed keys")
	}
}

func TestReadAllSettings_UserLayer(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	cwd := filepath.Join(tmp, "project")
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(cwd, ".claude"), 0755); err != nil {
		t.Fatal(err)
	}

	writeJSON(t, filepath.Join(home, ".claude", "settings.json"), map[string]interface{}{
		"model":             testOpusModel,
		"cleanupPeriodDays": 60,
	})

	layers, err := readAllSettings(home, cwd)
	if err != nil {
		t.Fatalf("readAllSettings: %v", err)
	}

	userLayer, ok := layers[ScopeUser]
	if !ok {
		t.Fatal("expected user layer to be present")
	}
	if _, ok := userLayer["model"]; !ok {
		t.Error("expected 'model' in user layer")
	}
}

func TestReadAllSettings_ProjectOverridesUser(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	cwd := filepath.Join(tmp, "project")

	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(cwd, ".claude"), 0755); err != nil {
		t.Fatal(err)
	}

	writeJSON(t, filepath.Join(home, ".claude", "settings.json"), map[string]interface{}{
		"model": "claude-sonnet-4-6",
	})
	writeJSON(t, filepath.Join(cwd, ".claude", "settings.json"), map[string]interface{}{
		"model": testOpusModel,
	})

	layers, err := readAllSettings(home, cwd)
	if err != nil {
		t.Fatalf("readAllSettings: %v", err)
	}

	cv := resolveSettingsKey("model", layers, "")
	if cv.EffectiveValue != testOpusModel {
		t.Errorf("effective model = %v, want %s", cv.EffectiveValue, testOpusModel)
	}
	if cv.Source != ScopeProject {
		t.Errorf("source = %q, want ScopeProject", cv.Source)
	}
}

func TestResolveSettingsKey_UsesDefault(t *testing.T) {
	layers := map[ConfigScope]map[string]json.RawMessage{}
	cv := resolveSettingsKey("cleanupPeriodDays", layers, "30")
	if cv.EffectiveValue != "30" {
		t.Errorf("expected default '30', got %v", cv.EffectiveValue)
	}
	if !cv.IsDefault {
		t.Error("expected IsDefault=true when no layer sets the key")
	}
	if cv.Source != ScopeDefault {
		t.Errorf("source = %q, want ScopeDefault", cv.Source)
	}
}

func TestResolveSettingsKey_LocalBeatsProject(t *testing.T) {
	layers := map[ConfigScope]map[string]json.RawMessage{
		ScopeProject: {"model": json.RawMessage(`"claude-sonnet-4-6"`)},
		ScopeLocal:   {"model": json.RawMessage(`"claude-haiku-4-5-20251001"`)},
	}
	cv := resolveSettingsKey("model", layers, "")
	// Local should win over project
	if cv.EffectiveValue != "claude-haiku-4-5-20251001" {
		t.Errorf("effective model = %v, want claude-haiku-4-5-20251001", cv.EffectiveValue)
	}
	if cv.Source != ScopeLocal {
		t.Errorf("source = %q, want ScopeLocal", cv.Source)
	}
}

func TestReadEnvVars_OSEnv(t *testing.T) {
	t.Setenv("ANTHROPIC_MODEL", "claude-test-model")
	layers := map[ConfigScope]map[string]json.RawMessage{}

	vars := readEnvVars(layers)
	cv, ok := vars["ANTHROPIC_MODEL"]
	if !ok {
		t.Fatal("ANTHROPIC_MODEL not found in env vars output")
	}
	if cv.EffectiveValue != "claude-test-model" {
		t.Errorf("effective value = %v, want claude-test-model", cv.EffectiveValue)
	}
	if cv.Source != ScopeEnv {
		t.Errorf("source = %q, want ScopeEnv", cv.Source)
	}
}

func TestReadEnvVars_SettingsEnvBlock(t *testing.T) {
	// ANTHROPIC_MODEL set in settings.json env block, not OS env
	t.Setenv("ANTHROPIC_MODEL", "") // ensure OS env is clear
	layers := map[ConfigScope]map[string]json.RawMessage{
		ScopeUser: {
			"env": json.RawMessage(`{"ANTHROPIC_MODEL": "from-settings-env"}`),
		},
	}
	vars := readEnvVars(layers)
	cv, ok := vars["ANTHROPIC_MODEL"]
	if !ok {
		t.Fatal("ANTHROPIC_MODEL not found")
	}
	if cv.EffectiveValue != "from-settings-env" {
		t.Errorf("effective value = %v, want from-settings-env", cv.EffectiveValue)
	}
}

func TestReadFileBased_CLAUDE_md(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	cwd := filepath.Join(tmp, "project")

	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(cwd, ".claude"), 0755); err != nil {
		t.Fatal(err)
	}

	// Write a project CLAUDE.md
	claudeMd := "# Instructions\nDo this.\nDo that.\n"
	if err := os.WriteFile(filepath.Join(cwd, ".claude", "CLAUDE.md"), []byte(claudeMd), 0644); err != nil {
		t.Fatal(err)
	}

	fileCfg := readFileBased(home, cwd)

	cv, ok := fileCfg["file:claudemd.project"]
	if !ok {
		t.Fatal("file:claudemd.project not found")
	}
	if cv.EffectiveValue == testNotFound {
		t.Error("expected CLAUDE.md to be found")
	}

	// User CLAUDE.md should be not found (we didn't create it)
	cvUser, ok := fileCfg["file:claudemd.user"]
	if !ok {
		t.Fatal("file:claudemd.user not found in map")
	}
	if cvUser.EffectiveValue != testNotFound {
		t.Errorf("expected user CLAUDE.md to be not found, got %v", cvUser.EffectiveValue)
	}
}

func TestReadFileBased_Agents(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	cwd := filepath.Join(tmp, "project")

	agentsDir := filepath.Join(cwd, ".claude", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "explorer.md"), []byte("# Explorer"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "planner.md"), []byte("# Planner"), 0644); err != nil {
		t.Fatal(err)
	}

	fileCfg := readFileBased(home, cwd)

	cv, ok := fileCfg["file:agents.project"]
	if !ok {
		t.Fatal("file:agents.project not found")
	}
	if cv.IsDefault {
		t.Error("expected IsDefault=false when agents directory exists")
	}
}

func TestParseRawValue(t *testing.T) {
	tests := []struct {
		raw  string
		want interface{}
	}{
		{`"hello"`, "hello"},
		{`true`, true},
		{`false`, false},
		{`42`, 42},
		{`["a","b"]`, []interface{}{"a", "b"}},
	}
	for _, tt := range tests {
		got := parseRawValue(json.RawMessage(tt.raw))
		switch want := tt.want.(type) {
		case string:
			if got != want {
				t.Errorf("parseRawValue(%s) = %v, want %v", tt.raw, got, want)
			}
		case bool:
			if got != want {
				t.Errorf("parseRawValue(%s) = %v, want %v", tt.raw, got, want)
			}
		case int:
			if got != want {
				t.Errorf("parseRawValue(%s) = %v, want %v", tt.raw, got, want)
			}
		case []interface{}:
			arr, ok := got.([]interface{})
			if !ok {
				t.Errorf("parseRawValue(%s) = %T, want []interface{}", tt.raw, got)
			} else if len(arr) != len(want) {
				t.Errorf("parseRawValue(%s) len = %d, want %d", tt.raw, len(arr), len(want))
			}
		}
	}
}

func TestIsEnvVar(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{"ANTHROPIC_MODEL", true},
		{"CLAUDE_CODE_MAX_OUTPUT_TOKENS", true},
		{"model", false},
		{"sandbox.enabled", false},
		{"file:claudemd.user", false},
		{"$schema", false},
	}
	for _, tt := range tests {
		got := isEnvVar(tt.key)
		if got != tt.want {
			t.Errorf("isEnvVar(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}
}
