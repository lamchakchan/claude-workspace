package setup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

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
	fmt.Println("\n=== Claude Code Platform Setup ===")

	// Step 1: Check if Claude Code CLI is installed
	fmt.Println("\n[1/6] Checking Claude Code installation...")
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
	fmt.Println("\n[2/6] API Key provisioning...")
	if err := provisionApiKey(); err != nil {
		return err
	}

	// Step 3: Create global user settings
	fmt.Println("\n[3/6] Setting up global user configuration...")
	if err := setupGlobalSettings(); err != nil {
		return err
	}

	// Step 4: Create global CLAUDE.md
	fmt.Println("\n[4/6] Setting up global CLAUDE.md...")
	if err := setupGlobalClaudeMd(); err != nil {
		return err
	}

	// Step 5: Install binary to PATH
	fmt.Println("\n[5/6] Installing claude-workspace to PATH...")
	installBinaryToPath()

	// Step 6: Check optional system tools
	fmt.Println("\n[6/6] Checking optional system tools...")
	checkOptionalTools()

	fmt.Println("\n=== Setup Complete ===")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Attach to a project:  claude-workspace attach /path/to/project")
	fmt.Println("  2. Start Claude Code:    cd /path/to/project && claude")
	fmt.Println("  3. Add MCP servers:      claude-workspace mcp add <name> -- <command>")
	fmt.Println()

	return nil
}

func installClaude() error {
	fmt.Println("  Installing Claude Code via official installer...")
	if err := platform.RunQuiet("bash", "-c", "curl -fsSL https://claude.ai/install.sh | bash"); err != nil {
		fmt.Fprintln(os.Stderr, "  Failed to install Claude Code automatically.")
		fmt.Println("  Please install manually: curl -fsSL https://claude.ai/install.sh | bash")
		fmt.Println("  Or visit: https://docs.anthropic.com/en/docs/claude-code")
		os.Exit(1)
	}
	fmt.Println("  Claude Code installed successfully.")
	return nil
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

	defaults := getDefaultGlobalSettings()

	if platform.FileExists(settingsPath) {
		fmt.Println("  Global settings already exist. Merging platform defaults...")
		var existing map[string]interface{}
		if err := platform.ReadJSONFile(settingsPath, &existing); err != nil {
			fmt.Println("  Could not merge settings. Skipping global settings update.")
			return nil
		}
		merged := mergeSettings(existing, defaults)
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

func getDefaultGlobalSettings() map[string]interface{} {
	return map[string]interface{}{
		"$schema": "https://json.schemastore.org/claude-code-settings.json",
		"env": map[string]interface{}{
			"CLAUDE_CODE_ENABLE_TELEMETRY":           "1",
			"CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS":   "1",
			"CLAUDE_CODE_ENABLE_TASKS":               "true",
			"CLAUDE_CODE_SUBAGENT_MODEL":             "sonnet",
			"CLAUDE_AUTOCOMPACT_PCT_OVERRIDE":        "80",
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

func mergeSettings(existing, defaults map[string]interface{}) map[string]interface{} {
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
	var pkgInstall func(name string) string
	if runtime.GOOS == "darwin" && platform.Exists("brew") {
		pkgInstall = func(name string) string { return "brew install " + name }
	} else if platform.Exists("apt-get") {
		pkgInstall = func(name string) string { return "sudo apt-get install -y " + name }
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
		fmt.Printf("  Found: %s\n", joinStrings(found, ", "))
	}

	if len(missing) > 0 {
		// Attempt to install missing tools via package manager
		var stillMissing []tool
		if pkgInstall != nil {
			var installable []string
			for _, t := range missing {
				if t.name != "prettier" { // prettier uses npm
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
					args := append([]string{"install", "-y"}, installable...)
					platform.RunQuiet("sudo", append([]string{"apt-get"}, args...)...)
				}
			}

			// Re-check what's still missing
			for _, t := range missing {
				if !platform.Exists(t.name) {
					stillMissing = append(stillMissing, t)
				}
			}
		} else {
			stillMissing = missing
		}

		if len(stillMissing) > 0 {
			fmt.Println("\n  Optional tools not found (not required, but useful):")
			for _, t := range stillMissing {
				fmt.Printf("    - %s: %s\n", t.name, t.purpose)
				fmt.Printf("      Install: %s\n", t.install)
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
