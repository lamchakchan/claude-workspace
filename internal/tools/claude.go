package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

// ClaudeInstallCmd is the shell command to install Claude Code CLI.
const ClaudeInstallCmd = "curl -fsSL https://claude.ai/install.sh | bash"

// Claude returns the Claude Code CLI tool definition.
func Claude() Tool {
	return Tool{
		Name:       "claude",
		Purpose:    "Claude Code CLI",
		Required:   true,
		InstallCmd: ClaudeInstallCmd,
		InstallFn:  installClaude,
		VersionFn: func() (string, error) {
			return platform.Output("claude", "--version")
		},
	}
}

func installClaude() error {
	fmt.Println("  Installing Claude Code via official installer...")
	if err := platform.Run("bash", "-c", ClaudeInstallCmd); err != nil {
		fmt.Fprintln(os.Stderr, "  Failed to install Claude Code automatically.")
		fmt.Println("  Please install manually: " + ClaudeInstallCmd)
		fmt.Println("  Or visit: https://docs.anthropic.com/en/docs/claude-code")
		os.Exit(1)
	}

	// The installer places claude in ~/.local/bin which may not be in PATH.
	// Add it to PATH for the current process so subsequent steps can find it.
	home, _ := os.UserHomeDir()
	localBin := filepath.Join(home, ".local", "bin")
	if platform.FileExists(filepath.Join(localBin, "claude")) {
		os.Setenv("PATH", localBin+":"+os.Getenv("PATH"))

		fmt.Println("  Configuring claude in PATH...")

		// Strategy 1: Symlink to /usr/local/bin (works immediately, no shell restart)
		if err := symlinkClaudeBinary(localBin); err == nil {
			fmt.Println("  Symlinked claude → /usr/local/bin/claude (available immediately).")
		} else {
			// Strategy 2: Append to shell RC file
			rcPath, shellName := detectShellRC(home)
			if modified, err := appendPathToRC(home, shellName, rcPath); err != nil {
				fmt.Printf("  Warning: could not auto-configure PATH: %v\n", err)
				fmt.Println("  To fix manually, run:")
				fmt.Println("    echo 'export PATH=\"$HOME/.local/bin:$PATH\"' >> ~/." + shellName + "rc")
			} else if modified {
				fmt.Printf("  Added ~/.local/bin to PATH in %s\n", filepath.Base(rcPath))
				fmt.Printf("  Restart your shell or run: source %s\n", rcPath)
			}
		}

		fmt.Println("  Note: Ignore any PATH warning above — already handled.")
	}

	fmt.Println("  Claude Code installed successfully.")
	return nil
}

// detectShellRC determines the user's shell RC file path and shell name.
// It checks $SHELL first, then falls back to file-existence checks.
func detectShellRC(home string) (rcPath, shellName string) {
	shell := os.Getenv("SHELL")

	switch {
	case strings.HasSuffix(shell, "zsh"):
		return filepath.Join(home, ".zshrc"), "zsh"
	case strings.HasSuffix(shell, "fish"):
		return filepath.Join(home, ".config", "fish", "config.fish"), "fish"
	case strings.HasSuffix(shell, "bash"):
		return filepath.Join(home, ".bashrc"), "bash"
	}

	// Fallback: check if .zshrc exists (common on macOS)
	if platform.FileExists(filepath.Join(home, ".zshrc")) {
		return filepath.Join(home, ".zshrc"), "zsh"
	}

	// Default to bash
	return filepath.Join(home, ".bashrc"), "bash"
}

// symlinkClaudeBinary creates a symlink from ~/.local/bin/claude to /usr/local/bin/claude.
// Returns nil on success; returns an error if both direct and sudo symlink fail.
func symlinkClaudeBinary(localBin string) error {
	src := filepath.Join(localBin, "claude")
	dst := "/usr/local/bin/claude"

	// Skip if /usr/local/bin/claude already exists and resolves to the right target
	if target, err := os.Readlink(dst); err == nil && target == src {
		return nil
	}

	// Try direct symlink
	if err := platform.SymlinkFile(src, dst); err == nil {
		return nil
	}

	// Fall back to sudo ln -sf
	if err := platform.RunQuiet("sudo", "ln", "-sf", src, dst); err != nil {
		return fmt.Errorf("symlink failed (direct and sudo): %w", err)
	}
	return nil
}

// appendPathToRC adds ~/.local/bin to PATH in the given shell RC file.
// For fish, it uses fish_add_path. For bash/zsh, it appends an export line.
// Returns (true, nil) if the file was modified, (false, nil) if already configured.
func appendPathToRC(home, shellName, rcPath string) (modified bool, err error) {
	// Fish uses a different mechanism
	if shellName == "fish" {
		fishPath := filepath.Join(home, ".local", "bin")
		if err := platform.RunQuiet("fish", "-c", "fish_add_path "+fishPath); err != nil {
			return false, fmt.Errorf("fish_add_path failed: %w", err)
		}
		return true, nil
	}

	// bash/zsh: check idempotency
	pathLine := "\n# Added by claude-workspace setup\nexport PATH=\"$HOME/.local/bin:$PATH\"\n"

	content, err := os.ReadFile(rcPath)
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("reading %s: %w", rcPath, err)
	}

	if strings.Contains(string(content), ".local/bin") {
		return false, nil // already configured
	}

	// Ensure parent directory exists (relevant for new files)
	if err := os.MkdirAll(filepath.Dir(rcPath), 0755); err != nil {
		return false, fmt.Errorf("creating directory for %s: %w", rcPath, err)
	}

	if err := os.WriteFile(rcPath, append(content, []byte(pathLine)...), 0644); err != nil {
		return false, fmt.Errorf("writing %s: %w", rcPath, err)
	}
	return true, nil
}
