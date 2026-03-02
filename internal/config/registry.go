// Package config implements the "config" command, providing a viewer and editor
// for all Claude Code configuration across every scope layer.
package config

import "strings"

// Category groups related config keys for display and navigation.
type Category string

const (
	// Settings.json categories
	CatCore        Category = "Core"
	CatUI          Category = "UI & Display"
	CatPermissions Category = "Permissions"
	CatSandbox     Category = "Sandbox"
	CatHooks       Category = "Hooks"
	CatMCP         Category = "MCP Servers"
	CatPlugins     Category = "Plugins"
	CatTelemetry   Category = "Telemetry"
	CatCloud       Category = "Cloud Providers"
	CatAttribution Category = "Attribution"
	CatAgentTeams  Category = "Agent Teams"

	// Environment variable categories
	CatEnvModel    Category = "Env: Model & Tokens"
	CatEnvAuth     Category = "Env: Authentication"
	CatEnvCloud    Category = "Env: Cloud Providers"
	CatEnvBash     Category = "Env: Shell & Bash"
	CatEnvFeatures Category = "Env: Feature Flags"
	CatEnvContext  Category = "Env: Context & Compaction"
	CatEnvMCP      Category = "Env: MCP"
	CatEnvUpdates  Category = "Env: Updates & Commands"
	CatEnvCaching  Category = "Env: Prompt Caching"
	CatEnvPaths    Category = "Env: Paths"
	CatEnvAccount  Category = "Env: Account Info"
	CatEnvMisc     Category = "Env: Miscellaneous"

	// File-based config
	CatFiles Category = "Config Files"
)

// ValueType describes the expected data type of a config key.
type ValueType string

const (
	TypeString      ValueType = "string"
	TypeBool        ValueType = "bool"
	TypeInt         ValueType = "int"
	TypeStringArray ValueType = "[]string"
	TypeObject      ValueType = "object"
	TypeEnum        ValueType = "enum"
)

// ConfigScope identifies which settings layer a value comes from.
type ConfigScope string //nolint:revive // stutter intentional: scope is ambiguous without package qualifier

const (
	ScopeManaged ConfigScope = "managed"
	ScopeUser    ConfigScope = "user"
	ScopeProject ConfigScope = "project"
	ScopeLocal   ConfigScope = "local"
	ScopeEnv     ConfigScope = "env"
	ScopeDefault ConfigScope = "default"
)

// ConfigKey describes a single Claude Code configuration option.
type ConfigKey struct { //nolint:revive // stutter intentional: Key is too generic without package qualifier
	Key         string        // Dot-path key name (e.g., "sandbox.enabled", "ANTHROPIC_MODEL")
	Category    Category      // Grouping for display
	Type        ValueType     // Data type
	Default     string        // Default value as a human-readable string ("" if no default)
	Description string        // One-sentence description of what this key does
	ValidScopes []ConfigScope // Which scopes can set this key (empty = all settings scopes)
	EnumValues  []string      // Valid values for TypeEnum keys
	ReadOnly    bool          // True if the key cannot be set via settings.json (e.g., managed-only)
}

// Registry holds all known Claude Code configuration keys and provides
// O(1) lookup, category grouping, and search.
type Registry struct {
	keys       []ConfigKey
	lookup     map[string]*ConfigKey
	categories []Category // ordered, deduplicated
}

// registryInstance is the singleton built at package init time.
var registryInstance = buildRegistry()

// GlobalRegistry returns the singleton config key registry.
func GlobalRegistry() *Registry { return registryInstance }

func buildRegistry() *Registry {
	keys := make([]ConfigKey, 0, 300)
	keys = append(keys, buildSettingsKeys()...)
	keys = append(keys, buildEnvKeys()...)
	keys = append(keys, buildFileKeys()...)

	lookup := make(map[string]*ConfigKey, len(keys))
	seen := make(map[string]bool, len(keys))
	for i := range keys {
		k := keys[i].Key
		if !seen[k] {
			seen[k] = true
			lookup[k] = &keys[i]
		}
	}

	cats := dedupCategories(keys)
	return &Registry{keys: keys, lookup: lookup, categories: cats}
}

// All returns every registered config key.
func (r *Registry) All() []ConfigKey { return r.keys }

// ByCategory returns all keys in the given category.
func (r *Registry) ByCategory(cat Category) []ConfigKey {
	out := make([]ConfigKey, 0, 16)
	for i := range r.keys {
		if r.keys[i].Category == cat {
			out = append(out, r.keys[i])
		}
	}
	return out
}

// Get returns the key descriptor for the given dot-path key name.
func (r *Registry) Get(key string) (*ConfigKey, bool) {
	k, ok := r.lookup[key]
	return k, ok
}

// Categories returns the ordered list of distinct categories.
func (r *Registry) Categories() []Category { return r.categories }

// Search returns all keys whose Key or Description contains query (case-insensitive).
func (r *Registry) Search(query string) []ConfigKey {
	q := strings.ToLower(query)
	out := make([]ConfigKey, 0, 8)
	for i := range r.keys {
		k := &r.keys[i]
		if strings.Contains(strings.ToLower(k.Key), q) ||
			strings.Contains(strings.ToLower(k.Description), q) {
			out = append(out, *k)
		}
	}
	return out
}

// dedupCategories returns categories in the order they first appear in keys.
func dedupCategories(keys []ConfigKey) []Category {
	seen := make(map[Category]bool)
	out := make([]Category, 0, 24)
	for i := range keys {
		cat := keys[i].Category
		if !seen[cat] {
			seen[cat] = true
			out = append(out, cat)
		}
	}
	return out
}

// allScopes is a convenience for keys valid in all settings scopes.
var allScopes = []ConfigScope{ScopeManaged, ScopeUser, ScopeProject, ScopeLocal}

// userScopes is for keys only valid in user/project/local (not managed).
var userScopes = []ConfigScope{ScopeUser, ScopeProject, ScopeLocal}

// managedOnly is for keys that only enterprise-managed settings can set.
var managedOnly = []ConfigScope{ScopeManaged}
