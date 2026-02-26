// Package setup implements the "setup" command, which handles first-time
// platform setup including Claude CLI installation, API key provisioning,
// global settings, Node.js verification, MCP server registration, and
// optional tool installation.
package setup

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/lamchakchan/claude-workspace/internal/platform"
	"github.com/lamchakchan/claude-workspace/internal/statusline"
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

// knownMemoryProviders is the set of memory MCP server keys this platform manages.
var knownMemoryProviders = []string{"mcp-memory-libsql", "engram", "memory"}

// Run executes the setup command, performing first-time platform configuration.
// Pass --force in args to overwrite existing settings.
func Run(args []string) error {
	force := false
	for _, a := range args {
		if a == "--force" {
			force = true
		}
	}

	platform.PrintBanner(os.Stdout, "Claude Code Platform Setup")

	// Step 1: Check if Claude Code CLI is installed
	platform.PrintStep(os.Stdout, 1, 9, "Checking Claude Code installation...")

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

		// Ensure ~/.local/bin/claude symlink exists. Claude Code expects its binary there
		// regardless of how it was installed (e.g. Homebrew). Missing symlink causes startup
		// warnings that claude update does not fix (known upstream issue).
		if claudePath, err := exec.LookPath("claude"); err == nil {
			home, _ := os.UserHomeDir()
			if err := ensureLocalBinClaude(home, claudePath); err != nil {
				platform.PrintWarningLine(os.Stdout, fmt.Sprintf("could not create ~/.local/bin/claude: %v", err))
			}
		}
	} else {
		fmt.Println("  Claude Code CLI not found. Installing...")
		if err := claudeTool.Install(); err != nil {
			return err
		}
	}

	// Step 2: API Key provisioning
	platform.PrintStep(os.Stdout, 2, 9, "API Key provisioning...")
	if err := provisionApiKey(); err != nil {
		return err
	}

	// Step 3: Create global user settings
	platform.PrintStep(os.Stdout, 3, 9, "Setting up global user configuration...")
	if err := setupGlobalSettings(force); err != nil {
		return err
	}

	// Step 4: Create global CLAUDE.md
	platform.PrintStep(os.Stdout, 4, 9, "Setting up global CLAUDE.md...")
	if err := setupGlobalClaudeMd(); err != nil {
		return err
	}

	// Step 5: Install binary to PATH
	platform.PrintStep(os.Stdout, 5, 9, "Installing claude-workspace to PATH...")
	installBinaryToPath()

	// Step 6: Ensure Node.js is available
	platform.PrintStep(os.Stdout, 6, 9, "Checking Node.js (required for filesystem MCP server)...")
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

	// Step 7: Register user-scoped MCP servers (mcp-memory-libsql via npx — no extra install needed)
	platform.PrintStep(os.Stdout, 7, 9, "Registering user-scoped MCP servers...")
	if err := setupUserMCPServers(force); err != nil {
		platform.PrintWarningLine(os.Stdout, fmt.Sprintf("MCP server registration skipped: %v", err))
	}

	// Step 8: Check optional system tools
	platform.PrintStep(os.Stdout, 8, 9, "Checking optional system tools...")
	tools.CheckAndInstall(tools.Optional())

	// Step 9: Statusline setup
	platform.PrintStep(os.Stdout, 9, 9, "Statusline setup (cost & context display)...")
	if err := statusline.Run([]string{}); err != nil {
		platform.PrintWarningLine(os.Stdout, fmt.Sprintf("statusline setup skipped: %v", err))
	}

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

func setupGlobalSettings(force bool) error {
	settingsPath := filepath.Join(claudeHome, "settings.json")

	defaults := GetDefaultGlobalSettings()

	if platform.FileExists(settingsPath) {
		fmt.Println("  Global settings already exist. Merging platform defaults...")
		var existing map[string]interface{}
		if err := platform.ReadJSONFile(settingsPath, &existing); err != nil {
			fmt.Println("  Could not merge settings. Skipping global settings update.")
			return nil
		}
		var merged map[string]interface{}
		if force {
			merged = MergeSettingsForce(existing, defaults)
		} else {
			merged = MergeSettings(existing, defaults)
		}
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

// GetDefaultGlobalSettings returns the default global settings map parsed from
// the embedded settings.json template.
func GetDefaultGlobalSettings() map[string]interface{} {
	data, err := platform.ReadGlobalAsset("settings.json")
	if err != nil {
		// Should never happen — file is embedded at compile time
		panic(fmt.Sprintf("reading embedded global settings.json: %v", err))
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		panic(fmt.Sprintf("parsing embedded global settings.json: %v", err))
	}

	return settings
}

// MergeSettings merges platform default settings into existing settings without
// overwriting user-customized values. Env vars and permission lists are unioned.
func MergeSettings(existing, defaults map[string]interface{}) map[string]interface{} {
	return mergeSettings(existing, defaults, false)
}

// MergeSettingsForce merges settings with force mode: replaces permissions wholesale
// from defaults instead of performing a union merge.
func MergeSettingsForce(existing, defaults map[string]interface{}) map[string]interface{} {
	return mergeSettings(existing, defaults, true)
}

func mergeSettings(existing, defaults map[string]interface{}, force bool) map[string]interface{} {
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

	// Merge permissions
	if defaultPerms, ok := defaults["permissions"].(map[string]interface{}); ok {
		if force {
			// Force mode: replace permissions entirely with defaults
			merged["permissions"] = defaultPerms
		} else {
			existingPerms, _ := existing["permissions"].(map[string]interface{})
			if existingPerms == nil {
				existingPerms = make(map[string]interface{})
			}

			mergedPerms := make(map[string]interface{})
			for k, v := range existingPerms {
				mergedPerms[k] = v
			}

			// Merge deny list (union)
			if defaultPerms["deny"] != nil {
				mergedPerms["deny"] = mergeStringList(existingPerms["deny"], extractStrings(defaultPerms["deny"]))
			}

			// Merge allow list (union)
			if defaultPerms["allow"] != nil {
				mergedPerms["allow"] = mergeStringList(existingPerms["allow"], extractStrings(defaultPerms["allow"]))
			}

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

// extractStrings converts a value that may be []string or []interface{} (from JSON
// deserialization) into a []string.
func extractStrings(v interface{}) []string {
	switch s := v.(type) {
	case []string:
		return s
	case []interface{}:
		out := make([]string, 0, len(s))
		for _, item := range s {
			if str, ok := item.(string); ok {
				out = append(out, str)
			}
		}
		return out
	}
	return nil
}

// mergeStringList computes the union of an existing list (which may be []string or
// []interface{} from JSON deserialization) and a defaults list of []string.
// Existing entries come first, then any new defaults are appended.
func mergeStringList(existing interface{}, defaults []string) []string {
	var existingList []string
	if existing != nil {
		switch v := existing.(type) {
		case []string:
			existingList = v
		case []interface{}:
			for _, item := range v {
				if s, ok := item.(string); ok {
					existingList = append(existingList, s)
				}
			}
		}
	}

	seen := make(map[string]bool)
	for _, rule := range existingList {
		seen[rule] = true
	}

	combined := make([]string, len(existingList))
	copy(combined, existingList)
	for _, rule := range defaults {
		if !seen[rule] {
			combined = append(combined, rule)
		}
	}

	return combined
}

func setupGlobalClaudeMd() error {
	claudeMdPath := filepath.Join(claudeHome, "CLAUDE.md")

	if platform.FileExists(claudeMdPath) {
		fmt.Println("  Global CLAUDE.md already exists. Skipping.")
		return nil
	}

	content, err := platform.ReadGlobalAsset("CLAUDE.md")
	if err != nil {
		return fmt.Errorf("reading global CLAUDE.md template: %w", err)
	}

	if err := os.WriteFile(claudeMdPath, content, 0644); err != nil {
		return fmt.Errorf("writing CLAUDE.md: %w", err)
	}
	fmt.Println("  Global CLAUDE.md created at ~/.claude/CLAUDE.md")
	return nil
}

// platformMCPServers returns the user-scoped MCP servers the platform registers by default.
func platformMCPServers(home string) map[string]interface{} {
	dbPath := filepath.Join(home, ".config", "claude-workspace", "memory.db")
	return map[string]interface{}{
		"mcp-memory-libsql": map[string]interface{}{
			"command": "npx",
			"args":    []string{"-y", "mcp-memory-libsql"},
			"env":     map[string]interface{}{"LIBSQL_URL": "file:" + dbPath},
		},
	}
}

// RemoveUserMCPServers removes the given server keys from the mcpServers map in config.
// Returns the modified config. Does not write to disk.
func RemoveUserMCPServers(config map[string]interface{}, keys []string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range config {
		result[k] = v
	}
	existing, _ := result["mcpServers"].(map[string]interface{})
	if existing == nil {
		return result
	}
	updated := make(map[string]interface{})
	for k, v := range existing {
		updated[k] = v
	}
	for _, key := range keys {
		delete(updated, key)
	}
	result["mcpServers"] = updated
	return result
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

func setupUserMCPServers(force bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
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

	// On force: remove all known memory provider keys first (clean slate for re-registration)
	if force {
		config = RemoveUserMCPServers(config, knownMemoryProviders)
	} else {
		// Check if any known memory provider is already configured — skip if so
		existing, _ := config["mcpServers"].(map[string]interface{})
		for _, key := range knownMemoryProviders {
			if existing != nil {
				if _, found := existing[key]; found {
					fmt.Printf("  Memory MCP already configured (provider: %s). Run 'claude-workspace memory configure' to change providers.\n", key)
					return nil
				}
			}
		}
	}

	// Ensure the DB parent directory exists
	dbDir := filepath.Join(home, ".config", "claude-workspace")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		platform.PrintWarningLine(os.Stdout, fmt.Sprintf("could not create %s: %v", dbDir, err))
	}

	servers := platformMCPServers(home)
	merged := MergeUserMCPServers(config, servers)

	if err := platform.WriteJSONFile(claudeConfig, merged); err != nil {
		return fmt.Errorf("writing %s: %w", claudeConfig, err)
	}

	// Report what was registered vs already present
	existing, _ := config["mcpServers"].(map[string]interface{})
	var added, skipped []string
	for name := range servers {
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
		fmt.Println("  Memory tools: mcp__mcp-memory-libsql__search_nodes, create_entities, read_graph")
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

// ensureLocalBinClaude creates ~/.local/bin/claude as a symlink to claudePath when it doesn't
// already exist, then ensures ~/.local/bin is present in the shell RC file.
// We do not resolve symlinks in claudePath so that package-manager managed paths (e.g.
// /opt/homebrew/bin/claude) remain valid across version upgrades.
func ensureLocalBinClaude(home, claudePath string) error {
	localBinClaude := filepath.Join(home, ".local", "bin", "claude")

	if platform.FileExists(localBinClaude) {
		return nil
	}

	if err := platform.SymlinkFile(claudePath, localBinClaude); err != nil {
		return fmt.Errorf("creating ~/.local/bin/claude symlink: %w", err)
	}
	fmt.Printf("  Created ~/.local/bin/claude → %s\n", claudePath)

	// AppendPathToRC is idempotent — it checks for ".local/bin" before writing.
	rcPath, shellName := platform.DetectShellRC(home)
	if modified, err := platform.AppendPathToRC(home, shellName, rcPath); err != nil {
		platform.PrintWarningLine(os.Stdout, fmt.Sprintf("could not update PATH in %s: %v", rcPath, err))
		fmt.Println(`  Add manually: export PATH="$HOME/.local/bin:$PATH"`)
	} else if modified {
		fmt.Printf("  Added ~/.local/bin to PATH in %s\n", filepath.Base(rcPath))
		fmt.Printf("  Restart your shell or run: source %s\n", rcPath)
	}

	return nil
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
