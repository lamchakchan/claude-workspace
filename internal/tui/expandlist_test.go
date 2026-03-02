package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// testItem implements ListItem for testing.
type testItem struct {
	title  string
	detail string
}

func (t *testItem) Title() string  { return t.title }
func (t *testItem) Detail() string { return t.detail }

func TestExpandList_FlattenSections(t *testing.T) {
	tests := []struct {
		name           string
		sections       []ListSection
		wantEntries    int
		wantSelectable int
	}{
		{
			name: "two sections with items",
			sections: []ListSection{
				{Title: "Section A", Items: []ListItem{
					&testItem{title: "a1"}, &testItem{title: "a2"},
				}},
				{Title: "Section B", Items: []ListItem{
					&testItem{title: "b1"},
				}},
			},
			wantEntries:    5, // 2 headers + 3 items
			wantSelectable: 3,
		},
		{
			name:           "empty sections",
			sections:       []ListSection{},
			wantEntries:    0,
			wantSelectable: 0,
		},
		{
			name: "section with no items",
			sections: []ListSection{
				{Title: "Empty", Items: nil},
			},
			wantEntries:    1, // header only
			wantSelectable: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			theme := DefaultTheme()
			m := &ExpandListModel{theme: &theme}
			m.flatten(tt.sections)
			if len(m.entries) != tt.wantEntries {
				t.Errorf("entries = %d, want %d", len(m.entries), tt.wantEntries)
			}
			if got := m.selectableCount(); got != tt.wantSelectable {
				t.Errorf("selectableCount = %d, want %d", got, tt.wantSelectable)
			}
		})
	}
}

func TestExpandList_CursorSkipsHeaders(t *testing.T) {
	theme := DefaultTheme()
	m := &ExpandListModel{theme: &theme}
	m.flatten([]ListSection{
		{Title: "Header A", Items: []ListItem{
			&testItem{title: "item-1"},
			&testItem{title: "item-2"},
		}},
		{Title: "Header B", Items: []ListItem{
			&testItem{title: "item-3"},
		}},
	})

	// Cursor should start on first selectable item (index 1, skipping header at 0)
	if m.cursor != 1 {
		t.Errorf("initial cursor = %d, want 1", m.cursor)
	}

	// Move down through items, should skip header at index 3
	m.cursor = m.nextSelectable(m.cursor+1, 1) // -> index 2 (item-2)
	if m.cursor != 2 {
		t.Errorf("after first down cursor = %d, want 2", m.cursor)
	}

	m.cursor = m.nextSelectable(m.cursor+1, 1) // -> index 4 (item-3, skipping header at 3)
	if m.cursor != 4 {
		t.Errorf("after second down cursor = %d, want 4", m.cursor)
	}

	// Move back up should skip header at index 3
	m.cursor = m.nextSelectable(m.cursor-1, -1) // -> index 2 (item-2)
	if m.cursor != 2 {
		t.Errorf("after up cursor = %d, want 2", m.cursor)
	}
}

func TestExpandList_EntryHeight(t *testing.T) {
	theme := DefaultTheme()
	m := &ExpandListModel{theme: &theme}
	m.flatten([]ListSection{
		{Title: "Section", Items: []ListItem{
			&testItem{title: "item", detail: "line1\nline2\nline3"},
		}},
	})

	// All entries are height 1 in list mode (no inline expansion)
	if h := m.entryHeight(0); h != 1 {
		t.Errorf("header height = %d, want 1", h)
	}
	if h := m.entryHeight(1); h != 1 {
		t.Errorf("item height = %d, want 1", h)
	}
}

func TestExpandList_LineOf(t *testing.T) {
	theme := DefaultTheme()
	m := &ExpandListModel{theme: &theme}
	m.flatten([]ListSection{
		{Title: "Section", Items: []ListItem{
			&testItem{title: "a", detail: "d1\nd2"},
			&testItem{title: "b", detail: "d3"},
		}},
	})

	// All entries are 1 line each, so lineOf(i) == i
	for i := range m.entries {
		if l := m.lineOf(i); l != i {
			t.Errorf("lineOf(%d) = %d, want %d", i, l, i)
		}
	}
}

func TestExpandList_ScrollClamp(t *testing.T) {
	theme := DefaultTheme()
	m := &ExpandListModel{theme: &theme, height: 10} // visibleLines = 10 - 4 - 3 = 3

	items := make([]ListItem, 20)
	for i := range items {
		items[i] = &testItem{title: strings.Repeat("x", i+1), detail: "detail"}
	}
	m.flatten([]ListSection{
		{Title: "Section", Items: items},
	})

	// Move cursor to last item
	m.cursor = len(m.entries) - 1
	m.clampScroll()

	visible := m.visibleLines()
	cursorLine := m.lineOf(m.cursor)

	// Cursor should be within visible window
	if cursorLine >= m.scroll+visible {
		t.Errorf("cursor line %d exceeds scroll window %d+%d", cursorLine, m.scroll, visible)
	}
	if cursorLine < m.scroll {
		t.Errorf("cursor line %d before scroll start %d", cursorLine, m.scroll)
	}
}

func TestExpandList_EmptyList(t *testing.T) {
	theme := DefaultTheme()
	m := &ExpandListModel{theme: &theme, height: 20}
	m.flatten([]ListSection{})

	if m.selectableCount() != 0 {
		t.Errorf("selectableCount = %d, want 0", m.selectableCount())
	}
	if m.totalLines() != 0 {
		t.Errorf("totalLines = %d, want 0", m.totalLines())
	}
}

func TestExpandList_SelectableIndex(t *testing.T) {
	theme := DefaultTheme()
	m := &ExpandListModel{theme: &theme}
	m.flatten([]ListSection{
		{Title: "A", Items: []ListItem{
			&testItem{title: "1"},
			&testItem{title: "2"},
		}},
		{Title: "B", Items: []ListItem{
			&testItem{title: "3"},
		}},
	})

	// Cursor on first item (entry index 1) -> selectable index 1
	m.cursor = 1
	if idx := m.selectableIndex(); idx != 1 {
		t.Errorf("selectableIndex at cursor=1 got %d, want 1", idx)
	}

	// Cursor on third item (entry index 4) -> selectable index 3
	m.cursor = 4
	if idx := m.selectableIndex(); idx != 3 {
		t.Errorf("selectableIndex at cursor=4 got %d, want 3", idx)
	}
}

func TestExpandList_EnterDetailMode(t *testing.T) {
	theme := DefaultTheme()
	m := &ExpandListModel{theme: &theme, width: 80, height: 30}
	m.flatten([]ListSection{
		{Title: "Section", Items: []ListItem{
			&testItem{title: "item-1", detail: "detail content\nline two\nline three"},
		}},
	})

	// Should not be in detail mode initially
	if m.detailMode {
		t.Error("should not start in detail mode")
	}

	// Enter detail mode
	m.enterDetail()
	if !m.detailMode {
		t.Error("should be in detail mode after enterDetail()")
	}

	// Viewport should be configured
	if m.detailVP.Width() != m.detailVPWidth() {
		t.Errorf("viewport width = %d, want %d", m.detailVP.Width(), m.detailVPWidth())
	}
	if m.detailVP.Height() != m.detailVPHeight() {
		t.Errorf("viewport height = %d, want %d", m.detailVP.Height(), m.detailVPHeight())
	}
}

func TestExpandList_EnterDetailMode_OnHeader(t *testing.T) {
	theme := DefaultTheme()
	m := &ExpandListModel{theme: &theme, width: 80, height: 30}
	m.flatten([]ListSection{
		{Title: "Section", Items: []ListItem{
			&testItem{title: "item-1", detail: "detail"},
		}},
	})

	// Force cursor onto header
	m.cursor = 0
	m.enterDetail()

	// Should NOT enter detail mode on a header
	if m.detailMode {
		t.Error("should not enter detail mode on a header entry")
	}
}

func TestExpandList_ExitDetailMode(t *testing.T) {
	theme := DefaultTheme()
	m := &ExpandListModel{theme: &theme, width: 80, height: 30}
	m.flatten([]ListSection{
		{Title: "Section", Items: []ListItem{
			&testItem{title: "item-1", detail: "detail content"},
		}},
	})

	m.enterDetail()
	if !m.detailMode {
		t.Fatal("expected detail mode after enterDetail()")
	}

	// Exit via esc key
	m.exitDetail()
	if m.detailMode {
		t.Error("should not be in detail mode after exitDetail()")
	}
}

func TestExpandList_DetailKeyHandling(t *testing.T) {
	theme := DefaultTheme()
	m := &ExpandListModel{theme: &theme, width: 80, height: 30}
	m.flatten([]ListSection{
		{Title: "Section", Items: []ListItem{
			&testItem{title: "item-1", detail: "detail content"},
		}},
	})
	m.enterDetail()

	// Enter key should exit detail mode
	m.handleDetailKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.detailMode {
		t.Error("enter key should exit detail mode")
	}

	// Space key should NOT exit detail mode (it pages down in viewport)
	m.enterDetail()
	m.handleDetailKey(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if !m.detailMode {
		t.Error("space key should not exit detail mode")
	}
}

func TestExpandList_DetailViewportResize(t *testing.T) {
	theme := DefaultTheme()
	m := &ExpandListModel{theme: &theme, width: 80, height: 30}
	m.flatten([]ListSection{
		{Title: "Section", Items: []ListItem{
			&testItem{title: "item-1", detail: "detail content\nline two"},
		}},
	})
	m.enterDetail()

	origWidth := m.detailVPWidth()
	origHeight := m.detailVPHeight()

	// Simulate window resize
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 50})

	newWidth := m.detailVPWidth()
	newHeight := m.detailVPHeight()

	if newWidth == origWidth {
		t.Error("viewport width should change after resize")
	}
	if newHeight == origHeight {
		t.Error("viewport height should change after resize")
	}
	if m.detailVP.Width() != newWidth {
		t.Errorf("viewport width = %d, want %d after resize", m.detailVP.Width(), newWidth)
	}
	if m.detailVP.Height() != newHeight {
		t.Errorf("viewport height = %d, want %d after resize", m.detailVP.Height(), newHeight)
	}
}
