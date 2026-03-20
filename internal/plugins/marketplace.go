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
func MarketplaceAdd(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: claude-workspace plugins marketplace add <owner/repo>")
	}
	repo := args[0]
	if !strings.Contains(repo, "/") || strings.Count(repo, "/") != 1 {
		return fmt.Errorf("invalid marketplace format: %q (expected owner/repo)", repo)
	}

	fmt.Printf("Adding marketplace %s...\n", repo)
	if err := platform.Run("claude", "plugin", "marketplace", "add", repo); err != nil {
		return fmt.Errorf("adding marketplace %s: %w", repo, err)
	}
	return nil
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
