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

// depInstallers is the ordered list of dependency installers for worktrees.
// Each installer returns true if it handled the language (even if install failed).
var depInstallers = []func(string) bool{
	installJSDeps,
	installRubyDeps,
	installPythonDeps,
	installMavenDeps,
	installGradleDeps,
	installDotNetDeps,
	installElixirDeps,
	installPHPDeps,
	installSwiftDeps,
	installScalaDeps,
}

func installWorktreeDeps(worktreeDir string) {
	installed := false
	for _, install := range depInstallers {
		if install(worktreeDir) {
			installed = true
		}
	}
	if !installed {
		fmt.Println("  No recognized dependency files found. Skipping dependency installation.")
	}
}

func installJSDeps(dir string) bool {
	switch {
	case platform.FileExists(filepath.Join(dir, "bun.lockb")) || platform.FileExists(filepath.Join(dir, "bun.lock")):
		if err := platform.RunQuietDir(dir, "bun", "install"); err == nil {
			platform.PrintSuccess(os.Stdout, "Dependencies installed (bun)")
		} else {
			platform.PrintWarningLine(os.Stdout, "Could not install bun dependencies")
		}
	case platform.FileExists(filepath.Join(dir, "package-lock.json")):
		if err := platform.RunQuietDir(dir, "npm", "ci"); err == nil {
			platform.PrintSuccess(os.Stdout, "Dependencies installed (npm)")
		} else {
			platform.PrintWarningLine(os.Stdout, "Could not install npm dependencies")
		}
	case platform.FileExists(filepath.Join(dir, "yarn.lock")):
		if err := platform.RunQuietDir(dir, "yarn", "install", "--frozen-lockfile"); err == nil {
			platform.PrintSuccess(os.Stdout, "Dependencies installed (yarn)")
		} else {
			platform.PrintWarningLine(os.Stdout, "Could not install yarn dependencies")
		}
	case platform.FileExists(filepath.Join(dir, "pnpm-lock.yaml")):
		if err := platform.RunQuietDir(dir, "pnpm", "install", "--frozen-lockfile"); err == nil {
			platform.PrintSuccess(os.Stdout, "Dependencies installed (pnpm)")
		} else {
			platform.PrintWarningLine(os.Stdout, "Could not install pnpm dependencies")
		}
	case platform.FileExists(filepath.Join(dir, "package.json")):
		fmt.Println("  No lockfile found. Run your package manager to install dependencies.")
	default:
		return false
	}
	return true
}

func installRubyDeps(dir string) bool {
	switch {
	case platform.FileExists(filepath.Join(dir, "Gemfile.lock")):
		if platform.Exists("bundle") {
			if err := platform.RunQuietDir(dir, "bundle", "install"); err == nil {
				platform.PrintSuccess(os.Stdout, "Dependencies installed (bundler)")
			} else {
				platform.PrintWarningLine(os.Stdout, "Could not install bundler dependencies")
			}
		} else {
			platform.PrintWarningLine(os.Stdout, "Gemfile.lock found but bundler not installed")
		}
	case platform.FileExists(filepath.Join(dir, "Gemfile")):
		fmt.Println("  Gemfile found but no Gemfile.lock. Run `bundle install` to install dependencies.")
	default:
		return false
	}
	return true
}

func installPythonDeps(dir string) bool {
	switch {
	case platform.FileExists(filepath.Join(dir, "poetry.lock")):
		if platform.Exists("poetry") {
			if err := platform.RunQuietDir(dir, "poetry", "install"); err == nil {
				platform.PrintSuccess(os.Stdout, "Dependencies installed (poetry)")
			} else {
				platform.PrintWarningLine(os.Stdout, "Could not install poetry dependencies")
			}
		}
	case platform.FileExists(filepath.Join(dir, "uv.lock")):
		if platform.Exists("uv") {
			if err := platform.RunQuietDir(dir, "uv", "sync"); err == nil {
				platform.PrintSuccess(os.Stdout, "Dependencies installed (uv)")
			} else {
				platform.PrintWarningLine(os.Stdout, "Could not install uv dependencies")
			}
		}
	case platform.FileExists(filepath.Join(dir, "requirements.txt")):
		if platform.Exists("pip") {
			if err := platform.RunQuietDir(dir, "pip", "install", "-r", "requirements.txt"); err == nil {
				platform.PrintSuccess(os.Stdout, "Dependencies installed (pip)")
			} else {
				platform.PrintWarningLine(os.Stdout, "Could not install pip dependencies")
			}
		} else {
			fmt.Println("  requirements.txt found. Run `pip install -r requirements.txt` to install dependencies.")
		}
	default:
		return false
	}
	return true
}

func installMavenDeps(dir string) bool {
	if !platform.FileExists(filepath.Join(dir, "pom.xml")) {
		return false
	}
	if platform.Exists("mvn") {
		if err := platform.RunQuietDir(dir, "mvn", "dependency:resolve", "-q"); err == nil {
			platform.PrintSuccess(os.Stdout, "Dependencies resolved (Maven)")
		} else {
			platform.PrintWarningLine(os.Stdout, "Could not resolve Maven dependencies")
		}
	} else {
		fmt.Println("  pom.xml found but mvn not installed.")
	}
	return true
}

func installGradleDeps(dir string) bool {
	if !platform.FileExists(filepath.Join(dir, "build.gradle")) && !platform.FileExists(filepath.Join(dir, "build.gradle.kts")) {
		return false
	}
	switch {
	case platform.FileExists(filepath.Join(dir, "gradlew")):
		if err := platform.RunQuietDir(dir, "./gradlew", "dependencies", "--quiet"); err == nil {
			platform.PrintSuccess(os.Stdout, "Dependencies resolved (Gradle)")
		} else {
			platform.PrintWarningLine(os.Stdout, "Could not resolve Gradle dependencies")
		}
	case platform.Exists("gradle"):
		if err := platform.RunQuietDir(dir, "gradle", "dependencies", "--quiet"); err == nil {
			platform.PrintSuccess(os.Stdout, "Dependencies resolved (Gradle)")
		} else {
			platform.PrintWarningLine(os.Stdout, "Could not resolve Gradle dependencies")
		}
	default:
		fmt.Println("  Gradle project found but no gradlew wrapper or gradle binary.")
	}
	return true
}

func installDotNetDeps(dir string) bool {
	csprojMatches, _ := filepath.Glob(filepath.Join(dir, "*.csproj"))
	slnMatches, _ := filepath.Glob(filepath.Join(dir, "*.sln"))
	if len(csprojMatches) == 0 && len(slnMatches) == 0 {
		return false
	}
	if platform.Exists("dotnet") {
		if err := platform.RunQuietDir(dir, "dotnet", "restore"); err == nil {
			platform.PrintSuccess(os.Stdout, "Dependencies restored (dotnet)")
		} else {
			platform.PrintWarningLine(os.Stdout, "Could not restore dotnet dependencies")
		}
	}
	return true
}

func installElixirDeps(dir string) bool {
	if !platform.FileExists(filepath.Join(dir, "mix.exs")) {
		return false
	}
	if platform.Exists("mix") {
		if err := platform.RunQuietDir(dir, "mix", "deps.get"); err == nil {
			platform.PrintSuccess(os.Stdout, "Dependencies installed (mix)")
		} else {
			platform.PrintWarningLine(os.Stdout, "Could not install mix dependencies")
		}
	}
	return true
}

func installPHPDeps(dir string) bool {
	switch {
	case platform.FileExists(filepath.Join(dir, "composer.lock")):
		if platform.Exists("composer") {
			if err := platform.RunQuietDir(dir, "composer", "install"); err == nil {
				platform.PrintSuccess(os.Stdout, "Dependencies installed (composer)")
			} else {
				platform.PrintWarningLine(os.Stdout, "Could not install composer dependencies")
			}
		}
	case platform.FileExists(filepath.Join(dir, "composer.json")):
		fmt.Println("  composer.json found but no composer.lock. Run `composer install` to install dependencies.")
	default:
		return false
	}
	return true
}

func installSwiftDeps(dir string) bool {
	if !platform.FileExists(filepath.Join(dir, "Package.swift")) {
		return false
	}
	if platform.Exists("swift") {
		if err := platform.RunQuietDir(dir, "swift", "package", "resolve"); err == nil {
			platform.PrintSuccess(os.Stdout, "Dependencies resolved (Swift PM)")
		} else {
			platform.PrintWarningLine(os.Stdout, "Could not resolve Swift package dependencies")
		}
	}
	return true
}

func installScalaDeps(dir string) bool {
	if !platform.FileExists(filepath.Join(dir, "build.sbt")) {
		return false
	}
	if platform.Exists("sbt") {
		if err := platform.RunQuietDir(dir, "sbt", "update"); err == nil {
			platform.PrintSuccess(os.Stdout, "Dependencies resolved (sbt)")
		} else {
			platform.PrintWarningLine(os.Stdout, "Could not resolve sbt dependencies")
		}
	}
	return true
}
