package plugins

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverMarketplaces_NoDir(t *testing.T) {
	// When the marketplaces directory does not exist, returns nil
	marketplaces, err := DiscoverMarketplaces()
	if err != nil {
		t.Fatalf("DiscoverMarketplaces() error: %v", err)
	}
	// If the user has no marketplaces dir, result is nil (or possibly non-nil if they do)
	// We primarily test that no error is returned for missing dir
	_ = marketplaces
}

func TestDiscoverMarketplaces_WithMarketplaces(t *testing.T) {
	// Create a temp directory structure simulating ~/.claude/plugins/marketplaces/
	tmpDir := t.TempDir()
	marketplacesDir := filepath.Join(tmpDir, ".claude", "plugins", "marketplaces")

	// Create a mock marketplace with 2 plugins
	mpDir := filepath.Join(marketplacesDir, "test-marketplace")
	pluginsDir := filepath.Join(mpDir, "plugins")
	if err := os.MkdirAll(filepath.Join(pluginsDir, "plugin-a", ".claude-plugin"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(pluginsDir, "plugin-b", ".claude-plugin"), 0755); err != nil {
		t.Fatal(err)
	}
	// Add a non-dir entry (should be ignored)
	if err := os.WriteFile(filepath.Join(pluginsDir, "README.md"), []byte("# test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create another marketplace with 0 plugins (no plugins/ dir)
	emptyMp := filepath.Join(marketplacesDir, "empty-marketplace")
	if err := os.MkdirAll(emptyMp, 0755); err != nil {
		t.Fatal(err)
	}

	// Override HOME to use our temp dir
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", origHome) }()

	marketplaces, err := DiscoverMarketplaces()
	if err != nil {
		t.Fatalf("DiscoverMarketplaces() error: %v", err)
	}

	if len(marketplaces) != 2 {
		t.Fatalf("marketplaces = %d, want 2", len(marketplaces))
	}

	// Sorted alphabetically: empty-marketplace, test-marketplace
	if marketplaces[0].Name != "empty-marketplace" {
		t.Errorf("marketplaces[0].Name = %q, want %q", marketplaces[0].Name, "empty-marketplace")
	}
	if marketplaces[0].PluginCount != 0 {
		t.Errorf("marketplaces[0].PluginCount = %d, want 0", marketplaces[0].PluginCount)
	}

	if marketplaces[1].Name != "test-marketplace" {
		t.Errorf("marketplaces[1].Name = %q, want %q", marketplaces[1].Name, "test-marketplace")
	}
	if marketplaces[1].PluginCount != 2 {
		t.Errorf("marketplaces[1].PluginCount = %d, want 2", marketplaces[1].PluginCount)
	}
}
