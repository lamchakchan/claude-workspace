// Package doctor implements the "doctor" command, which performs health checks
// on the platform configuration including CLI tools, global settings, project
// setup, agents, skills, hooks, MCP servers, and authentication.
package doctor

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lamchakchan/claude-workspace/internal/platform"
	"github.com/lamchakchan/claude-workspace/internal/setup"
	"github.com/lamchakchan/claude-workspace/internal/tools"
	"github.com/lamchakchan/claude-workspace/internal/upgrade"
)

// Run executes the doctor command, printing to os.Stdout.
func Run() error {
	return RunTo(os.Stdout)
}

// RunTo executes the doctor command, writing all output to w.
func RunTo(w io.Writer) error {
	platform.PrintBanner(w, "Claude Platform Health Check")

	issues := 0
	warnings := 0

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}

	cwd, _ := os.Getwd()

	i, wa := checkClaudeCLI(w, home)
	issues += i
	warnings += wa

	i, wa = checkClaudeWorkspace(w)
	issues += i
	warnings += wa

	i, wa = checkGit(w)
	issues += i
	warnings += wa

	i, wa = checkNode(w)
	issues += i
	warnings += wa

	i, wa = checkGlobalConfig(w, home)
	issues += i
	warnings += wa

	i, wa = checkProjectConfig(w, cwd)
	issues += i
	warnings += wa

	i, wa = checkAgents(w, cwd)
	issues += i
	warnings += wa

	i, wa = checkSkills(w, cwd)
	issues += i
	warnings += wa

	i, wa = checkHooks(w, cwd)
	issues += i
	warnings += wa

	i, wa = checkHookConfig(w, cwd)
	issues += i
	warnings += wa

	i, wa = checkMCPServers(w, cwd)
	issues += i
	warnings += wa

	i, wa = checkAuth(w, home)
	issues += i
	warnings += wa

	// Summary
	platform.PrintBanner(w, "Summary")
	if issues == 0 && warnings == 0 {
		fmt.Fprintln(w, platform.BoldGreen("All checks passed. Platform is healthy."))
	} else {
		if issues > 0 {
			fmt.Fprintln(w, platform.Red(fmt.Sprintf("Issues: %d (must fix)", issues)))
		}
		if warnings > 0 {
			fmt.Fprintln(w, platform.Yellow(fmt.Sprintf("Warnings: %d (optional)", warnings)))
		}
	}
	fmt.Fprintln(w)

	return nil
}

// checkClaudeCLI verifies the Claude Code CLI is installed and checks for npm shadow installs.
func checkClaudeCLI(w io.Writer, home string) (int, int) {
	issues := 0
	warnings := 0

	platform.PrintSectionLabel(w, "Claude Code CLI")
	claudeBin := "claude"
	if !platform.Exists(claudeBin) {
		// The official installer places claude in ~/.local/bin which may not be in PATH
		localBin := filepath.Join(home, ".local", "bin", "claude")
		if platform.FileExists(localBin) {
			claudeBin = localBin
		}
	}
	if ver, err := platform.Output(claudeBin, "--version"); err == nil {
		pass(w, "Installed: "+ver)
		// Check if installed via npm (may shadow official binary)
		npmInfo := setup.DetectNpmClaude()
		if npmInfo.Detected {
			warn(w, "Claude Code is installed via npm (@anthropic-ai/claude-code)")
			fmt.Fprintln(w, "    The npm version may shadow the official binary in PATH.")
			fmt.Fprintln(w, "    Fix: npm uninstall -g @anthropic-ai/claude-code")
			warnings++
		}
	} else {
		fail(w, "Claude Code CLI not found")
		fmt.Fprintln(w, "    Install: curl -fsSL https://claude.ai/install.sh | bash")
		issues++
	}

	return issues, warnings
}

// checkClaudeWorkspace verifies the claude-workspace binary is in PATH and checks for updates.
func checkClaudeWorkspace(w io.Writer) (int, int) {
	issues := 0
	warnings := 0

	platform.PrintSectionLabel(w, "claude-workspace CLI")
	if platform.Exists("claude-workspace") {
		pass(w, "claude-workspace is in PATH")
	} else {
		warn(w, "claude-workspace not found in PATH")
		fmt.Fprintln(w, "    Run: claude-workspace setup")
		warnings++
	}

	// Soft update check (3s timeout, skip on failure)
	checkForUpdate(w)

	return issues, warnings
}

// checkGit verifies Git is installed.
func checkGit(w io.Writer) (int, int) {
	issues := 0
	warnings := 0

	platform.PrintSectionLabel(w, "Git")
	if ver, err := platform.Output("git", "--version"); err == nil {
		pass(w, ver)
	} else {
		fail(w, "Git not found")
		issues++
	}

	return issues, warnings
}

// checkNode verifies Node.js is installed and meets minimum version requirements.
func checkNode(w io.Writer) (int, int) {
	issues := 0
	warnings := 0

	platform.PrintSectionLabel(w, "Node.js")
	nodeTool := tools.Node()
	switch {
	case nodeTool.IsInstalled():
		if ver, err := platform.Output("node", "--version"); err == nil {
			pass(w, "Node.js: "+ver)
		} else {
			pass(w, "Node.js: installed")
		}
		if platform.Exists("npx") {
			pass(w, "npx: available")
		} else {
			warn(w, "npx not found (required for MCP servers)")
			warnings++
		}
	case platform.Exists("node"):
		// node exists but doesn't meet minimum version
		ver, _ := platform.Output("node", "--version")
		warn(w, fmt.Sprintf("Node.js %s is below minimum version (v%d+)", ver, tools.NodeMinMajor))
		fmt.Fprintln(w, "    Upgrade Node.js: https://nodejs.org")
		warnings++
	default:
		fail(w, "Node.js not found (required for MCP servers)")
		fmt.Fprintln(w, "    Install: https://nodejs.org or run: claude-workspace setup")
		issues++
	}

	return issues, warnings
}

// checkGlobalConfig verifies global settings.json and CLAUDE.md exist and are valid.
func checkGlobalConfig(w io.Writer, home string) (int, int) {
	issues := 0
	warnings := 0

	platform.PrintSectionLabel(w, "Global Configuration")

	globalSettingsPath := filepath.Join(home, ".claude", "settings.json")
	if platform.FileExists(globalSettingsPath) {
		pass(w, "~/.claude/settings.json exists")
		var settings map[string]json.RawMessage
		if err := platform.ReadJSONFile(globalSettingsPath, &settings); err == nil {
			var env map[string]string
			if raw, ok := settings["env"]; ok {
				if json.Unmarshal(raw, &env) == nil {
					if model, ok := env["CLAUDE_CODE_SUBAGENT_MODEL"]; ok {
						pass(w, "Subagent model: "+model)
					}
					if env["CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS"] == "1" {
						pass(w, "Agent teams: enabled")
					}
				}
			}
		} else {
			warn(w, "Could not parse global settings")
			warnings++
		}
	} else {
		warn(w, "~/.claude/settings.json not found. Run 'claude-workspace setup'")
		warnings++
	}

	globalClaudeMd := filepath.Join(home, ".claude", "CLAUDE.md")
	if platform.FileExists(globalClaudeMd) {
		pass(w, "~/.claude/CLAUDE.md exists")
	} else {
		warn(w, "~/.claude/CLAUDE.md not found")
		warnings++
	}

	return issues, warnings
}

// checkProjectConfig runs table-driven checks for expected project configuration files.
func checkProjectConfig(w io.Writer, cwd string) (int, int) {
	issues := 0
	warnings := 0

	platform.PrintSectionLabel(w, "Project Configuration")

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
			pass(w, check.label+": "+check.path)
		case check.required:
			fail(w, check.label+" not found: "+check.path)
			issues++
		default:
			warn(w, check.label+" not found: "+check.path)
			warnings++
		}
	}

	return issues, warnings
}

// checkAgents scans the agents directory for .md agent definition files.
func checkAgents(w io.Writer, cwd string) (int, int) {
	issues := 0
	warnings := 0

	platform.PrintSectionLabel(w, "Agents")
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
				pass(w, fmt.Sprintf("Found %d agents: %s", len(agents), strings.Join(agents, ", ")))
			} else {
				warn(w, "No agent definitions found")
				warnings++
			}
		}
	}

	return issues, warnings
}

// checkSkills walks the skills directory for SKILL.md definition files.
func checkSkills(w io.Writer, cwd string) (int, int) {
	issues := 0
	warnings := 0

	platform.PrintSectionLabel(w, "Skills")
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
			pass(w, fmt.Sprintf("Found %d skills: %s", len(skills), strings.Join(skills, ", ")))
		} else {
			warn(w, "No skill definitions found")
			warnings++
		}
	}

	return issues, warnings
}

// checkHooks verifies hook shell scripts in the hooks directory are executable.
func checkHooks(w io.Writer, cwd string) (int, int) {
	issues := 0
	warnings := 0

	platform.PrintSectionLabel(w, "Hooks")
	hooksDir := filepath.Join(cwd, ".claude", "hooks")
	if platform.FileExists(hooksDir) {
		entries, err := os.ReadDir(hooksDir)
		if err == nil {
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".sh") {
					hookPath := filepath.Join(hooksDir, e.Name())
					if platform.IsExecutable(hookPath) {
						pass(w, e.Name()+": executable")
					} else {
						fail(w, fmt.Sprintf("%s: not executable. Run: chmod +x %s", e.Name(), hookPath))
						issues++
					}
				}
			}
		}
	}

	return issues, warnings
}

// checkHookConfig validates that settings.json hook references are parseable.
func checkHookConfig(w io.Writer, cwd string) (int, int) {
	issues := 0
	warnings := 0

	platform.PrintSectionLabel(w, "Hook Configuration")
	settingsPath := filepath.Join(cwd, ".claude", "settings.json")
	if platform.FileExists(settingsPath) {
		var settings map[string]json.RawMessage
		if err := platform.ReadJSONFile(settingsPath, &settings); err == nil {
			if raw, ok := settings["hooks"]; ok {
				var hooks map[string]json.RawMessage
				if json.Unmarshal(raw, &hooks) == nil {
					hookCount := countHookCommands(hooks)
					pass(w, fmt.Sprintf("%d hook commands configured", hookCount))
				}
			}
		} else {
			warn(w, "Could not validate hook configuration")
			warnings++
		}
	}

	return issues, warnings
}

// checkMCPServers validates the .mcp.json configuration file.
func checkMCPServers(w io.Writer, cwd string) (int, int) {
	issues := 0
	warnings := 0

	platform.PrintSectionLabel(w, "MCP Servers")
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
				pass(w, fmt.Sprintf("%d MCP servers configured: %s", len(servers), strings.Join(servers, ", ")))
			} else {
				warn(w, "No MCP servers configured in .mcp.json")
				warnings++
			}
		} else {
			fail(w, "Could not parse .mcp.json")
			issues++
		}
	}

	return issues, warnings
}

// checkAuth verifies API key or OAuth authentication is configured.
func checkAuth(w io.Writer, home string) (int, int) {
	issues := 0
	warnings := 0

	platform.PrintSectionLabel(w, "Authentication")
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		pass(w, "ANTHROPIC_API_KEY is set")
	} else {
		claudeConfig := filepath.Join(home, ".claude.json")
		if platform.FileExists(claudeConfig) {
			var config map[string]json.RawMessage
			if err := platform.ReadJSONFile(claudeConfig, &config); err == nil {
				if _, ok := config["oauthAccount"]; ok {
					pass(w, "OAuth authentication configured")
				} else {
					warn(w, "No API key or OAuth found. Run: claude-workspace setup")
					warnings++
				}
			} else {
				warn(w, "Could not read authentication config")
				warnings++
			}
		} else {
			warn(w, "No authentication configured. Run: claude-workspace setup")
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

func pass(w io.Writer, msg string) {
	platform.PrintOK(w, msg)
}

func fail(w io.Writer, msg string) {
	platform.PrintFail(w, msg)
}

func warn(w io.Writer, msg string) {
	platform.PrintWarn(w, msg)
}

func checkForUpdate(w io.Writer) {
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
			platform.PrintInfo(w, fmt.Sprintf("Update available: %s → %s", currentVer, res.release.TagName))
			fmt.Fprintln(w, "    Run: claude-workspace upgrade")
		}
	case <-time.After(3 * time.Second):
		return // timeout, skip silently
	}
}
