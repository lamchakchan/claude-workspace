package statusline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

// Run is the entry point for the statusline command.
// args is os.Args[2:] (everything after "statusline").
func Run(args []string) error {
	force := false
	for _, a := range args {
		if a == "--force" {
			force = true
		}
	}
	return configure(force)
}

// configure reads ~/.claude/settings.json, detects the best runtime, and writes the statusLine block.
func configure(force bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}
	settingsPath := filepath.Join(home, ".claude", "settings.json")

	var settings map[string]interface{}
	if platform.FileExists(settingsPath) {
		if err := platform.ReadJSONFile(settingsPath, &settings); err != nil {
			return fmt.Errorf("reading settings: %w", err)
		}
	} else {
		settings = make(map[string]interface{})
	}

	if !force {
		if _, exists := settings["statusLine"]; exists {
			platform.PrintOK(os.Stdout, "statusLine already configured in ~/.claude/settings.json (use --force to overwrite)")
			return nil
		}
	}

	cmd := detectStatusLineCommand()
	fmt.Printf("  Runtime detected: %s\n", runtimeLabel(cmd))

	if strings.HasPrefix(cmd, "jq") {
		platform.PrintWarningLine(os.Stdout, "bun and npx not found — using inline jq fallback. Ensure jq is installed.")
	}

	settings["statusLine"] = map[string]interface{}{
		"type":    "command",
		"command": cmd,
		"padding": 0,
	}

	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0755); err != nil {
		return fmt.Errorf("creating ~/.claude: %w", err)
	}

	if err := platform.WriteJSONFile(settingsPath, settings); err != nil {
		return fmt.Errorf("writing settings: %w", err)
	}

	platform.PrintOK(os.Stdout, "statusLine configured in ~/.claude/settings.json")
	fmt.Printf("  Command: %s\n", cmd)
	fmt.Println("  Restart Claude Code to activate the statusline.")
	return nil
}

// detectStatusLineCommand returns the best available statusline command string.
func detectStatusLineCommand() string {
	if platform.Exists("bun") {
		return "bun x ccusage statusline"
	}
	if platform.Exists("npx") {
		return "npx -y ccusage statusline"
	}
	return `jq -r '"\(.model.display_name) | $\(.cost.total_cost_usd | . * 1000 | round / 1000) | \(.context_window.used_percentage)% ctx"'`
}

// runtimeLabel returns a human-readable label for the detected command.
func runtimeLabel(cmd string) string {
	switch {
	case strings.HasPrefix(cmd, "bun"):
		return "bun (ccusage)"
	case strings.HasPrefix(cmd, "npx"):
		return "npx (ccusage)"
	default:
		return "inline jq (fallback)"
	}
}
