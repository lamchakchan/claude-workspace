package tui

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lamchakchan/claude-workspace/internal/config"
)

// configLoadedMsg carries the loaded config snapshot and registry.
type configLoadedMsg struct {
	snap *config.ConfigSnapshot
	reg  *config.Registry
}

// configErrorMsg carries an error from the async config loader.
type configErrorMsg struct{ err error }

// configSavedMsg signals a successful config write.
type configSavedMsg struct{ key string }

// configDeletedMsg signals a successful config key deletion.
type configDeletedMsg struct{ key string }

// ConfigModel displays configuration keys grouped by category with tab navigation,
// key list with source badges, detail panel, and inline edit support.
type ConfigModel struct {
	theme      *Theme
	categories []config.Category
	activeTab  int
	keys       []config.ConfigKey // keys in active category (filtered)
	allKeys    []config.ConfigKey // keys in active category (unfiltered)
	cursor     int
	scroll     int
	filter     string
	filterMode bool
	snapshot   *config.ConfigSnapshot
	registry   *config.Registry
	loading    bool
	err        string
	width      int
	height     int
	// Text/object/enum edit mode
	editMode     bool
	editValue    string
	editScope    config.ConfigScope
	editScopes   []config.ConfigScope
	editScopeIdx int
	editEnumIdx  int // cursor within TypeEnum EnumValues list
	// Delete mode
	deleteMode bool
	// Array editor mode (TypeStringArray)
	arrayEditMode bool
	arrayItems    []interface{} // items in the selected scope (not merged)
	arrayCursor   int
	arrayAddMode  bool   // sub-mode: typing a new item to append
	arrayAddValue string // text being entered for new item
}

// configHeaderLines is the number of lines used by the banner + tab bar.
const configHeaderLines = 4

// configFooterLines is the number of lines used by the help footer.
const configFooterLines = 2

// editableScopes are the scopes available for inline editing.
var editableScopes = []config.ConfigScope{config.ScopeUser, config.ScopeProject, config.ScopeLocal}

// NewConfigView creates a new configuration viewer.
func NewConfigView(theme *Theme) *ConfigModel {
	return &ConfigModel{
		theme:      theme,
		loading:    true,
		editScopes: editableScopes,
		editScope:  config.ScopeUser,
	}
}

func (m *ConfigModel) Init() tea.Cmd {
	return func() tea.Msg {
		snap, err := config.ReadAll()
		if err != nil {
			return configErrorMsg{err: err}
		}
		return configLoadedMsg{snap: snap, reg: config.GlobalRegistry()}
	}
}

func (m *ConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case configLoadedMsg:
		m.loading = false
		m.snapshot = msg.snap
		m.registry = msg.reg
		m.categories = msg.reg.Categories()
		if len(m.categories) > 0 {
			m.switchCategory(0)
		}
		return m, nil

	case configErrorMsg:
		m.loading = false
		m.err = msg.err.Error()
		return m, nil

	case configSavedMsg:
		m.editMode = false
		m.editValue = ""
		m.arrayEditMode = false
		// Reload snapshot
		return m, func() tea.Msg {
			snap, err := config.ReadAll()
			if err != nil {
				return configErrorMsg{err: err}
			}
			return configLoadedMsg{snap: snap, reg: config.GlobalRegistry()}
		}

	case configDeletedMsg:
		m.deleteMode = false
		m.arrayEditMode = false
		// Reload snapshot
		return m, func() tea.Msg {
			snap, err := config.ReadAll()
			if err != nil {
				return configErrorMsg{err: err}
			}
			return configLoadedMsg{snap: snap, reg: config.GlobalRegistry()}
		}

	case tea.KeyPressMsg:
		if m.arrayEditMode {
			return m.handleArrayKey(msg)
		}
		if m.deleteMode {
			return m.handleDeleteKey(msg)
		}
		if m.editMode {
			return m.handleEditKey(msg)
		}
		if m.filterMode {
			return m.handleFilterKey(msg)
		}
		return m.handleNormalKey(msg)
	}

	return m, nil
}

// handleNormalKey handles key events in normal (non-filter, non-edit) mode.
func (m *ConfigModel) handleNormalKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) { //nolint:gocyclo // key dispatch requires handling many distinct key bindings
	switch msg.String() {
	case "q", keyCtrlC:
		return m, func() tea.Msg { return PopViewMsg{} }
	case keyEsc:
		if m.filter != "" {
			m.filter = ""
			m.applyFilter()
			return m, nil
		}
		return m, func() tea.Msg { return PopViewMsg{} }
	case keyTab, "l", keyRight:
		if len(m.categories) > 0 {
			m.switchCategory((m.activeTab + 1) % len(m.categories))
		}
	case keyShiftTab, "h", keyLeft:
		if len(m.categories) > 0 {
			m.switchCategory((m.activeTab + len(m.categories) - 1) % len(m.categories))
		}
	case "j", keyDown:
		if m.cursor < len(m.keys)-1 {
			m.cursor++
			m.adjustScroll()
		}
	case "k", keyUp:
		if m.cursor > 0 {
			m.cursor--
			m.adjustScroll()
		}
	case "/":
		m.filterMode = true
	case "e":
		if len(m.keys) == 0 {
			break
		}
		key := m.keys[m.cursor]
		if key.ReadOnly || strings.HasPrefix(key.Key, "file:") {
			break
		}
		m.editScopeIdx = 0
		m.editScope = m.editScopes[0]
		switch key.Type {
		case config.TypeObject:
			// Open read-only object viewer
			m.editMode = true
		case config.TypeEnum:
			m.editMode = true
			m.editEnumIdx = 0
			// Pre-select current effective value in the enum list
			if len(key.EnumValues) > 0 {
				if cv, ok := m.snapshot.Values[key.Key]; ok && cv != nil && !cv.IsDefault {
					val := configFormatValue(cv.EffectiveValue)
					for i, v := range key.EnumValues {
						if v == val {
							m.editEnumIdx = i
							break
						}
					}
				}
			}
		case config.TypeStringArray:
			m.arrayEditMode = true
			m.arrayCursor = 0
			m.arrayAddMode = false
			m.arrayAddValue = ""
			m.refreshArrayItems()
		default:
			// TypeString, TypeBool, TypeInt — text editor
			m.editMode = true
			if cv, ok := m.snapshot.Values[key.Key]; ok && cv != nil && !cv.IsDefault {
				m.editValue = configFormatValue(cv.EffectiveValue)
			} else {
				m.editValue = ""
			}
		}
	case "d":
		if len(m.keys) == 0 {
			break
		}
		key := m.keys[m.cursor]
		if key.ReadOnly || strings.HasPrefix(key.Key, "file:") || key.Category == config.CatFiles {
			break
		}
		m.deleteMode = true
		m.editScopeIdx = 0
		m.editScope = m.editScopes[0]
	default:
		// Number keys 1-9 jump to category
		if len(msg.Text) == 1 && msg.Text[0] >= '1' && msg.Text[0] <= '9' {
			idx := int(msg.Text[0] - '1')
			if idx < len(m.categories) {
				m.switchCategory(idx)
			}
		}
	}
	return m, nil
}

// handleFilterKey handles key events in filter mode.
func (m *ConfigModel) handleFilterKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc:
		m.filterMode = false
		m.filter = ""
		m.applyFilter()
	case keyEnter:
		m.filterMode = false
	case "backspace":
		if len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
			m.applyFilter()
		}
	default:
		if len(msg.Text) > 0 && msg.Text[0] >= ' ' {
			m.filter += msg.Text
			m.applyFilter()
		}
	}
	return m, nil
}

// handleEditKey handles key events in edit mode, dispatching by key type.
func (m *ConfigModel) handleEditKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if len(m.keys) == 0 {
		return m, nil
	}
	key := m.keys[m.cursor]

	// TypeEnum: arrow-key selector
	if key.Type == config.TypeEnum {
		return m.handleEnumKey(msg)
	}
	// TypeObject: read-only viewer — only Esc closes it
	if key.Type == config.TypeObject {
		if msg.String() == keyEsc {
			m.editMode = false
		}
		return m, nil
	}
	// TypeString, TypeBool, TypeInt: free text input
	switch msg.String() {
	case keyEsc:
		m.editMode = false
		m.editValue = ""
	case keyEnter:
		val := m.editValue
		scope := m.editScope
		home, cwd := "", ""
		if m.snapshot != nil {
			home = m.snapshot.Home
			cwd = m.snapshot.Cwd
		}
		return m, func() tea.Msg {
			err := config.WriteSettingsValue(key.Key, val, scope, home, cwd)
			if err != nil {
				return configErrorMsg{err: err}
			}
			return configSavedMsg{key: key.Key}
		}
	case keyTab:
		m.editScopeIdx = (m.editScopeIdx + 1) % len(m.editScopes)
		m.editScope = m.editScopes[m.editScopeIdx]
	case "backspace":
		if len(m.editValue) > 0 {
			m.editValue = m.editValue[:len(m.editValue)-1]
		}
	default:
		if len(msg.Text) > 0 && msg.Text[0] >= ' ' {
			m.editValue += msg.Text
		}
	}
	return m, nil
}

// handleEnumKey handles key events in the enum-selector sub-mode.
func (m *ConfigModel) handleEnumKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if len(m.keys) == 0 {
		return m, nil
	}
	key := m.keys[m.cursor]
	switch msg.String() {
	case keyEsc:
		m.editMode = false
	case "j", keyDown:
		if m.editEnumIdx < len(key.EnumValues)-1 {
			m.editEnumIdx++
		}
	case "k", keyUp:
		if m.editEnumIdx > 0 {
			m.editEnumIdx--
		}
	case keyEnter:
		if len(key.EnumValues) == 0 {
			return m, nil
		}
		val := key.EnumValues[m.editEnumIdx]
		scope := m.editScope
		home, cwd := "", ""
		if m.snapshot != nil {
			home = m.snapshot.Home
			cwd = m.snapshot.Cwd
		}
		return m, func() tea.Msg {
			err := config.WriteSettingsValue(key.Key, val, scope, home, cwd)
			if err != nil {
				return configErrorMsg{err: err}
			}
			return configSavedMsg{key: key.Key}
		}
	case keyTab:
		m.editScopeIdx = (m.editScopeIdx + 1) % len(m.editScopes)
		m.editScope = m.editScopes[m.editScopeIdx]
	}
	return m, nil
}

// handleDeleteKey handles key events in delete confirmation mode.
func (m *ConfigModel) handleDeleteKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc:
		m.deleteMode = false
	case keyEnter:
		if len(m.keys) == 0 {
			return m, nil
		}
		key := m.keys[m.cursor]
		scope := m.editScope
		home, cwd := "", ""
		if m.snapshot != nil {
			home = m.snapshot.Home
			cwd = m.snapshot.Cwd
		}
		return m, func() tea.Msg {
			err := config.DeleteSettingsValue(key.Key, scope, home, cwd)
			if err != nil {
				return configErrorMsg{err: err}
			}
			return configDeletedMsg{key: key.Key}
		}
	case keyTab:
		m.editScopeIdx = (m.editScopeIdx + 1) % len(m.editScopes)
		m.editScope = m.editScopes[m.editScopeIdx]
	}
	return m, nil
}

// handleArrayKey dispatches array-editor key events between navigation and add-item modes.
func (m *ConfigModel) handleArrayKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.arrayAddMode {
		return m.handleArrayAddKey(msg)
	}
	return m.handleArrayNavKey(msg)
}

// handleArrayNavKey handles navigation within the array editor.
func (m *ConfigModel) handleArrayNavKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) { //nolint:gocyclo
	switch msg.String() {
	case keyEsc:
		m.arrayEditMode = false
		m.arrayItems = nil
		// Reload snapshot to reflect any changes made during array editing
		return m, func() tea.Msg {
			snap, err := config.ReadAll()
			if err != nil {
				return configErrorMsg{err: err}
			}
			return configLoadedMsg{snap: snap, reg: config.GlobalRegistry()}
		}
	case "j", keyDown:
		if m.arrayCursor < len(m.arrayItems)-1 {
			m.arrayCursor++
		}
	case "k", keyUp:
		if m.arrayCursor > 0 {
			m.arrayCursor--
		}
	case "a":
		m.arrayAddMode = true
		m.arrayAddValue = ""
	case "d":
		if len(m.arrayItems) == 0 || m.snapshot == nil || len(m.keys) == 0 {
			return m, nil
		}
		key := m.keys[m.cursor]
		item := fmt.Sprintf("%v", m.arrayItems[m.arrayCursor])
		scope := m.editScope
		home, cwd := m.snapshot.Home, m.snapshot.Cwd
		// Remove from local list immediately for responsive UX
		newItems := make([]interface{}, 0, len(m.arrayItems)-1)
		for i, v := range m.arrayItems {
			if i != m.arrayCursor {
				newItems = append(newItems, v)
			}
		}
		m.arrayItems = newItems
		if m.arrayCursor >= len(m.arrayItems) {
			m.arrayCursor = max(0, len(m.arrayItems)-1)
		}
		return m, func() tea.Msg {
			if err := config.RemoveFromArray(key.Key, item, scope, home, cwd); err != nil {
				return configErrorMsg{err: err}
			}
			return nil // local state already updated; no reload needed
		}
	case keyTab:
		m.editScopeIdx = (m.editScopeIdx + 1) % len(m.editScopes)
		m.editScope = m.editScopes[m.editScopeIdx]
		m.arrayCursor = 0
		m.refreshArrayItems()
	}
	return m, nil
}

// handleArrayAddKey handles key events while the user is typing a new array item.
func (m *ConfigModel) handleArrayAddKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc:
		m.arrayAddMode = false
		m.arrayAddValue = ""
	case keyEnter:
		val := strings.TrimSpace(m.arrayAddValue)
		if val == "" || m.snapshot == nil || len(m.keys) == 0 {
			return m, nil
		}
		key := m.keys[m.cursor]
		scope := m.editScope
		home, cwd := m.snapshot.Home, m.snapshot.Cwd
		// Append to local list immediately
		m.arrayItems = append(m.arrayItems, val)
		m.arrayAddMode = false
		m.arrayAddValue = ""
		return m, func() tea.Msg {
			if err := config.AppendToArray(key.Key, val, scope, home, cwd); err != nil {
				return configErrorMsg{err: err}
			}
			return nil // local state already updated; no reload needed
		}
	case "backspace":
		if len(m.arrayAddValue) > 0 {
			m.arrayAddValue = m.arrayAddValue[:len(m.arrayAddValue)-1]
		}
	default:
		if len(msg.Text) > 0 && msg.Text[0] >= ' ' {
			m.arrayAddValue += msg.Text
		}
	}
	return m, nil
}

// refreshArrayItems loads array items for the currently selected scope from the snapshot.
func (m *ConfigModel) refreshArrayItems() {
	if len(m.keys) == 0 || m.snapshot == nil {
		m.arrayItems = nil
		return
	}
	key := m.keys[m.cursor]
	cv := m.snapshot.Values[key.Key]
	if cv == nil {
		m.arrayItems = nil
		return
	}
	val, ok := cv.LayerValues[m.editScope]
	if !ok {
		m.arrayItems = nil
		return
	}
	if arr, ok := val.([]interface{}); ok {
		m.arrayItems = arr
	} else {
		m.arrayItems = nil
	}
	m.arrayCursor = 0
}

// switchCategory changes the active tab and resets the cursor.
func (m *ConfigModel) switchCategory(idx int) {
	m.activeTab = idx
	if m.registry != nil && idx < len(m.categories) {
		m.allKeys = m.registry.ByCategory(m.categories[idx])
	}
	m.cursor = 0
	m.scroll = 0
	m.applyFilter()
}

// applyFilter filters allKeys by the current filter string.
func (m *ConfigModel) applyFilter() {
	if m.filter == "" {
		m.keys = m.allKeys
	} else {
		q := strings.ToLower(m.filter)
		filtered := make([]config.ConfigKey, 0, len(m.allKeys))
		for i := range m.allKeys {
			key := &m.allKeys[i]
			if strings.Contains(strings.ToLower(key.Key), q) ||
				strings.Contains(strings.ToLower(key.Description), q) {
				filtered = append(filtered, *key)
			}
		}
		m.keys = filtered
	}
	if m.cursor >= len(m.keys) {
		m.cursor = max(0, len(m.keys)-1)
	}
	m.adjustScroll()
}

// visibleKeyRows returns the number of key rows visible in the key list.
func (m *ConfigModel) visibleKeyRows() int {
	v := m.height - configHeaderLines - configFooterLines
	if v < 1 {
		return 1
	}
	return v
}

// adjustScroll ensures the cursor is visible within the scroll window.
func (m *ConfigModel) adjustScroll() {
	visible := m.visibleKeyRows()
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+visible {
		m.scroll = m.cursor - visible + 1
	}
}

func (m *ConfigModel) View() tea.View {
	var b strings.Builder

	// Title banner
	b.WriteString(m.theme.SectionBanner("Configuration"))

	if m.loading {
		b.WriteString("\n  Loading...")
		return tea.NewView(b.String())
	}

	if m.err != "" {
		b.WriteString("\n  ")
		b.WriteString(lipgloss.NewStyle().Foreground(m.theme.Error).Render(m.err))
		b.WriteString("\n\n  Press q to go back.\n")
		return tea.NewView(b.String())
	}

	// Tab bar
	b.WriteString(m.renderConfigTabs())
	b.WriteString("\n")

	// Main content area
	if m.width >= 120 {
		b.WriteString(m.renderTwoColumn())
	} else {
		b.WriteString(m.renderSingleColumn())
	}

	// Edit overlay (text, enum, or object viewer)
	if m.editMode && len(m.keys) > 0 {
		b.WriteString("\n")
		b.WriteString(m.renderEditPanel())
	}

	// Array editor overlay
	if m.arrayEditMode && len(m.keys) > 0 {
		b.WriteString("\n")
		b.WriteString(m.renderArrayPanel())
	}

	// Delete overlay
	if m.deleteMode && len(m.keys) > 0 {
		b.WriteString("\n")
		b.WriteString(m.renderDeletePanel())
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(m.renderFooter())

	return tea.NewView(b.String())
}

// renderConfigTabs renders the category tab bar.
func (m *ConfigModel) renderConfigTabs() string {
	selected := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Primary)
	unselected := lipgloss.NewStyle().Foreground(m.theme.Muted)
	separator := lipgloss.NewStyle().Foreground(m.theme.Muted)

	tabs := make([]string, 0, len(m.categories))
	for i, cat := range m.categories {
		label := fmt.Sprintf("[%d]", i+1)
		name := string(cat)
		if len(name) > 12 {
			name = name[:12]
		}
		if i == m.activeTab {
			tabs = append(tabs, selected.Render(label+" "+name)+"  ")
		} else {
			tabs = append(tabs, unselected.Render(label+" "+name)+"  ")
		}
	}

	line := "  " + strings.Join(tabs, "")
	rule := separator.Render("  " + strings.Repeat("─", max(1, m.width-4)))
	return line + "\n" + rule
}

// renderTwoColumn renders side-by-side key list and detail panel.
func (m *ConfigModel) renderTwoColumn() string {
	leftWidth := m.width / 2
	rightWidth := m.width - leftWidth - 3 // gap

	var left strings.Builder
	m.renderKeyList(&left, leftWidth)

	var right strings.Builder
	if len(m.keys) > 0 {
		m.renderDetailPanel(&right, rightWidth)
	}

	leftStr := left.String()
	rightStr := right.String()

	return lipgloss.JoinHorizontal(lipgloss.Top, leftStr, "   ", rightStr)
}

// renderSingleColumn renders key list only; detail is shown inline for selected key.
func (m *ConfigModel) renderSingleColumn() string {
	var b strings.Builder
	m.renderKeyList(&b, m.width-2)

	if len(m.keys) > 0 {
		b.WriteString("\n")
		m.renderDetailPanel(&b, m.width-4)
	}

	return b.String()
}

// renderKeyList renders the scrollable key list with source badges.
func (m *ConfigModel) renderKeyList(b *strings.Builder, width int) {
	if len(m.keys) == 0 {
		b.WriteString("  (no keys")
		if m.filter != "" {
			b.WriteString(" matching filter")
		}
		b.WriteString(")\n")
		return
	}

	visible := m.visibleKeyRows()
	end := m.scroll + visible
	if end > len(m.keys) {
		end = len(m.keys)
	}

	for i := m.scroll; i < end; i++ {
		badge := m.scopeBadge(&m.keys[i])
		key := m.keys[i]
		name := key.Key

		// Truncate key name to fit
		maxName := width - 8 // badge(5) + spaces
		if maxName < 10 {
			maxName = 10
		}
		if len(name) > maxName {
			name = name[:maxName-3] + "..."
		}

		cursor := "  "
		if i == m.cursor {
			cursor = lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render("> ")
			name = lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render(name)
		}

		fmt.Fprintf(b, "%s%s %s\n", cursor, badge, name)
	}
}

// scopeBadge returns a styled source badge for a config key.
func (m *ConfigModel) scopeBadge(key *config.ConfigKey) string {
	cv := m.snapshot.Values[key.Key]
	if cv == nil || cv.IsDefault {
		return lipgloss.NewStyle().Foreground(m.theme.Muted).Render("[DEF]")
	}
	switch cv.Source {
	case config.ScopeManaged:
		return lipgloss.NewStyle().Foreground(m.theme.Error).Bold(true).Render("[MGD]")
	case config.ScopeUser:
		return lipgloss.NewStyle().Foreground(m.theme.Secondary).Render("[USR]")
	case config.ScopeProject:
		return lipgloss.NewStyle().Foreground(m.theme.Success).Render("[PRJ]")
	case config.ScopeLocal:
		return lipgloss.NewStyle().Foreground(m.theme.Warning).Render("[LOC]")
	case config.ScopeEnv:
		return lipgloss.NewStyle().Foreground(m.theme.Primary).Render("[ENV]")
	default:
		return lipgloss.NewStyle().Foreground(m.theme.Muted).Render("[DEF]")
	}
}

// renderDetailPanel renders the detail view for the selected key.
func (m *ConfigModel) renderDetailPanel(b *strings.Builder, width int) {
	if m.cursor >= len(m.keys) {
		return
	}
	key := m.keys[m.cursor]
	cv := m.snapshot.Values[key.Key]

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Secondary)
	labelStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)
	_ = width

	b.WriteString("  ")
	b.WriteString(headerStyle.Render(key.Key))
	b.WriteString("\n")

	b.WriteString("  ")
	b.WriteString(labelStyle.Render("Type: "))
	b.WriteString(string(key.Type))
	b.WriteString("\n")

	if key.Default != "" {
		b.WriteString("  ")
		b.WriteString(labelStyle.Render("Default: "))
		b.WriteString(key.Default)
		b.WriteString("\n")
	}

	b.WriteString("  ")
	b.WriteString(labelStyle.Render("Description: "))
	b.WriteString(key.Description)
	b.WriteString("\n")

	if len(key.EnumValues) > 0 {
		b.WriteString("  ")
		b.WriteString(labelStyle.Render("Values: "))
		b.WriteString(strings.Join(key.EnumValues, ", "))
		b.WriteString("\n")
	}

	// Effective value
	b.WriteString("\n  ")
	b.WriteString(labelStyle.Render("Effective: "))
	if cv == nil || cv.IsDefault {
		defVal := key.Default
		if defVal == "" {
			defVal = "(unset)"
		}
		b.WriteString(defVal)
	} else {
		b.WriteString(configFormatValue(cv.EffectiveValue))
	}
	b.WriteString("\n")

	// Layer values with effective marker
	if cv != nil && len(cv.LayerValues) > 0 {
		b.WriteString("\n  ")
		b.WriteString(labelStyle.Render("Layers:"))
		b.WriteString("\n")
		scopes := []config.ConfigScope{
			config.ScopeManaged, config.ScopeUser, config.ScopeProject,
			config.ScopeLocal, config.ScopeEnv,
		}
		effectiveStyle := lipgloss.NewStyle().Foreground(m.theme.Primary)
		for _, scope := range scopes {
			val, ok := cv.LayerValues[scope]
			if !ok {
				continue
			}
			marker := ""
			if scope == cv.Source {
				marker = "  " + effectiveStyle.Render("← effective")
			}
			fmt.Fprintf(b, "    %-8s %s%s\n", string(scope)+":", configFormatValue(val), marker)
		}
	}
}

// renderEditPanel dispatches to the appropriate edit sub-panel based on key type.
func (m *ConfigModel) renderEditPanel() string {
	if m.cursor >= len(m.keys) {
		return ""
	}
	key := m.keys[m.cursor]
	switch key.Type {
	case config.TypeObject:
		return m.renderObjectPanel()
	case config.TypeEnum:
		return m.renderEnumPanel()
	default:
		return m.renderTextEditPanel()
	}
}

// renderTextEditPanel renders the free-text edit overlay for string/bool/int keys.
func (m *ConfigModel) renderTextEditPanel() string {
	if m.cursor >= len(m.keys) {
		return ""
	}
	key := m.keys[m.cursor]

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Primary)
	labelStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)

	var b strings.Builder
	b.WriteString("  ")
	b.WriteString(headerStyle.Render("Edit: " + key.Key))
	b.WriteString("\n")

	// Scope selector
	b.WriteString("  ")
	b.WriteString(labelStyle.Render("Scope: "))
	for i, scope := range m.editScopes {
		name := string(scope)
		if i == m.editScopeIdx {
			b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(m.theme.Primary).Render("[" + name + "]"))
		} else {
			b.WriteString(lipgloss.NewStyle().Foreground(m.theme.Muted).Render(" " + name + " "))
		}
		b.WriteString(" ")
	}
	b.WriteString("\n")

	// Value input
	b.WriteString("  ")
	b.WriteString(labelStyle.Render("Value: "))
	b.WriteString(m.editValue)
	b.WriteString(lipgloss.NewStyle().Foreground(m.theme.Primary).Render("_"))
	b.WriteString("\n")

	// Help
	b.WriteString("  ")
	help := fmt.Sprintf(
		"%s save  %s cancel  %s scope",
		lipgloss.NewStyle().Foreground(m.theme.Muted).Bold(true).Render(keyEnter),
		lipgloss.NewStyle().Foreground(m.theme.Muted).Bold(true).Render(keyEsc),
		lipgloss.NewStyle().Foreground(m.theme.Muted).Bold(true).Render(keyTab),
	)
	b.WriteString(help)

	return b.String()
}

// renderEnumPanel renders the arrow-key selector for TypeEnum keys.
func (m *ConfigModel) renderEnumPanel() string {
	if m.cursor >= len(m.keys) {
		return ""
	}
	key := m.keys[m.cursor]

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Primary)
	labelStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)
	selectedStyle := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true)

	var b strings.Builder
	b.WriteString("  ")
	b.WriteString(headerStyle.Render("Edit: " + key.Key))
	b.WriteString("\n")

	// Scope selector
	b.WriteString("  ")
	b.WriteString(labelStyle.Render("Scope: "))
	for i, scope := range m.editScopes {
		name := string(scope)
		if i == m.editScopeIdx {
			b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(m.theme.Primary).Render("[" + name + "]"))
		} else {
			b.WriteString(lipgloss.NewStyle().Foreground(m.theme.Muted).Render(" " + name + " "))
		}
		b.WriteString(" ")
	}
	b.WriteString("\n\n")

	// Enum values list
	for i, val := range key.EnumValues {
		b.WriteString("  ")
		if i == m.editEnumIdx {
			b.WriteString(selectedStyle.Render("● " + val))
		} else {
			b.WriteString(labelStyle.Render("○ " + val))
		}
		b.WriteString("\n")
	}

	// Help
	b.WriteString("\n  ")
	help := fmt.Sprintf(
		"%s/%s select  %s save  %s cancel  %s scope",
		lipgloss.NewStyle().Foreground(m.theme.Muted).Bold(true).Render("↑"),
		lipgloss.NewStyle().Foreground(m.theme.Muted).Bold(true).Render("↓"),
		lipgloss.NewStyle().Foreground(m.theme.Muted).Bold(true).Render(keyEnter),
		lipgloss.NewStyle().Foreground(m.theme.Muted).Bold(true).Render(keyEsc),
		lipgloss.NewStyle().Foreground(m.theme.Muted).Bold(true).Render(keyTab),
	)
	b.WriteString(help)

	return b.String()
}

// renderObjectPanel renders a read-only JSON viewer for TypeObject keys.
func (m *ConfigModel) renderObjectPanel() string {
	if m.cursor >= len(m.keys) {
		return ""
	}
	key := m.keys[m.cursor]

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Secondary)
	labelStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)
	mutedStyle := lipgloss.NewStyle().Foreground(m.theme.Muted).Italic(true)

	var b strings.Builder
	b.WriteString("  ")
	b.WriteString(headerStyle.Render("View: " + key.Key))
	b.WriteString("\n")

	// Show effective value as pretty-printed JSON
	cv := m.snapshot.Values[key.Key]
	if cv != nil && !cv.IsDefault {
		if obj, ok := cv.EffectiveValue.(map[string]interface{}); ok {
			data, err := json.MarshalIndent(obj, "    ", "  ")
			if err == nil {
				b.WriteString("\n")
				b.WriteString("    ")
				b.WriteString(string(data))
				b.WriteString("\n")
			}
		}
	} else {
		b.WriteString("  ")
		b.WriteString(mutedStyle.Render("(using default — not explicitly set)"))
		b.WriteString("\n")
	}

	b.WriteString("\n  ")
	b.WriteString(mutedStyle.Render("Object values cannot be edited in the TUI."))
	b.WriteString("\n  ")
	b.WriteString(labelStyle.Render("Edit the settings file directly to modify this value."))
	b.WriteString("\n  ")
	b.WriteString(lipgloss.NewStyle().Foreground(m.theme.Muted).Bold(true).Render(keyEsc))
	b.WriteString(" close")

	return b.String()
}

// renderArrayPanel renders the per-item list editor for TypeStringArray keys.
func (m *ConfigModel) renderArrayPanel() string {
	if m.cursor >= len(m.keys) {
		return ""
	}
	key := m.keys[m.cursor]

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Primary)
	labelStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)
	selectedStyle := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)

	var b strings.Builder
	b.WriteString("  ")
	b.WriteString(headerStyle.Render("Edit array: " + key.Key))
	b.WriteString("\n")

	// Scope selector
	b.WriteString("  ")
	b.WriteString(labelStyle.Render("Scope: "))
	for i, scope := range m.editScopes {
		name := string(scope)
		if i == m.editScopeIdx {
			b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(m.theme.Primary).Render("[" + name + "]"))
		} else {
			b.WriteString(lipgloss.NewStyle().Foreground(m.theme.Muted).Render(" " + name + " "))
		}
		b.WriteString(" ")
	}
	b.WriteString("\n")

	// Items list
	fmt.Fprintf(&b, "\n  ")
	b.WriteString(labelStyle.Render(fmt.Sprintf("Items in this scope (%d):", len(m.arrayItems))))
	b.WriteString("\n")

	if len(m.arrayItems) == 0 {
		b.WriteString("    ")
		b.WriteString(mutedStyle.Render("(none)"))
		b.WriteString("\n")
	} else {
		for i, item := range m.arrayItems {
			s := fmt.Sprintf("%v", item)
			b.WriteString("  ")
			if i == m.arrayCursor {
				b.WriteString(selectedStyle.Render("> " + s))
			} else {
				b.WriteString("  " + s)
			}
			b.WriteString("\n")
		}
	}

	// Add-item input or navigation help
	if m.arrayAddMode {
		b.WriteString("\n  ")
		b.WriteString(labelStyle.Render("New item: "))
		b.WriteString(m.arrayAddValue)
		b.WriteString(lipgloss.NewStyle().Foreground(m.theme.Primary).Render("_"))
		b.WriteString("\n  ")
		help := fmt.Sprintf(
			"%s add  %s cancel",
			lipgloss.NewStyle().Foreground(m.theme.Muted).Bold(true).Render(keyEnter),
			lipgloss.NewStyle().Foreground(m.theme.Muted).Bold(true).Render(keyEsc),
		)
		b.WriteString(help)
	} else {
		b.WriteString("\n  ")
		help := fmt.Sprintf(
			"%s add  %s remove  %s/%s navigate  %s scope  %s done",
			lipgloss.NewStyle().Foreground(m.theme.Muted).Bold(true).Render("a"),
			lipgloss.NewStyle().Foreground(m.theme.Muted).Bold(true).Render("d"),
			lipgloss.NewStyle().Foreground(m.theme.Muted).Bold(true).Render("j"),
			lipgloss.NewStyle().Foreground(m.theme.Muted).Bold(true).Render("k"),
			lipgloss.NewStyle().Foreground(m.theme.Muted).Bold(true).Render(keyTab),
			lipgloss.NewStyle().Foreground(m.theme.Muted).Bold(true).Render(keyEsc),
		)
		b.WriteString(help)
	}

	return b.String()
}

// renderDeletePanel renders the inline delete confirmation overlay.
func (m *ConfigModel) renderDeletePanel() string {
	if m.cursor >= len(m.keys) {
		return ""
	}
	key := m.keys[m.cursor]

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Error)
	labelStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)

	var b strings.Builder
	b.WriteString("  ")
	b.WriteString(headerStyle.Render("Delete: " + key.Key))
	b.WriteString("\n")

	// Scope selector
	b.WriteString("  ")
	b.WriteString(labelStyle.Render("From scope: "))
	for i, scope := range m.editScopes {
		name := string(scope)
		if i == m.editScopeIdx {
			b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(m.theme.Error).Render("[" + name + "]"))
		} else {
			b.WriteString(lipgloss.NewStyle().Foreground(m.theme.Muted).Render(" " + name + " "))
		}
		b.WriteString(" ")
	}
	b.WriteString("\n")

	// Show current value in selected scope
	b.WriteString("  ")
	b.WriteString(labelStyle.Render("Value: "))
	if cv, ok := m.snapshot.Values[key.Key]; ok && cv != nil {
		if val, hasLayer := cv.LayerValues[m.editScope]; hasLayer {
			b.WriteString(configFormatValue(val))
		} else {
			b.WriteString(lipgloss.NewStyle().Foreground(m.theme.Muted).Render("(not set in this scope)"))
		}
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(m.theme.Muted).Render("(not set in this scope)"))
	}
	b.WriteString("\n")

	// Help
	b.WriteString("  ")
	help := fmt.Sprintf(
		"%s confirm  %s cancel  %s scope",
		lipgloss.NewStyle().Foreground(m.theme.Error).Bold(true).Render(keyEnter),
		lipgloss.NewStyle().Foreground(m.theme.Muted).Bold(true).Render(keyEsc),
		lipgloss.NewStyle().Foreground(m.theme.Muted).Bold(true).Render(keyTab),
	)
	b.WriteString(help)

	return b.String()
}

// renderFooter renders the help footer bar.
func (m *ConfigModel) renderFooter() string {
	if m.filterMode {
		return fmt.Sprintf(
			"  /%s  %s apply  %s clear",
			lipgloss.NewStyle().Foreground(m.theme.Primary).Render(m.filter+"_"),
			m.theme.HelpKey.Render(keyEnter),
			m.theme.HelpKey.Render(keyEsc),
		)
	}

	parts := []string{ //nolint:gocritic // appendCombine: each entry uses method calls evaluated at call time
		m.theme.HelpKey.Render("/") + m.theme.HelpDesc.Render(" filter"),
		m.theme.HelpKey.Render("e") + m.theme.HelpDesc.Render(" edit"),
		m.theme.HelpKey.Render("d") + m.theme.HelpDesc.Render(" delete"),
		m.theme.HelpKey.Render("j/k") + m.theme.HelpDesc.Render(" scroll"),
		m.theme.HelpKey.Render(keyTab) + m.theme.HelpDesc.Render(" next tab"),
		m.theme.HelpKey.Render("q") + m.theme.HelpDesc.Render(" back"),
	}
	return "  " + strings.Join(parts, "  ")
}

// configFormatValue converts an interface{} to a display string for the TUI.
func configFormatValue(v interface{}) string {
	if v == nil {
		return "(unset)"
	}
	switch val := v.(type) {
	case string:
		if len(val) > 60 {
			return val[:57] + "..."
		}
		return val
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(val)
	case float64:
		return fmt.Sprintf("%g", val)
	case []interface{}:
		if len(val) == 0 {
			return "[]"
		}
		items := make([]string, 0, len(val))
		for _, item := range val {
			items = append(items, fmt.Sprintf("%v", item))
		}
		joined := "[" + strings.Join(items, ", ") + "]"
		if len(joined) > 60 {
			return fmt.Sprintf("[%d items]", len(val))
		}
		return joined
	case map[string]interface{}:
		return "{object}"
	default:
		return fmt.Sprintf("%v", val)
	}
}
