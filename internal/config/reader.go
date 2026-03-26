package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

// ConfigValue holds the resolved value for a single config key across all scopes.
type ConfigValue struct { //nolint:revive // stutter intentional: Value is too generic without package qualifier
	Key            string
	EffectiveValue interface{}                 // The winning value after applying precedence
	Source         ConfigScope                 // Which scope the effective value comes from
	IsDefault      bool                        // True if no scope defines this key (effective = registry default)
	LayerValues    map[ConfigScope]interface{} // Value at each scope that defines this key
	RawJSON        json.RawMessage             // Raw JSON for complex objects/arrays (nil for scalars)
}

// ConfigSnapshot holds all resolved config values at a point in time.
type ConfigSnapshot struct { //nolint:revive // stutter intentional: Snapshot is too generic without package qualifier
	Values    map[string]*ConfigValue
	Timestamp time.Time
	Home      string
	Cwd       string
}

// ReadAll reads all Claude Code configuration layers and returns a snapshot
// with provenance tracking for every known key plus detected file-based configs.
func ReadAll() (*ConfigSnapshot, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home directory: %w", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}

	snap := &ConfigSnapshot{
		Values:    make(map[string]*ConfigValue, 300),
		Timestamp: time.Now(),
		Home:      home,
		Cwd:       cwd,
	}

	// Read settings.json from all scopes
	layers, err := readAllSettings(home, cwd)
	if err != nil {
		return nil, fmt.Errorf("reading settings layers: %w", err)
	}

	reg := GlobalRegistry()

	// Resolve settings.json keys from registry
	allKeys := reg.All()
	for i := range allKeys {
		key := &allKeys[i]
		if key.Category == CatFiles {
			continue // handled separately
		}
		if isEnvVar(key.Key) {
			continue // handled by readEnvVars
		}
		cv := resolveSettingsKey(key.Key, layers, key.Default)
		snap.Values[key.Key] = cv
	}

	// Add env var values
	for key, cv := range readEnvVars(layers) {
		snap.Values[key] = cv
	}

	// Add file-based config summaries
	for key, cv := range readFileBased(home, cwd) {
		snap.Values[key] = cv
	}

	return snap, nil
}

// isEnvVar returns true for keys that look like environment variables (all-caps with underscores).
func isEnvVar(key string) bool {
	if strings.HasPrefix(key, "file:") || strings.HasPrefix(key, "$") {
		return false
	}
	for _, ch := range key {
		if ch == '.' {
			return false
		}
		if ch >= 'a' && ch <= 'z' {
			return false
		}
	}
	return true
}

// --- Settings layers ---

// managedSettingsPath returns the platform-specific enterprise managed settings path.
func managedSettingsPath() string {
	if runtime.GOOS == "darwin" {
		return "/Library/Application Support/ClaudeCode/managed-settings.json"
	}
	return "/etc/claude-code/managed-settings.json"
}

// readAllSettings reads settings.json from every scope (managed, user, project, local).
// Returns a map of scope → flattened key → json.RawMessage.
func readAllSettings(home, cwd string) (map[ConfigScope]map[string]json.RawMessage, error) {
	paths := map[ConfigScope]string{
		ScopeManaged: managedSettingsPath(),
		ScopeUser:    filepath.Join(home, ".claude", "settings.json"),
		ScopeProject: filepath.Join(cwd, ".claude", "settings.json"),
		ScopeLocal:   filepath.Join(cwd, ".claude", "settings.local.json"),
	}

	layers := make(map[ConfigScope]map[string]json.RawMessage, 4)
	for scope, path := range paths {
		if !platform.FileExists(path) {
			continue
		}
		raw, err := platform.ReadJSONFileRaw(path)
		if err != nil {
			// Skip unreadable files silently; they may require elevated privileges
			continue
		}
		layers[scope] = flattenJSON("", raw)
	}
	return layers, nil
}

// flattenJSON recursively flattens a nested JSON map into dot-path keys.
// e.g., {"sandbox": {"enabled": true}} → {"sandbox.enabled": true}
func flattenJSON(prefix string, m map[string]json.RawMessage) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(m))
	for k, v := range m {
		fullKey := k
		if prefix != "" {
			fullKey = prefix + "." + k
		}

		// Try to descend into nested objects (but not arrays or scalars)
		var nested map[string]json.RawMessage
		if json.Unmarshal(v, &nested) == nil && len(nested) > 0 {
			// Check it's not an array by verifying the raw bytes start with {
			trimmed := strings.TrimSpace(string(v))
			if len(trimmed) > 0 && trimmed[0] == '{' {
				for nk, nv := range flattenJSON(fullKey, nested) {
					out[nk] = nv
				}
				// Also store the object itself at the parent key for object-type display
				out[fullKey] = v
				continue
			}
		}
		out[fullKey] = v
	}
	return out
}

// resolveSettingsKey computes the effective value for a settings.json key across all scopes.
// Precedence (highest to lowest): managed > local > project > user > default
func resolveSettingsKey(key string, layers map[ConfigScope]map[string]json.RawMessage, defaultVal string) *ConfigValue {
	cv := &ConfigValue{
		Key:         key,
		LayerValues: make(map[ConfigScope]interface{}),
		IsDefault:   true,
		Source:      ScopeDefault,
	}

	// Collect values from every scope into LayerValues
	allScopes := []ConfigScope{ScopeManaged, ScopeLocal, ScopeProject, ScopeUser}
	for _, scope := range allScopes {
		layer, ok := layers[scope]
		if !ok {
			continue
		}
		raw, ok := layer[key]
		if !ok {
			continue
		}
		cv.LayerValues[scope] = parseRawValue(raw)
	}

	// Apply precedence from lowest to highest so the highest-priority scope wins last.
	// Order (lowest → highest): user → project → local → managed
	lowToHigh := []ConfigScope{ScopeUser, ScopeProject, ScopeLocal, ScopeManaged}
	for _, scope := range lowToHigh {
		if val, ok := cv.LayerValues[scope]; ok {
			cv.EffectiveValue = val
			cv.Source = scope
			cv.IsDefault = false
		}
	}

	// Store raw JSON for the winning scope's value
	if cv.Source != ScopeDefault {
		if layer, ok := layers[cv.Source]; ok {
			if raw, ok := layer[key]; ok {
				cv.RawJSON = raw
			}
		}
	}

	// Fall back to registry default if no scope defines this key
	if cv.IsDefault && defaultVal != "" {
		cv.EffectiveValue = defaultVal
	}

	return cv
}

// parseRawValue converts a json.RawMessage to its native Go type.
func parseRawValue(raw json.RawMessage) interface{} {
	trimmed := strings.TrimSpace(string(raw))
	if len(trimmed) == 0 {
		return nil
	}
	switch trimmed[0] {
	case '"':
		var s string
		if json.Unmarshal(raw, &s) == nil {
			return s
		}
	case 't', 'f':
		var b bool
		if json.Unmarshal(raw, &b) == nil {
			return b
		}
	case '[':
		var arr []interface{}
		if json.Unmarshal(raw, &arr) == nil {
			return arr
		}
	case '{':
		var obj map[string]interface{}
		if json.Unmarshal(raw, &obj) == nil {
			return obj
		}
	default:
		// Numbers
		var n json.Number
		if json.Unmarshal(raw, &n) == nil {
			if i, err := n.Int64(); err == nil {
				return int(i)
			}
			if f, err := n.Float64(); err == nil {
				return f
			}
		}
	}
	return string(raw) // fallback: return raw string
}

// --- Env vars ---

// readEnvVars resolves environment variable keys from the OS environment and
// from settings.json `env` blocks.
func readEnvVars(layers map[ConfigScope]map[string]json.RawMessage) map[string]*ConfigValue { //nolint:gocyclo // env precedence logic requires handling multiple scopes and sources
	reg := GlobalRegistry()
	out := make(map[string]*ConfigValue, 120)

	// Collect env block values from each settings scope
	// settings.json `env` key maps variable names to values
	type envEntry struct {
		scope ConfigScope
		val   string
	}
	settingsEnv := make(map[string][]envEntry)

	order := []ConfigScope{ScopeManaged, ScopeUser, ScopeProject, ScopeLocal}
	for _, scope := range order {
		layer, ok := layers[scope]
		if !ok {
			continue
		}
		rawEnv, ok := layer["env"]
		if !ok {
			continue
		}
		var envMap map[string]string
		if json.Unmarshal(rawEnv, &envMap) != nil {
			continue
		}
		for k, v := range envMap {
			settingsEnv[k] = append(settingsEnv[k], envEntry{scope, v})
		}
	}

	envKeys := reg.All()
	for i := range envKeys {
		key := &envKeys[i]
		if !isEnvVar(key.Key) {
			continue
		}

		cv := &ConfigValue{
			Key:         key.Key,
			LayerValues: make(map[ConfigScope]interface{}),
			IsDefault:   true,
			Source:      ScopeDefault,
		}

		// Check settings.json env blocks (lower priority than OS env)
		for _, entry := range settingsEnv[key.Key] {
			cv.LayerValues[entry.scope] = entry.val
			if cv.IsDefault {
				cv.EffectiveValue = entry.val
				cv.Source = entry.scope
				cv.IsDefault = false
			}
		}
		// Apply env block precedence
		for _, scope := range order {
			if entries, ok := settingsEnv[key.Key]; ok {
				for _, entry := range entries {
					if entry.scope == scope {
						cv.EffectiveValue = entry.val
						cv.Source = scope
					}
				}
			}
		}

		// OS environment wins over settings.json env blocks
		if osVal := os.Getenv(key.Key); osVal != "" {
			cv.LayerValues[ScopeEnv] = osVal
			cv.EffectiveValue = osVal
			cv.Source = ScopeEnv
			cv.IsDefault = false
		}

		// Fall back to registry default
		if cv.IsDefault && key.Default != "" {
			cv.EffectiveValue = key.Default
		}

		out[key.Key] = cv
	}
	return out
}

// --- File-based config ---

// FileConfigValue summarizes a file-based config artifact (e.g., CLAUDE.md, MCP servers).
type FileConfigValue struct {
	Exists    bool
	Path      string
	LineCount int      // for text files (CLAUDE.md, rules)
	Items     []string // for directories (agents, skills, hooks) or server names (MCP)
}

// readFileBased discovers and summarizes file-based Claude Code config artifacts.
func readFileBased(home, cwd string) map[string]*ConfigValue {
	out := make(map[string]*ConfigValue, 20)

	add := func(key string, fcv FileConfigValue) {
		val := formatFileValue(fcv)
		out[key] = &ConfigValue{
			Key:            key,
			EffectiveValue: val,
			Source:         ScopeProject, // file-based have no "scope" hierarchy — use ScopeProject
			IsDefault:      !fcv.Exists,
			LayerValues:    map[ConfigScope]interface{}{ScopeProject: val},
		}
	}

	// CLAUDE.md layers
	add("file:claudemd.user", discoverTextFile(filepath.Join(home, ".claude", "CLAUDE.md")))
	projectClaudeMd := discoverTextFile(filepath.Join(cwd, ".claude", "CLAUDE.md"))
	if !projectClaudeMd.Exists {
		projectClaudeMd = discoverTextFile(filepath.Join(cwd, "CLAUDE.md"))
	}
	add("file:claudemd.project", projectClaudeMd)
	add("file:claudemd.local", discoverTextFile(filepath.Join(cwd, "CLAUDE.local.md"))) // deprecated: use .claude/rules/ instead

	// MCP servers
	add("file:mcp.project", discoverMCPProject(cwd))
	add("file:mcp.user", discoverMCPUser(home))
	add("file:mcp.managed", discoverMCPManaged())

	// Keybindings
	add("file:keybindings", discoverTextFile(filepath.Join(home, ".claude", "keybindings.json")))

	// Agents
	add("file:agents.project", discoverDirectory(filepath.Join(cwd, ".claude", "agents"), ".md"))
	add("file:agents.user", discoverDirectory(filepath.Join(home, ".claude", "agents"), ".md"))

	// Skills
	add("file:skills.project", discoverSkills(filepath.Join(cwd, ".claude", "skills")))
	add("file:skills.user", discoverSkills(filepath.Join(home, ".claude", "skills")))

	// Hooks
	add("file:hooks.project", discoverDirectory(filepath.Join(cwd, ".claude", "hooks"), ".sh"))
	add("file:hooks.user", discoverDirectory(filepath.Join(home, ".claude", "hooks"), ".sh"))

	// Rules
	add("file:rules.project", discoverDirectory(filepath.Join(cwd, ".claude", "rules"), ".md"))
	add("file:rules.user", discoverDirectory(filepath.Join(home, ".claude", "rules"), ".md"))

	// Settings files (presence)
	add("file:settings.managed", discoverTextFile(managedSettingsPath()))
	add("file:settings.user", discoverTextFile(filepath.Join(home, ".claude", "settings.json")))
	add("file:settings.project", discoverTextFile(filepath.Join(cwd, ".claude", "settings.json")))
	add("file:settings.local", discoverTextFile(filepath.Join(cwd, ".claude", "settings.local.json")))

	// Main claude.json
	add("file:claude.json", discoverTextFile(filepath.Join(home, ".claude.json")))

	return out
}

// discoverTextFile checks a text file and counts its lines.
func discoverTextFile(path string) FileConfigValue {
	if !platform.FileExists(path) {
		return FileConfigValue{Path: path}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return FileConfigValue{Exists: true, Path: path}
	}
	lines := strings.Count(string(data), "\n")
	if len(data) > 0 && !strings.HasSuffix(string(data), "\n") {
		lines++
	}
	return FileConfigValue{Exists: true, Path: path, LineCount: lines}
}

// discoverDirectory scans a directory for files with the given extension.
func discoverDirectory(dir, ext string) FileConfigValue {
	if !platform.FileExists(dir) {
		return FileConfigValue{Path: dir}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return FileConfigValue{Exists: true, Path: dir}
	}
	items := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ext) {
			items = append(items, strings.TrimSuffix(e.Name(), ext))
		}
	}
	return FileConfigValue{Exists: true, Path: dir, Items: items}
}

// discoverSkills scans a skills directory, returning the skill names
// (each skill may be a subdirectory with SKILL.md or a direct .md file).
func discoverSkills(dir string) FileConfigValue {
	if !platform.FileExists(dir) {
		return FileConfigValue{Path: dir}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return FileConfigValue{Exists: true, Path: dir}
	}
	items := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			skillMd := filepath.Join(dir, e.Name(), "SKILL.md")
			if platform.FileExists(skillMd) {
				items = append(items, e.Name())
			}
		} else if strings.HasSuffix(e.Name(), ".md") {
			items = append(items, strings.TrimSuffix(e.Name(), ".md"))
		}
	}
	return FileConfigValue{Exists: true, Path: dir, Items: items}
}

// discoverMCPProject reads .mcp.json and returns MCP server names.
func discoverMCPProject(cwd string) FileConfigValue {
	path := filepath.Join(cwd, ".mcp.json")
	if !platform.FileExists(path) {
		return FileConfigValue{Path: path}
	}
	return FileConfigValue{Exists: true, Path: path, Items: readMCPServerNames(path)}
}

// discoverMCPUser reads ~/.claude.json and extracts mcpServers names.
func discoverMCPUser(home string) FileConfigValue {
	path := filepath.Join(home, ".claude.json")
	if !platform.FileExists(path) {
		return FileConfigValue{Path: path}
	}
	var root map[string]json.RawMessage
	if err := platform.ReadJSONFile(path, &root); err != nil {
		return FileConfigValue{Exists: true, Path: path}
	}
	rawServers, ok := root["mcpServers"]
	if !ok {
		return FileConfigValue{Exists: true, Path: path}
	}
	var servers map[string]json.RawMessage
	if json.Unmarshal(rawServers, &servers) != nil {
		return FileConfigValue{Exists: true, Path: path}
	}
	names := make([]string, 0, len(servers))
	for name := range servers {
		names = append(names, name)
	}
	return FileConfigValue{Exists: true, Path: path, Items: names}
}

// discoverMCPManaged reads the managed MCP config file.
func discoverMCPManaged() FileConfigValue {
	var path string
	if runtime.GOOS == "darwin" {
		path = "/Library/Application Support/ClaudeCode/managed-mcp.json"
	} else {
		path = "/etc/claude-code/managed-mcp.json"
	}
	if !platform.FileExists(path) {
		return FileConfigValue{Path: path}
	}
	return FileConfigValue{Exists: true, Path: path, Items: readMCPServerNames(path)}
}

// readMCPServerNames reads a .mcp.json file and returns the list of server names.
func readMCPServerNames(path string) []string {
	var root struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if err := platform.ReadJSONFile(path, &root); err != nil {
		return nil
	}
	names := make([]string, 0, len(root.MCPServers))
	for name := range root.MCPServers {
		names = append(names, name)
	}
	return names
}

// formatFileValue converts a FileConfigValue to a human-readable string.
func formatFileValue(fcv FileConfigValue) string {
	if !fcv.Exists {
		return "(not found)"
	}
	var b strings.Builder
	b.WriteString("exists")
	switch {
	case len(fcv.Items) > 0:
		fmt.Fprintf(&b, " — %d item(s): %s", len(fcv.Items), strings.Join(fcv.Items, ", "))
	case fcv.LineCount > 0:
		fmt.Fprintf(&b, " — %d lines", fcv.LineCount)
	}
	return b.String()
}
