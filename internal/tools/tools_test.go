package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRegistryRequired(t *testing.T) {
	required := Required()
	if len(required) != 1 {
		t.Fatalf("expected 1 required tool, got %d", len(required))
	}
	if required[0].Name != "claude" {
		t.Errorf("expected required tool to be claude, got %s", required[0].Name)
	}
	if !required[0].Required {
		t.Error("claude should have Required=true")
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
	if len(all) != 5 {
		t.Fatalf("expected 5 total tools, got %d", len(all))
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

// Tests moved from setup_test.go

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
