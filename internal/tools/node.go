package tools

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

const (
	// NodeMinMajor is the minimum Node.js major version for MCP compatibility.
	NodeMinMajor = 18
	// NodeLTSVersion is the pinned LTS version for binary download fallback.
	NodeLTSVersion = "22.14.0"
)

// Node returns the Node.js tool definition.
func Node() Tool {
	return Tool{
		Name:     "node",
		Purpose:  "JavaScript runtime for MCP servers",
		Required: true,
		CheckFn: func() bool {
			return platform.Exists("node") && platform.Exists("npx") && nodeMeetsMinimum()
		},
		InstallFn: installNode,
		VersionFn: func() (string, error) {
			return platform.Output("node", "--version")
		},
	}
}

// nodeMeetsMinimum checks if the installed node version >= NodeMinMajor.
func nodeMeetsMinimum() bool {
	ver, err := platform.Output("node", "--version")
	if err != nil {
		return false
	}
	// ver looks like "v20.5.1"
	ver = strings.TrimPrefix(ver, "v")
	parts := strings.SplitN(ver, ".", 3)
	if len(parts) == 0 {
		return false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}
	return major >= NodeMinMajor
}

// installNode tries multiple strategies to install Node.js.
func installNode() error {
	// Priority 1: asdf (if present, user chose it as their runtime manager)
	if platform.Exists("asdf") {
		fmt.Println("  Detected asdf — installing Node.js via asdf...")
		if err := installNodeViaAsdf(); err == nil {
			return nil
		}
		fmt.Println("  asdf install failed, trying other methods...")
	}

	// Priority 2-6: system package manager
	pm := platform.DetectPackageManager()
	switch pm {
	case platform.PMBrew:
		fmt.Println("  Installing Node.js via Homebrew...")
		if err := platform.RunQuiet("brew", "install", "node"); err == nil {
			return nil
		}
	case platform.PMApt:
		fmt.Println("  Installing Node.js via NodeSource + apt...")
		if err := installNodeViaApt(); err == nil {
			return nil
		}
	case platform.PMDnf:
		fmt.Println("  Installing Node.js via dnf...")
		if err := platform.InstallSystemPackages(pm, []string{"nodejs", "npm"}); err == nil {
			return nil
		}
	case platform.PMPacman:
		fmt.Println("  Installing Node.js via pacman...")
		if err := platform.InstallSystemPackages(pm, []string{"nodejs", "npm"}); err == nil {
			return nil
		}
	case platform.PMApk:
		fmt.Println("  Installing Node.js via apk...")
		if err := platform.InstallSystemPackages(pm, []string{"nodejs", "npm"}); err == nil {
			return nil
		}
	}

	// Priority 7: binary download fallback
	fmt.Println("  Downloading Node.js binary from nodejs.org...")
	return installNodeBinary()
}

// installNodeViaAsdf installs Node.js using asdf version manager.
func installNodeViaAsdf() error {
	// Check if nodejs plugin is installed
	out, err := platform.Output("asdf", "plugin", "list")
	if err != nil || !strings.Contains(out, "nodejs") {
		if err := platform.RunQuiet("asdf", "plugin", "add", "nodejs"); err != nil {
			return fmt.Errorf("asdf plugin add nodejs: %w", err)
		}
	}

	// Install latest Node.js
	if err := platform.Run("asdf", "install", "nodejs", "latest"); err != nil {
		return fmt.Errorf("asdf install nodejs latest: %w", err)
	}

	// Set as home default
	if err := platform.RunQuiet("asdf", "set", "--home", "nodejs", "latest"); err != nil {
		return fmt.Errorf("asdf set nodejs: %w", err)
	}

	// Reshim to create shims for node, npm, npx
	if err := platform.RunQuiet("asdf", "reshim", "nodejs"); err != nil {
		return fmt.Errorf("asdf reshim nodejs: %w", err)
	}

	return nil
}

// installNodeViaApt installs Node.js using the NodeSource repository for modern LTS.
func installNodeViaApt() error {
	// Run NodeSource setup script
	setupCmd := "curl -fsSL https://deb.nodesource.com/setup_22.x | "
	if platform.Exists("sudo") {
		setupCmd += "sudo -E bash -"
	} else {
		setupCmd += "bash -"
	}
	if err := platform.RunQuiet("bash", "-c", setupCmd); err != nil {
		return fmt.Errorf("NodeSource setup: %w", err)
	}

	// Install nodejs (includes npm)
	return platform.InstallSystemPackages(platform.PMApt, []string{"nodejs"})
}

// installNodeBinary downloads and installs a Node.js binary from nodejs.org.
func installNodeBinary() error {
	osName := runtime.GOOS
	arch := runtime.GOARCH

	// Map Go arch names to Node.js naming
	switch arch {
	case "amd64":
		arch = "x64"
	case "386":
		arch = "x86"
	}

	// Linux uses .tar.xz, macOS uses .tar.gz
	ext := "tar.gz"
	if osName == "linux" {
		ext = "tar.xz"
	}

	url := fmt.Sprintf("https://nodejs.org/dist/v%s/node-v%s-%s-%s.%s",
		NodeLTSVersion, NodeLTSVersion, osName, arch, ext)

	// Download to temp directory
	tmpDir, err := os.MkdirTemp("", "node-install-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, "node."+ext)
	if err := downloadFile(url, archivePath); err != nil {
		return fmt.Errorf("downloading Node.js: %w", err)
	}

	// Install to ~/.local/lib/nodejs
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home dir: %w", err)
	}

	nodeDir := filepath.Join(home, ".local", "lib", "nodejs")
	if err := os.MkdirAll(nodeDir, 0755); err != nil {
		return fmt.Errorf("creating nodejs dir: %w", err)
	}

	// Extract archive
	prefix := fmt.Sprintf("node-v%s-%s-%s", NodeLTSVersion, osName, arch)
	if err := extractNodeArchive(archivePath, nodeDir, prefix); err != nil {
		return fmt.Errorf("extracting Node.js: %w", err)
	}

	// Create symlinks in ~/.local/bin
	localBin := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(localBin, 0755); err != nil {
		return fmt.Errorf("creating local bin dir: %w", err)
	}

	nodeBinDir := filepath.Join(nodeDir, "bin")
	for _, bin := range []string{"node", "npm", "npx"} {
		src := filepath.Join(nodeBinDir, bin)
		dst := filepath.Join(localBin, bin)
		if platform.FileExists(src) {
			os.Remove(dst) // remove existing symlink if any
			if err := os.Symlink(src, dst); err != nil {
				return fmt.Errorf("symlinking %s: %w", bin, err)
			}
		}
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

// downloadFile downloads a URL to a local file path.
func downloadFile(url, dest string) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, url)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// extractNodeArchive extracts bin/ contents from a Node.js tar.gz archive into destDir.
// The prefix is the top-level directory name inside the archive (e.g., "node-v22.14.0-darwin-arm64").
func extractNodeArchive(archive, destDir, prefix string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()

	var reader io.Reader
	if strings.HasSuffix(archive, ".tar.gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer gz.Close()
		reader = gz
	} else {
		// .tar.xz — use external xz command
		return extractNodeArchiveXZ(archive, destDir, prefix)
	}

	tr := tar.NewReader(reader)
	binPrefix := prefix + "/bin/"

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Only extract files from bin/ directory
		if !strings.HasPrefix(header.Name, binPrefix) {
			continue
		}

		relPath := strings.TrimPrefix(header.Name, binPrefix)
		if relPath == "" {
			continue
		}

		destPath := filepath.Join(destDir, "bin", relPath)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return err
			}
			outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return err
			}
			os.Remove(destPath)
			if err := os.Symlink(header.Linkname, destPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// extractNodeArchiveXZ handles .tar.xz archives using the system xz command.
func extractNodeArchiveXZ(archive, destDir, prefix string) error {
	// Extract to a temp directory, then move the bin/ contents
	tmpDir, err := os.MkdirTemp("", "node-extract-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// Use tar with xz support
	if err := platform.RunQuiet("tar", "-xf", archive, "-C", tmpDir); err != nil {
		return fmt.Errorf("tar extract: %w", err)
	}

	// Move bin/ contents to destDir
	srcBin := filepath.Join(tmpDir, prefix, "bin")
	dstBin := filepath.Join(destDir, "bin")
	if err := os.MkdirAll(dstBin, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(srcBin)
	if err != nil {
		return fmt.Errorf("reading extracted bin dir: %w", err)
	}

	for _, entry := range entries {
		src := filepath.Join(srcBin, entry.Name())
		dst := filepath.Join(dstBin, entry.Name())
		os.Remove(dst) // remove existing if any

		// Read symlink target or copy file
		if entry.Type()&os.ModeSymlink != 0 {
			target, err := os.Readlink(src)
			if err != nil {
				return err
			}
			if err := os.Symlink(target, dst); err != nil {
				return err
			}
		} else {
			if err := platform.CopyFile(src, dst); err != nil {
				return err
			}
		}
	}

	return nil
}
