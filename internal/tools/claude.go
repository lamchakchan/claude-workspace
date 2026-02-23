package tools

import (
	"fmt"
	"os"
	"path/filepath"

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
			rcPath, shellName := platform.DetectShellRC(home)
			if modified, err := platform.AppendPathToRC(home, shellName, rcPath); err != nil {
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
