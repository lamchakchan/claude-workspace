package tools

import (
	"fmt"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

// Prettier returns the prettier tool definition.
func Prettier() Tool {
	return Tool{
		Name:       "prettier",
		Purpose:    "Auto-format hook (JS/TS/JSON/CSS)",
		InstallCmd: "npm install -g prettier",
		InstallFn: func() error {
			if !platform.Exists("npm") {
				return fmt.Errorf("npm not available")
			}
			// Try without sudo first
			if err := platform.RunQuiet("npm", "install", "-g", "prettier"); err == nil {
				return nil
			}
			// Retry with sudo (needed when node is installed system-wide via apt)
			if platform.Exists("sudo") {
				return platform.RunQuiet("sudo", "npm", "install", "-g", "prettier")
			}
			return fmt.Errorf("npm install -g prettier failed (no write access to global npm directory)")
		},
	}
}
