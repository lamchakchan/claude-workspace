package tui

import (
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
	theme        *Theme
	categories   []config.Category
	activeTab    int
	keys         []config.ConfigKey // keys in active category (filtered)
	allKeys      []config.ConfigKey // keys in active category (unfiltered)
	cursor       int
	scroll       int
	filter       string
	filterMode   bool
	snapshot     *config.ConfigSnapshot
	registry     *config.Registry
	loading      bool
	err          string
	width        int
	height       int
	editMode     bool
	editValue    string
	editScope    config.ConfigScope
	editScopes   []config.ConfigScope
	editScopeIdx int
	deleteMode   bool
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
		// Reload snapshot
		return m, func() tea.Msg {
			snap, err := config.ReadAll()
			if err != nil {
				return configErrorMsg{err: err}
			}
			return configLoadedMsg{snap: snap, reg: config.GlobalRegistry()}
		}

	case tea.KeyPressMsg:
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
		if len(m.keys) > 0 {
			key := m.keys[m.cursor]
			if key.Type != config.TypeObject && !strings.HasPrefix(key.Key, "file:") {
				m.editMode = true
				m.editScopeIdx = 0
				m.editScope = m.editScopes[0]
				// Pre-fill with current effective value
				if cv, ok := m.snapshot.Values[key.Key]; ok && cv != nil && !cv.IsDefault {
					m.editValue = configFormatValue(cv.EffectiveValue)
				} else {
					m.editValue = ""
				}
			}
		}
	case "d":
		if len(m.keys) > 0 {
			key := m.keys[m.cursor]
			if !strings.HasPrefix(key.Key, "file:") && key.Category != config.CatFiles {
				m.deleteMode = true
				m.editScopeIdx = 0
				m.editScope = m.editScopes[0]
			}
		}
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

// handleEditKey handles key events in edit mode.
func (m *ConfigModel) handleEditKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc:
		m.editMode = false
		m.editValue = ""
	case keyEnter:
		if len(m.keys) == 0 {
			return m, nil
		}
		key := m.keys[m.cursor]
		val := m.editValue
		scope := m.editScope
		home := ""
		cwd := ""
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
		home := ""
		cwd := ""
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

	// Edit overlay
	if m.editMode && len(m.keys) > 0 {
		b.WriteString("\n")
		b.WriteString(m.renderEditPanel())
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

	// Layer values
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

// renderEditPanel renders the inline edit overlay.
func (m *ConfigModel) renderEditPanel() string {
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

	// Edit help
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
