// Package doctor implements the "doctor" command, which performs health checks
// on the platform configuration including CLI tools, global settings, project
// setup, agents, skills, hooks, MCP servers, and authentication.
package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lamchakchan/claude-workspace/internal/platform"
	"github.com/lamchakchan/claude-workspace/internal/setup"
	"github.com/lamchakchan/claude-workspace/internal/tools"
	"github.com/lamchakchan/claude-workspace/internal/upgrade"
)

// Run executes the doctor command, checking platform configuration health
// and printing a summary of issues and warnings.
func Run() error {
	platform.PrintBanner(os.Stdout, "Claude Platform Health Check")

	issues := 0
	warnings := 0

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}

	cwd, _ := os.Getwd()

	i, w := checkClaudeCLI(home)
	issues += i
	warnings += w

	i, w = checkClaudeWorkspace()
	issues += i
	warnings += w

	i, w = checkGit()
	issues += i
	warnings += w

	i, w = checkNode()
	issues += i
	warnings += w

	i, w = checkGlobalConfig(home)
	issues += i
	warnings += w

	i, w = checkProjectConfig(cwd)
	issues += i
	warnings += w

	i, w = checkAgents(cwd)
	issues += i
	warnings += w

	i, w = checkSkills(cwd)
	issues += i
	warnings += w

	i, w = checkHooks(cwd)
	issues += i
	warnings += w

	i, w = checkHookConfig(cwd)
	issues += i
	warnings += w

	i, w = checkMCPServers(cwd)
	issues += i
	warnings += w

	i, w = checkAuth(home)
	issues += i
	warnings += w

	// Summary
	platform.PrintBanner(os.Stdout, "Summary")
	if issues == 0 && warnings == 0 {
		fmt.Println(platform.BoldGreen("All checks passed. Platform is healthy."))
	} else {
		if issues > 0 {
			fmt.Println(platform.Red(fmt.Sprintf("Issues: %d (must fix)", issues)))
		}
		if warnings > 0 {
			fmt.Println(platform.Yellow(fmt.Sprintf("Warnings: %d (optional)", warnings)))
		}
	}
	fmt.Println()

	return nil
}

// checkClaudeCLI verifies the Claude Code CLI is installed and checks for npm shadow installs.
func checkClaudeCLI(home string) (int, int) {
	issues := 0
	warnings := 0

	platform.PrintSectionLabel(os.Stdout, "Claude Code CLI")
	claudeBin := "claude"
	if !platform.Exists(claudeBin) {
		// The official installer places claude in ~/.local/bin which may not be in PATH
		localBin := filepath.Join(home, ".local", "bin", "claude")
		if platform.FileExists(localBin) {
			claudeBin = localBin
		}
	}
	if ver, err := platform.Output(claudeBin, "--version"); err == nil {
		pass("Installed: " + ver)
		// Check if installed via npm (may shadow official binary)
		npmInfo := setup.DetectNpmClaude()
		if npmInfo.Detected {
			warn("Claude Code is installed via npm (@anthropic-ai/claude-code)")
			fmt.Println("    The npm version may shadow the official binary in PATH.")
			fmt.Println("    Fix: npm uninstall -g @anthropic-ai/claude-code")
			warnings++
		}
	} else {
		fail("Claude Code CLI not found")
		fmt.Println("    Install: curl -fsSL https://claude.ai/install.sh | bash")
		issues++
	}

	return issues, warnings
}

// checkClaudeWorkspace verifies the claude-workspace binary is in PATH and checks for updates.
func checkClaudeWorkspace() (int, int) {
	issues := 0
	warnings := 0

	platform.PrintSectionLabel(os.Stdout, "claude-workspace CLI")
	if platform.Exists("claude-workspace") {
		pass("claude-workspace is in PATH")
	} else {
		warn("claude-workspace not found in PATH")
		fmt.Println("    Run: claude-workspace setup")
		warnings++
	}

	// Soft update check (3s timeout, skip on failure)
	checkForUpdate()

	return issues, warnings
}

// checkGit verifies Git is installed.
func checkGit() (int, int) {
	issues := 0
	warnings := 0

	platform.PrintSectionLabel(os.Stdout, "Git")
	if ver, err := platform.Output("git", "--version"); err == nil {
		pass(ver)
	} else {
		fail("Git not found")
		issues++
	}

	return issues, warnings
}

// checkNode verifies Node.js is installed and meets minimum version requirements.
func checkNode() (int, int) {
	issues := 0
	warnings := 0

	platform.PrintSectionLabel(os.Stdout, "Node.js")
	nodeTool := tools.Node()
	switch {
	case nodeTool.IsInstalled():
		if ver, err := platform.Output("node", "--version"); err == nil {
			pass("Node.js: " + ver)
		} else {
			pass("Node.js: installed")
		}
		if platform.Exists("npx") {
			pass("npx: available")
		} else {
			warn("npx not found (required for MCP servers)")
			warnings++
		}
	case platform.Exists("node"):
		// node exists but doesn't meet minimum version
		ver, _ := platform.Output("node", "--version")
		warn(fmt.Sprintf("Node.js %s is below minimum version (v%d+)", ver, tools.NodeMinMajor))
		fmt.Println("    Upgrade Node.js: https://nodejs.org")
		warnings++
	default:
		fail("Node.js not found (required for MCP servers)")
		fmt.Println("    Install: https://nodejs.org or run: claude-workspace setup")
		issues++
	}

	return issues, warnings
}

// checkGlobalConfig verifies global settings.json and CLAUDE.md exist and are valid.
func checkGlobalConfig(home string) (int, int) {
	issues := 0
	warnings := 0

	platform.PrintSectionLabel(os.Stdout, "Global Configuration")

	globalSettingsPath := filepath.Join(home, ".claude", "settings.json")
	if platform.FileExists(globalSettingsPath) {
		pass("~/.claude/settings.json exists")
		var settings map[string]json.RawMessage
		if err := platform.ReadJSONFile(globalSettingsPath, &settings); err == nil {
			var env map[string]string
			if raw, ok := settings["env"]; ok {
				if json.Unmarshal(raw, &env) == nil {
					if model, ok := env["CLAUDE_CODE_SUBAGENT_MODEL"]; ok {
						pass("Subagent model: " + model)
					}
					if env["CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS"] == "1" {
						pass("Agent teams: enabled")
					}
				}
			}
		} else {
			warn("Could not parse global settings")
			warnings++
		}
	} else {
		warn("~/.claude/settings.json not found. Run 'claude-workspace setup'")
		warnings++
	}

	globalClaudeMd := filepath.Join(home, ".claude", "CLAUDE.md")
	if platform.FileExists(globalClaudeMd) {
		pass("~/.claude/CLAUDE.md exists")
	} else {
		warn("~/.claude/CLAUDE.md not found")
		warnings++
	}

	return issues, warnings
}

// checkProjectConfig runs table-driven checks for expected project configuration files.
func checkProjectConfig(cwd string) (int, int) {
	issues := 0
	warnings := 0

	platform.PrintSectionLabel(os.Stdout, "Project Configuration")

	checks := []struct {
		path     string
		label    string
		required bool
	}{
		{".claude/settings.json", "Project settings", true},
		{".claude/CLAUDE.md", "Project CLAUDE.md", true},
		{".mcp.json", "MCP configuration", false},
		{".claude/agents", "Agents directory", false},
		{".claude/skills", "Skills directory", false},
		{".claude/hooks", "Hooks directory", false},
		{"plans", "Plans directory", false},
	}

	for _, check := range checks {
		fullPath := filepath.Join(cwd, check.path)
		switch {
		case platform.FileExists(fullPath):
			pass(check.label + ": " + check.path)
		case check.required:
			fail(check.label + " not found: " + check.path)
			issues++
		default:
			warn(check.label + " not found: " + check.path)
			warnings++
		}
	}

	return issues, warnings
}

// checkAgents scans the agents directory for .md agent definition files.
func checkAgents(cwd string) (int, int) {
	issues := 0
	warnings := 0

	platform.PrintSectionLabel(os.Stdout, "Agents")
	agentsDir := filepath.Join(cwd, ".claude", "agents")
	if platform.FileExists(agentsDir) {
		entries, err := os.ReadDir(agentsDir)
		if err == nil {
			var agents []string
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
					agents = append(agents, strings.TrimSuffix(e.Name(), ".md"))
				}
			}
			if len(agents) > 0 {
				pass(fmt.Sprintf("Found %d agents: %s", len(agents), strings.Join(agents, ", ")))
			} else {
				warn("No agent definitions found")
				warnings++
			}
		}
	}

	return issues, warnings
}

// checkSkills walks the skills directory for SKILL.md definition files.
func checkSkills(cwd string) (int, int) {
	issues := 0
	warnings := 0

	platform.PrintSectionLabel(os.Stdout, "Skills")
	skillsDir := filepath.Join(cwd, ".claude", "skills")
	if platform.FileExists(skillsDir) {
		var skills []string
		_ = filepath.WalkDir(skillsDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.Name() == "SKILL.md" {
				rel, _ := filepath.Rel(skillsDir, filepath.Dir(path))
				skills = append(skills, rel)
			}
			return nil
		})
		if len(skills) > 0 {
			pass(fmt.Sprintf("Found %d skills: %s", len(skills), strings.Join(skills, ", ")))
		} else {
			warn("No skill definitions found")
			warnings++
		}
	}

	return issues, warnings
}

// checkHooks verifies hook shell scripts in the hooks directory are executable.
func checkHooks(cwd string) (int, int) {
	issues := 0
	warnings := 0

	platform.PrintSectionLabel(os.Stdout, "Hooks")
	hooksDir := filepath.Join(cwd, ".claude", "hooks")
	if platform.FileExists(hooksDir) {
		entries, err := os.ReadDir(hooksDir)
		if err == nil {
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".sh") {
					hookPath := filepath.Join(hooksDir, e.Name())
					if platform.IsExecutable(hookPath) {
						pass(e.Name() + ": executable")
					} else {
						fail(fmt.Sprintf("%s: not executable. Run: chmod +x %s", e.Name(), hookPath))
						issues++
					}
				}
			}
		}
	}

	return issues, warnings
}

// checkHookConfig validates that settings.json hook references are parseable.
func checkHookConfig(cwd string) (int, int) {
	issues := 0
	warnings := 0

	platform.PrintSectionLabel(os.Stdout, "Hook Configuration")
	settingsPath := filepath.Join(cwd, ".claude", "settings.json")
	if platform.FileExists(settingsPath) {
		var settings map[string]json.RawMessage
		if err := platform.ReadJSONFile(settingsPath, &settings); err == nil {
			if raw, ok := settings["hooks"]; ok {
				var hooks map[string]json.RawMessage
				if json.Unmarshal(raw, &hooks) == nil {
					hookCount := countHookCommands(hooks)
					pass(fmt.Sprintf("%d hook commands configured", hookCount))
				}
			}
		} else {
			warn("Could not validate hook configuration")
			warnings++
		}
	}

	return issues, warnings
}

// checkMCPServers validates the .mcp.json configuration file.
func checkMCPServers(cwd string) (int, int) {
	issues := 0
	warnings := 0

	platform.PrintSectionLabel(os.Stdout, "MCP Servers")
	mcpPath := filepath.Join(cwd, ".mcp.json")
	if platform.FileExists(mcpPath) {
		var mcpConfig struct {
			MCPServers map[string]json.RawMessage `json:"mcpServers"`
		}
		if err := platform.ReadJSONFile(mcpPath, &mcpConfig); err == nil {
			servers := make([]string, 0, len(mcpConfig.MCPServers))
			for name := range mcpConfig.MCPServers {
				servers = append(servers, name)
			}
			if len(servers) > 0 {
				pass(fmt.Sprintf("%d MCP servers configured: %s", len(servers), strings.Join(servers, ", ")))
			} else {
				warn("No MCP servers configured in .mcp.json")
				warnings++
			}
		} else {
			fail("Could not parse .mcp.json")
			issues++
		}
	}

	return issues, warnings
}

// checkAuth verifies API key or OAuth authentication is configured.
func checkAuth(home string) (int, int) {
	issues := 0
	warnings := 0

	platform.PrintSectionLabel(os.Stdout, "Authentication")
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		pass("ANTHROPIC_API_KEY is set")
	} else {
		claudeConfig := filepath.Join(home, ".claude.json")
		if platform.FileExists(claudeConfig) {
			var config map[string]json.RawMessage
			if err := platform.ReadJSONFile(claudeConfig, &config); err == nil {
				if _, ok := config["oauthAccount"]; ok {
					pass("OAuth authentication configured")
				} else {
					warn("No API key or OAuth found. Run: claude-workspace setup")
					warnings++
				}
			} else {
				warn("Could not read authentication config")
				warnings++
			}
		} else {
			warn("No authentication configured. Run: claude-workspace setup")
			warnings++
		}
	}

	return issues, warnings
}

func countHookCommands(hooks map[string]json.RawMessage) int {
	count := 0
	for _, raw := range hooks {
		var matchers []struct {
			Hooks []struct {
				Type string `json:"type"`
			} `json:"hooks"`
		}
		if json.Unmarshal(raw, &matchers) == nil {
			for _, m := range matchers {
				for _, h := range m.Hooks {
					if h.Type == "command" {
						count++
					}
				}
			}
		}
	}
	return count
}

func pass(msg string) {
	platform.PrintOK(os.Stdout, msg)
}

func fail(msg string) {
	platform.PrintFail(os.Stdout, msg)
}

func warn(msg string) {
	platform.PrintWarn(os.Stdout, msg)
}

func checkForUpdate() {
	type result struct {
		release *upgrade.Release
		err     error
	}
	ch := make(chan result, 1)
	go func() {
		r, err := upgrade.FetchLatest()
		ch <- result{r, err}
	}()

	select {
	case res := <-ch:
		if res.err != nil {
			return // silently skip on failure
		}
		currentVer, _ := platform.Output("claude-workspace", "--version")
		// currentVer looks like "claude-workspace vX.Y.Z" — extract the version
		currentVer = strings.TrimPrefix(currentVer, "claude-workspace ")
		if currentVer != "" && currentVer != res.release.TagName {
			platform.PrintInfo(os.Stdout, fmt.Sprintf("Update available: %s → %s", currentVer, res.release.TagName))
			fmt.Println("    Run: claude-workspace upgrade")
		}
	case <-time.After(3 * time.Second):
		return // timeout, skip silently
	}
}
