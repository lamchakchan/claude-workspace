package sandbox

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_NonexistentProject(t *testing.T) {
	err := Run("/nonexistent/project/path/xyz", "feature-branch")
	if err == nil {
		t.Fatal("Run() expected error for nonexistent project")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Run() error = %q, want contains 'not found'", err.Error())
	}
}

func TestRun_NotGitRepo(t *testing.T) {
	dir := t.TempDir()
	err := Run(dir, "feature-branch")
	if err == nil {
		t.Fatal("Run() expected error for non-git directory")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("Run() error = %q, want contains 'not a git repository'", err.Error())
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

func TestRun_CreatesWorktree(t *testing.T) {
	parent := t.TempDir()
	projectDir := filepath.Join(parent, "myproject")
	_ = os.MkdirAll(projectDir, 0755)

	initGitRepo(t, projectDir)

	worktreeDir := filepath.Join(parent, "myproject-worktrees", "test-branch")
	t.Cleanup(func() {
		_ = exec.Command("git", "-C", projectDir, "worktree", "remove", "--force", worktreeDir).Run()
	})

	err := Run(projectDir, "test-branch")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
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

func TestRun_ExistingWorktreeReturnsEarly(t *testing.T) {
	parent := t.TempDir()
	projectDir := filepath.Join(parent, "myproject")
	_ = os.MkdirAll(projectDir, 0755)

	initGitRepo(t, projectDir)

	worktreeDir := filepath.Join(parent, "myproject-worktrees", "test-branch")
	t.Cleanup(func() {
		_ = exec.Command("git", "-C", projectDir, "worktree", "remove", "--force", worktreeDir).Run()
	})

	// First run creates the worktree
	if err := Run(projectDir, "test-branch"); err != nil {
		t.Fatalf("first Run() error = %v", err)
	}

	// Second run should return nil (worktree already exists)
	if err := Run(projectDir, "test-branch"); err != nil {
		t.Errorf("second Run() error = %v, want nil", err)
	}
}

func TestRun_CopiesClaudeConfig(t *testing.T) {
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

	if err := Run(projectDir, "config-branch"); err != nil {
		t.Fatalf("Run() error = %v", err)
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

func TestRun_CopiesMcpJson(t *testing.T) {
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

	if err := Run(projectDir, "mcp-branch"); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(worktreeDir, ".mcp.json"))
	if err != nil {
		t.Fatalf(".mcp.json not copied: %v", err)
	}
	if string(content) != mcpContent {
		t.Errorf(".mcp.json content = %q, want %q", content, mcpContent)
	}
}

func TestRun_RelativePath(t *testing.T) {
	parent := t.TempDir()
	projectDir := filepath.Join(parent, "myproject")
	_ = os.MkdirAll(projectDir, 0755)

	initGitRepo(t, projectDir)

	worktreeDir := filepath.Join(parent, "myproject-worktrees", "rel-branch")
	t.Cleanup(func() {
		_ = exec.Command("git", "-C", projectDir, "worktree", "remove", "--force", worktreeDir).Run()
	})

	// Run with the absolute path (simulating that filepath.Abs resolves it)
	err := Run(projectDir, "rel-branch")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if _, err := os.Stat(worktreeDir); os.IsNotExist(err) {
		t.Error("worktree not created with resolved path")
	}
}
