package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteSettingsValue_NewKey(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	cwd := filepath.Join(tmp, "project")

	if err := WriteSettingsValue("model", testOpusModel, ScopeUser, home, cwd); err != nil {
		t.Fatalf("WriteSettingsValue: %v", err)
	}

	path := filepath.Join(home, ".claude", "settings.json")
	var got map[string]interface{}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got["model"] != testOpusModel {
		t.Errorf("model = %v, want %s", got["model"], testOpusModel)
	}
}

func TestWriteSettingsValue_NestedKey(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	cwd := filepath.Join(tmp, "project")

	if err := WriteSettingsValue("sandbox.enabled", "true", ScopeProject, home, cwd); err != nil {
		t.Fatalf("WriteSettingsValue: %v", err)
	}

	path := filepath.Join(cwd, ".claude", "settings.json")
	var got map[string]interface{}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	sandbox, ok := got["sandbox"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected nested 'sandbox' object, got %T", got["sandbox"])
	}
	if sandbox["enabled"] != true {
		t.Errorf("sandbox.enabled = %v, want true", sandbox["enabled"])
	}
}

func TestWriteSettingsValue_OverwriteExisting(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	cwd := filepath.Join(tmp, "project")

	dir := filepath.Join(cwd, ".claude")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	initial := `{"model": "claude-sonnet-4-6", "cleanupPeriodDays": 30}`
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}

	if err := WriteSettingsValue("model", "claude-opus-4-6", ScopeProject, home, cwd); err != nil {
		t.Fatalf("WriteSettingsValue: %v", err)
	}

	var got map[string]interface{}
	data, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got["model"] != "claude-opus-4-6" {
		t.Errorf("model = %v, want claude-opus-4-6", got["model"])
	}
	// cleanupPeriodDays should be preserved (as float64 from JSON)
	if got["cleanupPeriodDays"] == nil {
		t.Error("cleanupPeriodDays was lost after overwrite")
	}
}

func TestWriteSettingsValue_ManagedScope(t *testing.T) {
	err := WriteSettingsValue("model", "claude-opus-4-6", ScopeManaged, "/home", "/project")
	if err == nil {
		t.Fatal("expected error for managed scope write")
	}
}

func TestWriteSettingsValue_EnvScope(t *testing.T) {
	err := WriteSettingsValue("model", "claude-opus-4-6", ScopeEnv, "/home", "/project")
	if err == nil {
		t.Fatal("expected error for env scope write")
	}
}

func TestWriteSettingsValue_DefaultScope(t *testing.T) {
	err := WriteSettingsValue("model", "claude-opus-4-6", ScopeDefault, "/home", "/project")
	if err == nil {
		t.Fatal("expected error for default scope write")
	}
}

func TestWriteSettingsValue_LocalScope(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	cwd := filepath.Join(tmp, "project")

	if err := WriteSettingsValue("model", "claude-opus-4-6", ScopeLocal, home, cwd); err != nil {
		t.Fatalf("WriteSettingsValue: %v", err)
	}

	path := filepath.Join(cwd, ".claude", "settings.local.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("expected settings.local.json to be created")
	}
}

func TestParseWriteValue_Bool(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{"true", true},
		{"false", false},
		{"1", true},
		{"0", false},
	}
	for _, tt := range tests {
		got, err := parseWriteValue("sandbox.enabled", tt.value)
		if err != nil {
			t.Errorf("parseWriteValue(%q): %v", tt.value, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseWriteValue(%q) = %v, want %v", tt.value, got, tt.want)
		}
	}
}

func TestParseWriteValue_Int(t *testing.T) {
	tests := []struct {
		value   string
		want    int
		wantErr bool
	}{
		{"42", 42, false},
		{"0", 0, false},
		{"abc", 0, true},
	}
	for _, tt := range tests {
		got, err := parseWriteValue("cleanupPeriodDays", tt.value)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseWriteValue(%q): expected error", tt.value)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseWriteValue(%q): %v", tt.value, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseWriteValue(%q) = %v, want %v", tt.value, got, tt.want)
		}
	}
}

func TestParseWriteValue_Enum(t *testing.T) {
	tests := []struct {
		value   string
		wantErr bool
	}{
		{"low", false},
		{"medium", false},
		{"high", false},
		{"extreme", true},
	}
	for _, tt := range tests {
		got, err := parseWriteValue("effortLevel", tt.value)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseWriteValue(%q): expected error for invalid enum", tt.value)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseWriteValue(%q): %v", tt.value, err)
			continue
		}
		if got != tt.value {
			t.Errorf("parseWriteValue(%q) = %v, want %v", tt.value, got, tt.value)
		}
	}
}

func TestParseWriteValue_Array(t *testing.T) {
	got, err := parseWriteValue("permissions.allow", "Bash(*),Read(*),Write(*)")
	if err != nil {
		t.Fatalf("parseWriteValue: %v", err)
	}
	arr, ok := got.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", got)
	}
	if len(arr) != 3 {
		t.Fatalf("len = %d, want 3", len(arr))
	}
	want := []string{"Bash(*)", "Read(*)", "Write(*)"}
	for i, w := range want {
		if arr[i] != w {
			t.Errorf("arr[%d] = %v, want %q", i, arr[i], w)
		}
	}
}

func TestParseWriteValue_Object(t *testing.T) {
	got, err := parseWriteValue("env", `{"NODE_ENV": "development"}`)
	if err != nil {
		t.Fatalf("parseWriteValue: %v", err)
	}
	obj, ok := got.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map[string]interface{}, got %T", got)
	}
	if obj["NODE_ENV"] != "development" {
		t.Errorf("NODE_ENV = %v, want development", obj["NODE_ENV"])
	}
}

func TestParseWriteValue_Unknown(t *testing.T) {
	tests := []struct {
		key   string
		value string
		want  interface{}
	}{
		{"unknown.bool", "true", true},
		{"unknown.int", "42", 42},
		{"unknown.string", "hello", "hello"},
	}
	for _, tt := range tests {
		got, err := parseWriteValue(tt.key, tt.value)
		if err != nil {
			t.Errorf("parseWriteValue(%q, %q): %v", tt.key, tt.value, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseWriteValue(%q, %q) = %v (%T), want %v (%T)", tt.key, tt.value, got, got, tt.want, tt.want)
		}
	}
}

func TestAppendToArray(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	cwd := filepath.Join(tmp, "project")

	// Append to nonexistent key creates array
	if err := AppendToArray("permissions.allow", "Bash(*)", ScopeUser, home, cwd); err != nil {
		t.Fatalf("AppendToArray: %v", err)
	}

	// Append another item
	if err := AppendToArray("permissions.allow", "Read(*)", ScopeUser, home, cwd); err != nil {
		t.Fatalf("AppendToArray second: %v", err)
	}

	path := filepath.Join(home, ".claude", "settings.json")
	var got map[string]interface{}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}

	perms, ok := got["permissions"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected nested permissions object, got %T", got["permissions"])
	}
	arr, ok := perms["allow"].([]interface{})
	if !ok {
		t.Fatalf("expected allow array, got %T", perms["allow"])
	}
	if len(arr) != 2 {
		t.Fatalf("len = %d, want 2", len(arr))
	}
	if arr[0] != "Bash(*)" || arr[1] != "Read(*)" {
		t.Errorf("arr = %v, want [Bash(*), Read(*)]", arr)
	}
}

func TestRemoveFromArray(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	cwd := filepath.Join(tmp, "project")

	// Set up initial array
	dir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	initial := `{"permissions": {"allow": ["Bash(*)", "Read(*)", "Write(*)"]}}`
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}

	if err := RemoveFromArray("permissions.allow", "Read(*)", ScopeUser, home, cwd); err != nil {
		t.Fatalf("RemoveFromArray: %v", err)
	}

	var got map[string]interface{}
	data, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}

	perms := got["permissions"].(map[string]interface{})
	arr := perms["allow"].([]interface{})
	if len(arr) != 2 {
		t.Fatalf("len = %d, want 2 after removal", len(arr))
	}
	for _, item := range arr {
		if item == "Read(*)" {
			t.Error("Read(*) should have been removed")
		}
	}
}

func TestRemoveFromArray_NotFound(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	cwd := filepath.Join(tmp, "project")

	dir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	initial := `{"permissions": {"allow": ["Bash(*)"]}}`
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}

	// Removing a nonexistent item should be a no-op
	if err := RemoveFromArray("permissions.allow", "NotHere", ScopeUser, home, cwd); err != nil {
		t.Fatalf("RemoveFromArray: %v", err)
	}

	var got map[string]interface{}
	data, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}

	perms := got["permissions"].(map[string]interface{})
	arr := perms["allow"].([]interface{})
	if len(arr) != 1 {
		t.Errorf("len = %d, want 1 (no change)", len(arr))
	}
}

func TestSetNestedValue(t *testing.T) {
	tests := []struct {
		name    string
		dotPath string
		val     interface{}
		check   func(map[string]interface{}) bool
	}{
		{
			name:    "flat key",
			dotPath: "model",
			val:     "claude-opus-4-6",
			check: func(m map[string]interface{}) bool {
				return m["model"] == "claude-opus-4-6"
			},
		},
		{
			name:    "two-level nested",
			dotPath: "sandbox.enabled",
			val:     true,
			check: func(m map[string]interface{}) bool {
				sb, ok := m["sandbox"].(map[string]interface{})
				return ok && sb["enabled"] == true
			},
		},
		{
			name:    "three-level nested",
			dotPath: "sandbox.network.allowedDomains",
			val:     []string{"*.example.com"},
			check: func(m map[string]interface{}) bool {
				sb, ok := m["sandbox"].(map[string]interface{})
				if !ok {
					return false
				}
				net, ok := sb["network"].(map[string]interface{})
				return ok && net["allowedDomains"] != nil
			},
		},
	}
	for _, tt := range tests {
		root := make(map[string]interface{})
		setNestedValue(root, tt.dotPath, tt.val)
		if !tt.check(root) {
			t.Errorf("%s: check failed on result %v", tt.name, root)
		}
	}
}
