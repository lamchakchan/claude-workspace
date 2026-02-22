package setup

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

// NpmInstallInfo describes whether Claude Code was installed via npm.
type NpmInstallInfo struct {
	Detected bool
	Path     string // resolved path of claude binary (if detected via path)
	Source   string // "path-heuristic", "npm-list", or ""
}

// DetectNpmClaude checks whether the claude binary in PATH was installed via npm.
// It uses a three-tier strategy:
//  1. Path heuristic: check if resolved path contains "node_modules"
//  2. asdf shim heuristic: if path contains ".asdf/shims", fall through to npm list
//  3. npm list fallback: run "npm list -g @anthropic-ai/claude-code --depth=0"
func DetectNpmClaude() NpmInstallInfo {
	claudePath, err := exec.LookPath("claude")
	if err == nil {
		// Tier 1: path contains node_modules → definitely npm
		if strings.Contains(claudePath, "node_modules") {
			return NpmInstallInfo{Detected: true, Path: claudePath, Source: "path-heuristic"}
		}
		// Tier 2: asdf shim — can't tell from path alone, fall through to npm list
		// (no early return here)
	}

	// Tier 3: npm list fallback — definitive check
	if !platform.Exists("npm") {
		return NpmInstallInfo{}
	}
	out, listErr := platform.Output("npm", "list", "-g", "@anthropic-ai/claude-code", "--depth=0")
	if listErr == nil && strings.Contains(out, "@anthropic-ai/claude-code") {
		return NpmInstallInfo{Detected: true, Path: claudePath, Source: "npm-list"}
	}

	return NpmInstallInfo{}
}

// UninstallNpmClaude removes the npm-installed Claude Code package.
// It tries without sudo first, then retries with sudo on failure.
// After a successful uninstall, it runs "asdf reshim nodejs" if asdf is present.
func UninstallNpmClaude() error {
	// Try without sudo
	err := platform.RunQuiet("npm", "uninstall", "-g", "@anthropic-ai/claude-code")
	if err != nil {
		// Retry with sudo
		err = platform.RunQuiet("sudo", "npm", "uninstall", "-g", "@anthropic-ai/claude-code")
		if err != nil {
			return fmt.Errorf("npm uninstall failed (tried with and without sudo): %w", err)
		}
	}

	// Clean up asdf shims if asdf is present
	if platform.Exists("asdf") {
		_ = platform.RunQuiet("asdf", "reshim", "nodejs")
	}

	return nil
}
