// Package hooks discovers and lists Claude Code hook scripts and hook configuration.
package hooks

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

// HookScript represents a discovered hook shell script.
type HookScript struct {
	Name        string
	Description string
	Path        string
}

// HookConfig represents a single hook entry from settings.json.
type HookConfig struct {
	Event         string
	Matcher       string
	Command       string
	StatusMessage string
}

// Run routes the hooks subcommand.
func Run(args []string) error {
	subcmd := "list"
	if len(args) > 0 {
		subcmd = args[0]
	}
	switch subcmd {
	case "list":
		return list()
	default:
		fmt.Fprintf(os.Stderr, "Unknown hooks subcommand: %s\n", subcmd)
		fmt.Fprintln(os.Stderr, "Usage: claude-workspace hooks [list]")
		return fmt.Errorf("unknown subcommand: %s", subcmd)
	}
}

// list discovers hook scripts and configuration from multiple sources and prints them.
func list() error {
	platform.PrintBanner(os.Stdout, "Hooks")
	fmt.Println()

	anyFound := false

	// 1. Project hook scripts
	cwd, err := os.Getwd()
	if err == nil {
		hooksDir := filepath.Join(cwd, ".claude", "hooks")
		if platform.FileExists(hooksDir) {
			scripts := DiscoverHookScripts(hooksDir)
			if len(scripts) > 0 {
				anyFound = true
				platform.PrintSection(os.Stdout, "Project Hook Scripts (.claude/hooks/)")
				printScriptTable(scripts)
			}
		}
	}

	// 2. Hook configuration from settings.json
	if err == nil {
		settingsPath := filepath.Join(cwd, ".claude", "settings.json")
		if platform.FileExists(settingsPath) {
			configs := DiscoverHookConfig(settingsPath)
			if len(configs) > 0 {
				anyFound = true
				platform.PrintSection(os.Stdout, "Hook Configuration (settings.json)")
				printConfigTable(configs)
			}
		}
	}

	if !anyFound {
		fmt.Println("  No hooks found.")
		fmt.Println()
		fmt.Println("  Create a hook script: .claude/hooks/my-hook.sh (must be executable)")
		fmt.Println("  Configure hooks:      .claude/settings.json under \"hooks\" key")
		fmt.Println()
		return nil
	}

	// Tips
	platform.PrintSection(os.Stdout, "Tips")
	fmt.Println("  Hooks are shell scripts that run before/after tool use or on events.")
	fmt.Println("  Create new:  .claude/hooks/my-hook.sh (must be executable)")
	fmt.Println("  Configure:   .claude/settings.json under \"hooks\" key")
	fmt.Println()

	return nil
}

// DiscoverHookScripts walks a directory for .sh files and extracts their descriptions.
func DiscoverHookScripts(root string) []HookScript {
	var scripts []HookScript
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".sh") {
			return nil
		}
		desc := parseScriptDescription(path)
		scripts = append(scripts, HookScript{Name: d.Name(), Description: desc, Path: path})
		return nil
	})
	return scripts
}

// DiscoverEmbeddedHookScripts walks the embedded FS for .sh files.
func DiscoverEmbeddedHookScripts(efs fs.FS, root string) []HookScript {
	var scripts []HookScript
	_ = fs.WalkDir(efs, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".sh") {
			return nil
		}
		data, readErr := fs.ReadFile(efs, path)
		if readErr != nil {
			return nil
		}
		desc := parseScriptDescriptionBytes(data)
		scripts = append(scripts, HookScript{Name: d.Name(), Description: desc})
		return nil
	})
	return scripts
}

// hookEntry represents the JSON structure for a hook event entry in settings.json.
type hookEntry struct {
	Matcher string `json:"matcher"`
	Hooks   []struct {
		Type          string `json:"type"`
		Command       string `json:"command"`
		StatusMessage string `json:"statusMessage"`
	} `json:"hooks"`
}

// DiscoverHookConfig parses settings.json and extracts hook configuration.
func DiscoverHookConfig(settingsPath string) []HookConfig {
	raw, err := platform.ReadJSONFileRaw(settingsPath)
	if err != nil {
		return nil
	}

	hooksRaw, ok := raw["hooks"]
	if !ok {
		return nil
	}

	var events map[string][]hookEntry
	if err := json.Unmarshal(hooksRaw, &events); err != nil {
		return nil
	}

	var configs []HookConfig
	for event, entries := range events {
		for _, entry := range entries {
			matcher := entry.Matcher
			if matcher == "" {
				matcher = "(any)"
			}
			for _, h := range entry.Hooks {
				configs = append(configs, HookConfig{
					Event:         event,
					Matcher:       matcher,
					Command:       h.Command,
					StatusMessage: h.StatusMessage,
				})
			}
		}
	}
	return configs
}

// parseScriptDescription reads a file and extracts the description comment.
func parseScriptDescription(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return parseScriptDescriptionBytes(data)
}

// parseScriptDescriptionBytes extracts a description from the first comment line
// after skipping blank lines, the shebang, and set directives.
func parseScriptDescriptionBytes(data []byte) string {
	lines := strings.Split(string(data), "\n")
	limit := len(lines)
	if limit > 10 {
		limit = 10
	}
	for i := 0; i < limit; i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#!") {
			continue
		}
		if strings.HasPrefix(line, "set ") {
			continue
		}
		if strings.HasPrefix(line, "#") {
			return strings.TrimSpace(strings.TrimPrefix(line, "#"))
		}
		// Non-comment, non-skippable line — stop looking
		return ""
	}
	return ""
}

// printScriptTable prints hook scripts in aligned columns.
func printScriptTable(scripts []HookScript) {
	if len(scripts) == 0 {
		return
	}

	maxName := 0
	for _, s := range scripts {
		if len(s.Name) > maxName {
			maxName = len(s.Name)
		}
	}

	for _, s := range scripts {
		desc := s.Description
		if len(desc) > 70 {
			desc = desc[:67] + "..."
		}
		fmt.Printf("  %-*s  %s\n", maxName, s.Name, desc)
	}
	fmt.Println()
}

// printConfigTable prints hook configuration in aligned columns.
func printConfigTable(configs []HookConfig) {
	if len(configs) == 0 {
		return
	}

	maxEvent := len("EVENT")
	maxMatcher := len("MATCHER")
	for _, c := range configs {
		if len(c.Event) > maxEvent {
			maxEvent = len(c.Event)
		}
		if len(c.Matcher) > maxMatcher {
			maxMatcher = len(c.Matcher)
		}
	}

	fmt.Printf("  %-*s  %-*s  %s\n", maxEvent, "EVENT", maxMatcher, "MATCHER", "STATUS MESSAGE")
	for _, c := range configs {
		msg := c.StatusMessage
		if len(msg) > 50 {
			msg = msg[:47] + "..."
		}
		fmt.Printf("  %-*s  %-*s  %s\n", maxEvent, c.Event, maxMatcher, c.Matcher, msg)
	}
	fmt.Println()
}
