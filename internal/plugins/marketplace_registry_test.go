package plugins

import (
	"testing"
	"testing/fstest"
)

func TestLoadMarketplaces(t *testing.T) {
	fs := fstest.MapFS{
		"marketplaces.json": &fstest.MapFile{Data: []byte(`{
			"_description": "Test registry",
			"marketplaces": {
				"beta-marketplace": {
					"repo": "org/beta-marketplace",
					"description": "Beta plugins",
					"category": "community"
				},
				"alpha-marketplace": {
					"repo": "org/alpha-marketplace",
					"description": "Alpha plugins",
					"category": "official"
				}
			}
		}`)},
	}

	recipes, err := LoadMarketplaces(fs)
	if err != nil {
		t.Fatalf("LoadMarketplaces() error: %v", err)
	}

	if len(recipes) != 2 {
		t.Fatalf("recipes = %d, want 2", len(recipes))
	}

	// Should be sorted alphabetically
	if recipes[0].Key != "alpha-marketplace" {
		t.Errorf("recipes[0].Key = %q, want %q", recipes[0].Key, "alpha-marketplace")
	}
	if recipes[0].Repo != "org/alpha-marketplace" {
		t.Errorf("recipes[0].Repo = %q, want %q", recipes[0].Repo, "org/alpha-marketplace")
	}
	if recipes[0].Description != "Alpha plugins" {
		t.Errorf("recipes[0].Description = %q, want %q", recipes[0].Description, "Alpha plugins")
	}
	if recipes[0].Category != "official" {
		t.Errorf("recipes[0].Category = %q, want %q", recipes[0].Category, "official")
	}

	if recipes[1].Key != "beta-marketplace" {
		t.Errorf("recipes[1].Key = %q, want %q", recipes[1].Key, "beta-marketplace")
	}
}

func TestLoadMarketplaces_NilFS(t *testing.T) {
	recipes, err := LoadMarketplaces(nil)
	if err != nil {
		t.Fatalf("LoadMarketplaces(nil) error: %v", err)
	}
	if recipes != nil {
		t.Errorf("LoadMarketplaces(nil) = %v, want nil", recipes)
	}
}

func TestLoadMarketplaces_EmptyFS(t *testing.T) {
	fs := fstest.MapFS{
		"marketplaces.json": &fstest.MapFile{Data: []byte(`{
			"_description": "Empty registry",
			"marketplaces": {}
		}`)},
	}

	recipes, err := LoadMarketplaces(fs)
	if err != nil {
		t.Fatalf("LoadMarketplaces(empty) error: %v", err)
	}
	if len(recipes) != 0 {
		t.Errorf("LoadMarketplaces(empty) = %d recipes, want 0", len(recipes))
	}
}

func TestLoadMarketplaces_MalformedJSON(t *testing.T) {
	fs := fstest.MapFS{
		"marketplaces.json": &fstest.MapFile{Data: []byte(`{invalid json`)},
	}

	_, err := LoadMarketplaces(fs)
	if err == nil {
		t.Fatal("LoadMarketplaces(malformed) should return error")
	}
	if got := err.Error(); got == "" {
		t.Error("error message should not be empty")
	}
}
