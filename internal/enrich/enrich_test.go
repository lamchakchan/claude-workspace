package enrich

import (
	"os"
	"path/filepath"
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
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)

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
	os.MkdirAll(claudeDir, 0755)

	// Write an existing CLAUDE.md
	existing := "# Existing content"
	os.WriteFile(filepath.Join(claudeDir, "CLAUDE.md"), []byte(existing), 0644)

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

func TestRun_DefaultsToCwd(t *testing.T) {
	// Run with empty path — should use cwd and not error on directory resolution
	// This will fail on enrichment (no claude CLI) but should not fail on path resolution
	err := Run("", []string{"--scaffold-only"})
	if err != nil {
		t.Fatalf("Run() with empty path unexpected error: %v", err)
	}
}
