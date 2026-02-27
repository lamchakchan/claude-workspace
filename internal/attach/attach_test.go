package attach

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

func TestContains(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		item  string
		want  bool
	}{
		{"found", []string{"a", "b", "c"}, "b", true},
		{"not found", []string{"a", "b", "c"}, "d", false},
		{"empty slice", []string{}, "a", false},
		{"nil slice", nil, "a", false},
		{"first element", []string{"a", "b"}, "a", true},
		{"last element", []string{"a", "b"}, "b", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(tt.slice, tt.item)
			if got != tt.want {
				t.Errorf("contains(%v, %q) = %v, want %v", tt.slice, tt.item, got, tt.want)
			}
		})
	}
}

const rootGitignoreTemplate = `.claude/settings.local.json
.claude/CLAUDE.local.md
.claude/agent-memory-local/
.claude/MEMORY.md
.claude/*.jsonl
plans/*.md
!plans/.gitkeep
`

func setupMockFS(claudeGitignoreContent string) func() {
	oldFS := platform.FS
	platform.FS = fstest.MapFS{
		".claude/.gitignore": &fstest.MapFile{Data: []byte(claudeGitignoreContent)},
		".gitignore":         &fstest.MapFile{Data: []byte(rootGitignoreTemplate)},
	}
	return func() { platform.FS = oldFS }
}

func setupMockFSWithRoot(rootContent string) func() {
	oldFS := platform.FS
	platform.FS = fstest.MapFS{
		".gitignore": &fstest.MapFile{Data: []byte(rootContent)},
	}
	return func() { platform.FS = oldFS }
}

func TestSetupGitignore_CreatesFromTemplate(t *testing.T) {
	templateContent := "settings.local.json\nCLAUDE.local.md\nagent-memory-local/\nMEMORY.md\n*.jsonl\naudits/\n!*.example\n"
	restore := setupMockFS(templateContent)
	defer restore()

	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	_ = os.MkdirAll(claudeDir, 0755)

	setupGitignore(claudeDir)

	content, err := os.ReadFile(filepath.Join(claudeDir, ".gitignore"))
	if err != nil {
		t.Fatalf("expected .gitignore to be created: %v", err)
	}

	s := string(content)
	for _, entry := range []string{"settings.local.json", "MEMORY.md", "*.jsonl", "audits/", "!*.example"} {
		if !strings.Contains(s, entry) {
			t.Errorf("should contain %q", entry)
		}
	}
}

func TestSetupGitignore_UpdatesExisting(t *testing.T) {
	templateContent := "settings.local.json\nCLAUDE.local.md\nagent-memory-local/\nMEMORY.md\n*.jsonl\naudits/\n!*.example\n"
	restore := setupMockFS(templateContent)
	defer restore()

	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	_ = os.MkdirAll(claudeDir, 0755)

	// Write an existing gitignore missing some entries
	existing := "# Personal settings\nsettings.local.json\nCLAUDE.local.md\nagent-memory-local/\n!*.example\n"
	_ = os.WriteFile(filepath.Join(claudeDir, ".gitignore"), []byte(existing), 0644)

	setupGitignore(claudeDir)

	content, _ := os.ReadFile(filepath.Join(claudeDir, ".gitignore"))
	s := string(content)

	// Should have the original entries plus the missing ones
	if !strings.HasPrefix(s, existing) {
		t.Error("should preserve existing content")
	}
	for _, entry := range []string{"MEMORY.md", "*.jsonl", "audits/"} {
		if !strings.Contains(s, entry) {
			t.Errorf("should add missing entry %q", entry)
		}
	}
	// Should not duplicate existing entries
	if strings.Count(s, "settings.local.json") != 1 {
		t.Error("should not duplicate settings.local.json")
	}
}

func TestSetupRootGitignore_Creates(t *testing.T) {
	restore := setupMockFSWithRoot(rootGitignoreTemplate)
	defer restore()

	dir := t.TempDir()

	setupRootGitignore(dir)

	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("expected .gitignore to be created: %v", err)
	}

	s := string(content)
	for _, entry := range []string{
		".claude/settings.local.json",
		".claude/CLAUDE.local.md",
		".claude/agent-memory-local/",
		".claude/MEMORY.md",
		".claude/*.jsonl",
		"plans/*.md",
		"!plans/.gitkeep",
	} {
		if !strings.Contains(s, entry) {
			t.Errorf("should contain %q", entry)
		}
	}
}

func TestSetupRootGitignore_Idempotent(t *testing.T) {
	restore := setupMockFSWithRoot(rootGitignoreTemplate)
	defer restore()

	dir := t.TempDir()

	setupRootGitignore(dir)
	first, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))

	setupRootGitignore(dir)
	second, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))

	if !bytes.Equal(first, second) {
		t.Error("second call should not modify file")
	}
}
