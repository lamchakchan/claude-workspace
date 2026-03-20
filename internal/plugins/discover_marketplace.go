package plugins

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

// Marketplace represents a configured plugin marketplace.
type Marketplace struct {
	Name        string
	Repo        string
	PluginCount int
	Path        string
}

// DiscoverMarketplaces returns all configured marketplaces by reading ~/.claude/plugins/marketplaces/.
func DiscoverMarketplaces() ([]Marketplace, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	marketplacesDir := filepath.Join(home, ".claude", "plugins", "marketplaces")
	entries, err := os.ReadDir(marketplacesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	marketplaces := make([]Marketplace, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		mp := Marketplace{
			Name: name,
			Path: filepath.Join(marketplacesDir, name),
		}

		// Count plugins
		pluginsDir := filepath.Join(mp.Path, "plugins")
		pluginEntries, err := os.ReadDir(pluginsDir)
		if err == nil {
			for _, pe := range pluginEntries {
				if pe.IsDir() {
					mp.PluginCount++
				}
			}
		}

		// Try to read repo URL from .git/config
		mp.Repo = readGitRemote(mp.Path)

		marketplaces = append(marketplaces, mp)
	}

	sort.Slice(marketplaces, func(i, j int) bool {
		return marketplaces[i].Name < marketplaces[j].Name
	})
	return marketplaces, nil
}

// MarketplaceListTo writes a formatted list of configured marketplaces to w.
func MarketplaceListTo(w io.Writer) error {
	marketplaces, err := DiscoverMarketplaces()
	if err != nil {
		return fmt.Errorf("listing marketplaces: %w", err)
	}

	if len(marketplaces) == 0 {
		fmt.Fprintln(w, "No plugin marketplaces configured.")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Add one with: claude-workspace plugins marketplace add <owner/repo>")
		return nil
	}

	platform.PrintBanner(w, "Plugin Marketplaces")

	for _, mp := range marketplaces {
		platform.PrintOK(w, mp.Name)
		if mp.Repo != "" {
			fmt.Fprintf(w, "    Repo: %s", mp.Repo)
		}
		fmt.Fprintf(w, "    Plugins: %d\n", mp.PluginCount)
	}
	return nil
}

// readGitRemote reads the remote origin URL from a .git/config file.
func readGitRemote(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, ".git", "config"))
	if err != nil {
		return ""
	}
	// Simple parse: look for url = line after [remote "origin"]
	lines := strings.Split(string(data), "\n")
	inOrigin := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == `[remote "origin"]` {
			inOrigin = true
			continue
		}
		if inOrigin && strings.HasPrefix(trimmed, "[") {
			break
		}
		if inOrigin && strings.HasPrefix(trimmed, "url = ") {
			return strings.TrimPrefix(trimmed, "url = ")
		}
	}
	return ""
}
