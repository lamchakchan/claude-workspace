package plugins

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

// Plugin represents a Claude Code plugin.
type Plugin struct {
	Name        string
	Marketplace string
	Scope       string
	Version     string
	Enabled     bool
	Description string
}

// installedPluginsFile is the JSON structure of ~/.claude/plugins/installed_plugins.json.
type installedPluginsFile struct {
	Plugins map[string][]installedEntry `json:"plugins"`
}

type installedEntry struct {
	Scope       string `json:"scope"`
	Version     string `json:"version"`
	InstallPath string `json:"installPath"`
}

// pluginManifest is the structure of .claude-plugin/plugin.json inside a plugin.
type pluginManifest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// DiscoverInstalled returns all installed plugins by reading the installed_plugins.json file.
func DiscoverInstalled() ([]Plugin, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(home, ".claude", "plugins", "installed_plugins.json")
	if !platform.FileExists(path) {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var file installedPluginsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}

	// Read enabled state from settings
	enabledMap := readEnabledPlugins(home)

	var plugins []Plugin
	for key, entries := range file.Plugins {
		name, marketplace := splitPluginKey(key)
		for _, e := range entries {
			desc := readPluginDescription(e.InstallPath)
			enabled := true
			if v, ok := enabledMap[key]; ok {
				enabled = v
			}
			plugins = append(plugins, Plugin{
				Name:        name,
				Marketplace: marketplace,
				Scope:       e.Scope,
				Version:     e.Version,
				Enabled:     enabled,
				Description: desc,
			})
		}
	}

	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].Name < plugins[j].Name
	})
	return plugins, nil
}

// DiscoverAvailable returns plugins from all configured marketplace directories.
func DiscoverAvailable() ([]Plugin, error) {
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

	var plugins []Plugin
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		marketplace := entry.Name()
		pluginsDir := filepath.Join(marketplacesDir, marketplace, "plugins")
		pluginEntries, err := os.ReadDir(pluginsDir)
		if err != nil {
			continue
		}
		for _, pe := range pluginEntries {
			if !pe.IsDir() {
				continue
			}
			desc := readPluginDescription(filepath.Join(pluginsDir, pe.Name()))
			plugins = append(plugins, Plugin{
				Name:        pe.Name(),
				Marketplace: marketplace,
				Enabled:     true,
				Description: desc,
			})
		}
	}

	sort.Slice(plugins, func(i, j int) bool {
		if plugins[i].Marketplace != plugins[j].Marketplace {
			return plugins[i].Marketplace < plugins[j].Marketplace
		}
		return plugins[i].Name < plugins[j].Name
	})
	return plugins, nil
}

// splitPluginKey splits "name@marketplace" into its components.
func splitPluginKey(key string) (name, marketplace string) {
	if i := strings.LastIndex(key, "@"); i >= 0 {
		return key[:i], key[i+1:]
	}
	return key, ""
}

// readPluginDescription reads the description from a plugin's manifest file.
func readPluginDescription(installPath string) string {
	manifestPath := filepath.Join(installPath, ".claude-plugin", "plugin.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return ""
	}
	var m pluginManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return ""
	}
	return m.Description
}

// readEnabledPlugins reads the enabledPlugins map from ~/.claude/settings.json.
func readEnabledPlugins(home string) map[string]bool {
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil
	}
	var settings map[string]json.RawMessage
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil
	}
	raw, ok := settings["enabledPlugins"]
	if !ok {
		return nil
	}
	var enabled map[string]bool
	if err := json.Unmarshal(raw, &enabled); err != nil {
		return nil
	}
	return enabled
}
