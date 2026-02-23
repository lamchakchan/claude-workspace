package setup

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

// NpmInstallInfo describes whether Claude Code was installed via npm.
type NpmInstallInfo struct {
	Detected         bool
	Path             string   // resolved path of claude binary (if detected via path)
	Source           string   // "path-heuristic", "asdf-shim", "npm-list", or ""
	AsdfNodeVersions []string // nodejs versions with claude installed (asdf only)
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
		// Tier 2: asdf shim — scan filesystem for npm-installed claude across all node versions
		if strings.Contains(claudePath, ".asdf/shims") {
			versions := findAsdfNodejsVersionsWithClaude()
			if len(versions) > 0 {
				return NpmInstallInfo{Detected: true, Path: claudePath, Source: "asdf-shim", AsdfNodeVersions: versions}
			}
			// Fall through to npm list if no versions found
		}
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
// When asdf node versions are specified, it sets ASDF_NODEJS_VERSION for each uninstall.
// After a successful uninstall, it runs "asdf reshim nodejs" if asdf is present.
func UninstallNpmClaude(info NpmInstallInfo) error {
	if len(info.AsdfNodeVersions) > 0 {
		// Uninstall from each asdf node version that has claude
		for _, ver := range info.AsdfNodeVersions {
			env := []string{"ASDF_NODEJS_VERSION=" + ver}
			err := platform.RunQuietWithEnv(env, "npm", "uninstall", "-g", "@anthropic-ai/claude-code")
			if err != nil {
				// Retry with sudo
				err = platform.RunQuietWithEnv(env, "sudo", "npm", "uninstall", "-g", "@anthropic-ai/claude-code")
				if err != nil {
					return fmt.Errorf("npm uninstall failed for node %s (tried with and without sudo): %w", ver, err)
				}
			}
		}
	} else {
		// Try without sudo
		err := platform.RunQuiet("npm", "uninstall", "-g", "@anthropic-ai/claude-code")
		if err != nil {
			// Retry with sudo
			err = platform.RunQuiet("sudo", "npm", "uninstall", "-g", "@anthropic-ai/claude-code")
			if err != nil {
				return fmt.Errorf("npm uninstall failed (tried with and without sudo): %w", err)
			}
		}
	}

	// Clean up asdf shims if asdf is present
	if platform.Exists("asdf") {
		_ = platform.RunQuiet("asdf", "reshim", "nodejs")
	}

	return nil
}

// findAsdfNodejsVersionsWithClaude scans all asdf-installed Node.js versions
// for a claude binary that was installed via npm.
func findAsdfNodejsVersionsWithClaude() []string {
	asdfDataDir := platform.AsdfDataDir()
	if asdfDataDir == "" {
		return nil
	}

	pattern := filepath.Join(asdfDataDir, "installs", "nodejs", "*", "bin", "claude")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return nil
	}

	var versions []string
	for _, match := range matches {
		// Resolve symlinks to confirm this is an npm install (path should contain node_modules)
		resolved, err := filepath.EvalSymlinks(match)
		if err != nil {
			continue
		}
		if !strings.Contains(resolved, "node_modules") {
			continue
		}

		ver := extractAsdfNodeVersion(match)
		if ver != "" {
			versions = append(versions, ver)
		}
	}

	return versions
}

// extractAsdfNodeVersion extracts the Node.js version from an asdf install path.
// For example, "/home/user/.asdf/installs/nodejs/20.5.1/bin/claude" returns "20.5.1".
func extractAsdfNodeVersion(path string) string {
	parts := strings.Split(path, string(filepath.Separator))
	for i, part := range parts {
		if part == "nodejs" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}
