package upgrade

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

// ReplaceBinary replaces the currently installed binary with a new one using
// atomic rename. If the install directory requires elevated permissions,
// it falls back to sudo mv.
func ReplaceBinary(newBinaryPath string) error {
	// Resolve the actual install path (follow symlinks)
	currentExec, err := os.Executable()
	if err != nil {
		return fmt.Errorf("determining current executable: %w", err)
	}
	installPath, err := filepath.EvalSymlinks(currentExec)
	if err != nil {
		return fmt.Errorf("resolving symlinks: %w", err)
	}

	// Stage new binary next to install path (same filesystem for atomic rename)
	tmpPath := installPath + ".upgrade-tmp"
	defer os.Remove(tmpPath) // clean up on any failure path

	if err := platform.CopyFile(newBinaryPath, tmpPath); err != nil {
		return fmt.Errorf("staging new binary: %w", err)
	}
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return fmt.Errorf("setting permissions: %w", err)
	}

	// Verify new binary works
	ver, err := platform.Output(tmpPath, "--version")
	if err != nil {
		return fmt.Errorf("new binary failed --version check: %w", err)
	}
	fmt.Printf("  Verified new binary: %s\n", ver)

	// Atomic rename
	if err := os.Rename(tmpPath, installPath); err != nil {
		// Fallback to sudo mv for permission-restricted directories
		fmt.Println("  Attempting elevated install (sudo)...")
		if sudoErr := platform.Run("sudo", "mv", tmpPath, installPath); sudoErr != nil {
			return fmt.Errorf("could not replace binary (tried rename and sudo mv): %w", err)
		}
	}

	return nil
}
