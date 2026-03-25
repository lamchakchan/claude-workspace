package platform

import (
	"os"
	"path/filepath"
)

// IsClaudeAuthenticated checks whether Claude CLI credentials are available.
// It returns true if any of the following hold:
//   - ANTHROPIC_API_KEY environment variable is set
//   - ~/.claude.json contains an "oauthAccount" key
//   - ~/.claude.json contains a "primaryApiKey" key
func IsClaudeAuthenticated() bool {
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		return true
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	return hasClaudeConfigAuth(filepath.Join(home, ".claude.json"))
}

// hasClaudeConfigAuth checks whether a Claude config file contains authentication
// credentials (oauthAccount or primaryApiKey).
func hasClaudeConfigAuth(configPath string) bool {
	if !FileExists(configPath) {
		return false
	}

	config, err := ReadJSONFileRaw(configPath)
	if err != nil {
		return false
	}
	if _, hasOAuth := config["oauthAccount"]; hasOAuth {
		return true
	}
	if _, hasKey := config["primaryApiKey"]; hasKey {
		return true
	}

	return false
}
