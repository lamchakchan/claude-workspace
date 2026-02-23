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
			return platform.RunQuiet("npm", "install", "-g", "prettier")
		},
	}
}
