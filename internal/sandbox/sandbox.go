// Package sandbox implements the "sandbox" command, which creates isolated
// git worktrees for parallel development with Claude Code configuration
// automatically copied into each worktree.
package sandbox

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

// Run creates a git worktree sandbox for the given project path and branch name.
// It copies Claude configuration and installs dependencies in the new worktree.
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

	platform.PrintBanner(os.Stdout, fmt.Sprintf("Creating Sandboxed Branch: %s", branchName))
	fmt.Println()

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
	platform.PrintStep(os.Stdout, 1, 4, "Creating git worktree...")
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
	platform.PrintStep(os.Stdout, 2, 4, "Setting up Claude configuration...")
	claudeDir := filepath.Join(projectDir, ".claude")
	if platform.FileExists(claudeDir) {
		worktreeClaudeDir := filepath.Join(worktreeDir, ".claude")
		_ = os.MkdirAll(worktreeClaudeDir, 0755)

		localSettings := filepath.Join(claudeDir, "settings.local.json")
		if platform.FileExists(localSettings) {
			if err := platform.CopyFile(localSettings, filepath.Join(worktreeClaudeDir, "settings.local.json")); err == nil {
				platform.PrintSuccess(os.Stdout, "Copied local settings to worktree")
			}
		}

		localClaudeMd := filepath.Join(claudeDir, "CLAUDE.local.md")
		if platform.FileExists(localClaudeMd) {
			if err := platform.CopyFile(localClaudeMd, filepath.Join(worktreeClaudeDir, "CLAUDE.local.md")); err == nil {
				platform.PrintSuccess(os.Stdout, "Copied local CLAUDE.md to worktree")
			}
		}
	}

	// Copy .mcp.json if not tracked by git
	platform.PrintStep(os.Stdout, 3, 4, "Setting up MCP configuration...")
	mcpJSON := filepath.Join(projectDir, ".mcp.json")
	worktreeMcp := filepath.Join(worktreeDir, ".mcp.json")
	if platform.FileExists(mcpJSON) && !platform.FileExists(worktreeMcp) {
		if err := platform.CopyFile(mcpJSON, worktreeMcp); err == nil {
			platform.PrintSuccess(os.Stdout, "Copied .mcp.json to worktree")
		}
	}

	// Install dependencies if needed
	platform.PrintStep(os.Stdout, 4, 4, "Setting up dependencies...")
	installWorktreeDeps(worktreeDir)

	platform.PrintBanner(os.Stdout, "Sandbox Ready")
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
	switch {
	case platform.FileExists(filepath.Join(worktreeDir, "bun.lockb")) || platform.FileExists(filepath.Join(worktreeDir, "bun.lock")):
		if err := platform.RunQuietDir(worktreeDir, "bun", "install"); err == nil {
			platform.PrintSuccess(os.Stdout, "Dependencies installed (bun)")
		} else {
			platform.PrintWarningLine(os.Stdout, "Could not install bun dependencies")
		}
	case platform.FileExists(filepath.Join(worktreeDir, "package-lock.json")):
		if err := platform.RunQuietDir(worktreeDir, "npm", "ci"); err == nil {
			platform.PrintSuccess(os.Stdout, "Dependencies installed (npm)")
		} else {
			platform.PrintWarningLine(os.Stdout, "Could not install npm dependencies")
		}
	case platform.FileExists(filepath.Join(worktreeDir, "yarn.lock")):
		if err := platform.RunQuietDir(worktreeDir, "yarn", "install", "--frozen-lockfile"); err == nil {
			platform.PrintSuccess(os.Stdout, "Dependencies installed (yarn)")
		} else {
			platform.PrintWarningLine(os.Stdout, "Could not install yarn dependencies")
		}
	case platform.FileExists(filepath.Join(worktreeDir, "pnpm-lock.yaml")):
		if err := platform.RunQuietDir(worktreeDir, "pnpm", "install", "--frozen-lockfile"); err == nil {
			platform.PrintSuccess(os.Stdout, "Dependencies installed (pnpm)")
		} else {
			platform.PrintWarningLine(os.Stdout, "Could not install pnpm dependencies")
		}
	case platform.FileExists(filepath.Join(worktreeDir, "package.json")):
		fmt.Println("  No lockfile found. Run your package manager to install dependencies.")
	default:
		fmt.Println("  No package.json found. Skipping dependency installation.")
	}
}
