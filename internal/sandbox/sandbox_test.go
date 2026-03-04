package sandbox

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreate_NonexistentProject(t *testing.T) {
	err := Create("/nonexistent/project/path/xyz", "feature-branch")
	if err == nil {
		t.Fatal("Create() expected error for nonexistent project")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Create() error = %q, want contains 'not found'", err.Error())
	}
}

func TestCreate_NotGitRepo(t *testing.T) {
	dir := t.TempDir()
	err := Create(dir, "feature-branch")
	if err == nil {
		t.Fatal("Create() expected error for non-git directory")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("Create() error = %q, want contains 'not a git repository'", err.Error())
	}
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	cmds := [][]string{
		{"git", "init", dir},
		{"git", "-C", dir, "config", "user.email", "test@test.com"},
		{"git", "-C", dir, "config", "user.name", "Test"},
		{"git", "-C", dir, "commit", "--allow-empty", "-m", "initial commit"},
	}
	for _, args := range cmds {
		if err := exec.Command(args[0], args[1:]...).Run(); err != nil {
			t.Fatalf("git setup %v: %v", args, err)
		}
	}
}

func TestCreate_CreatesWorktree(t *testing.T) {
	parent := t.TempDir()
	projectDir := filepath.Join(parent, "myproject")
	_ = os.MkdirAll(projectDir, 0755)

	initGitRepo(t, projectDir)

	worktreeDir := filepath.Join(parent, "myproject-worktrees", "test-branch")
	t.Cleanup(func() {
		_ = exec.Command("git", "-C", projectDir, "worktree", "remove", "--force", worktreeDir).Run()
	})

	err := Create(projectDir, "test-branch")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if _, err := os.Stat(worktreeDir); os.IsNotExist(err) {
		t.Error("worktree directory was not created")
	}

	// Verify the branch exists
	out, err := exec.Command("git", "-C", projectDir, "branch", "--list", "test-branch").Output()
	if err != nil {
		t.Fatalf("git branch --list: %v", err)
	}
	if !strings.Contains(string(out), "test-branch") {
		t.Error("branch 'test-branch' was not created")
	}
}

func TestCreate_ExistingWorktreeReturnsEarly(t *testing.T) {
	parent := t.TempDir()
	projectDir := filepath.Join(parent, "myproject")
	_ = os.MkdirAll(projectDir, 0755)

	initGitRepo(t, projectDir)

	worktreeDir := filepath.Join(parent, "myproject-worktrees", "test-branch")
	t.Cleanup(func() {
		_ = exec.Command("git", "-C", projectDir, "worktree", "remove", "--force", worktreeDir).Run()
	})

	// First call creates the worktree
	if err := Create(projectDir, "test-branch"); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	// Second call should return nil (worktree already exists)
	if err := Create(projectDir, "test-branch"); err != nil {
		t.Errorf("second Create() error = %v, want nil", err)
	}
}

func TestCreate_CopiesClaudeConfig(t *testing.T) {
	parent := t.TempDir()
	projectDir := filepath.Join(parent, "myproject")
	_ = os.MkdirAll(projectDir, 0755)

	initGitRepo(t, projectDir)

	// Create .claude config files in the project
	claudeDir := filepath.Join(projectDir, ".claude")
	_ = os.MkdirAll(claudeDir, 0755)
	_ = os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), []byte(`{"key":"value"}`), 0644)
	_ = os.WriteFile(filepath.Join(claudeDir, "CLAUDE.local.md"), []byte("# Local instructions"), 0644)

	worktreeDir := filepath.Join(parent, "myproject-worktrees", "config-branch")
	t.Cleanup(func() {
		_ = exec.Command("git", "-C", projectDir, "worktree", "remove", "--force", worktreeDir).Run()
	})

	if err := Create(projectDir, "config-branch"); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify config files were copied
	settingsPath := filepath.Join(worktreeDir, ".claude", "settings.local.json")
	if content, err := os.ReadFile(settingsPath); err != nil {
		t.Errorf("settings.local.json not copied: %v", err)
	} else if string(content) != `{"key":"value"}` {
		t.Errorf("settings.local.json content = %q", content)
	}

	claudeMdPath := filepath.Join(worktreeDir, ".claude", "CLAUDE.local.md")
	if content, err := os.ReadFile(claudeMdPath); err != nil {
		t.Errorf("CLAUDE.local.md not copied: %v", err)
	} else if string(content) != "# Local instructions" {
		t.Errorf("CLAUDE.local.md content = %q", content)
	}
}

func TestCreate_CopiesMcpJson(t *testing.T) {
	parent := t.TempDir()
	projectDir := filepath.Join(parent, "myproject")
	_ = os.MkdirAll(projectDir, 0755)

	initGitRepo(t, projectDir)

	// Create .mcp.json
	mcpContent := `{"mcpServers":{}}`
	_ = os.WriteFile(filepath.Join(projectDir, ".mcp.json"), []byte(mcpContent), 0644)

	worktreeDir := filepath.Join(parent, "myproject-worktrees", "mcp-branch")
	t.Cleanup(func() {
		_ = exec.Command("git", "-C", projectDir, "worktree", "remove", "--force", worktreeDir).Run()
	})

	if err := Create(projectDir, "mcp-branch"); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(worktreeDir, ".mcp.json"))
	if err != nil {
		t.Fatalf(".mcp.json not copied: %v", err)
	}
	if string(content) != mcpContent {
		t.Errorf(".mcp.json content = %q, want %q", content, mcpContent)
	}
}

func TestCreate_RelativePath(t *testing.T) {
	parent := t.TempDir()
	projectDir := filepath.Join(parent, "myproject")
	_ = os.MkdirAll(projectDir, 0755)

	initGitRepo(t, projectDir)

	worktreeDir := filepath.Join(parent, "myproject-worktrees", "rel-branch")
	t.Cleanup(func() {
		_ = exec.Command("git", "-C", projectDir, "worktree", "remove", "--force", worktreeDir).Run()
	})

	err := Create(projectDir, "rel-branch")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if _, err := os.Stat(worktreeDir); os.IsNotExist(err) {
		t.Error("worktree not created with resolved path")
	}
}

func TestRemove_NonexistentProject(t *testing.T) {
	err := Remove("/nonexistent/project/path/xyz", "feature-branch")
	if err == nil {
		t.Fatal("Remove() expected error for nonexistent project")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Remove() error = %q, want contains 'not found'", err.Error())
	}
}

func TestRemove_NotGitRepo(t *testing.T) {
	dir := t.TempDir()
	err := Remove(dir, "feature-branch")
	if err == nil {
		t.Fatal("Remove() expected error for non-git directory")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("Remove() error = %q, want contains 'not a git repository'", err.Error())
	}
}

func TestRemove_MissingSandbox(t *testing.T) {
	parent := t.TempDir()
	projectDir := filepath.Join(parent, "myproject")
	_ = os.MkdirAll(projectDir, 0755)

	initGitRepo(t, projectDir)

	err := Remove(projectDir, "nonexistent-branch")
	if err == nil {
		t.Fatal("Remove() expected error for missing sandbox")
	}
	if !strings.Contains(err.Error(), "sandbox not found") {
		t.Errorf("Remove() error = %q, want contains 'sandbox not found'", err.Error())
	}
}

func TestListTo_NotGitRepo(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	err := ListTo(&buf, dir)
	if err == nil {
		t.Fatal("ListTo() expected error for non-git directory")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("ListTo() error = %q, want contains 'not a git repository'", err.Error())
	}
}

func TestListTo_NoSandboxes(t *testing.T) {
	parent := t.TempDir()
	projectDir := filepath.Join(parent, "myproject")
	_ = os.MkdirAll(projectDir, 0755)

	initGitRepo(t, projectDir)

	var buf bytes.Buffer
	if err := ListTo(&buf, projectDir); err != nil {
		t.Fatalf("ListTo() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "No sandboxes found") {
		t.Errorf("ListTo() output = %q, want contains 'No sandboxes found'", out)
	}
}

func TestListTo_ShowsSandboxes(t *testing.T) {
	parent := t.TempDir()
	projectDir := filepath.Join(parent, "myproject")
	_ = os.MkdirAll(projectDir, 0755)

	initGitRepo(t, projectDir)

	worktreeDir := filepath.Join(parent, "myproject-worktrees", "list-branch")
	t.Cleanup(func() {
		_ = exec.Command("git", "-C", projectDir, "worktree", "remove", "--force", worktreeDir).Run()
	})

	if err := Create(projectDir, "list-branch"); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	var buf bytes.Buffer
	if err := ListTo(&buf, projectDir); err != nil {
		t.Fatalf("ListTo() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "list-branch") {
		t.Errorf("ListTo() output missing branch name, got %q", out)
	}
	if !strings.Contains(out, "Total: 1 sandbox(es)") {
		t.Errorf("ListTo() output missing total count, got %q", out)
	}
}

func TestRemove_RemovesWorktree(t *testing.T) {
	parent := t.TempDir()
	projectDir := filepath.Join(parent, "myproject")
	_ = os.MkdirAll(projectDir, 0755)

	initGitRepo(t, projectDir)

	// Create a sandbox first
	if err := Create(projectDir, "remove-branch"); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	worktreeDir := filepath.Join(parent, "myproject-worktrees", "remove-branch")
	if _, err := os.Stat(worktreeDir); os.IsNotExist(err) {
		t.Fatal("worktree was not created")
	}

	// Remove it
	if err := Remove(projectDir, "remove-branch"); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	if _, err := os.Stat(worktreeDir); !os.IsNotExist(err) {
		t.Error("worktree directory still exists after removal")
	}
}
