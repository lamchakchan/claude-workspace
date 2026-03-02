package config

import (
	"strings"
	"testing"
)

func TestRunTo_ViewSubcommand(t *testing.T) {
	var buf strings.Builder
	err := RunTo(&buf, []string{"view"})
	if err != nil {
		t.Fatalf("RunTo(view): %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Configuration") {
		t.Error("view output missing 'Configuration' banner")
	}
	// At least one category should appear
	if !strings.Contains(out, "Core") {
		t.Error("view output missing 'Core' category")
	}
}

func TestRunTo_GetSubcommand_Found(t *testing.T) {
	var buf strings.Builder
	err := RunTo(&buf, []string{"get", "model"})
	if err != nil {
		t.Fatalf("RunTo(get model): %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "model") {
		t.Error("get output missing key name 'model'")
	}
}

func TestRunTo_GetSubcommand_Missing(t *testing.T) {
	var buf strings.Builder
	err := RunTo(&buf, []string{"get"})
	if err == nil {
		t.Error("expected error for missing key argument")
	}
}

func TestRunTo_GetSubcommand_UnknownKey(t *testing.T) {
	var buf strings.Builder
	err := RunTo(&buf, []string{"get", "nonexistent.key.xyz"})
	if err == nil {
		t.Error("expected error for unknown key")
	}
	if !strings.Contains(err.Error(), "nonexistent.key.xyz") {
		t.Errorf("error should mention the unknown key, got: %v", err)
	}
}

func TestRunTo_UnknownSubcommand(t *testing.T) {
	var buf strings.Builder
	err := RunTo(&buf, []string{"frobnicate"})
	if err == nil {
		t.Error("expected error for unknown subcommand")
	}
	if !strings.Contains(err.Error(), "frobnicate") {
		t.Errorf("error should mention the subcommand, got: %v", err)
	}
}

func TestRunTo_NoArgs(t *testing.T) {
	var buf strings.Builder
	err := RunTo(&buf, []string{})
	if err != nil {
		t.Errorf("RunTo with no args should return nil (TUI mode), got: %v", err)
	}
}

func TestRunSet_InvalidScope(t *testing.T) {
	err := runSet([]string{"--scope", "managed", "model", "claude-opus-4-6"})
	if err == nil {
		t.Error("expected error for managed scope")
	}
	if !strings.Contains(err.Error(), "managed") {
		t.Errorf("error should mention 'managed', got: %v", err)
	}
}

func TestRunSet_MissingArgs(t *testing.T) {
	err := runSet([]string{"model"}) // only key, no value
	if err == nil {
		t.Error("expected error for missing value")
	}
}

func TestRunTo_DeleteSubcommand_MissingKey(t *testing.T) {
	var buf strings.Builder
	err := RunTo(&buf, []string{"delete"})
	if err == nil {
		t.Error("expected error for missing key argument")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("error should mention usage, got: %v", err)
	}
}

func TestRunTo_DeleteSubcommand_InvalidScope(t *testing.T) {
	err := runDelete([]string{"--scope", "managed", "model"})
	if err == nil {
		t.Error("expected error for managed scope")
	}
	if !strings.Contains(err.Error(), "managed") {
		t.Errorf("error should mention 'managed', got: %v", err)
	}
}

func TestRunTo_UnsetAlias(t *testing.T) {
	// "unset" is an alias for "delete"
	var buf strings.Builder
	err := RunTo(&buf, []string{"unset"})
	if err == nil {
		t.Error("expected error for missing key argument (unset alias)")
	}
}
