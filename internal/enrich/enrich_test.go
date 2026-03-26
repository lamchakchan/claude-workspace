package enrich

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_MissingProjectDir(t *testing.T) {
	err := Run("/nonexistent/path/to/project", nil)
	if err == nil {
		t.Fatal("Run() expected error for missing project dir")
	}
	if got := err.Error(); got != "project directory not found: /nonexistent/path/to/project" {
		t.Errorf("error = %q, want 'project directory not found: /nonexistent/path/to/project'", got)
	}
}

func TestRun_ScaffoldOnly(t *testing.T) {
	dir := t.TempDir()

	// Create a go.mod so scaffold detects Go
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)

	err := Run(dir, []string{"--scaffold-only"})
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	// Verify CLAUDE.md was created
	claudeMdPath := filepath.Join(dir, ".claude", "CLAUDE.md")
	data, err := os.ReadFile(claudeMdPath)
	if err != nil {
		t.Fatalf("CLAUDE.md not created: %v", err)
	}

	content := string(data)
	if content == "" {
		t.Error("CLAUDE.md should not be empty")
	}
}

func TestRun_ScaffoldOnlyExistingFile(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	_ = os.MkdirAll(claudeDir, 0755)

	// Write an existing CLAUDE.md
	existing := "# Existing content"
	_ = os.WriteFile(filepath.Join(claudeDir, "CLAUDE.md"), []byte(existing), 0644)

	err := Run(dir, []string{"--scaffold-only"})
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	// Verify existing content is preserved
	data, _ := os.ReadFile(filepath.Join(claudeDir, "CLAUDE.md"))
	if string(data) != existing {
		t.Errorf("existing CLAUDE.md was modified: got %q, want %q", string(data), existing)
	}
}

func TestRun_ScaffoldOnlyExistingFile_WritesToRules(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	_ = os.MkdirAll(claudeDir, 0755)

	// Write an existing CLAUDE.md
	existing := "# Existing content"
	_ = os.WriteFile(filepath.Join(claudeDir, "CLAUDE.md"), []byte(existing), 0644)

	// Create a go.mod so scaffold detects Go
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)

	err := Run(dir, []string{"--scaffold-only"})
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	// Existing CLAUDE.md should be preserved
	data, _ := os.ReadFile(filepath.Join(claudeDir, "CLAUDE.md"))
	if string(data) != existing {
		t.Errorf("existing CLAUDE.md was modified: got %q, want %q", string(data), existing)
	}

	// Scaffold should have been written to rules/platform.md
	rulesPath := filepath.Join(claudeDir, "rules", "platform.md")
	rulesData, err := os.ReadFile(rulesPath)
	if err != nil {
		t.Fatalf("rules/platform.md not created: %v", err)
	}
	if !strings.Contains(string(rulesData), "# Project Instructions") {
		t.Error("rules/platform.md should contain scaffold content")
	}
	if !strings.Contains(string(rulesData), "Tech Stack: Go") {
		t.Error("rules/platform.md should detect Go tech stack")
	}
}

func TestRun_DefaultsToCwd(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	err = Run("", []string{"--scaffold-only"})
	if err != nil {
		t.Fatalf("Run() with empty path unexpected error: %v", err)
	}
}
