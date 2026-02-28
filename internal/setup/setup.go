// Package setup implements the "setup" command, which handles first-time
// platform setup including Claude CLI installation, API key provisioning,
// global settings, Node.js verification, MCP server registration, and
// optional tool installation.
package setup

import (
	"encoding/json"
	"fmt"
	"io"
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
	return runTo(os.Stdout, force, true)
}

// RunTo is like Run but writes all output to w instead of os.Stdout and skips
// interactive steps (e.g., API key provisioning that requires stdin).
func RunTo(w io.Writer, args []string) error {
	force := false
	for _, a := range args {
		if a == "--force" {
			force = true
		}
	}
	return runTo(w, force, false)
}

func runTo(w io.Writer, force, interactive bool) error {
	platform.PrintBanner(w, "Claude Code Platform Setup")

	platform.PrintStep(w, 1, 9, "Checking Claude Code installation...")
	if err := ensureClaudeCLITo(w); err != nil {
		return err
	}

	platform.PrintStep(w, 2, 9, "API Key provisioning...")
	if err := provisionAPIKeyTo(w, interactive); err != nil {
		return err
	}

	platform.PrintStep(w, 3, 9, "Setting up global user configuration...")
	if err := setupGlobalSettingsTo(w, force); err != nil {
		return err
	}

	platform.PrintStep(w, 4, 9, "Setting up global CLAUDE.md...")
	if err := setupGlobalClaudeMdTo(w); err != nil {
		return err
	}

	platform.PrintStep(w, 5, 9, "Installing claude-workspace to PATH...")
	installBinaryToPathTo(w)

	platform.PrintStep(w, 6, 9, "Checking Node.js (required for filesystem MCP server)...")
	ensureNodeTo(w)

	platform.PrintStep(w, 7, 9, "Registering user-scoped MCP servers...")
	if err := setupUserMCPServersTo(w, force); err != nil {
		platform.PrintWarningLine(w, fmt.Sprintf("MCP server registration skipped: %v", err))
	}

	platform.PrintStep(w, 8, 9, "Checking optional system tools...")
	tools.CheckAndInstallTo(w, tools.Optional())

	platform.PrintStep(w, 9, 9, "Statusline setup (cost & context display)...")
	if err := statusline.RunTo(w, []string{}); err != nil {
		platform.PrintWarningLine(w, fmt.Sprintf("statusline setup skipped: %v", err))
	}

	platform.PrintBanner(w, "Setup Complete")
	fmt.Fprintln(w, "\nNext steps:")
	fmt.Fprintln(w)
	platform.PrintCommand(w, "claude-workspace attach /path/to/project")
	platform.PrintCommand(w, "cd /path/to/project && claude")
	platform.PrintCommand(w, "claude-workspace mcp add <name> -- <command>")
	fmt.Fprintln(w)

	return nil
}

func ensureClaudeCLITo(w io.Writer) error {
	npmInfo := DetectNpmClaude()
	if npmInfo.Detected {
		fmt.Fprintf(w, "  Detected Claude Code installed via npm (source: %s).\n", npmInfo.Source)
		fmt.Fprintln(w, "  Removing npm version before installing official binary...")
		if err := UninstallNpmClaude(npmInfo); err != nil {
			platform.PrintWarningLine(w, fmt.Sprintf("could not remove npm Claude: %v", err))
			fmt.Fprintln(w, "  Please run manually: npm uninstall -g @anthropic-ai/claude-code")
			fmt.Fprintln(w, "  Then re-run: claude-workspace setup")
			return fmt.Errorf("npm Claude uninstall failed: %w", err)
		}
		fmt.Fprintln(w, "  npm Claude Code removed successfully.")
	}

	claudeTool := tools.Claude()
	if claudeTool.IsInstalled() {
		ver, _ := platform.Output("claude", "--version")
		fmt.Fprintf(w, "  Claude Code CLI found: %s\n", ver)

		if claudePath, err := exec.LookPath("claude"); err == nil {
			home, _ := os.UserHomeDir()
			if err := ensureLocalBinClaudeTo(w, home, claudePath); err != nil {
				platform.PrintWarningLine(w, fmt.Sprintf("could not create ~/.local/bin/claude: %v", err))
			}
		}
		return nil
	}

	fmt.Fprintln(w, "  Claude Code CLI not found. Installing...")
	return claudeTool.Install()
}

func ensureNodeTo(w io.Writer) {
	nodeTool := tools.Node()
	if nodeTool.IsInstalled() {
		ver, _ := platform.Output("node", "--version")
		fmt.Fprintf(w, "  Node.js found: %s\n", ver)
		return
	}
	fmt.Fprintln(w, "  Node.js not found or below minimum version. Installing...")
	if err := nodeTool.Install(); err != nil {
		platform.PrintWarningLine(w, fmt.Sprintf("Node.js install failed: %v", err))
		fmt.Fprintln(w, "  MCP servers require Node.js. Install manually: https://nodejs.org")
	} else if ver, err := platform.Output("node", "--version"); err == nil {
		fmt.Fprintf(w, "  Node.js installed: %s\n", ver)
	}
}

func provisionAPIKeyTo(w io.Writer, interactive bool) error {
	if platform.FileExists(claudeConfig) {
		var config map[string]json.RawMessage
		if err := platform.ReadJSONFile(claudeConfig, &config); err == nil {
			if _, hasOAuth := config["oauthAccount"]; hasOAuth {
				fmt.Fprintln(w, "  Already authenticated. Skipping API key provisioning.")
				return nil
			}
			if _, hasKey := config["primaryApiKey"]; hasKey {
				fmt.Fprintln(w, "  Already authenticated. Skipping API key provisioning.")
				return nil
			}
		}
	}

	if !interactive {
		fmt.Fprintln(w, "  API key provisioning requires interactive setup.")
		fmt.Fprintln(w, "  Run 'claude' directly to complete the login flow.")
		fmt.Fprintln(w, "  You can set ANTHROPIC_API_KEY in your environment as an alternative.")
		return nil
	}

	fmt.Fprintln(w, "  Starting self-service API key provisioning (Option 2)...")
	fmt.Fprintln(w, "  This will open Claude Code's interactive login flow.")
	fmt.Fprintln(w, "  Select 'Use an API key' when prompted.")
	fmt.Fprintln(w)

	exitCode, err := platform.RunSpawn("claude", "--print-api-key-config")
	if err != nil || exitCode != 0 {
		fmt.Fprintln(w, "\n  API key provisioning requires interactive setup.")
		fmt.Fprintln(w, "  Run 'claude' directly to complete the login flow.")
		fmt.Fprintln(w, "  You can set ANTHROPIC_API_KEY in your environment as an alternative.")
	}

	return nil
}

func setupGlobalSettingsTo(w io.Writer, force bool) error {
	settingsPath := filepath.Join(claudeHome, "settings.json")

	defaults := GetDefaultGlobalSettings()

	if platform.FileExists(settingsPath) {
		fmt.Fprintln(w, "  Global settings already exist. Merging platform defaults...")
		var existing map[string]interface{}
		if err := platform.ReadJSONFile(settingsPath, &existing); err != nil {
			fmt.Fprintln(w, "  Could not merge settings. Skipping global settings update.")
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
		fmt.Fprintln(w, "  Global settings updated.")
		return nil
	}

	// Create ~/.claude/ directory if needed
	if err := os.MkdirAll(claudeHome, 0755); err != nil {
		return fmt.Errorf("creating ~/.claude: %w", err)
	}

	if err := platform.WriteJSONFile(settingsPath, defaults); err != nil {
		return fmt.Errorf("writing global settings: %w", err)
	}
	fmt.Fprintln(w, "  Global settings created at ~/.claude/settings.json")
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

func setupGlobalClaudeMdTo(w io.Writer) error {
	claudeMdPath := filepath.Join(claudeHome, "CLAUDE.md")

	if platform.FileExists(claudeMdPath) {
		fmt.Fprintln(w, "  Global CLAUDE.md already exists. Skipping.")
		return nil
	}

	content, err := platform.ReadGlobalAsset("CLAUDE.md")
	if err != nil {
		return fmt.Errorf("reading global CLAUDE.md template: %w", err)
	}

	if err := os.WriteFile(claudeMdPath, content, 0644); err != nil {
		return fmt.Errorf("writing CLAUDE.md: %w", err)
	}
	fmt.Fprintln(w, "  Global CLAUDE.md created at ~/.claude/CLAUDE.md")
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

func setupUserMCPServersTo(w io.Writer, force bool) error {
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

	if force {
		config = RemoveUserMCPServers(config, knownMemoryProviders)
	} else {
		existing, _ := config["mcpServers"].(map[string]interface{})
		for _, key := range knownMemoryProviders {
			if existing != nil {
				if _, found := existing[key]; found {
					fmt.Fprintf(w, "  Memory MCP already configured (provider: %s). Run 'claude-workspace memory configure' to change providers.\n", key)
					return nil
				}
			}
		}
	}

	dbDir := filepath.Join(home, ".config", "claude-workspace")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		platform.PrintWarningLine(w, fmt.Sprintf("could not create %s: %v", dbDir, err))
	}

	servers := platformMCPServers(home)
	merged := MergeUserMCPServers(config, servers)

	if err := platform.WriteJSONFile(claudeConfig, merged); err != nil {
		return fmt.Errorf("writing %s: %w", claudeConfig, err)
	}

	reportMCPRegistrationTo(w, config, servers)
	return nil
}

func reportMCPRegistrationTo(w io.Writer, config map[string]interface{}, servers map[string]interface{}) {
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
		platform.PrintOK(w, fmt.Sprintf("Registered: %s", joinStrings(added, ", ")))
		fmt.Fprintln(w, "  Memory tools: mcp__mcp-memory-libsql__search_nodes, create_entities, read_graph")
	}
	if len(skipped) > 0 {
		fmt.Fprintf(w, "  Already registered: %s\n", joinStrings(skipped, ", "))
	}
}

func installBinaryToPathTo(w io.Writer) {
	execPath, err := os.Executable()
	if err != nil {
		fmt.Fprintln(w, "  Could not determine binary path. Skipping PATH installation.")
		return
	}

	// Check if already in a standard PATH location
	if _, err := platform.Output("which", "claude-workspace"); err == nil {
		fmt.Fprintln(w, "  claude-workspace is already in PATH.")
		return
	}

	installDir := "/usr/local/bin"
	destPath := filepath.Join(installDir, "claude-workspace")

	// Try to copy the binary
	fmt.Fprintf(w, "  Installing to %s...\n", destPath)
	if err := platform.CopyFile(execPath, destPath); err != nil {
		// Try with sudo
		if err := platform.Run("sudo", "cp", execPath, destPath); err != nil {
			fmt.Fprintf(w, "  Could not install to %s (permission denied).\n", installDir)
			fmt.Fprintf(w, "  To install manually:\n")
			fmt.Fprintf(w, "    sudo cp %s %s\n", execPath, destPath)
			return
		}
		// Make executable
		_ = platform.RunQuiet("sudo", "chmod", "+x", destPath)
	} else {
		_ = os.Chmod(destPath, 0755)
	}
	fmt.Fprintln(w, "  Installed: claude-workspace is now available globally.")
}

// ensureLocalBinClaude creates ~/.local/bin/claude as a symlink to claudePath when it doesn't
// already exist, then ensures ~/.local/bin is present in the shell RC file.
// We do not resolve symlinks in claudePath so that package-manager managed paths (e.g.
// /opt/homebrew/bin/claude) remain valid across version upgrades.
func ensureLocalBinClaude(home, claudePath string) error {
	return ensureLocalBinClaudeTo(os.Stdout, home, claudePath)
}

func ensureLocalBinClaudeTo(w io.Writer, home, claudePath string) error {
	localBinClaude := filepath.Join(home, ".local", "bin", "claude")

	if platform.FileExists(localBinClaude) {
		return nil
	}

	if err := platform.SymlinkFile(claudePath, localBinClaude); err != nil {
		return fmt.Errorf("creating ~/.local/bin/claude symlink: %w", err)
	}
	fmt.Fprintf(w, "  Created ~/.local/bin/claude → %s\n", claudePath)

	// AppendPathToRC is idempotent — it checks for ".local/bin" before writing.
	rcPath, shellName := platform.DetectShellRC(home)
	if modified, err := platform.AppendPathToRC(home, shellName, rcPath); err != nil {
		platform.PrintWarningLine(w, fmt.Sprintf("could not update PATH in %s: %v", rcPath, err))
		fmt.Fprintln(w, `  Add manually: export PATH="$HOME/.local/bin:$PATH"`)
	} else if modified {
		fmt.Fprintf(w, "  Added ~/.local/bin to PATH in %s\n", filepath.Base(rcPath))
		fmt.Fprintf(w, "  Restart your shell or run: source %s\n", rcPath)
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
