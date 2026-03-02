package config

import (
	"strings"
	"testing"
)

func TestRegistryAll(t *testing.T) {
	reg := GlobalRegistry()
	all := reg.All()
	if len(all) < 200 {
		t.Errorf("expected at least 200 keys, got %d", len(all))
	}
}

func TestRegistryNoDuplicates(t *testing.T) {
	reg := GlobalRegistry()
	seen := make(map[string]int)
	for _, k := range reg.All() {
		seen[k.Key]++
	}
	for key, count := range seen {
		if count > 1 {
			t.Errorf("duplicate key %q appears %d times", key, count)
		}
	}
}

func TestRegistryByCategory(t *testing.T) {
	reg := GlobalRegistry()
	for _, cat := range reg.Categories() {
		keys := reg.ByCategory(cat)
		if len(keys) == 0 {
			t.Errorf("category %q has no keys", cat)
		}
	}
}

func TestRegistryCategories(t *testing.T) {
	reg := GlobalRegistry()
	cats := reg.Categories()
	if len(cats) < 10 {
		t.Errorf("expected at least 10 categories, got %d", len(cats))
	}

	// Verify categories we know must exist
	required := []Category{
		CatCore, CatUI, CatPermissions, CatSandbox,
		CatEnvModel, CatEnvAuth, CatEnvBash, CatFiles,
	}
	catSet := make(map[Category]bool, len(cats))
	for _, c := range cats {
		catSet[c] = true
	}
	for _, want := range required {
		if !catSet[want] {
			t.Errorf("required category %q not found in registry", want)
		}
	}
}

func TestRegistryGet_Known(t *testing.T) {
	reg := GlobalRegistry()
	tests := []struct {
		key      string
		wantType ValueType
	}{
		{"model", TypeString},
		{"effortLevel", TypeEnum},
		{"sandbox.enabled", TypeBool},
		{"permissions.allow", TypeStringArray},
		{"CLAUDE_CODE_MAX_OUTPUT_TOKENS", TypeInt},
		{"ANTHROPIC_API_KEY", TypeString},
		{"file:claudemd.user", TypeString},
	}
	for _, tt := range tests {
		k, ok := reg.Get(tt.key)
		if !ok {
			t.Errorf("Get(%q): key not found", tt.key)
			continue
		}
		if k.Type != tt.wantType {
			t.Errorf("Get(%q): type = %q, want %q", tt.key, k.Type, tt.wantType)
		}
	}
}

func TestRegistryGet_Missing(t *testing.T) {
	reg := GlobalRegistry()
	_, ok := reg.Get("nonexistent.key.that.does.not.exist")
	if ok {
		t.Error("Get on missing key should return false")
	}
}

func TestRegistrySearch_ByKeyName(t *testing.T) {
	reg := GlobalRegistry()
	results := reg.Search("sandbox")
	if len(results) < 5 {
		t.Errorf("search for 'sandbox' expected >= 5 results, got %d", len(results))
	}
	for _, k := range results {
		if !strings.Contains(strings.ToLower(k.Key), "sandbox") &&
			!strings.Contains(strings.ToLower(k.Description), "sandbox") {
			t.Errorf("search result %q does not contain 'sandbox'", k.Key)
		}
	}
}

func TestRegistrySearch_ByDescription(t *testing.T) {
	reg := GlobalRegistry()
	// Search for something in description but not key
	results := reg.Search("truncation")
	if len(results) == 0 {
		t.Error("search for 'truncation' expected at least 1 result")
	}
}

func TestRegistrySearch_CaseInsensitive(t *testing.T) {
	reg := GlobalRegistry()
	lower := reg.Search("bedrock")
	upper := reg.Search("BEDROCK")
	if len(lower) != len(upper) {
		t.Errorf("case-insensitive search mismatch: lower=%d upper=%d", len(lower), len(upper))
	}
}

func TestRegistrySearch_NoResults(t *testing.T) {
	reg := GlobalRegistry()
	results := reg.Search("zzznomatch_xyzzy_impossible")
	if len(results) != 0 {
		t.Errorf("expected no results, got %d", len(results))
	}
}

func TestRegistryEnumKeys_HaveEnumValues(t *testing.T) {
	reg := GlobalRegistry()
	for _, k := range reg.All() {
		if k.Type == TypeEnum && len(k.EnumValues) == 0 {
			t.Errorf("key %q is TypeEnum but has no EnumValues", k.Key)
		}
	}
}

func TestRegistryEnvKeys_HaveEnvScope(t *testing.T) {
	reg := GlobalRegistry()
	for _, k := range reg.All() {
		// Env var keys (uppercase with underscores, not file:) should have ScopeEnv
		if len(k.Key) > 0 && k.Key[0] >= 'A' && k.Key[0] <= 'Z' && !strings.HasPrefix(k.Key, "$") {
			hasEnvScope := false
			for _, s := range k.ValidScopes {
				if s == ScopeEnv {
					hasEnvScope = true
					break
				}
			}
			if !hasEnvScope {
				t.Errorf("env var key %q does not have ScopeEnv in ValidScopes", k.Key)
			}
		}
	}
}
