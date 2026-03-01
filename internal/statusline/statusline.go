// Package statusline implements the "statusline" command, which configures the
// Claude Code status bar to display session cost, context usage, model name,
// and weekly subscription reset countdown.
package statusline

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

// Run is the entry point for the statusline command.
// args is os.Args[2:] (everything after "statusline").
func Run(args []string) error {
	if len(args) > 0 && args[0] == "render" {
		return RunRender(args[1:])
	}
	force := false
	for _, a := range args {
		if a == "--force" {
			force = true
		}
	}
	return configureTo(os.Stdout, force)
}

// RunTo is like Run but writes all output to w instead of os.Stdout.
func RunTo(w io.Writer, args []string) error {
	if len(args) > 0 && args[0] == "render" {
		return RunRender(args[1:])
	}
	force := false
	for _, a := range args {
		if a == "--force" {
			force = true
		}
	}
	return configureTo(w, force)
}

// configureTo writes ~/.claude/statusline.sh and registers it in ~/.claude/settings.json.
func configureTo(w io.Writer, force bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}

	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("creating ~/.claude: %w", err)
	}

	settingsPath := filepath.Join(claudeDir, "settings.json")

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
			platform.PrintOK(w, "statusLine already configured in ~/.claude/settings.json (use --force to overwrite)")
			return nil
		}
	}

	script, err := platform.ReadGlobalAsset("statusline.sh")
	if err != nil {
		return fmt.Errorf("reading statusline template: %w", err)
	}

	scriptPath := filepath.Join(claudeDir, "statusline.sh")
	if err := writeWrapperScript(scriptPath, script); err != nil {
		return fmt.Errorf("writing statusline script: %w", err)
	}
	fmt.Fprintf(w, "  Script written: %s\n", scriptPath)

	cmd := "bash " + scriptPath
	settings["statusLine"] = map[string]interface{}{
		"type":    "command",
		"command": cmd,
		"padding": 0,
	}

	if err := platform.WriteJSONFile(settingsPath, settings); err != nil {
		return fmt.Errorf("writing settings: %w", err)
	}

	platform.PrintOK(w, "statusLine configured in ~/.claude/settings.json")
	fmt.Fprintln(w, "  Restart Claude Code to activate the statusline.")
	return nil
}

// writeWrapperScript writes the statusline shell script content to the given path.
func writeWrapperScript(path string, content []byte) error {
	return os.WriteFile(path, content, 0755)
}
