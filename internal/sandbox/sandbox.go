package sandbox

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

func Run(projectPath, branchName string) error {
	if projectPath == "" || branchName == "" {
		fmt.Fprintln(os.Stderr, "Usage: claude-workspace sandbox <project-path> <branch-name>")
		fmt.Println("\nExamples:")
		fmt.Println("  claude-workspace sandbox ./my-project feature-auth")
		fmt.Println("  claude-workspace sandbox ./my-project feature-api")
		fmt.Println("  claude-workspace sandbox ./my-project bugfix-login")
		os.Exit(1)
	}

	projectDir, err := filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	if !platform.FileExists(projectDir) {
		return fmt.Errorf("project directory not found: %s", projectDir)
	}

	// Verify it's a git repo
	if err := platform.RunQuietDir(projectDir, "git", "rev-parse", "--git-dir"); err != nil {
		return fmt.Errorf("not a git repository: %s\nInitialize git first: git init", projectDir)
	}

	projectName := filepath.Base(projectDir)
	worktreeBase := filepath.Join(filepath.Dir(projectDir), projectName+"-worktrees")
	worktreeDir := filepath.Join(worktreeBase, branchName)

	fmt.Printf("\n=== Creating Sandboxed Branch: %s ===\n\n", branchName)

	// Create worktrees directory
	if err := os.MkdirAll(worktreeBase, 0755); err != nil {
		return fmt.Errorf("creating worktrees directory: %w", err)
	}

	// Check if worktree already exists
	if platform.FileExists(worktreeDir) {
		fmt.Printf("Worktree already exists at: %s\n", worktreeDir)
		fmt.Printf("To use it: cd %s && claude\n", worktreeDir)
		return nil
	}

	// Check if branch already exists
	branchExists := platform.RunQuietDir(projectDir, "git", "rev-parse", "--verify", branchName) == nil

	// Create the worktree
	fmt.Println("[1/4] Creating git worktree...")
	if branchExists {
		if err := platform.RunDir(projectDir, "git", "worktree", "add", worktreeDir, branchName); err != nil {
			return fmt.Errorf("creating worktree: %w", err)
		}
	} else {
		if err := platform.RunDir(projectDir, "git", "worktree", "add", "-b", branchName, worktreeDir); err != nil {
			return fmt.Errorf("creating worktree: %w", err)
		}
	}
	fmt.Printf("  Worktree created at: %s\n", worktreeDir)

	// Copy .claude configuration to worktree if it exists in main project
	fmt.Println("[2/4] Setting up Claude configuration...")
	claudeDir := filepath.Join(projectDir, ".claude")
	if platform.FileExists(claudeDir) {
		worktreeClaudeDir := filepath.Join(worktreeDir, ".claude")
		os.MkdirAll(worktreeClaudeDir, 0755)

		localSettings := filepath.Join(claudeDir, "settings.local.json")
		if platform.FileExists(localSettings) {
			if err := platform.CopyFile(localSettings, filepath.Join(worktreeClaudeDir, "settings.local.json")); err == nil {
				fmt.Println("  Copied local settings to worktree")
			}
		}

		localClaudeMd := filepath.Join(claudeDir, "CLAUDE.local.md")
		if platform.FileExists(localClaudeMd) {
			if err := platform.CopyFile(localClaudeMd, filepath.Join(worktreeClaudeDir, "CLAUDE.local.md")); err == nil {
				fmt.Println("  Copied local CLAUDE.md to worktree")
			}
		}
	}

	// Copy .mcp.json if not tracked by git
	fmt.Println("[3/4] Setting up MCP configuration...")
	mcpJson := filepath.Join(projectDir, ".mcp.json")
	worktreeMcp := filepath.Join(worktreeDir, ".mcp.json")
	if platform.FileExists(mcpJson) && !platform.FileExists(worktreeMcp) {
		if err := platform.CopyFile(mcpJson, worktreeMcp); err == nil {
			fmt.Println("  Copied .mcp.json to worktree")
		}
	}

	// Install dependencies if needed
	fmt.Println("[4/4] Setting up dependencies...")
	installWorktreeDeps(worktreeDir)

	fmt.Println("\n=== Sandbox Ready ===")
	fmt.Printf("\nBranch:    %s\n", branchName)
	fmt.Printf("Directory: %s\n", worktreeDir)
	fmt.Println("\nTo start working:")
	fmt.Printf("  cd %s\n", worktreeDir)
	fmt.Println("  claude")
	fmt.Println("\nTo list all worktrees:")
	fmt.Printf("  git -C %s worktree list\n", projectDir)
	fmt.Println("\nTo remove this sandbox when done:")
	fmt.Printf("  git -C %s worktree remove %s\n", projectDir, worktreeDir)
	fmt.Println()

	return nil
}

func installWorktreeDeps(worktreeDir string) {
	if platform.FileExists(filepath.Join(worktreeDir, "bun.lockb")) || platform.FileExists(filepath.Join(worktreeDir, "bun.lock")) {
		if err := platform.RunQuietDir(worktreeDir, "bun", "install"); err == nil {
			fmt.Println("  Dependencies installed (bun)")
		} else {
			fmt.Println("  Warning: Could not install bun dependencies")
		}
	} else if platform.FileExists(filepath.Join(worktreeDir, "package-lock.json")) {
		if err := platform.RunQuietDir(worktreeDir, "npm", "ci"); err == nil {
			fmt.Println("  Dependencies installed (npm)")
		} else {
			fmt.Println("  Warning: Could not install npm dependencies")
		}
	} else if platform.FileExists(filepath.Join(worktreeDir, "yarn.lock")) {
		if err := platform.RunQuietDir(worktreeDir, "yarn", "install", "--frozen-lockfile"); err == nil {
			fmt.Println("  Dependencies installed (yarn)")
		} else {
			fmt.Println("  Warning: Could not install yarn dependencies")
		}
	} else if platform.FileExists(filepath.Join(worktreeDir, "pnpm-lock.yaml")) {
		if err := platform.RunQuietDir(worktreeDir, "pnpm", "install", "--frozen-lockfile"); err == nil {
			fmt.Println("  Dependencies installed (pnpm)")
		} else {
			fmt.Println("  Warning: Could not install pnpm dependencies")
		}
	} else if platform.FileExists(filepath.Join(worktreeDir, "package.json")) {
		fmt.Println("  No lockfile found. Run your package manager to install dependencies.")
	} else {
		fmt.Println("  No package.json found. Skipping dependency installation.")
	}
}
