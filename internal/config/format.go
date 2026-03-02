package config

import (
	"fmt"
	"io"
	"strings"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

const (
	valNone  = "(none)"
	valTrue  = "true"
	valFalse = "false"
)

// FormatView writes a formatted multi-section config report to w.
// Keys are grouped by category; each key shows its effective value and source badge.
func FormatView(w io.Writer, snap *ConfigSnapshot, reg *Registry) error {
	platform.PrintBanner(w, "Claude Code Configuration")

	for _, cat := range reg.Categories() {
		keys := reg.ByCategory(cat)
		if len(keys) == 0 {
			continue
		}
		platform.PrintSection(w, string(cat))
		for i := range keys {
			key := &keys[i]
			cv := snap.Values[key.Key]
			badge, val := formatKeyLine(cv, key)
			keyName := key.Key
			if cv != nil && !cv.IsDefault {
				keyName = platform.Bold(key.Key)
			}
			desc := truncate(key.Description, 50)
			fmt.Fprintf(w, "  %s %s = %s  (%s)\n", badge, keyName, truncate(formatValue(val), 60), desc)
		}
	}
	return nil
}

// FormatGet writes detailed information about a single key to w.
// Shows the value at each scope layer, effective value, type, default, and description.
// Returns an error if the key is not found in the registry.
func FormatGet(w io.Writer, key string, snap *ConfigSnapshot, reg *Registry) error {
	ck, ok := reg.Get(key)
	if !ok {
		return fmt.Errorf("unknown config key %q; use 'config view' to see all keys", key)
	}

	platform.PrintBanner(w, key)

	fmt.Fprintf(w, "  Type:        %s\n", ck.Type)
	if ck.Default != "" {
		fmt.Fprintf(w, "  Default:     %s\n", ck.Default)
	}
	fmt.Fprintf(w, "  Description: %s\n", ck.Description)

	if len(ck.EnumValues) > 0 {
		fmt.Fprintf(w, "  Values:      %s\n", strings.Join(ck.EnumValues, ", "))
	}

	cv := snap.Values[key]

	platform.PrintSectionLabel(w, "Effective Value")
	if cv == nil || cv.IsDefault {
		defVal := ck.Default
		if defVal == "" {
			defVal = valNone
		}
		fmt.Fprintf(w, "  %s %s\n", scopeBadge(ScopeDefault), defVal)
	} else {
		fmt.Fprintf(w, "  %s %s\n", scopeBadge(cv.Source), formatValue(cv.EffectiveValue))
	}

	platform.PrintSectionLabel(w, "Layer Values")
	scopes := []ConfigScope{ScopeManaged, ScopeUser, ScopeProject, ScopeLocal, ScopeEnv}
	if cv != nil {
		for _, scope := range scopes {
			val, ok := cv.LayerValues[scope]
			if !ok {
				continue
			}
			fmt.Fprintf(w, "  %s %s\n", scopeBadge(scope), formatValue(val))
		}
	}
	if cv == nil || len(cv.LayerValues) == 0 {
		fmt.Fprintf(w, "  (no layer overrides)\n")
	}

	return nil
}

// formatKeyLine returns the source badge and effective value for a config key line.
func formatKeyLine(cv *ConfigValue, key *ConfigKey) (string, interface{}) {
	if cv == nil || cv.IsDefault {
		defVal := key.Default
		if defVal == "" {
			defVal = valNone
		}
		return scopeBadge(ScopeDefault), defVal
	}
	return scopeBadge(cv.Source), cv.EffectiveValue
}

// scopeBadge returns a colored badge string for the given scope.
func scopeBadge(scope ConfigScope) string {
	switch scope {
	case ScopeManaged:
		return platform.BoldRed("[MGD]")
	case ScopeUser:
		return platform.Cyan("[USR]")
	case ScopeProject:
		return platform.Green("[PRJ]")
	case ScopeLocal:
		return platform.Yellow("[LOC]")
	case ScopeEnv:
		return platform.Cyan("[ENV]")
	default:
		return "[DEF]"
	}
}

// formatValue converts an interface{} config value to a human-readable string.
func formatValue(v interface{}) string {
	if v == nil {
		return valNone
	}
	switch val := v.(type) {
	case string:
		return val
	case bool:
		if val {
			return valTrue
		}
		return valFalse
	case int:
		return fmt.Sprintf("%d", val)
	case float64:
		return fmt.Sprintf("%g", val)
	case []interface{}:
		items := make([]string, 0, len(val))
		for _, item := range val {
			items = append(items, fmt.Sprintf("%v", item))
		}
		return "[" + strings.Join(items, ", ") + "]"
	case map[string]interface{}:
		return fmt.Sprintf("%v", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// truncate shortens s to maxLen characters, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 4 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
