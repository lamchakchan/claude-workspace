package tools

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

// Engram returns the engram tool definition.
func Engram() Tool {
	return Tool{
		Name:       "engram",
		Purpose:    "Persistent cross-project memory (FTS5 SQLite)",
		Required:   true,
		InstallCmd: "brew install gentleman-programming/tap/engram",
		CheckFn: func() bool {
			return platform.Exists("engram")
		},
		InstallFn: installEngram,
		VersionFn: func() (string, error) {
			return platform.Output("engram", "--version")
		},
	}
}

// installEngram tries Homebrew first, then falls back to GitHub binary download.
func installEngram() error {
	// Priority 1: Homebrew (macOS or Linux with brew)
	if platform.Exists("brew") {
		fmt.Println("  Installing engram via Homebrew...")
		if err := platform.RunQuiet("brew", "install", "gentleman-programming/tap/engram"); err == nil {
			return nil
		}
		fmt.Println("  Homebrew install failed, trying binary download...")
	}

	// Priority 2: GitHub binary download
	fmt.Println("  Downloading engram binary from GitHub...")
	return installEngramBinary()
}

// installEngramBinary downloads the latest engram release from GitHub.
func installEngramBinary() error {
	// Fetch latest release tag
	version, err := engramLatestVersion()
	if err != nil {
		return fmt.Errorf("fetching latest engram version: %w", err)
	}

	osName := runtime.GOOS
	arch := runtime.GOARCH

	// Download pattern: engram_{version}_{os}_{arch}.tar.gz
	// Strip leading "v" from version for the filename
	ver := strings.TrimPrefix(version, "v")
	url := fmt.Sprintf("https://github.com/Gentleman-Programming/engram/releases/download/%s/engram_%s_%s_%s.tar.gz",
		version, ver, osName, arch)

	// Download to temp directory
	tmpDir, err := os.MkdirTemp("", "engram-install-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, "engram.tar.gz")
	if err := downloadFile(url, archivePath); err != nil {
		return fmt.Errorf("downloading engram: %w", err)
	}

	// Extract the binary
	if err := platform.RunQuiet("tar", "-xzf", archivePath, "-C", tmpDir); err != nil {
		return fmt.Errorf("extracting engram: %w", err)
	}

	// Install to ~/.local/bin
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home dir: %w", err)
	}

	localBin := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(localBin, 0755); err != nil {
		return fmt.Errorf("creating local bin dir: %w", err)
	}

	src := filepath.Join(tmpDir, "engram")
	dst := filepath.Join(localBin, "engram")
	if err := platform.CopyFile(src, dst); err != nil {
		return fmt.Errorf("copying engram binary: %w", err)
	}
	if err := os.Chmod(dst, 0755); err != nil {
		return fmt.Errorf("making engram executable: %w", err)
	}

	// Update PATH for current process
	currentPath := os.Getenv("PATH")
	if !strings.Contains(currentPath, localBin) {
		os.Setenv("PATH", localBin+":"+currentPath)
	}

	// Configure shell RC for persistence
	rcPath, shellName := platform.DetectShellRC(home)
	if modified, err := platform.AppendPathToRC(home, shellName, rcPath); err != nil {
		fmt.Printf("  Warning: could not auto-configure PATH: %v\n", err)
	} else if modified {
		fmt.Printf("  Added ~/.local/bin to PATH in %s\n", filepath.Base(rcPath))
	}

	return nil
}

// engramLatestVersion fetches the latest release tag from GitHub.
func engramLatestVersion() (string, error) {
	resp, err := http.Get("https://api.github.com/repos/Gentleman-Programming/engram/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("parsing release JSON: %w", err)
	}

	if release.TagName == "" {
		return "", fmt.Errorf("no tag_name in release response")
	}

	return release.TagName, nil
}
