// Package plugins implements the "plugins" command for managing Claude Code plugins.
package plugins

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

// Run dispatches the plugins subcommand.
func Run(args []string) error {
	if len(args) == 0 {
		return ListTo(os.Stdout)
	}
	switch args[0] {
	case "list":
		return ListTo(os.Stdout)
	case "add", "install":
		return Add(args[1:])
	case "remove", "uninstall":
		return Remove(args[1:])
	case "available":
		return AvailableTo(os.Stdout)
	default:
		return fmt.Errorf("unknown plugins subcommand: %s (available: list, add, remove, available)", args[0])
	}
}

// ListTo writes a formatted list of installed plugins to w.
func ListTo(w io.Writer) error {
	plugins, err := DiscoverInstalled()
	if err != nil {
		return fmt.Errorf("listing installed plugins: %w", err)
	}

	if len(plugins) == 0 {
		fmt.Fprintln(w, "No plugins installed.")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Install one with: claude-workspace plugins add <plugin>@<marketplace>")
		fmt.Fprintln(w, "Browse available:  claude-workspace plugins available")
		return nil
	}

	platform.PrintBanner(w, "Installed Plugins")

	for _, p := range plugins {
		scope := p.Scope
		if scope == "" {
			scope = "user"
		}
		name := p.Name
		if p.Marketplace != "" {
			name += "@" + p.Marketplace
		}
		platform.PrintOK(w, name)
		fmt.Fprintf(w, "    Scope: %s", scope)
		if p.Version != "" {
			fmt.Fprintf(w, "  Version: %s", p.Version)
		}
		if !p.Enabled {
			fmt.Fprintf(w, "  (disabled)")
		}
		fmt.Fprintln(w)
		if p.Description != "" {
			fmt.Fprintf(w, "    %s\n", p.Description)
		}
	}
	return nil
}

// Add installs a plugin via the Claude CLI.
func Add(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: claude-workspace plugins add <plugin[@marketplace]> [--scope user|project]")
	}

	plugin := args[0]
	scope := "user"
	for i := 1; i < len(args); i++ {
		if args[i] == "--scope" && i+1 < len(args) {
			scope = args[i+1]
			i++
		}
	}

	fmt.Printf("Installing %s (scope: %s)...\n", plugin, scope)
	if err := platform.Run("claude", "plugin", "install", plugin, "--scope", scope); err != nil {
		return fmt.Errorf("installing plugin %s: %w", plugin, err)
	}
	return nil
}

// Remove uninstalls a plugin via the Claude CLI.
func Remove(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: claude-workspace plugins remove <plugin[@marketplace]> [--scope user|project]")
	}

	plugin := args[0]
	scope := "user"
	for i := 1; i < len(args); i++ {
		if args[i] == "--scope" && i+1 < len(args) {
			scope = args[i+1]
			i++
		}
	}

	fmt.Printf("Removing %s (scope: %s)...\n", plugin, scope)
	if err := platform.Run("claude", "plugin", "uninstall", plugin, "--scope", scope); err != nil {
		return fmt.Errorf("removing plugin %s: %w", plugin, err)
	}
	return nil
}

// AvailableTo writes a formatted list of available plugins from configured marketplaces.
func AvailableTo(w io.Writer) error {
	plugins, err := DiscoverAvailable()
	if err != nil {
		return fmt.Errorf("listing available plugins: %w", err)
	}

	if len(plugins) == 0 {
		fmt.Fprintln(w, "No marketplace plugins found.")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Add a marketplace first:")
		fmt.Fprintln(w, "  claude plugin marketplace add anthropics/claude-plugins-official")
		return nil
	}

	installed, _ := DiscoverInstalled()
	installedSet := make(map[string]bool, len(installed))
	for _, p := range installed {
		key := p.Name
		if p.Marketplace != "" {
			key += "@" + p.Marketplace
		}
		installedSet[key] = true
	}

	platform.PrintBanner(w, "Available Plugins")

	// Group by marketplace
	byMarketplace := make(map[string][]Plugin)
	var order []string
	for _, p := range plugins {
		mp := p.Marketplace
		if mp == "" {
			mp = "unknown"
		}
		if _, seen := byMarketplace[mp]; !seen {
			order = append(order, mp)
		}
		byMarketplace[mp] = append(byMarketplace[mp], p)
	}

	for _, mp := range order {
		fmt.Fprintf(w, "\n  %s\n", strings.ToUpper(mp))
		for _, p := range byMarketplace[mp] {
			key := p.Name + "@" + mp
			marker := "  "
			if installedSet[key] {
				marker = "✓ "
			}
			fmt.Fprintf(w, "  %s%s", marker, p.Name)
			if p.Description != "" {
				fmt.Fprintf(w, "  — %s", p.Description)
			}
			fmt.Fprintln(w)
		}
	}
	return nil
}
