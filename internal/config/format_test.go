package config

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestFormatView(t *testing.T) {
	reg := &Registry{
		keys: []ConfigKey{
			{Key: "model", Category: CatCore, Type: TypeString, Default: "claude-sonnet-4-6", Description: "Override model"},
			{Key: "sandbox.enabled", Category: CatSandbox, Type: TypeBool, Default: "false", Description: "Enable sandboxing"},
		},
		lookup:     make(map[string]*ConfigKey),
		categories: []Category{CatCore, CatSandbox},
	}
	for i := range reg.keys {
		reg.lookup[reg.keys[i].Key] = &reg.keys[i]
	}

	snap := &ConfigSnapshot{
		Values: map[string]*ConfigValue{
			"model": {
				Key:            "model",
				EffectiveValue: "claude-opus-4-6",
				Source:         ScopeUser,
				IsDefault:      false,
				LayerValues:    map[ConfigScope]interface{}{ScopeUser: "claude-opus-4-6"},
			},
			"sandbox.enabled": {
				Key:       "sandbox.enabled",
				Source:    ScopeDefault,
				IsDefault: true,
			},
		},
		Timestamp: time.Now(),
	}

	var buf bytes.Buffer
	if err := FormatView(&buf, snap, reg); err != nil {
		t.Fatalf("FormatView: %v", err)
	}
	out := buf.String()

	tests := []struct {
		name    string
		want    string
		present bool
	}{
		{"banner", "Claude Code Configuration", true},
		{"core category", "Core", true},
		{"sandbox category", "Sandbox", true},
		{"model key", "model", true},
		{"model value", "claude-opus-4-6", true},
		{"sandbox key", "sandbox.enabled", true},
		{"user badge", "[USR]", true},
		{"default badge", "[DEF]", true},
	}
	for _, tt := range tests {
		if strings.Contains(out, tt.want) != tt.present {
			t.Errorf("%s: contains(%q) = %v, want %v", tt.name, tt.want, !tt.present, tt.present)
		}
	}
}

func TestFormatGet_Found(t *testing.T) {
	reg := &Registry{
		keys: []ConfigKey{
			{Key: "model", Category: CatCore, Type: TypeString, Default: "claude-sonnet-4-6", Description: "Override the default model"},
		},
		lookup:     make(map[string]*ConfigKey),
		categories: []Category{CatCore},
	}
	reg.lookup["model"] = &reg.keys[0]

	snap := &ConfigSnapshot{
		Values: map[string]*ConfigValue{
			"model": {
				Key:            "model",
				EffectiveValue: "claude-opus-4-6",
				Source:         ScopeProject,
				IsDefault:      false,
				LayerValues:    map[ConfigScope]interface{}{ScopeProject: "claude-opus-4-6"},
			},
		},
		Timestamp: time.Now(),
	}

	var buf bytes.Buffer
	if err := FormatGet(&buf, "model", snap, reg); err != nil {
		t.Fatalf("FormatGet: %v", err)
	}
	out := buf.String()

	for _, want := range []string{"model", "string", "Override the default model", "claude-opus-4-6", "[PRJ]"} {
		if !strings.Contains(out, want) {
			t.Errorf("FormatGet output missing %q", want)
		}
	}
}

func TestFormatGet_NotFound(t *testing.T) {
	reg := &Registry{
		keys:       nil,
		lookup:     make(map[string]*ConfigKey),
		categories: nil,
	}
	snap := &ConfigSnapshot{
		Values:    make(map[string]*ConfigValue),
		Timestamp: time.Now(),
	}

	var buf bytes.Buffer
	err := FormatGet(&buf, "nonexistent.key", snap, reg)
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
	if !strings.Contains(err.Error(), "unknown config key") {
		t.Errorf("error = %q, want it to contain 'unknown config key'", err.Error())
	}
}

func TestFormatGet_AllLayers(t *testing.T) {
	reg := &Registry{
		keys: []ConfigKey{
			{Key: "model", Category: CatCore, Type: TypeString, Description: "Override model"},
		},
		lookup:     make(map[string]*ConfigKey),
		categories: []Category{CatCore},
	}
	reg.lookup["model"] = &reg.keys[0]

	snap := &ConfigSnapshot{
		Values: map[string]*ConfigValue{
			"model": {
				Key:            "model",
				EffectiveValue: "claude-opus-4-6",
				Source:         ScopeLocal,
				IsDefault:      false,
				LayerValues: map[ConfigScope]interface{}{
					ScopeUser:    "claude-sonnet-4-6",
					ScopeProject: "claude-haiku-4-5-20251001",
					ScopeLocal:   "claude-opus-4-6",
				},
			},
		},
		Timestamp: time.Now(),
	}

	var buf bytes.Buffer
	if err := FormatGet(&buf, "model", snap, reg); err != nil {
		t.Fatalf("FormatGet: %v", err)
	}
	out := buf.String()

	for _, want := range []string{
		"[USR]", "claude-sonnet-4-6",
		"[PRJ]", "claude-haiku-4-5-20251001",
		"[LOC]", "claude-opus-4-6",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("FormatGet output missing %q", want)
		}
	}
}

func TestFormatGet_DefaultOnly(t *testing.T) {
	reg := &Registry{
		keys: []ConfigKey{
			{Key: "cleanupPeriodDays", Category: CatCore, Type: TypeInt, Default: "30", Description: "Days to retain"},
		},
		lookup:     make(map[string]*ConfigKey),
		categories: []Category{CatCore},
	}
	reg.lookup["cleanupPeriodDays"] = &reg.keys[0]

	snap := &ConfigSnapshot{
		Values: map[string]*ConfigValue{
			"cleanupPeriodDays": {
				Key:       "cleanupPeriodDays",
				Source:    ScopeDefault,
				IsDefault: true,
			},
		},
		Timestamp: time.Now(),
	}

	var buf bytes.Buffer
	if err := FormatGet(&buf, "cleanupPeriodDays", snap, reg); err != nil {
		t.Fatalf("FormatGet: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "[DEF]") {
		t.Error("expected [DEF] badge for default-only key")
	}
	if !strings.Contains(out, "30") {
		t.Error("expected default value '30' in output")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"this is a very long string", 10, "this is..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc"},
	}
	for _, tt := range tests {
		got := truncate(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		input interface{}
		want  string
	}{
		{nil, "(none)"},
		{"hello", "hello"},
		{true, "true"},
		{false, "false"},
		{42, "42"},
		{3.14, "3.14"},
		{[]interface{}{"a", "b"}, "[a, b]"},
	}
	for _, tt := range tests {
		got := formatValue(tt.input)
		if got != tt.want {
			t.Errorf("formatValue(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestScopeBadge(t *testing.T) {
	tests := []struct {
		scope ConfigScope
		want  string
	}{
		{ScopeManaged, "[MGD]"},
		{ScopeUser, "[USR]"},
		{ScopeProject, "[PRJ]"},
		{ScopeLocal, "[LOC]"},
		{ScopeEnv, "[ENV]"},
		{ScopeDefault, "[DEF]"},
	}
	for _, tt := range tests {
		got := scopeBadge(tt.scope)
		if !strings.Contains(got, tt.want) {
			t.Errorf("scopeBadge(%q) = %q, want it to contain %q", tt.scope, got, tt.want)
		}
	}
}
