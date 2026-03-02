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

func TestConfigModel_PageDown(t *testing.T) {
	theme := DefaultTheme()
	m := loadedConfigModel(&theme)
	m.height = 20

	if len(m.keys) < 2 {
		t.Skip("need at least 2 keys to test page down")
	}

	visible := m.visibleKeyRows()
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
	cm := updated.(*ConfigModel)

	want := visible
	if want > len(cm.keys)-1 {
		want = len(cm.keys) - 1
	}
	if cm.cursor != want {
		t.Errorf("after pgdn: cursor = %d, want %d (visible=%d, keys=%d)", cm.cursor, want, visible, len(cm.keys))
	}
}

func TestConfigModel_PageUp(t *testing.T) {
	theme := DefaultTheme()
	m := loadedConfigModel(&theme)
	m.height = 20

	if len(m.keys) < 2 {
		t.Skip("need at least 2 keys to test page up")
	}

	// Move cursor to the middle first
	mid := len(m.keys) / 2
	m.cursor = mid
	m.adjustScroll()

	visible := m.visibleKeyRows()
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyPgUp})
	cm := updated.(*ConfigModel)

	want := mid - visible
	if want < 0 {
		want = 0
	}
	if cm.cursor != want {
		t.Errorf("after pgup: cursor = %d, want %d", cm.cursor, want)
	}
}

func TestConfigModel_PageDownClamp(t *testing.T) {
	theme := DefaultTheme()
	m := loadedConfigModel(&theme)
	m.height = 20

	if len(m.keys) < 2 {
		t.Skip("need at least 2 keys")
	}

	// Move cursor near the end so page down would exceed bounds
	m.cursor = len(m.keys) - 2
	m.adjustScroll()

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
	cm := updated.(*ConfigModel)

	if cm.cursor != len(cm.keys)-1 {
		t.Errorf("page down at end: cursor = %d, want %d", cm.cursor, len(cm.keys)-1)
	}
}

func TestConfigModel_PageUpClamp(t *testing.T) {
	theme := DefaultTheme()
	m := loadedConfigModel(&theme)
	m.height = 20

	if len(m.keys) < 2 {
		t.Skip("need at least 2 keys")
	}

	// Set cursor to 1, page up should clamp to 0
	m.cursor = 1
	m.adjustScroll()

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyPgUp})
	cm := updated.(*ConfigModel)

	if cm.cursor != 0 {
		t.Errorf("page up near top: cursor = %d, want 0", cm.cursor)
	}
}

func TestConfigModel_GoToTop(t *testing.T) {
	theme := DefaultTheme()
	m := loadedConfigModel(&theme)

	if len(m.keys) < 3 {
		t.Skip("need at least 3 keys")
	}

	m.cursor = len(m.keys) / 2
	m.adjustScroll()

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	cm := updated.(*ConfigModel)

	if cm.cursor != 0 {
		t.Errorf("after g: cursor = %d, want 0", cm.cursor)
	}
}

func TestConfigModel_GoToBottom(t *testing.T) {
	theme := DefaultTheme()
	m := loadedConfigModel(&theme)

	if len(m.keys) < 3 {
		t.Skip("need at least 3 keys")
	}

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'G', Text: "G", Mod: tea.ModShift})
	cm := updated.(*ConfigModel)

	want := len(cm.keys) - 1
	if cm.cursor != want {
		t.Errorf("after G: cursor = %d, want %d", cm.cursor, want)
	}
}

func TestConfigModel_ArrayPageDown(t *testing.T) {
	theme := DefaultTheme()
	m := NewConfigView(&theme)
	m.height = 40
	m.arrayEditMode = true
	m.arrayItems = make([]interface{}, 50)
	m.arrayCursor = 0
	m.arrayScroll = 0
	m.snapshot = &config.ConfigSnapshot{
		Values: map[string]*config.ConfigValue{
			"test.key": {Key: "test.key", LayerValues: map[config.ConfigScope]interface{}{}},
		},
	}
	m.keys = []config.ConfigKey{{Key: "test.key", Type: config.TypeStringArray}}

	visible := m.visibleArrayRows()
	m.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})

	if m.arrayCursor != visible {
		t.Errorf("array pgdn: cursor = %d, want %d", m.arrayCursor, visible)
	}
}

func TestConfigModel_ArrayPageUp(t *testing.T) {
	theme := DefaultTheme()
	m := NewConfigView(&theme)
	m.height = 40
	m.arrayEditMode = true
	m.arrayItems = make([]interface{}, 50)
	m.arrayCursor = 25
	m.arrayScroll = 20
	m.snapshot = &config.ConfigSnapshot{
		Values: map[string]*config.ConfigValue{
			"test.key": {Key: "test.key", LayerValues: map[config.ConfigScope]interface{}{}},
		},
	}
	m.keys = []config.ConfigKey{{Key: "test.key", Type: config.TypeStringArray}}

	visible := m.visibleArrayRows()
	m.Update(tea.KeyPressMsg{Code: tea.KeyPgUp})

	want := 25 - visible
	if want < 0 {
		want = 0
	}
	if m.arrayCursor != want {
		t.Errorf("array pgup: cursor = %d, want %d", m.arrayCursor, want)
	}
}

func TestConfigModel_ArrayGoToTopBottom(t *testing.T) {
	theme := DefaultTheme()
	m := NewConfigView(&theme)
	m.height = 40
	m.arrayEditMode = true
	m.arrayItems = make([]interface{}, 50)
	m.arrayCursor = 25
	m.arrayScroll = 20
	m.snapshot = &config.ConfigSnapshot{
		Values: map[string]*config.ConfigValue{
			"test.key": {Key: "test.key", LayerValues: map[config.ConfigScope]interface{}{}},
		},
	}
	m.keys = []config.ConfigKey{{Key: "test.key", Type: config.TypeStringArray}}

	// Go to top
	m.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	if m.arrayCursor != 0 {
		t.Errorf("array g: cursor = %d, want 0", m.arrayCursor)
	}

	// Go to bottom
	m.Update(tea.KeyPressMsg{Code: 'G', Text: "G", Mod: tea.ModShift})
	if m.arrayCursor != 49 {
		t.Errorf("array G: cursor = %d, want 49", m.arrayCursor)
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

	items := make([]TabItem, 0, len(m.categories))
	for _, cat := range m.categories {
		name := string(cat)
		if len(name) > 12 {
			name = name[:12]
		}
		items = append(items, TabItem{Label: name})
	}

	out := renderTabBar(items, 0, 120, &theme)
	if out == "" {
		t.Error("renderTabBar returned empty string")
	}
	// First category should appear
	if len(m.categories) > 0 {
		first := string(m.categories[0])
		if len(first) > 12 {
			first = first[:12]
		}
		if !strings.Contains(out, first) {
			t.Errorf("renderTabBar missing first category %q", first)
		}
	}
}

func TestConfigModel_VisibleArrayRows(t *testing.T) {
	theme := DefaultTheme()
	tests := []struct {
		name   string
		height int
		items  int
		want   int
	}{
		// 40 tall, 50 items: available=40-4-2=34, half=17, maxItems=17-7=10
		{"normal terminal", 40, 50, 10},
		// 60 tall, 50 items: available=54, half=27, maxItems=27-7=20
		{"tall terminal", 60, 50, 20},
		// 30 tall, 5 items: available=24, half=12, maxItems=5 (capped to item count)
		{"short list fits", 30, 5, 5},
		// 10 tall: available=4, half=2, maxItems=2-7=-5 → clamp to 3
		{"tiny terminal clamps to 3", 10, 50, 3},
		// 0 tall: available=-6, half=-3, maxItems=-3-7=-10 → clamp to 3
		{"zero height clamps to minimum", 0, 50, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewConfigView(&theme)
			m.height = tt.height
			m.arrayItems = make([]interface{}, tt.items)
			got := m.visibleArrayRows()
			if got != tt.want {
				t.Errorf("visibleArrayRows() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestConfigModel_AdjustArrayScroll(t *testing.T) {
	theme := DefaultTheme()
	tests := []struct {
		name       string
		items      int
		cursor     int
		scroll     int
		height     int
		wantScroll int
	}{
		{"cursor in view, no change", 50, 5, 0, 40, 0},
		{"cursor above view, scroll up", 50, 2, 10, 40, 2},
		{"cursor at start", 50, 0, 5, 40, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewConfigView(&theme)
			m.height = tt.height
			m.arrayItems = make([]interface{}, tt.items)
			m.arrayCursor = tt.cursor
			m.arrayScroll = tt.scroll
			m.adjustArrayScroll()
			if m.arrayScroll != tt.wantScroll {
				t.Errorf("arrayScroll = %d, want %d", m.arrayScroll, tt.wantScroll)
			}
		})
	}
}

func TestConfigModel_AdjustArrayScroll_CursorBelowView(t *testing.T) {
	theme := DefaultTheme()
	m := NewConfigView(&theme)
	m.height = 40
	m.arrayItems = make([]interface{}, 50)
	m.arrayCursor = 30
	m.arrayScroll = 0
	m.adjustArrayScroll()

	visible := m.visibleArrayRows()
	wantScroll := 30 - visible + 1
	if m.arrayScroll != wantScroll {
		t.Errorf("arrayScroll = %d, want %d (visible=%d)", m.arrayScroll, wantScroll, visible)
	}
}

func TestConfigModel_KeyListShrinksForArrayPanel(t *testing.T) {
	theme := DefaultTheme()
	m := NewConfigView(&theme)
	m.height = 40

	// Without array mode, key list gets full space
	rowsWithout := m.visibleKeyRows()

	// With array mode, key list should shrink
	m.arrayEditMode = true
	m.arrayItems = make([]interface{}, 50)
	rowsWith := m.visibleKeyRows()

	if rowsWith >= rowsWithout {
		t.Errorf("key list should shrink when array panel active: without=%d, with=%d", rowsWithout, rowsWith)
	}
	// Total rendered lines should fit in terminal height
	totalUsed := configHeaderLines + configFooterLines + rowsWith + m.arrayPanelHeight()
	if totalUsed > m.height {
		t.Errorf("total content (%d) exceeds terminal height (%d)", totalUsed, m.height)
	}
}

func TestConfigModel_ArrayPanelWindowed(t *testing.T) {
	theme := DefaultTheme()
	m := loadedConfigModel(&theme)
	m.width = 80
	m.height = 30

	// Simulate entering array edit with many items
	items := make([]interface{}, 100)
	for i := range items {
		items[i] = "item-" + strings.Repeat("x", 3) + "-" + string(rune('0'+i%10))
	}
	m.arrayEditMode = true
	m.arrayItems = items
	m.arrayCursor = 0
	m.arrayScroll = 0
	m.editScopes = []config.ConfigScope{config.ScopeUser}
	m.editScopeIdx = 0
	m.editScope = config.ScopeUser

	view := m.View()
	output := view.Content

	// Should contain the first item (in the visible window)
	if !strings.Contains(output, "item-xxx-0") {
		t.Error("windowed output should contain item at scroll position 0")
	}

	// Count rendered item lines — should be at most visible rows
	visible := m.visibleArrayRows()
	lines := strings.Split(output, "\n")
	itemLines := 0
	for _, line := range lines {
		if strings.Contains(line, "item-xxx-") {
			itemLines++
		}
	}
	if itemLines > visible {
		t.Errorf("rendered %d item lines, want at most %d", itemLines, visible)
	}

	// Help bar should be present (not pushed off screen)
	if !strings.Contains(output, "navigate") {
		t.Error("help footer should be visible in windowed output")
	}
}

func TestConfigModel_ArrayScrollResetOnScopeChange(t *testing.T) {
	theme := DefaultTheme()
	m := NewConfigView(&theme)
	m.height = 30
	m.arrayEditMode = true
	m.arrayItems = make([]interface{}, 50)
	m.arrayCursor = 25
	m.arrayScroll = 20
	m.editScopes = []config.ConfigScope{config.ScopeUser, config.ScopeLocal}
	m.editScopeIdx = 0
	m.editScope = config.ScopeUser

	// Provide a minimal snapshot and keys so tab doesn't panic
	m.snapshot = &config.ConfigSnapshot{
		Values: map[string]*config.ConfigValue{
			"test.key": {
				Key:         "test.key",
				LayerValues: map[config.ConfigScope]interface{}{},
			},
		},
		Home: "/tmp",
		Cwd:  "/tmp",
	}
	m.keys = []config.ConfigKey{{Key: "test.key", Type: config.TypeStringArray}}
	m.cursor = 0

	// Press tab to switch scope
	m.Update(tea.KeyPressMsg{Code: tea.KeyTab})

	if m.arrayScroll != 0 {
		t.Errorf("arrayScroll after scope tab = %d, want 0", m.arrayScroll)
	}
	if m.arrayCursor != 0 {
		t.Errorf("arrayCursor after scope tab = %d, want 0", m.arrayCursor)
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
