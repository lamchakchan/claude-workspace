package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/lamchakchan/claude-workspace/internal/config"
)

func TestNewConfigView(t *testing.T) {
	theme := DefaultTheme()
	m := NewConfigView(&theme)
	if !m.loading {
		t.Error("NewConfigView should start in loading state")
	}
	if len(m.categories) != 0 {
		t.Errorf("NewConfigView categories = %d, want 0", len(m.categories))
	}
	if m.theme == nil {
		t.Error("NewConfigView theme is nil")
	}
}

func TestConfigModel_LoadedMsg(t *testing.T) {
	theme := DefaultTheme()
	m := NewConfigView(&theme)

	reg := config.GlobalRegistry()
	snap := &config.ConfigSnapshot{
		Values: make(map[string]*config.ConfigValue),
	}
	// Populate minimal values so scopeBadge doesn't panic
	allKeys := reg.All()
	for i := range allKeys {
		key := &allKeys[i]
		snap.Values[key.Key] = &config.ConfigValue{
			Key:         key.Key,
			IsDefault:   true,
			Source:      config.ScopeDefault,
			LayerValues: make(map[config.ConfigScope]interface{}),
		}
	}

	msg := configLoadedMsg{snap: snap, reg: reg}
	updated, _ := m.Update(msg)
	cm := updated.(*ConfigModel)

	if cm.loading {
		t.Error("loading should be false after configLoadedMsg")
	}
	if len(cm.categories) == 0 {
		t.Error("categories should be populated after configLoadedMsg")
	}
	if cm.registry == nil {
		t.Error("registry should be set after configLoadedMsg")
	}
	if cm.snapshot == nil {
		t.Error("snapshot should be set after configLoadedMsg")
	}
	if len(cm.keys) == 0 {
		t.Error("keys should be populated for the first category")
	}
}

func TestConfigModel_TabNavigation(t *testing.T) {
	theme := DefaultTheme()
	m := loadedConfigModel(&theme)

	if m.activeTab != 0 {
		t.Fatalf("initial activeTab = %d, want 0", m.activeTab)
	}

	// Tab forward
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	cm := updated.(*ConfigModel)
	if cm.activeTab != 1 {
		t.Errorf("after tab: activeTab = %d, want 1", cm.activeTab)
	}

	// Tab wraps around
	for range len(cm.categories) - 1 {
		updated, _ = cm.Update(tea.KeyPressMsg{Code: tea.KeyTab})
		cm = updated.(*ConfigModel)
	}
	if cm.activeTab != 0 {
		t.Errorf("after full tab cycle: activeTab = %d, want 0", cm.activeTab)
	}

	// Shift+tab goes backward
	updated, _ = cm.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	cm = updated.(*ConfigModel)
	expected := len(cm.categories) - 1
	if cm.activeTab != expected {
		t.Errorf("after shift+tab: activeTab = %d, want %d", cm.activeTab, expected)
	}
}

func TestConfigModel_Filter(t *testing.T) {
	theme := DefaultTheme()
	m := loadedConfigModel(&theme)

	totalKeys := len(m.keys)
	if totalKeys == 0 {
		t.Skip("no keys in first category to test filter")
	}

	// Press / to enter filter mode
	updated, _ := m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	cm := updated.(*ConfigModel)
	if !cm.filterMode {
		t.Error("filterMode should be true after pressing /")
	}

	// Type a filter string that matches a subset
	// Use a very specific string that likely won't match all keys
	updated, _ = cm.Update(tea.KeyPressMsg{Code: 'z', Text: "zzzznotfound"})
	cm = updated.(*ConfigModel)
	if cm.filter != "zzzznotfound" {
		t.Errorf("filter = %q, want %q", cm.filter, "zzzznotfound")
	}
	if len(cm.keys) != 0 {
		t.Errorf("filtered keys = %d, want 0 for nonsense filter", len(cm.keys))
	}

	// Backspace clears filter char by char
	for range len("zzzznotfound") {
		updated, _ = cm.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
		cm = updated.(*ConfigModel)
	}
	if len(cm.keys) != totalKeys {
		t.Errorf("after clearing filter: keys = %d, want %d", len(cm.keys), totalKeys)
	}
}

func TestConfigModel_FilterEscape(t *testing.T) {
	theme := DefaultTheme()
	m := loadedConfigModel(&theme)

	// Enter filter mode
	updated, _ := m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	cm := updated.(*ConfigModel)
	if !cm.filterMode {
		t.Fatal("expected filterMode after /")
	}

	// Type something
	updated, _ = cm.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	cm = updated.(*ConfigModel)

	// Escape should exit filter mode and clear filter
	updated, _ = cm.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	cm = updated.(*ConfigModel)
	if cm.filterMode {
		t.Error("filterMode should be false after esc")
	}
	if cm.filter != "" {
		t.Errorf("filter should be cleared after esc, got %q", cm.filter)
	}
}

func TestConfigModel_CursorMovement(t *testing.T) {
	theme := DefaultTheme()
	m := loadedConfigModel(&theme)

	if len(m.keys) < 2 {
		t.Skip("need at least 2 keys to test cursor movement")
	}

	// Move down
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	cm := updated.(*ConfigModel)
	if cm.cursor != 1 {
		t.Errorf("after j: cursor = %d, want 1", cm.cursor)
	}

	// Move back up
	updated, _ = cm.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	cm = updated.(*ConfigModel)
	if cm.cursor != 0 {
		t.Errorf("after k: cursor = %d, want 0", cm.cursor)
	}

	// Can't go above 0
	updated, _ = cm.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	cm = updated.(*ConfigModel)
	if cm.cursor != 0 {
		t.Errorf("cursor should stay at 0, got %d", cm.cursor)
	}
}

func TestConfigModel_NumberJump(t *testing.T) {
	theme := DefaultTheme()
	m := loadedConfigModel(&theme)

	if len(m.categories) < 2 {
		t.Skip("need at least 2 categories to test number jump")
	}

	// Press "2" to jump to second category
	updated, _ := m.Update(tea.KeyPressMsg{Code: '2', Text: "2"})
	cm := updated.(*ConfigModel)
	if cm.activeTab != 1 {
		t.Errorf("after pressing 2: activeTab = %d, want 1", cm.activeTab)
	}
}

func TestConfigModel_View(t *testing.T) {
	theme := DefaultTheme()
	m := loadedConfigModel(&theme)
	m.width = 80
	m.height = 24

	view := m.View()
	content := view.Content
	if content == "" {
		t.Error("View() returned empty content")
	}
	if !strings.Contains(content, "Configuration") {
		t.Error("View() should contain 'Configuration' header")
	}
}

func TestConfigModel_ViewWide(t *testing.T) {
	theme := DefaultTheme()
	m := loadedConfigModel(&theme)
	m.width = 140
	m.height = 30

	view := m.View()
	if view.Content == "" {
		t.Error("View() returned empty content for wide layout")
	}
}

func TestConfigModel_ViewLoading(t *testing.T) {
	theme := DefaultTheme()
	m := NewConfigView(&theme)
	m.width = 80
	m.height = 24

	view := m.View()
	if !strings.Contains(view.Content, "Loading") {
		t.Error("loading view should show 'Loading'")
	}
}

func TestConfigModel_ViewError(t *testing.T) {
	theme := DefaultTheme()
	m := NewConfigView(&theme)
	m.loading = false
	m.err = "test error"
	m.width = 80
	m.height = 24

	view := m.View()
	if !strings.Contains(view.Content, "test error") {
		t.Error("error view should show the error message")
	}
}

func TestConfigFormatValue(t *testing.T) {
	tests := []struct {
		name string
		val  interface{}
		want string
	}{
		{name: "nil", val: nil, want: "(unset)"},
		{name: "string", val: "hello", want: "hello"},
		{name: "bool_true", val: true, want: "true"},
		{name: "bool_false", val: false, want: "false"},
		{name: "int", val: 42, want: "42"},
		{name: "float", val: 3.14, want: "3.14"},
		{name: "empty_array", val: []interface{}{}, want: "[]"},
		{name: "short_array", val: []interface{}{"a", "b"}, want: "[a, b]"},
		{name: "object", val: map[string]interface{}{"key": "val"}, want: "{object}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := configFormatValue(tt.val)
			if got != tt.want {
				t.Errorf("configFormatValue(%v) = %q, want %q", tt.val, got, tt.want)
			}
		})
	}
}

func TestConfigFormatValue_LongString(t *testing.T) {
	long := strings.Repeat("x", 100)
	got := configFormatValue(long)
	if len(got) > 60 {
		t.Errorf("long string should be truncated to 60 chars, got %d", len(got))
	}
	if !strings.HasSuffix(got, "...") {
		t.Error("truncated string should end with ...")
	}
}

func TestConfigModel_RenderConfigTabs(t *testing.T) {
	theme := DefaultTheme()
	m := loadedConfigModel(&theme)
	m.width = 120

	tabs := m.renderConfigTabs()
	if tabs == "" {
		t.Error("renderConfigTabs returned empty string")
	}
	// First category should appear
	if len(m.categories) > 0 {
		first := string(m.categories[0])
		if len(first) > 12 {
			first = first[:12]
		}
		if !strings.Contains(tabs, first) {
			t.Errorf("renderConfigTabs missing first category %q", first)
		}
	}
}

// loadedConfigModel creates a ConfigModel with snapshot and registry loaded,
// suitable for testing navigation and rendering.
func loadedConfigModel(theme *Theme) *ConfigModel {
	m := NewConfigView(theme)
	reg := config.GlobalRegistry()
	snap := &config.ConfigSnapshot{
		Values: make(map[string]*config.ConfigValue),
		Home:   "/tmp/test-home",
		Cwd:    "/tmp/test-cwd",
	}
	regKeys := reg.All()
	for i := range regKeys {
		key := &regKeys[i]
		snap.Values[key.Key] = &config.ConfigValue{
			Key:         key.Key,
			IsDefault:   true,
			Source:      config.ScopeDefault,
			LayerValues: make(map[config.ConfigScope]interface{}),
		}
	}

	msg := configLoadedMsg{snap: snap, reg: reg}
	updated, _ := m.Update(msg)
	return updated.(*ConfigModel)
}
