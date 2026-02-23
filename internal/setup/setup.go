package setup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

var (
	claudeHome   string
	claudeConfig string
)

func init() {
	home, _ := os.UserHomeDir()
	claudeHome = filepath.Join(home, ".claude")
	claudeConfig = filepath.Join(home, ".claude.json")
}

func Run() error {
	platform.PrintBanner(os.Stdout, "Claude Code Platform Setup")

	// Step 1: Check if Claude Code CLI is installed
	platform.PrintStep(os.Stdout, 1, 7, "Checking Claude Code installation...")

	// Check for npm-installed Claude that needs cleanup
	npmInfo := DetectNpmClaude()
	if npmInfo.Detected {
		fmt.Printf("  Detected Claude Code installed via npm (source: %s).\n", npmInfo.Source)
		fmt.Println("  Removing npm version before installing official binary...")
		if err := UninstallNpmClaude(npmInfo); err != nil {
			platform.PrintWarningLine(os.Stdout, fmt.Sprintf("could not remove npm Claude: %v", err))
			fmt.Println("  Please run manually: npm uninstall -g @anthropic-ai/claude-code")
			fmt.Println("  Then re-run: claude-workspace setup")
			return fmt.Errorf("npm Claude uninstall failed: %w", err)
		}
		fmt.Println("  npm Claude Code removed successfully.")
	}

	if platform.Exists("claude") {
		ver, _ := platform.Output("claude", "--version")
		fmt.Printf("  Claude Code CLI found: %s\n", ver)
	} else {
		fmt.Println("  Claude Code CLI not found. Installing...")
		if err := installClaude(); err != nil {
			return err
		}
	}

	// Step 2: API Key provisioning
	platform.PrintStep(os.Stdout, 2, 7, "API Key provisioning...")
	if err := provisionApiKey(); err != nil {
		return err
	}

	// Step 3: Create global user settings
	platform.PrintStep(os.Stdout, 3, 7, "Setting up global user configuration...")
	if err := setupGlobalSettings(); err != nil {
		return err
	}

	// Step 4: Create global CLAUDE.md
	platform.PrintStep(os.Stdout, 4, 7, "Setting up global CLAUDE.md...")
	if err := setupGlobalClaudeMd(); err != nil {
		return err
	}

	// Step 5: Install binary to PATH
	platform.PrintStep(os.Stdout, 5, 7, "Installing claude-workspace to PATH...")
	installBinaryToPath()

	// Step 6: Register user-scoped MCP servers
	platform.PrintStep(os.Stdout, 6, 7, "Registering user-scoped MCP servers...")
	if err := setupUserMCPServers(); err != nil {
		platform.PrintWarningLine(os.Stdout, fmt.Sprintf("MCP server registration skipped: %v", err))
	}

	// Step 7: Check optional system tools
	platform.PrintStep(os.Stdout, 7, 7, "Checking optional system tools...")
	checkOptionalTools()

	platform.PrintBanner(os.Stdout, "Setup Complete")
	fmt.Println("\nNext steps:")
	fmt.Println()
	platform.PrintCommand(os.Stdout, "claude-workspace attach /path/to/project")
	platform.PrintCommand(os.Stdout, "cd /path/to/project && claude")
	platform.PrintCommand(os.Stdout, "claude-workspace mcp add <name> -- <command>")
	fmt.Println()

	return nil
}

func installClaude() error {
	fmt.Println("  Installing Claude Code via official installer...")
	if err := platform.Run("bash", "-c", "curl -fsSL https://claude.ai/install.sh | bash"); err != nil {
		fmt.Fprintln(os.Stderr, "  Failed to install Claude Code automatically.")
		fmt.Println("  Please install manually: curl -fsSL https://claude.ai/install.sh | bash")
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

func provisionApiKey() error {
	if platform.FileExists(claudeConfig) {
		var config map[string]json.RawMessage
		if err := platform.ReadJSONFile(claudeConfig, &config); err == nil {
			if _, hasOAuth := config["oauthAccount"]; hasOAuth {
				fmt.Println("  Already authenticated. Skipping API key provisioning.")
				return nil
			}
			if _, hasKey := config["primaryApiKey"]; hasKey {
				fmt.Println("  Already authenticated. Skipping API key provisioning.")
				return nil
			}
		}
	}

	fmt.Println("  Starting self-service API key provisioning (Option 2)...")
	fmt.Println("  This will open Claude Code's interactive login flow.")
	fmt.Println("  Select 'Use an API key' when prompted.")
	fmt.Println()

	exitCode, err := platform.RunSpawn("claude", "--print-api-key-config")
	if err != nil || exitCode != 0 {
		fmt.Println("\n  API key provisioning requires interactive setup.")
		fmt.Println("  Run 'claude' directly to complete the login flow.")
		fmt.Println("  You can set ANTHROPIC_API_KEY in your environment as an alternative.")
	}

	return nil
}

func setupGlobalSettings() error {
	settingsPath := filepath.Join(claudeHome, "settings.json")

	defaults := GetDefaultGlobalSettings()

	if platform.FileExists(settingsPath) {
		fmt.Println("  Global settings already exist. Merging platform defaults...")
		var existing map[string]interface{}
		if err := platform.ReadJSONFile(settingsPath, &existing); err != nil {
			fmt.Println("  Could not merge settings. Skipping global settings update.")
			return nil
		}
		merged := MergeSettings(existing, defaults)
		if err := platform.WriteJSONFile(settingsPath, merged); err != nil {
			return fmt.Errorf("writing global settings: %w", err)
		}
		fmt.Println("  Global settings updated.")
		return nil
	}

	// Create ~/.claude/ directory if needed
	if err := os.MkdirAll(claudeHome, 0755); err != nil {
		return fmt.Errorf("creating ~/.claude: %w", err)
	}

	if err := platform.WriteJSONFile(settingsPath, defaults); err != nil {
		return fmt.Errorf("writing global settings: %w", err)
	}
	fmt.Println("  Global settings created at ~/.claude/settings.json")
	return nil
}

func GetDefaultGlobalSettings() map[string]interface{} {
	return map[string]interface{}{
		"$schema": "https://json.schemastore.org/claude-code-settings.json",
		"env": map[string]interface{}{
			"CLAUDE_CODE_ENABLE_TELEMETRY":         "1",
			"CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS": "1",
			"CLAUDE_CODE_ENABLE_TASKS":             "true",
			"CLAUDE_CODE_SUBAGENT_MODEL":           "sonnet",
			"CLAUDE_AUTOCOMPACT_PCT_OVERRIDE":      "80",
		},
		"permissions": map[string]interface{}{
			"deny": []string{
				"Bash(rm -rf /)",
				"Bash(rm -rf /*)",
				"Bash(git push --force * main)",
				"Bash(git push --force * master)",
				"Bash(git push -f * main)",
				"Bash(git push -f * master)",
				"Read(./.env)",
				"Read(./.env.*)",
				"Read(./secrets/**)",
			},
		},
		"alwaysThinkingEnabled": true,
		"showTurnDuration":      true,
	}
}

func MergeSettings(existing, defaults map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	for k, v := range existing {
		merged[k] = v
	}

	// Merge env vars (don't overwrite existing)
	if defaultEnv, ok := defaults["env"].(map[string]interface{}); ok {
		existingEnv, _ := existing["env"].(map[string]interface{})
		if existingEnv == nil {
			existingEnv = make(map[string]interface{})
		}
		mergedEnv := make(map[string]interface{})
		for k, v := range defaultEnv {
			mergedEnv[k] = v
		}
		for k, v := range existingEnv {
			mergedEnv[k] = v // existing takes precedence
		}
		merged["env"] = mergedEnv
	}

	// Merge deny permissions (union)
	if defaultPerms, ok := defaults["permissions"].(map[string]interface{}); ok {
		if defaultDeny, ok := defaultPerms["deny"].([]string); ok {
			existingPerms, _ := existing["permissions"].(map[string]interface{})
			if existingPerms == nil {
				existingPerms = make(map[string]interface{})
			}

			var existingDeny []string
			if ed, ok := existingPerms["deny"]; ok {
				switch v := ed.(type) {
				case []string:
					existingDeny = v
				case []interface{}:
					for _, item := range v {
						if s, ok := item.(string); ok {
							existingDeny = append(existingDeny, s)
						}
					}
				}
			}

			denySet := make(map[string]bool)
			for _, rule := range existingDeny {
				denySet[rule] = true
			}

			combined := make([]string, len(existingDeny))
			copy(combined, existingDeny)
			for _, rule := range defaultDeny {
				if !denySet[rule] {
					combined = append(combined, rule)
				}
			}

			mergedPerms := make(map[string]interface{})
			for k, v := range existingPerms {
				mergedPerms[k] = v
			}
			mergedPerms["deny"] = combined
			merged["permissions"] = mergedPerms
		}
	}

	// Set boolean flags only if not already set
	for _, key := range []string{"alwaysThinkingEnabled", "showTurnDuration"} {
		if _, exists := merged[key]; !exists {
			if v, ok := defaults[key]; ok {
				merged[key] = v
			}
		}
	}

	return merged
}

func setupGlobalClaudeMd() error {
	claudeMdPath := filepath.Join(claudeHome, "CLAUDE.md")

	if platform.FileExists(claudeMdPath) {
		fmt.Println("  Global CLAUDE.md already exists. Skipping.")
		return nil
	}

	content := `# Global Claude Code Instructions

## Identity
You are an AI coding agent operating within a governed platform environment.
Follow the platform conventions, use subagents for delegation, and plan before implementing.

## Defaults
- Always use TodoWrite for multi-step tasks
- Prefer Sonnet for coding, Haiku for exploration
- Read files before modifying them
- Run tests after making changes
- Never commit secrets or credentials

## Git Conventions
- Work on feature branches, never main/master
- Commit messages: imperative mood, explain "why"
- Create PRs with clear descriptions
`

	if err := os.WriteFile(claudeMdPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing CLAUDE.md: %w", err)
	}
	fmt.Println("  Global CLAUDE.md created at ~/.claude/CLAUDE.md")
	return nil
}

// platformMCPServers returns the user-scoped MCP servers the platform registers by default.
func platformMCPServers() map[string]interface{} {
	return map[string]interface{}{
		"memory": map[string]interface{}{
			"command": "npx",
			"args":    []string{"-y", "@anthropic/claude-code-memory-server"},
			"env":     map[string]interface{}{},
		},
		"git": map[string]interface{}{
			"command": "npx",
			"args":    []string{"-y", "@modelcontextprotocol/server-git"},
			"env":     map[string]interface{}{},
		},
	}
}

// MergeUserMCPServers merges platform MCP servers into an existing ~/.claude.json config map.
// Only adds servers that are not already present; never overwrites existing entries.
func MergeUserMCPServers(config map[string]interface{}, servers map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	for k, v := range config {
		merged[k] = v
	}

	existing, _ := merged["mcpServers"].(map[string]interface{})
	if existing == nil {
		existing = make(map[string]interface{})
	}

	result := make(map[string]interface{})
	for k, v := range existing {
		result[k] = v
	}
	for name, cfg := range servers {
		if _, alreadySet := result[name]; !alreadySet {
			result[name] = cfg
		}
	}

	merged["mcpServers"] = result
	return merged
}

func setupUserMCPServers() error {
	if !platform.Exists("npx") {
		fmt.Println("  npx not found — skipping automatic MCP server registration.")
		fmt.Println("  Install Node.js, then run manually:")
		fmt.Println("    claude mcp add --scope user memory -- npx -y @anthropic/claude-code-memory-server")
		fmt.Println("    claude mcp add --scope user git -- npx -y @modelcontextprotocol/server-git")
		return nil
	}

	var config map[string]interface{}
	if platform.FileExists(claudeConfig) {
		if err := platform.ReadJSONFile(claudeConfig, &config); err != nil {
			return fmt.Errorf("reading %s: %w", claudeConfig, err)
		}
	}
	if config == nil {
		config = make(map[string]interface{})
	}

	merged := MergeUserMCPServers(config, platformMCPServers())

	if err := platform.WriteJSONFile(claudeConfig, merged); err != nil {
		return fmt.Errorf("writing %s: %w", claudeConfig, err)
	}

	// Report what was registered vs already present
	existing, _ := config["mcpServers"].(map[string]interface{})
	var added, skipped []string
	for name := range platformMCPServers() {
		if existing != nil {
			if _, found := existing[name]; found {
				skipped = append(skipped, name)
				continue
			}
		}
		added = append(added, name)
	}

	if len(added) > 0 {
		platform.PrintOK(os.Stdout, fmt.Sprintf("Registered: %s", joinStrings(added, ", ")))
	}
	if len(skipped) > 0 {
		fmt.Printf("  Already registered: %s\n", joinStrings(skipped, ", "))
	}

	return nil
}

func installBinaryToPath() {
	execPath, err := os.Executable()
	if err != nil {
		fmt.Println("  Could not determine binary path. Skipping PATH installation.")
		return
	}

	// Check if already in a standard PATH location
	if _, err := platform.Output("which", "claude-workspace"); err == nil {
		fmt.Println("  claude-workspace is already in PATH.")
		return
	}

	installDir := "/usr/local/bin"
	destPath := filepath.Join(installDir, "claude-workspace")

	// Try to copy the binary
	fmt.Printf("  Installing to %s...\n", destPath)
	if err := platform.CopyFile(execPath, destPath); err != nil {
		// Try with sudo
		if err := platform.Run("sudo", "cp", execPath, destPath); err != nil {
			fmt.Printf("  Could not install to %s (permission denied).\n", installDir)
			fmt.Printf("  To install manually:\n")
			fmt.Printf("    sudo cp %s %s\n", execPath, destPath)
			return
		}
		// Make executable
		platform.RunQuiet("sudo", "chmod", "+x", destPath)
	} else {
		os.Chmod(destPath, 0755)
	}
	fmt.Println("  Installed: claude-workspace is now available globally.")
}

func checkOptionalTools() {
	type tool struct {
		name    string
		purpose string
		install string
	}

	tools := []tool{
		{"shellcheck", "Hook script validation", ""},
		{"jq", "JSON processing in hooks", ""},
		{"prettier", "Auto-format hook (JS/TS/JSON/CSS)", "npm install -g prettier"},
		{"tmux", "Agent teams split-pane mode", ""},
	}

	// Detect package manager for system tools
	hasSudo := platform.Exists("sudo")
	var pkgInstall func(name string) string
	if runtime.GOOS == "darwin" && platform.Exists("brew") {
		pkgInstall = func(name string) string { return "brew install " + name }
	} else if platform.Exists("apt-get") {
		if hasSudo {
			pkgInstall = func(name string) string { return "sudo apt-get install -y " + name }
		} else {
			pkgInstall = func(name string) string { return "apt-get install -y " + name }
		}
	}

	// Set install commands
	for i := range tools {
		if tools[i].install == "" {
			if pkgInstall != nil {
				tools[i].install = pkgInstall(tools[i].name)
			} else if runtime.GOOS == "darwin" {
				tools[i].install = "brew install " + tools[i].name + " (macOS) / apt install " + tools[i].name + " (Linux)"
			} else {
				tools[i].install = "apt install " + tools[i].name
			}
		}
	}

	var missing []tool
	var found []string

	for _, t := range tools {
		if platform.Exists(t.name) {
			found = append(found, t.name)
		} else {
			missing = append(missing, t)
		}
	}

	if len(found) > 0 {
		platform.PrintSuccess(os.Stdout, fmt.Sprintf("Found: %s", joinStrings(found, ", ")))
	}

	if len(missing) > 0 {
		// Attempt to install missing tools via package manager
		var stillMissing []tool
		if pkgInstall != nil {
			var installable []string
			for _, t := range missing {
				if t.name != "prettier" { // prettier uses npm, handled below
					installable = append(installable, t.name)
				}
			}
			if len(installable) > 0 {
				fmt.Printf("\n  Attempting to install: %s\n", joinStrings(installable, ", "))
				if runtime.GOOS == "darwin" && platform.Exists("brew") {
					args := append([]string{"install"}, installable...)
					if err := platform.RunQuiet("brew", args...); err == nil {
						fmt.Println("  Installed successfully via brew.")
					}
				} else if platform.Exists("apt-get") {
					args := append([]string{"apt-get", "install", "-y"}, installable...)
					if hasSudo {
						args = append([]string{"sudo"}, args...)
					}
					platform.RunQuiet(args[0], args[1:]...)
				}
			}
		}

		// Auto-install prettier via npm if npm is available
		if !platform.Exists("prettier") && platform.Exists("npm") {
			fmt.Println("\n  Installing prettier via npm (used by auto-format hook)...")
			if err := platform.RunQuiet("npm", "install", "-g", "prettier"); err == nil {
				platform.PrintOK(os.Stdout, "Installed prettier globally")
			} else {
				platform.PrintWarn(os.Stdout, "Failed to install prettier via npm")
			}
		}

		// Re-check what's still missing
		for _, t := range missing {
			if !platform.Exists(t.name) {
				stillMissing = append(stillMissing, t)
			}
		}

		if len(stillMissing) > 0 {
			fmt.Println("\n  Optional tools not found (not required, but useful):")
			for _, t := range stillMissing {
				fmt.Printf("    - %s: %s\n", platform.Bold(t.name), t.purpose)
				platform.PrintCommand(os.Stdout, t.install)
			}
		} else {
			fmt.Println("  All optional tools are available.")
		}
	} else {
		fmt.Println("  All optional tools are available.")
	}
}

func joinStrings(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
