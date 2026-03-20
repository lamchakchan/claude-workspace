package plugins

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
)

// MarketplaceRecipe describes a curated marketplace from the embedded registry.
type MarketplaceRecipe struct {
	Key         string
	Repo        string
	Description string
	Category    string
}

// marketplaceRegistryFile is the JSON schema for marketplaces.json.
type marketplaceRegistryFile struct {
	Description  string                              `json:"_description"`
	Marketplaces map[string]marketplaceRegistryEntry `json:"marketplaces"`
}

// marketplaceRegistryEntry is a single marketplace entry in the JSON file.
type marketplaceRegistryEntry struct {
	Repo        string `json:"repo"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// LoadMarketplaces reads the curated marketplace registry from the embedded FS.
// Returns nil on nil FS. Returns empty slice if no marketplaces found.
func LoadMarketplaces(configFS fs.FS) ([]MarketplaceRecipe, error) {
	if configFS == nil {
		return nil, nil
	}

	data, err := fs.ReadFile(configFS, "marketplaces.json")
	if err != nil {
		return nil, fmt.Errorf("reading marketplaces.json: %w", err)
	}

	var file marketplaceRegistryFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parsing marketplaces.json: %w", err)
	}

	if len(file.Marketplaces) == 0 {
		return []MarketplaceRecipe{}, nil
	}

	recipes := make([]MarketplaceRecipe, 0, len(file.Marketplaces))
	for key, entry := range file.Marketplaces {
		recipes = append(recipes, MarketplaceRecipe{
			Key:         key,
			Repo:        entry.Repo,
			Description: entry.Description,
			Category:    entry.Category,
		})
	}

	sort.Slice(recipes, func(i, j int) bool {
		return recipes[i].Key < recipes[j].Key
	})
	return recipes, nil
}
