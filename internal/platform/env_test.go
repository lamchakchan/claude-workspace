package platform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	shellZsh  = "zsh"
	shellBash = "bash"
	shellFish = "fish"
)

func TestDetectShellRC_ZshFromEnv(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")
	home := t.TempDir()

	rcPath, shellName := DetectShellRC(home)

	if shellName != shellZsh {
		t.Errorf("expected shellName=%s, got %s", shellZsh, shellName)
	}
	if filepath.Base(rcPath) != ".zshrc" {
		t.Errorf("expected .zshrc, got %s", rcPath)
	}
}

func TestDetectShellRC_BashFromEnv(t *testing.T) {
	t.Setenv("SHELL", "/usr/bin/bash")
	home := t.TempDir()

	rcPath, shellName := DetectShellRC(home)

	if shellName != shellBash {
		t.Errorf("expected shellName=%s, got %s", shellBash, shellName)
	}
	if filepath.Base(rcPath) != ".bashrc" {
		t.Errorf("expected .bashrc, got %s", rcPath)
	}
}

func TestDetectShellRC_FishFromEnv(t *testing.T) {
	t.Setenv("SHELL", "/usr/bin/fish")
	home := t.TempDir()

	rcPath, shellName := DetectShellRC(home)

	if shellName != shellFish {
		t.Errorf("expected shellName=%s, got %s", shellFish, shellName)
	}
	if !strings.HasSuffix(rcPath, filepath.Join(".config", "fish", "config.fish")) {
		t.Errorf("expected config.fish path, got %s", rcPath)
	}
}

func TestDetectShellRC_FallbackFileExists(t *testing.T) {
	t.Setenv("SHELL", "")
	home := t.TempDir()

	// Create .zshrc so file-existence fallback triggers
	_ = os.WriteFile(filepath.Join(home, ".zshrc"), []byte(""), 0644)

	rcPath, shellName := DetectShellRC(home)

	if shellName != shellZsh {
		t.Errorf("expected shellName=%s from fallback, got %s", shellZsh, shellName)
	}
	if filepath.Base(rcPath) != ".zshrc" {
		t.Errorf("expected .zshrc, got %s", rcPath)
	}
}

func TestDetectShellRC_FallbackDefault(t *testing.T) {
	t.Setenv("SHELL", "")
	home := t.TempDir()

	rcPath, shellName := DetectShellRC(home)

	if shellName != shellBash {
		t.Errorf("expected shellName=%s as default, got %s", shellBash, shellName)
	}
	if filepath.Base(rcPath) != ".bashrc" {
		t.Errorf("expected .bashrc, got %s", rcPath)
	}
}

func TestAppendPathToRC_AddsWhenAbsent(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".bashrc")
	_ = os.WriteFile(rcPath, []byte("# existing content\n"), 0644)

	modified, err := AppendPathToRC(home, shellBash, rcPath)
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
	_ = os.WriteFile(rcPath, []byte("export PATH=\"$HOME/.local/bin:$PATH\"\n"), 0644)

	modified, err := AppendPathToRC(home, shellBash, rcPath)
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
	// Don't create the file — AppendPathToRC should create it

	modified, err := AppendPathToRC(home, shellBash, rcPath)
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

func TestAsdfDataDir_FromEnv(t *testing.T) {
	t.Setenv("ASDF_DATA_DIR", "/custom/asdf")
	dir := AsdfDataDir()
	if dir != "/custom/asdf" {
		t.Errorf("expected /custom/asdf, got %s", dir)
	}
}

func TestAsdfDataDir_Default(t *testing.T) {
	t.Setenv("ASDF_DATA_DIR", "")
	dir := AsdfDataDir()
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".asdf")
	if dir != expected {
		t.Errorf("expected %s, got %s", expected, dir)
	}
}
