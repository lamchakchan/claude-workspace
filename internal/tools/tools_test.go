package tools

import (
	"strings"
	"testing"
)

func TestRegistryRequired(t *testing.T) {
	required := Required()
	if len(required) != 3 {
		t.Fatalf("expected 3 required tools, got %d", len(required))
	}
	names := make(map[string]bool)
	for _, tool := range required {
		names[tool.Name] = true
		if !tool.Required {
			t.Errorf("required tool %s should have Required=true", tool.Name)
		}
	}
	for _, name := range []string{"claude", "node", "engram"} {
		if !names[name] {
			t.Errorf("expected required tool %s to be in registry", name)
		}
	}
}

func TestRegistryOptional(t *testing.T) {
	optional := Optional()
	if len(optional) != 4 {
		t.Fatalf("expected 4 optional tools, got %d", len(optional))
	}
	names := make(map[string]bool)
	for _, tool := range optional {
		names[tool.Name] = true
		if tool.Required {
			t.Errorf("optional tool %s should not have Required=true", tool.Name)
		}
	}
	for _, name := range []string{"shellcheck", "jq", "prettier", "tmux"} {
		if !names[name] {
			t.Errorf("expected optional tool %s to be in registry", name)
		}
	}
}

func TestRegistryAll(t *testing.T) {
	all := All()
	if len(all) != 7 {
		t.Fatalf("expected 7 total tools, got %d", len(all))
	}
}

func TestToolIsInstalled_Default(t *testing.T) {
	// "go" should exist in PATH during tests
	tool := Tool{Name: "go"}
	if !tool.IsInstalled() {
		t.Error("expected 'go' to be found in PATH")
	}

	// A nonsense binary should not exist
	tool = Tool{Name: "nonexistent-binary-xyz-12345"}
	if tool.IsInstalled() {
		t.Error("expected nonexistent binary to not be found")
	}
}

func TestToolIsInstalled_CustomCheckFn(t *testing.T) {
	tool := Tool{
		Name:    "anything",
		CheckFn: func() bool { return true },
	}
	if !tool.IsInstalled() {
		t.Error("expected custom CheckFn returning true to report installed")
	}

	tool.CheckFn = func() bool { return false }
	if tool.IsInstalled() {
		t.Error("expected custom CheckFn returning false to report not installed")
	}
}

func TestToolInstallHint_WithInstallCmd(t *testing.T) {
	tool := Tool{
		Name:       "prettier",
		InstallCmd: "npm install -g prettier",
	}
	hint := tool.InstallHint()
	if hint != "npm install -g prettier" {
		t.Errorf("expected explicit InstallCmd, got %q", hint)
	}
}

func TestToolInstallHint_AutoDetect(t *testing.T) {
	// Tool with no InstallCmd uses auto-detection
	tool := Tool{Name: "jq"}
	hint := tool.InstallHint()
	// Should return something non-empty regardless of platform
	if hint == "" {
		t.Error("expected non-empty auto-detected install hint")
	}
	if !strings.Contains(hint, "jq") {
		t.Errorf("expected install hint to contain tool name, got %q", hint)
	}
}

func TestPrettierHasCustomInstallFn(t *testing.T) {
	p := Prettier()
	if p.InstallFn == nil {
		t.Error("prettier should have a custom InstallFn")
	}
	if p.InstallCmd != "npm install -g prettier" {
		t.Errorf("unexpected InstallCmd: %s", p.InstallCmd)
	}
}

func TestClaudeHasCustomInstallFn(t *testing.T) {
	c := Claude()
	if c.InstallFn == nil {
		t.Error("claude should have a custom InstallFn")
	}
	if c.VersionFn == nil {
		t.Error("claude should have a VersionFn")
	}
	if c.InstallCmd != ClaudeInstallCmd {
		t.Errorf("unexpected InstallCmd: %s", c.InstallCmd)
	}
}

func TestNodeHasCustomFns(t *testing.T) {
	n := Node()
	if n.InstallFn == nil {
		t.Error("node should have a custom InstallFn")
	}
	if n.CheckFn == nil {
		t.Error("node should have a custom CheckFn")
	}
	if n.VersionFn == nil {
		t.Error("node should have a VersionFn")
	}
	if !n.Required {
		t.Error("node should have Required=true")
	}
	if n.Purpose == "" {
		t.Error("node should have a Purpose")
	}
}

func TestEngramHasCustomFns(t *testing.T) {
	e := Engram()
	if e.InstallFn == nil {
		t.Error("engram should have a custom InstallFn")
	}
	if e.CheckFn == nil {
		t.Error("engram should have a custom CheckFn")
	}
	if e.VersionFn == nil {
		t.Error("engram should have a VersionFn")
	}
	if !e.Required {
		t.Error("engram should have Required=true")
	}
	if e.Purpose == "" {
		t.Error("engram should have a Purpose")
	}
}

func TestSimpleToolsHaveNoCustomFns(t *testing.T) {
	for _, fn := range []func() Tool{Shellcheck, JQ, Tmux} {
		tool := fn()
		if tool.InstallFn != nil {
			t.Errorf("%s should not have a custom InstallFn", tool.Name)
		}
		if tool.CheckFn != nil {
			t.Errorf("%s should not have a custom CheckFn", tool.Name)
		}
		if tool.Purpose == "" {
			t.Errorf("%s should have a Purpose", tool.Name)
		}
	}
}
