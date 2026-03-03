// Package mcpregistry loads pre-defined MCP server recipes from embedded JSON
// config files. Each recipe describes a server that can be added via the TUI
// picker, with pre-filled form values for name, command/URL, env vars, and scope.
package mcpregistry

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

// Transport identifies how a client connects to an MCP server.
type Transport string

const (
	// TransportStdio indicates a local stdio-based MCP server (command + args).
	TransportStdio Transport = "stdio"
	// TransportHTTP indicates a remote HTTP/SSE-based MCP server (URL).
	TransportHTTP Transport = "http"
)

// Recipe describes a single pre-defined MCP server configuration.
type Recipe struct {
	Key       string            // server name key (e.g. "brave-search")
	Category  string            // category name (e.g. "search")
	Transport Transport         // stdio or http
	Command   string            // stdio: binary name (e.g. "npx")
	Args      []string          // stdio: command arguments
	URL       string            // http: server URL
	EnvVars   map[string]string // environment variables needed
	Notes     string            // human-readable description
	SetupCmd  string            // CLI command to add this server
	Scope     string            // suggested scope ("user" or "local")
}

// Category groups recipes under a display name.
type Category struct {
	Name    string
	Recipes []Recipe
}

// categoryOrder defines the display order for categories.
var categoryOrder = []string{
	"collaboration",
	"search",
	"observability",
	"database",
	"memory",
}

// configFile is the JSON schema for each docs/mcp-configs/*.json file.
type configFile struct {
	Description   string                     `json:"_description"`
	Examples      map[string]json.RawMessage `json:"examples"`
	SetupCommands map[string]string          `json:"setup_commands"`
	Notes         map[string]string          `json:"notes"`
}

// serverExample is the parsed example entry for a single server.
type serverExample struct {
	Type    string            `json:"type"`
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	URL     string            `json:"url"`
	Env     map[string]string `json:"env"`
}

// LoadAll reads all JSON config files from configFS and returns categorized recipes.
// Returns empty categories (not an error) if configFS is nil or contains no JSON files.
func LoadAll(configFS fs.FS) ([]Category, error) {
	if configFS == nil {
		return nil, nil
	}

	entries, err := fs.ReadDir(configFS, ".")
	if err != nil {
		return nil, nil
	}

	catMap := make(map[string][]Recipe, len(entries))

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := fs.ReadFile(configFS, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", entry.Name(), err)
		}

		var cf configFile
		if err := json.Unmarshal(data, &cf); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", entry.Name(), err)
		}

		categoryName := strings.TrimSuffix(entry.Name(), ".json")

		for key, raw := range cf.Examples {
			var ex serverExample
			if err := json.Unmarshal(raw, &ex); err != nil {
				return nil, fmt.Errorf("parsing example %q in %s: %w", key, entry.Name(), err)
			}

			recipe := Recipe{
				Key:      key,
				Category: categoryName,
				Notes:    cf.Notes[key],
				SetupCmd: cf.SetupCommands[key],
				Scope:    parseScope(cf.Notes[key]),
			}

			if ex.Type == "http" || ex.URL != "" {
				recipe.Transport = TransportHTTP
				recipe.URL = ex.URL
			} else {
				recipe.Transport = TransportStdio
				recipe.Command = ex.Command
				recipe.Args = ex.Args
			}

			// Collect env vars, filtering out empty ones
			if len(ex.Env) > 0 {
				recipe.EnvVars = make(map[string]string, len(ex.Env))
				for k, v := range ex.Env {
					recipe.EnvVars[k] = v
				}
			}

			catMap[categoryName] = append(catMap[categoryName], recipe)
		}
	}

	// Build ordered categories
	categories := make([]Category, 0, len(catMap))
	for _, name := range categoryOrder {
		recipes, ok := catMap[name]
		if !ok {
			continue
		}
		// Sort recipes by key within each category for stable ordering
		sort.Slice(recipes, func(i, j int) bool {
			return recipes[i].Key < recipes[j].Key
		})
		categories = append(categories, Category{Name: name, Recipes: recipes})
	}

	// Append any categories not in the predefined order
	for name, recipes := range catMap {
		found := false
		for _, ordered := range categoryOrder {
			if name == ordered {
				found = true
				break
			}
		}
		if !found {
			sort.Slice(recipes, func(i, j int) bool {
				return recipes[i].Key < recipes[j].Key
			})
			categories = append(categories, Category{Name: name, Recipes: recipes})
		}
	}

	return categories, nil
}

// parseScope extracts the recommended scope from a notes string.
// Looks for "Recommended scope: local" or "Recommended scope: user".
// Defaults to "user" if not found.
func parseScope(notes string) string {
	lower := strings.ToLower(notes)
	if strings.Contains(lower, "recommended scope: local") {
		return "local"
	}
	return "user"
}

// FirstEnvVar returns the first env var key from the recipe, or empty string.
// Useful for pre-filling the "API key env var" field in stdio forms.
func (r *Recipe) FirstEnvVar() string {
	for k := range r.EnvVars {
		return k
	}
	return ""
}

// CommandString returns the full command string (command + args) for stdio recipes.
func (r *Recipe) CommandString() string {
	if r.Command == "" {
		return ""
	}
	parts := make([]string, 0, 1+len(r.Args))
	parts = append(parts, r.Command)
	parts = append(parts, r.Args...)
	return strings.Join(parts, " ")
}

// NotesFirstLine returns the first sentence of the notes, trimming scope prefix.
func (r *Recipe) NotesFirstLine() string {
	if r.Notes == "" {
		return ""
	}
	// Skip the "Recommended scope: ..." prefix
	s := r.Notes
	if idx := strings.Index(s, ")."); idx >= 0 {
		s = strings.TrimSpace(s[idx+2:])
	} else if idx := strings.Index(s, ". "); idx >= 0 {
		// Take first sentence
		s = s[:idx+1]
	}
	return s
}
