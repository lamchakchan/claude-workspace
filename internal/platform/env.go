package platform

import (
	"os"
	"path/filepath"
	"strings"
)

// DetectShellRC determines the user's shell RC file path and shell name.
// It checks $SHELL first, then falls back to file-existence checks.
func DetectShellRC(home string) (rcPath, shellName string) {
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
	if FileExists(filepath.Join(home, ".zshrc")) {
		return filepath.Join(home, ".zshrc"), "zsh"
	}

	// Default to bash
	return filepath.Join(home, ".bashrc"), "bash"
}

// AppendPathToRC adds ~/.local/bin to PATH in the given shell RC file.
// For fish, it uses fish_add_path. For bash/zsh, it appends an export line.
// Returns (true, nil) if the file was modified, (false, nil) if already configured.
func AppendPathToRC(home, shellName, rcPath string) (modified bool, err error) {
	// Fish uses a different mechanism
	if shellName == "fish" {
		fishPath := filepath.Join(home, ".local", "bin")
		if err := RunQuiet("fish", "-c", "fish_add_path "+fishPath); err != nil {
			return false, err
		}
		return true, nil
	}

	// bash/zsh: check idempotency
	pathLine := "\n# Added by claude-workspace setup\nexport PATH=\"$HOME/.local/bin:$PATH\"\n"

	content, err := os.ReadFile(rcPath)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}

	if strings.Contains(string(content), ".local/bin") {
		return false, nil // already configured
	}

	// Ensure parent directory exists (relevant for new files)
	if err := os.MkdirAll(filepath.Dir(rcPath), 0755); err != nil {
		return false, err
	}

	if err := os.WriteFile(rcPath, append(content, []byte(pathLine)...), 0644); err != nil {
		return false, err
	}
	return true, nil
}

// AsdfDataDir returns the asdf data directory, checking $ASDF_DATA_DIR first
// and falling back to ~/.asdf.
func AsdfDataDir() string {
	if dir := os.Getenv("ASDF_DATA_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".asdf")
}
