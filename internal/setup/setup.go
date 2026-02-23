package setup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lamchakchan/claude-workspace/internal/platform"
	"github.com/lamchakchan/claude-workspace/internal/tools"
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
	platform.PrintStep(os.Stdout, 1, 8, "Checking Claude Code installation...")

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

	claudeTool := tools.Claude()
	if claudeTool.IsInstalled() {
		ver, _ := platform.Output("claude", "--version")
		fmt.Printf("  Claude Code CLI found: %s\n", ver)
	} else {
		fmt.Println("  Claude Code CLI not found. Installing...")
		if err := claudeTool.Install(); err != nil {
			return err
		}
	}

	// Step 2: API Key provisioning
	platform.PrintStep(os.Stdout, 2, 8, "API Key provisioning...")
	if err := provisionApiKey(); err != nil {
		return err
	}

	// Step 3: Create global user settings
	platform.PrintStep(os.Stdout, 3, 8, "Setting up global user configuration...")
	if err := setupGlobalSettings(); err != nil {
		return err
	}

	// Step 4: Create global CLAUDE.md
	platform.PrintStep(os.Stdout, 4, 8, "Setting up global CLAUDE.md...")
	if err := setupGlobalClaudeMd(); err != nil {
		return err
	}

	// Step 5: Install binary to PATH
	platform.PrintStep(os.Stdout, 5, 8, "Installing claude-workspace to PATH...")
	installBinaryToPath()

	// Step 6: Ensure Node.js is available
	platform.PrintStep(os.Stdout, 6, 8, "Checking Node.js for MCP servers...")
	nodeTool := tools.Node()
	if nodeTool.IsInstalled() {
		ver, _ := platform.Output("node", "--version")
		fmt.Printf("  Node.js found: %s\n", ver)
	} else {
		fmt.Println("  Node.js not found or below minimum version. Installing...")
		if err := nodeTool.Install(); err != nil {
			platform.PrintWarningLine(os.Stdout, fmt.Sprintf("Node.js install failed: %v", err))
			fmt.Println("  MCP servers require Node.js. Install manually: https://nodejs.org")
		} else if ver, err := platform.Output("node", "--version"); err == nil {
			fmt.Printf("  Node.js installed: %s\n", ver)
		}
	}

	// Step 7: Register user-scoped MCP servers
	platform.PrintStep(os.Stdout, 7, 8, "Registering user-scoped MCP servers...")
	if err := setupUserMCPServers(); err != nil {
		platform.PrintWarningLine(os.Stdout, fmt.Sprintf("MCP server registration skipped: %v", err))
	}

	// Step 8: Check optional system tools
	platform.PrintStep(os.Stdout, 8, 8, "Checking optional system tools...")
	tools.CheckAndInstall(tools.Optional())

	platform.PrintBanner(os.Stdout, "Setup Complete")
	fmt.Println("\nNext steps:")
	fmt.Println()
	platform.PrintCommand(os.Stdout, "claude-workspace attach /path/to/project")
	platform.PrintCommand(os.Stdout, "cd /path/to/project && claude")
	platform.PrintCommand(os.Stdout, "claude-workspace mcp add <name> -- <command>")
	fmt.Println()

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
		fmt.Println("  Node.js installation may have failed or npx is not in PATH.")
		fmt.Println("  Install Node.js (https://nodejs.org), then run manually:")
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
