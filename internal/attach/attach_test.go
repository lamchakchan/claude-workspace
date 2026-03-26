package attach

import (
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

func setupMockFS(claudeGitignoreContent string) func() {
	oldFS := platform.FS
	platform.FS = fstest.MapFS{
		".claude/.gitignore": &fstest.MapFile{Data: []byte(claudeGitignoreContent)},
	}
	return func() { platform.FS = oldFS }
}

func TestSetupGitignore_CreatesFromTemplate(t *testing.T) {
	templateContent := "settings.local.json\nCLAUDE.local.md\nagent-memory-local/\nMEMORY.md\n*.jsonl\naudits/\nplans/*.md\n!plans/.gitkeep\n!*.example\n"
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
	for _, entry := range []string{"settings.local.json", "MEMORY.md", "*.jsonl", "audits/", "plans/*.md", "!plans/.gitkeep", "!*.example"} {
		if !strings.Contains(s, entry) {
			t.Errorf("should contain %q", entry)
		}
	}
}

func TestSetupGitignore_UpdatesExisting(t *testing.T) {
	templateContent := "settings.local.json\nCLAUDE.local.md\nagent-memory-local/\nMEMORY.md\n*.jsonl\naudits/\nplans/*.md\n!plans/.gitkeep\n!*.example\n"
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

func TestSetupProjectInstructions_NoCLAUDEMd(t *testing.T) {
	oldFS := platform.FS
	platform.FS = fstest.MapFS{
		".claude/rules/platform.md": &fstest.MapFile{Data: []byte("# Platform Rules")},
	}
	defer func() { platform.FS = oldFS }()

	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	_ = os.MkdirAll(filepath.Join(claudeDir, "rules"), 0755)

	// No existing CLAUDE.md — should write scaffold to CLAUDE.md
	got := setupProjectInstructions(dir, claudeDir, false)

	if got != filepath.Join(claudeDir, "CLAUDE.md") {
		t.Errorf("target = %q, want CLAUDE.md path", got)
	}
	data, err := os.ReadFile(filepath.Join(claudeDir, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("CLAUDE.md not created: %v", err)
	}
	if !strings.Contains(string(data), "# Project Instructions") {
		t.Error("CLAUDE.md should contain scaffold content")
	}
	// Rules template should also be copied
	if _, err := os.ReadFile(filepath.Join(claudeDir, "rules", "platform.md")); err != nil {
		t.Error("rules/platform.md should be created")
	}
}

func TestSetupProjectInstructions_ExistingCLAUDEMd(t *testing.T) {
	oldFS := platform.FS
	platform.FS = fstest.MapFS{
		".claude/rules/platform.md": &fstest.MapFile{Data: []byte("# Platform Rules")},
	}
	defer func() { platform.FS = oldFS }()

	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	_ = os.MkdirAll(filepath.Join(claudeDir, "rules"), 0755)

	// Write existing CLAUDE.md
	existing := "# My Custom Instructions"
	_ = os.WriteFile(filepath.Join(claudeDir, "CLAUDE.md"), []byte(existing), 0644)

	got := setupProjectInstructions(dir, claudeDir, false)

	// Should target rules/platform.md
	if got != filepath.Join(claudeDir, "rules", "platform.md") {
		t.Errorf("target = %q, want rules/platform.md path", got)
	}
	// Existing CLAUDE.md should be preserved
	data, _ := os.ReadFile(filepath.Join(claudeDir, "CLAUDE.md"))
	if string(data) != existing {
		t.Errorf("CLAUDE.md was modified: got %q, want %q", string(data), existing)
	}
	// rules/platform.md should have platform rules template content
	rulesData, err := os.ReadFile(filepath.Join(claudeDir, "rules", "platform.md"))
	if err != nil {
		t.Fatalf("rules/platform.md not created: %v", err)
	}
	if !strings.Contains(string(rulesData), "# Platform Rules") {
		t.Error("rules/platform.md should contain platform rules template content")
	}
}

func TestSetupProjectInstructions_Force(t *testing.T) {
	oldFS := platform.FS
	platform.FS = fstest.MapFS{
		".claude/rules/platform.md": &fstest.MapFile{Data: []byte("# Platform Rules")},
	}
	defer func() { platform.FS = oldFS }()

	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	_ = os.MkdirAll(filepath.Join(claudeDir, "rules"), 0755)

	// Write existing CLAUDE.md
	_ = os.WriteFile(filepath.Join(claudeDir, "CLAUDE.md"), []byte("# Old"), 0644)

	got := setupProjectInstructions(dir, claudeDir, true)

	// --force should always write to CLAUDE.md
	if got != filepath.Join(claudeDir, "CLAUDE.md") {
		t.Errorf("target = %q, want CLAUDE.md path with --force", got)
	}
	data, _ := os.ReadFile(filepath.Join(claudeDir, "CLAUDE.md"))
	if !strings.Contains(string(data), "# Project Instructions") {
		t.Error("CLAUDE.md should be overwritten with scaffold content")
	}
}

func TestSetupGitignore_SkipsDenyAll(t *testing.T) {
	templateContent := "settings.local.json\nCLAUDE.local.md\nagent-memory-local/\nMEMORY.md\n*.jsonl\naudits/\nplans/*.md\n!plans/.gitkeep\n!*.example\n"
	restore := setupMockFS(templateContent)
	defer restore()

	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	_ = os.MkdirAll(claudeDir, 0755)

	// Write a deny-all gitignore
	denyAll := "*\n!.gitignore\n!CLAUDE.md\n"
	_ = os.WriteFile(filepath.Join(claudeDir, ".gitignore"), []byte(denyAll), 0644)

	setupGitignore(claudeDir)

	content, _ := os.ReadFile(filepath.Join(claudeDir, ".gitignore"))
	if string(content) != denyAll {
		t.Error("should not modify a deny-all .gitignore")
	}
}
