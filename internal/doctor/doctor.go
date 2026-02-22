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
	"github.com/lamchakchan/claude-workspace/internal/upgrade"
)

func Run() error {
	fmt.Println("\n=== Claude Platform Health Check ===")

	issues := 0
	warnings := 0

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}

	// 1. Check Claude Code CLI
	fmt.Println("\n[Claude Code CLI]")
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

	// 2. Check claude-workspace in PATH
	fmt.Println("\n[claude-workspace CLI]")
	if platform.Exists("claude-workspace") {
		pass("claude-workspace is in PATH")
	} else {
		warn("claude-workspace not found in PATH")
		fmt.Println("    Run: claude-workspace setup")
		warnings++
	}

	// Soft update check (3s timeout, skip on failure)
	checkForUpdate()

	// 3. Check Git
	fmt.Println("\n[Git]")
	if ver, err := platform.Output("git", "--version"); err == nil {
		pass(ver)
	} else {
		fail("Git not found")
		issues++
	}

	// 4. Check Global Settings
	fmt.Println("\n[Global Configuration]")

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

	// 5. Check Project Configuration
	fmt.Println("\n[Project Configuration]")
	cwd, _ := os.Getwd()

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
		if platform.FileExists(fullPath) {
			pass(check.label + ": " + check.path)
		} else if check.required {
			fail(check.label + " not found: " + check.path)
			issues++
		} else {
			warn(check.label + " not found: " + check.path)
			warnings++
		}
	}

	// 6. Check Agents
	fmt.Println("\n[Agents]")
	agentsDir := filepath.Join(cwd, ".claude/agents")
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

	// 7. Check Skills
	fmt.Println("\n[Skills]")
	skillsDir := filepath.Join(cwd, ".claude/skills")
	if platform.FileExists(skillsDir) {
		var skills []string
		filepath.WalkDir(skillsDir, func(path string, d os.DirEntry, err error) error {
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

	// 8. Check Hooks
	fmt.Println("\n[Hooks]")
	hooksDir := filepath.Join(cwd, ".claude/hooks")
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

	// 9. Check settings.json hooks reference valid scripts
	fmt.Println("\n[Hook Configuration]")
	settingsPath := filepath.Join(cwd, ".claude/settings.json")
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

	// 10. Check MCP servers
	fmt.Println("\n[MCP Servers]")
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

	// 11. Check Authentication
	fmt.Println("\n[Authentication]")
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

	// Summary
	fmt.Println("\n=== Summary ===")
	if issues == 0 && warnings == 0 {
		fmt.Println("All checks passed. Platform is healthy.")
	} else {
		if issues > 0 {
			fmt.Printf("Issues: %d (must fix)\n", issues)
		}
		if warnings > 0 {
			fmt.Printf("Warnings: %d (optional)\n", warnings)
		}
	}
	fmt.Println()

	return nil
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
	fmt.Printf("  [OK] %s\n", msg)
}

func fail(msg string) {
	fmt.Printf("  [FAIL] %s\n", msg)
}

func warn(msg string) {
	fmt.Printf("  [WARN] %s\n", msg)
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
			fmt.Printf("  [INFO] Update available: %s → %s\n", currentVer, res.release.TagName)
			fmt.Println("    Run: claude-workspace upgrade")
		}
	case <-time.After(3 * time.Second):
		return // timeout, skip silently
	}
}
