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

	// Create the worktree
	platform.PrintStep(os.Stdout, 1, 4, "Creating git worktree...")
	if err := createWorktree(projectDir, worktreeDir, branchName); err != nil {
		return err
	}
	fmt.Printf("  Worktree created at: %s\n", worktreeDir)

	// Copy .claude configuration to worktree if it exists in main project
	platform.PrintStep(os.Stdout, 2, 4, "Setting up Claude configuration...")
	copyClaudeConfig(projectDir, worktreeDir)

	// Copy .mcp.json if not tracked by git
	platform.PrintStep(os.Stdout, 3, 4, "Setting up MCP configuration...")
	copyMCPConfig(projectDir, worktreeDir)

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

func createWorktree(projectDir, worktreeDir, branchName string) error {
	branchExists := platform.RunQuietDir(projectDir, "git", "rev-parse", "--verify", branchName) == nil
	if branchExists {
		if err := platform.RunDir(projectDir, "git", "worktree", "add", worktreeDir, branchName); err != nil {
			return fmt.Errorf("creating worktree: %w", err)
		}
	} else {
		if err := platform.RunDir(projectDir, "git", "worktree", "add", "-b", branchName, worktreeDir); err != nil {
			return fmt.Errorf("creating worktree: %w", err)
		}
	}
	return nil
}

func copyClaudeConfig(projectDir, worktreeDir string) {
	claudeDir := filepath.Join(projectDir, ".claude")
	if !platform.FileExists(claudeDir) {
		return
	}
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

func copyMCPConfig(projectDir, worktreeDir string) {
	mcpJSON := filepath.Join(projectDir, ".mcp.json")
	worktreeMcp := filepath.Join(worktreeDir, ".mcp.json")
	if platform.FileExists(mcpJSON) && !platform.FileExists(worktreeMcp) {
		if err := platform.CopyFile(mcpJSON, worktreeMcp); err == nil {
			platform.PrintSuccess(os.Stdout, "Copied .mcp.json to worktree")
		}
	}
}

func installWorktreeDeps(worktreeDir string) {
	installed := false

	// --- JavaScript/TypeScript ---
	switch {
	case platform.FileExists(filepath.Join(worktreeDir, "bun.lockb")) || platform.FileExists(filepath.Join(worktreeDir, "bun.lock")):
		if err := platform.RunQuietDir(worktreeDir, "bun", "install"); err == nil {
			platform.PrintSuccess(os.Stdout, "Dependencies installed (bun)")
		} else {
			platform.PrintWarningLine(os.Stdout, "Could not install bun dependencies")
		}
		installed = true
	case platform.FileExists(filepath.Join(worktreeDir, "package-lock.json")):
		if err := platform.RunQuietDir(worktreeDir, "npm", "ci"); err == nil {
			platform.PrintSuccess(os.Stdout, "Dependencies installed (npm)")
		} else {
			platform.PrintWarningLine(os.Stdout, "Could not install npm dependencies")
		}
		installed = true
	case platform.FileExists(filepath.Join(worktreeDir, "yarn.lock")):
		if err := platform.RunQuietDir(worktreeDir, "yarn", "install", "--frozen-lockfile"); err == nil {
			platform.PrintSuccess(os.Stdout, "Dependencies installed (yarn)")
		} else {
			platform.PrintWarningLine(os.Stdout, "Could not install yarn dependencies")
		}
		installed = true
	case platform.FileExists(filepath.Join(worktreeDir, "pnpm-lock.yaml")):
		if err := platform.RunQuietDir(worktreeDir, "pnpm", "install", "--frozen-lockfile"); err == nil {
			platform.PrintSuccess(os.Stdout, "Dependencies installed (pnpm)")
		} else {
			platform.PrintWarningLine(os.Stdout, "Could not install pnpm dependencies")
		}
		installed = true
	case platform.FileExists(filepath.Join(worktreeDir, "package.json")):
		fmt.Println("  No lockfile found. Run your package manager to install dependencies.")
		installed = true
	}

	// --- Ruby ---
	if platform.FileExists(filepath.Join(worktreeDir, "Gemfile.lock")) {
		if platform.Exists("bundle") {
			if err := platform.RunQuietDir(worktreeDir, "bundle", "install"); err == nil {
				platform.PrintSuccess(os.Stdout, "Dependencies installed (bundler)")
			} else {
				platform.PrintWarningLine(os.Stdout, "Could not install bundler dependencies")
			}
		} else {
			platform.PrintWarningLine(os.Stdout, "Gemfile.lock found but bundler not installed")
		}
		installed = true
	} else if platform.FileExists(filepath.Join(worktreeDir, "Gemfile")) {
		fmt.Println("  Gemfile found but no Gemfile.lock. Run `bundle install` to install dependencies.")
		installed = true
	}

	// --- Python ---
	if platform.FileExists(filepath.Join(worktreeDir, "poetry.lock")) {
		if platform.Exists("poetry") {
			if err := platform.RunQuietDir(worktreeDir, "poetry", "install"); err == nil {
				platform.PrintSuccess(os.Stdout, "Dependencies installed (poetry)")
			} else {
				platform.PrintWarningLine(os.Stdout, "Could not install poetry dependencies")
			}
		}
		installed = true
	} else if platform.FileExists(filepath.Join(worktreeDir, "uv.lock")) {
		if platform.Exists("uv") {
			if err := platform.RunQuietDir(worktreeDir, "uv", "sync"); err == nil {
				platform.PrintSuccess(os.Stdout, "Dependencies installed (uv)")
			} else {
				platform.PrintWarningLine(os.Stdout, "Could not install uv dependencies")
			}
		}
		installed = true
	} else if platform.FileExists(filepath.Join(worktreeDir, "requirements.txt")) {
		if platform.Exists("pip") {
			if err := platform.RunQuietDir(worktreeDir, "pip", "install", "-r", "requirements.txt"); err == nil {
				platform.PrintSuccess(os.Stdout, "Dependencies installed (pip)")
			} else {
				platform.PrintWarningLine(os.Stdout, "Could not install pip dependencies")
			}
		} else {
			fmt.Println("  requirements.txt found. Run `pip install -r requirements.txt` to install dependencies.")
		}
		installed = true
	}

	// --- Java Maven ---
	if platform.FileExists(filepath.Join(worktreeDir, "pom.xml")) {
		if platform.Exists("mvn") {
			if err := platform.RunQuietDir(worktreeDir, "mvn", "dependency:resolve", "-q"); err == nil {
				platform.PrintSuccess(os.Stdout, "Dependencies resolved (Maven)")
			} else {
				platform.PrintWarningLine(os.Stdout, "Could not resolve Maven dependencies")
			}
		} else {
			fmt.Println("  pom.xml found but mvn not installed.")
		}
		installed = true
	}

	// --- Java/Kotlin Gradle ---
	if platform.FileExists(filepath.Join(worktreeDir, "build.gradle")) || platform.FileExists(filepath.Join(worktreeDir, "build.gradle.kts")) {
		wrapper := filepath.Join(worktreeDir, "gradlew")
		if platform.FileExists(wrapper) {
			if err := platform.RunQuietDir(worktreeDir, "./gradlew", "dependencies", "--quiet"); err == nil {
				platform.PrintSuccess(os.Stdout, "Dependencies resolved (Gradle)")
			} else {
				platform.PrintWarningLine(os.Stdout, "Could not resolve Gradle dependencies")
			}
		} else if platform.Exists("gradle") {
			if err := platform.RunQuietDir(worktreeDir, "gradle", "dependencies", "--quiet"); err == nil {
				platform.PrintSuccess(os.Stdout, "Dependencies resolved (Gradle)")
			} else {
				platform.PrintWarningLine(os.Stdout, "Could not resolve Gradle dependencies")
			}
		} else {
			fmt.Println("  Gradle project found but no gradlew wrapper or gradle binary.")
		}
		installed = true
	}

	// --- C# .NET ---
	csprojMatches, _ := filepath.Glob(filepath.Join(worktreeDir, "*.csproj"))
	slnMatches, _ := filepath.Glob(filepath.Join(worktreeDir, "*.sln"))
	if len(csprojMatches) > 0 || len(slnMatches) > 0 {
		if platform.Exists("dotnet") {
			if err := platform.RunQuietDir(worktreeDir, "dotnet", "restore"); err == nil {
				platform.PrintSuccess(os.Stdout, "Dependencies restored (dotnet)")
			} else {
				platform.PrintWarningLine(os.Stdout, "Could not restore dotnet dependencies")
			}
		}
		installed = true
	}

	// --- Elixir ---
	if platform.FileExists(filepath.Join(worktreeDir, "mix.exs")) {
		if platform.Exists("mix") {
			if err := platform.RunQuietDir(worktreeDir, "mix", "deps.get"); err == nil {
				platform.PrintSuccess(os.Stdout, "Dependencies installed (mix)")
			} else {
				platform.PrintWarningLine(os.Stdout, "Could not install mix dependencies")
			}
		}
		installed = true
	}

	// --- PHP ---
	if platform.FileExists(filepath.Join(worktreeDir, "composer.lock")) {
		if platform.Exists("composer") {
			if err := platform.RunQuietDir(worktreeDir, "composer", "install"); err == nil {
				platform.PrintSuccess(os.Stdout, "Dependencies installed (composer)")
			} else {
				platform.PrintWarningLine(os.Stdout, "Could not install composer dependencies")
			}
		}
		installed = true
	} else if platform.FileExists(filepath.Join(worktreeDir, "composer.json")) {
		fmt.Println("  composer.json found but no composer.lock. Run `composer install` to install dependencies.")
		installed = true
	}

	// --- Swift ---
	if platform.FileExists(filepath.Join(worktreeDir, "Package.swift")) {
		if platform.Exists("swift") {
			if err := platform.RunQuietDir(worktreeDir, "swift", "package", "resolve"); err == nil {
				platform.PrintSuccess(os.Stdout, "Dependencies resolved (Swift PM)")
			} else {
				platform.PrintWarningLine(os.Stdout, "Could not resolve Swift package dependencies")
			}
		}
		installed = true
	}

	// --- Scala ---
	if platform.FileExists(filepath.Join(worktreeDir, "build.sbt")) {
		if platform.Exists("sbt") {
			if err := platform.RunQuietDir(worktreeDir, "sbt", "update"); err == nil {
				platform.PrintSuccess(os.Stdout, "Dependencies resolved (sbt)")
			} else {
				platform.PrintWarningLine(os.Stdout, "Could not resolve sbt dependencies")
			}
		}
		installed = true
	}

	if !installed {
		fmt.Println("  No recognized dependency files found. Skipping dependency installation.")
	}
}
