package plugins

import (
	"fmt"
	"os"
	"strings"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

// RunMarketplace dispatches the marketplace subcommand.
func RunMarketplace(args []string) error {
	if len(args) == 0 {
		return MarketplaceListTo(os.Stdout)
	}
	switch args[0] {
	case "list":
		return MarketplaceListTo(os.Stdout)
	case "add":
		return MarketplaceAdd(args[1:])
	case "remove":
		return MarketplaceRemove(args[1:])
	default:
		return fmt.Errorf("unknown marketplace subcommand: %s (available: list, add, remove)", args[0])
	}
}

// MarketplaceAdd adds a plugin marketplace via the Claude CLI.
// Accepts either owner/repo format or a local filesystem path.
func MarketplaceAdd(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: claude-workspace plugins marketplace add <owner/repo | /path/to/repo>")
	}
	target := args[0]

	if !isLocalPath(target) {
		if !strings.Contains(target, "/") || strings.Count(target, "/") != 1 {
			return fmt.Errorf("invalid marketplace format: %q (expected owner/repo or local path)", target)
		}
	}

	fmt.Printf("Adding marketplace %s...\n", target)
	if err := platform.Run("claude", "plugin", "marketplace", "add", target); err != nil {
		return fmt.Errorf("adding marketplace %s: %w", target, err)
	}
	return nil
}

// isLocalPath returns true if the argument looks like a filesystem path
// (absolute, relative, or home-relative) rather than an owner/repo identifier.
func isLocalPath(s string) bool {
	return strings.HasPrefix(s, "/") ||
		strings.HasPrefix(s, "./") ||
		strings.HasPrefix(s, "../") ||
		strings.HasPrefix(s, "~/") ||
		s == "." || s == ".." || s == "~"
}

// MarketplaceRemove removes a plugin marketplace via the Claude CLI.
func MarketplaceRemove(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: claude-workspace plugins marketplace remove <name>")
	}
	name := args[0]

	fmt.Printf("Removing marketplace %s...\n", name)
	if err := platform.Run("claude", "plugin", "marketplace", "remove", name); err != nil {
		return fmt.Errorf("removing marketplace %s: %w", name, err)
	}
	return nil
}
