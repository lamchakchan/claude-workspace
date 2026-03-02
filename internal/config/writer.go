package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

// WriteSettingsValue writes a single key=value to the appropriate settings.json file
// for the given scope (user, project, or local). Managed, env, and default scopes return an error.
// It reads the existing file, sets the value (creating nested objects for dot-paths),
// and writes back using platform.WriteJSONFile.
func WriteSettingsValue(key, value string, scope ConfigScope, home, cwd string) error {
	path, err := settingsPath(scope, home, cwd)
	if err != nil {
		return err
	}

	root, err := readOrCreateJSON(path)
	if err != nil {
		return err
	}

	parsed, err := parseWriteValue(key, value)
	if err != nil {
		return fmt.Errorf("parsing value for %q: %w", key, err)
	}

	setNestedValue(root, key, parsed)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", path, err)
	}
	return platform.WriteJSONFile(path, root)
}

// AppendToArray appends a string value to a settings.json array key.
// Creates the array if it doesn't exist.
func AppendToArray(key, value string, scope ConfigScope, home, cwd string) error {
	path, err := settingsPath(scope, home, cwd)
	if err != nil {
		return err
	}

	root, err := readOrCreateJSON(path)
	if err != nil {
		return err
	}

	arr := getNestedArray(root, key)
	arr = append(arr, value)
	setNestedValue(root, key, arr)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", path, err)
	}
	return platform.WriteJSONFile(path, root)
}

// RemoveFromArray removes a matching string value from a settings.json array key.
func RemoveFromArray(key, value string, scope ConfigScope, home, cwd string) error {
	path, err := settingsPath(scope, home, cwd)
	if err != nil {
		return err
	}

	root, err := readOrCreateJSON(path)
	if err != nil {
		return err
	}

	arr := getNestedArray(root, key)
	filtered := make([]interface{}, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok && s == value {
			continue
		}
		filtered = append(filtered, item)
	}
	setNestedValue(root, key, filtered)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", path, err)
	}
	return platform.WriteJSONFile(path, root)
}

// settingsPath returns the filesystem path for the given scope's settings.json file.
func settingsPath(scope ConfigScope, home, cwd string) (string, error) {
	switch scope {
	case ScopeUser:
		return filepath.Join(home, ".claude", "settings.json"), nil
	case ScopeProject:
		return filepath.Join(cwd, ".claude", "settings.json"), nil
	case ScopeLocal:
		return filepath.Join(cwd, ".claude", "settings.local.json"), nil
	default:
		return "", fmt.Errorf("cannot write to %q scope; only user, project, and local scopes are writable", scope)
	}
}

// readOrCreateJSON reads an existing JSON file or returns an empty map if it doesn't exist.
func readOrCreateJSON(path string) (map[string]interface{}, error) {
	if !platform.FileExists(path) {
		return make(map[string]interface{}), nil
	}
	var root map[string]interface{}
	if err := platform.ReadJSONFile(path, &root); err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return root, nil
}

// parseWriteValue parses a string value into the appropriate Go type based on
// the registry's type for the given key. For unknown keys, it infers the type.
func parseWriteValue(key, value string) (interface{}, error) { //nolint:gocyclo
	reg := GlobalRegistry()
	ck, ok := reg.Get(key)
	if !ok {
		return inferType(value), nil
	}

	switch ck.Type {
	case TypeBool:
		switch strings.ToLower(value) {
		case "true", "1":
			return true, nil
		case "false", "0":
			return false, nil
		default:
			return nil, fmt.Errorf("invalid bool value %q; expected true/false/1/0", value)
		}
	case TypeInt:
		n, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("invalid int value %q: %w", value, err)
		}
		return n, nil
	case TypeEnum:
		for _, valid := range ck.EnumValues {
			if value == valid {
				return value, nil
			}
		}
		return nil, fmt.Errorf("invalid enum value %q; valid values: %s", value, strings.Join(ck.EnumValues, ", "))
	case TypeStringArray:
		if strings.TrimSpace(value) == "" {
			return []interface{}{}, nil
		}
		parts := strings.Split(value, ",")
		arr := make([]interface{}, 0, len(parts))
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				arr = append(arr, trimmed)
			}
		}
		return arr, nil
	case TypeObject:
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(value), &obj); err != nil {
			return nil, fmt.Errorf("invalid JSON object %q: %w", value, err)
		}
		return obj, nil
	default:
		return value, nil
	}
}

// inferType guesses the Go type for a value string when the key is unknown.
func inferType(value string) interface{} {
	switch strings.ToLower(value) {
	case "true":
		return true
	case "false":
		return false
	}
	if n, err := strconv.Atoi(value); err == nil {
		return n
	}
	return value
}

// setNestedValue writes val into root at the dot-separated path.
// Intermediate maps are created as needed.
func setNestedValue(root map[string]interface{}, dotPath string, val interface{}) {
	parts := strings.Split(dotPath, ".")
	current := root
	for _, part := range parts[:len(parts)-1] {
		next, ok := current[part]
		if !ok {
			m := make(map[string]interface{})
			current[part] = m
			current = m
			continue
		}
		if m, ok := next.(map[string]interface{}); ok {
			current = m
		} else {
			m := make(map[string]interface{})
			current[part] = m
			current = m
		}
	}
	current[parts[len(parts)-1]] = val
}

// getNestedArray retrieves the array at the dot-path, returning nil if not found.
func getNestedArray(root map[string]interface{}, dotPath string) []interface{} {
	parts := strings.Split(dotPath, ".")
	current := root
	for _, part := range parts[:len(parts)-1] {
		next, ok := current[part]
		if !ok {
			return nil
		}
		m, ok := next.(map[string]interface{})
		if !ok {
			return nil
		}
		current = m
	}
	last := parts[len(parts)-1]
	arr, ok := current[last].([]interface{})
	if !ok {
		return nil
	}
	return arr
}
