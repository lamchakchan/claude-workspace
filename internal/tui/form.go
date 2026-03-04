package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// PathType controls path autocomplete behavior and validation for a form field.
type PathType int

const (
	PathNone PathType = iota // no path autocomplete
	PathDir                  // directories only
	PathFile                 // files only
	PathAny                  // files and directories
)

// FormField defines a single field in a form.
type FormField struct {
	Label       string
	Placeholder string
	Password    bool
	Required    bool
	PathType    PathType // enables path autocomplete with type filtering and validation
	Choices     []string // if non-nil, renders as a cycle picker instead of a text input
}

// isPath returns true if the field has any path autocomplete behavior.
func (f FormField) isPath() bool { return f.PathType != PathNone }

// FormResult is sent when the user submits or cancels the form.
type FormResult struct {
	Values    []string // values indexed by field position
	Cancelled bool
}

// pathSuggestionsMsg carries directory completion suggestions for a path field.
type pathSuggestionsMsg struct {
	fieldIndex  int
	suggestions []string
}

// FormModel is a generic multi-field text input form component.
type FormModel struct {
	Title     string
	Fields    []FormField
	inputs    []textinput.Model
	choiceIdx []int // selected index per field; only meaningful when Field.Choices != nil
	cursor    int   // focused field index
	done      bool
	theme     *Theme
}

// NewForm creates a new form with the given fields.
func NewForm(title string, fields []FormField, theme *Theme) *FormModel {
	inputs := make([]textinput.Model, len(fields))
	choiceIdx := make([]int, len(fields))
	for i, f := range fields {
		ti := textinput.New()
		ti.Placeholder = f.Placeholder
		if f.Password {
			ti.EchoMode = textinput.EchoPassword
		}
		if f.isPath() {
			ti.ShowSuggestions = true
		}
		if i == 0 && len(f.Choices) == 0 {
			ti.Focus()
		}
		inputs[i] = ti
	}

	return &FormModel{
		Title:     title,
		Fields:    fields,
		inputs:    inputs,
		choiceIdx: choiceIdx,
		theme:     theme,
	}
}

// Values returns the current values of all fields.
func (m *FormModel) Values() []string {
	vals := make([]string, len(m.Fields))
	for i, f := range m.Fields {
		if len(f.Choices) > 0 {
			vals[i] = f.Choices[m.choiceIdx[i]]
		} else {
			vals[i] = m.inputs[i].Value()
		}
	}
	return vals
}

// SetValue sets the text input value at field index i.
func (m *FormModel) SetValue(i int, value string) {
	if i >= 0 && i < len(m.inputs) {
		m.inputs[i].SetValue(value)
	}
}

// SetChoice selects the choice matching value at field index i.
func (m *FormModel) SetChoice(i int, value string) {
	if i >= 0 && i < len(m.Fields) && len(m.Fields[i].Choices) > 0 {
		for j, c := range m.Fields[i].Choices {
			if c == value {
				m.choiceIdx[i] = j
				break
			}
		}
	}
}

func (m *FormModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	if len(m.Fields) > 0 && len(m.Fields[0].Choices) == 0 {
		cmds = append(cmds, textinput.Blink)
	}
	if cmd := m.pathSuggestionsForCurrent(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

func (m *FormModel) Update(msg tea.Msg) (*FormModel, tea.Cmd) {
	// Handle path suggestion responses
	if msg, ok := msg.(pathSuggestionsMsg); ok {
		if msg.fieldIndex >= 0 && msg.fieldIndex < len(m.inputs) {
			m.inputs[msg.fieldIndex].SetSuggestions(msg.suggestions)
		}
		return m, nil
	}

	if msg, ok := msg.(tea.KeyPressMsg); ok {
		return m.handleFormKey(msg)
	}

	// Select fields handle all input via key events; don't route other messages to them.
	if len(m.Fields[m.cursor].Choices) > 0 {
		return m, nil
	}

	var cmd tea.Cmd
	prevValue := m.inputs[m.cursor].Value()
	m.inputs[m.cursor], cmd = m.inputs[m.cursor].Update(msg)

	// If this is a path field and the value changed, refresh suggestions
	if m.Fields[m.cursor].isPath() && m.inputs[m.cursor].Value() != prevValue {
		sugCmd := readDirSuggestions(m.cursor, m.inputs[m.cursor].Value(), m.Fields[m.cursor].PathType)
		return m, tea.Batch(cmd, sugCmd)
	}

	return m, cmd
}

// handleFormKey processes key presses in the form.
func (m *FormModel) handleFormKey(msg tea.KeyPressMsg) (*FormModel, tea.Cmd) {
	switch msg.String() {
	case keyCtrlC, keyEsc:
		m.done = true
		return m, func() tea.Msg { return FormResult{Cancelled: true} }

	case keyLeft:
		if len(m.Fields[m.cursor].Choices) > 0 {
			n := len(m.Fields[m.cursor].Choices)
			m.choiceIdx[m.cursor] = (m.choiceIdx[m.cursor] - 1 + n) % n
			return m, nil
		}
		var cmd tea.Cmd
		m.inputs[m.cursor], cmd = m.inputs[m.cursor].Update(msg)
		return m, cmd

	case keyRight:
		if len(m.Fields[m.cursor].Choices) > 0 {
			n := len(m.Fields[m.cursor].Choices)
			m.choiceIdx[m.cursor] = (m.choiceIdx[m.cursor] + 1) % n
			return m, nil
		}
		var cmd tea.Cmd
		m.inputs[m.cursor], cmd = m.inputs[m.cursor].Update(msg)
		return m, cmd

	case keyTab:
		return m.handleTabKey(msg)

	case keyDown:
		return m.handleDownKey(msg)

	case keyUp:
		return m.handleUpKey(msg)

	case keyShiftTab:
		m.focusPrev()
		cmd := m.pathSuggestionsForCurrent()
		return m, cmd

	case keyEnter:
		if m.cursor == len(m.Fields)-1 {
			return m.submit()
		}
		if len(m.Fields[m.cursor].Choices) == 0 {
			m.inputs[m.cursor].Blur()
		}
		m.cursor++
		if len(m.Fields[m.cursor].Choices) == 0 {
			m.inputs[m.cursor].Focus()
		}
		cmd := m.pathSuggestionsForCurrent()
		return m, cmd

	default:
		if len(m.Fields[m.cursor].Choices) == 0 {
			var cmd tea.Cmd
			m.inputs[m.cursor], cmd = m.inputs[m.cursor].Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

// handleTabKey handles tab in the form (path suggestion accept or field advance).
func (m *FormModel) handleTabKey(msg tea.KeyPressMsg) (*FormModel, tea.Cmd) {
	if m.Fields[m.cursor].isPath() && m.inputs[m.cursor].CurrentSuggestion() != "" {
		var cmd tea.Cmd
		m.inputs[m.cursor], cmd = m.inputs[m.cursor].Update(msg)
		return m, tea.Batch(cmd, readDirSuggestions(m.cursor, m.inputs[m.cursor].Value(), m.Fields[m.cursor].PathType))
	}
	m.focusNext()
	sugCmd := m.pathSuggestionsForCurrent()
	return m, sugCmd
}

// handleDownKey handles down arrow in the form (suggestion cycling or field advance).
func (m *FormModel) handleDownKey(msg tea.KeyPressMsg) (*FormModel, tea.Cmd) {
	if m.Fields[m.cursor].isPath() && len(m.inputs[m.cursor].MatchedSuggestions()) > 0 {
		var cmd tea.Cmd
		m.inputs[m.cursor], cmd = m.inputs[m.cursor].Update(msg)
		return m, cmd
	}
	m.focusNext()
	sugCmd := m.pathSuggestionsForCurrent()
	return m, sugCmd
}

// handleUpKey handles up arrow in the form (suggestion cycling or field retreat).
func (m *FormModel) handleUpKey(msg tea.KeyPressMsg) (*FormModel, tea.Cmd) {
	if m.Fields[m.cursor].isPath() && len(m.inputs[m.cursor].MatchedSuggestions()) > 0 {
		var cmd tea.Cmd
		m.inputs[m.cursor], cmd = m.inputs[m.cursor].Update(msg)
		return m, cmd
	}
	m.focusPrev()
	sugCmd := m.pathSuggestionsForCurrent()
	return m, sugCmd
}

// focusNext moves focus to the next input field.
func (m *FormModel) focusNext() {
	if len(m.Fields[m.cursor].Choices) == 0 {
		m.inputs[m.cursor].Blur()
	}
	m.cursor = (m.cursor + 1) % len(m.Fields)
	if len(m.Fields[m.cursor].Choices) == 0 {
		m.inputs[m.cursor].Focus()
	}
}

// focusPrev moves focus to the previous input field.
func (m *FormModel) focusPrev() {
	if len(m.Fields[m.cursor].Choices) == 0 {
		m.inputs[m.cursor].Blur()
	}
	m.cursor = (m.cursor - 1 + len(m.Fields)) % len(m.Fields)
	if len(m.Fields[m.cursor].Choices) == 0 {
		m.inputs[m.cursor].Focus()
	}
}

func (m *FormModel) submit() (*FormModel, tea.Cmd) {
	for i, f := range m.Fields {
		if f.Required && len(f.Choices) == 0 && strings.TrimSpace(m.inputs[i].Value()) == "" {
			m.focusField(i)
			return m, nil
		}
	}

	// Validate path fields: check existence and type
	for i, f := range m.Fields {
		if f.PathType == PathNone {
			continue
		}
		val := strings.TrimSpace(m.inputs[i].Value())
		if val == "" {
			continue // optional empty path is fine
		}
		expanded, _ := expandTilde(val)
		absPath, err := filepath.Abs(expanded)
		if err != nil {
			m.focusField(i)
			return m, nil
		}
		info, err := os.Stat(absPath)
		if err != nil {
			m.focusField(i)
			return m, nil
		}
		if f.PathType == PathDir && !info.IsDir() {
			m.focusField(i)
			return m, nil
		}
		if f.PathType == PathFile && info.IsDir() {
			m.focusField(i)
			return m, nil
		}
	}

	vals := m.Values()
	m.done = true
	return m, func() tea.Msg { return FormResult{Values: vals} }
}

// focusField moves focus to field i, blurring the current field.
func (m *FormModel) focusField(i int) {
	if len(m.Fields[m.cursor].Choices) == 0 {
		m.inputs[m.cursor].Blur()
	}
	m.cursor = i
	if len(m.Fields[i].Choices) == 0 {
		m.inputs[i].Focus()
	}
}

func (m *FormModel) View() string {
	var b strings.Builder

	b.WriteString(m.theme.Title.Render(m.Title))
	b.WriteString("\n\n")

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Secondary)

	activeChoiceStyle := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Primary)
	mutedChoiceStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)

	for i, f := range m.Fields {
		label := labelStyle.Render(f.Label)
		if f.Required {
			label += lipgloss.NewStyle().Foreground(m.theme.Error).Render(" *")
		}
		b.WriteString(label + "\n")

		if len(f.Choices) > 0 {
			// Render as a cycle picker: < choice >
			choice := f.Choices[m.choiceIdx[i]]
			if i == m.cursor {
				b.WriteString("  " + activeChoiceStyle.Render("< "+choice+" >") + "\n")
			} else {
				b.WriteString("  " + mutedChoiceStyle.Render("  "+choice+"  ") + "\n")
			}
		} else {
			b.WriteString("  " + m.inputs[i].View() + "\n")

			// Render suggestion dropdown for focused path fields
			if i == m.cursor && f.isPath() {
				matches := m.inputs[i].MatchedSuggestions()
				if len(matches) > 0 {
					selectedIdx := m.inputs[i].CurrentSuggestionIndex()
					b.WriteString(m.renderSuggestionList(matches, selectedIdx))
				}
			}
		}

		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Build help text based on the focused field type
	switch {
	case len(m.Fields[m.cursor].Choices) > 0:
		help := m.theme.HelpKey.Render("←/→") + " " + m.theme.HelpDesc.Render("cycle") + "  " +
			m.theme.HelpKey.Render("tab") + " " + m.theme.HelpDesc.Render("next field") + "  " +
			m.theme.HelpKey.Render(keyEnter) + " " + m.theme.HelpDesc.Render("submit") + "  " +
			m.theme.HelpKey.Render("esc") + " " + m.theme.HelpDesc.Render("cancel")
		b.WriteString(help)
	case m.Fields[m.cursor].isPath():
		help := m.theme.HelpKey.Render("↑/↓") + " " + m.theme.HelpDesc.Render("select") + "  " +
			m.theme.HelpKey.Render("tab") + " " + m.theme.HelpDesc.Render("accept") + "  " +
			m.theme.HelpKey.Render(keyEnter) + " " + m.theme.HelpDesc.Render("submit") + "  " +
			m.theme.HelpKey.Render("esc") + " " + m.theme.HelpDesc.Render("cancel")
		b.WriteString(help)
	default:
		help := m.theme.HelpKey.Render("tab") + " " + m.theme.HelpDesc.Render("next field") + "  " +
			m.theme.HelpKey.Render(keyEnter) + " " + m.theme.HelpDesc.Render("submit") + "  " +
			m.theme.HelpKey.Render("esc") + " " + m.theme.HelpDesc.Render("cancel")
		b.WriteString(help)
	}

	return b.String()
}

const maxVisibleSuggestions = 8 // max items shown in suggestion dropdown

// renderSuggestionList renders a dropdown-style list of path suggestions below
// the input field, highlighting the currently selected item.
func (m *FormModel) renderSuggestionList(matches []string, selectedIdx int) string {
	var b strings.Builder

	total := len(matches)

	// Determine visible window around the selected index
	start := 0
	end := total
	if total > maxVisibleSuggestions {
		// Center the window on the selected item
		half := maxVisibleSuggestions / 2
		start = selectedIdx - half
		if start < 0 {
			start = 0
		}
		end = start + maxVisibleSuggestions
		if end > total {
			end = total
			start = end - maxVisibleSuggestions
		}
	}

	selectedStyle := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)
	countStyle := lipgloss.NewStyle().Foreground(m.theme.Muted).Italic(true)

	// Show scroll indicator at top if truncated
	if start > 0 {
		b.WriteString("    " + countStyle.Render("...") + "\n")
	}

	for i := start; i < end; i++ {
		entry := matches[i]
		if i == selectedIdx {
			b.WriteString("  " + selectedStyle.Render("> "+entry) + "\n")
		} else {
			b.WriteString("    " + normalStyle.Render(entry) + "\n")
		}
	}

	// Show scroll indicator at bottom if truncated
	if end < total {
		b.WriteString("    " + countStyle.Render("...") + "\n")
	}

	// Show count when there are many matches
	if total > maxVisibleSuggestions {
		b.WriteString("    " + countStyle.Render(fmt.Sprintf("(%d/%d)", selectedIdx+1, total)) + "\n")
	}

	return b.String()
}

// pathSuggestionsForCurrent returns a Cmd that loads path suggestions for the
// currently focused field, or nil if the field is not a path field.
func (m *FormModel) pathSuggestionsForCurrent() tea.Cmd {
	if m.cursor >= 0 && m.cursor < len(m.Fields) && m.Fields[m.cursor].isPath() {
		return readDirSuggestions(m.cursor, m.inputs[m.cursor].Value(), m.Fields[m.cursor].PathType)
	}
	return nil
}

// readDirSuggestions returns a Cmd that reads directory entries and produces
// path suggestions matching the current input value.
func readDirSuggestions(fieldIndex int, value string, pt PathType) tea.Cmd {
	return func() tea.Msg {
		suggestions := listPathSuggestions(value, pt)
		return pathSuggestionsMsg{
			fieldIndex:  fieldIndex,
			suggestions: suggestions,
		}
	}
}

// listPathSuggestions generates file/directory completion suggestions for the
// given path prefix. It expands ~ to the home directory, hides dotfiles unless
// the user explicitly types a dot, and appends / to directories.
func listPathSuggestions(prefix string, pt PathType) []string {
	if prefix == "" {
		return listCurrentDir(pt)
	}

	expanded, tilde := expandTilde(prefix)
	if expanded == "" {
		return nil
	}

	dir, partial := splitDirPartial(prefix, expanded)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	return buildSuggestions(entries, prefix, partial, tilde, pt)
}

// expandTilde expands ~ to the home directory. Returns the expanded path and
// whether tilde expansion was performed.
func expandTilde(prefix string) (string, bool) {
	if strings.HasPrefix(prefix, "~/") || prefix == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", false
		}
		return filepath.Join(home, prefix[1:]), true
	}
	return prefix, false
}

// splitDirPartial splits the expanded path into the directory to read and the
// partial filename to match against.
func splitDirPartial(prefix, expanded string) (string, string) {
	if strings.HasSuffix(prefix, "/") || strings.HasSuffix(prefix, string(filepath.Separator)) {
		return expanded, ""
	}
	return filepath.Dir(expanded), filepath.Base(expanded)
}

// buildSuggestions filters directory entries and builds suggestion strings.
func buildSuggestions(entries []os.DirEntry, prefix, partial string, tilde bool, pt PathType) []string {
	showDotfiles := strings.HasPrefix(partial, ".")
	endsWithSep := strings.HasSuffix(prefix, "/") || strings.HasSuffix(prefix, string(filepath.Separator))

	suggestions := make([]string, 0, len(entries))
	for _, e := range entries {
		if pt == PathDir && !e.IsDir() {
			continue
		}
		if pt == PathFile && e.IsDir() {
			continue
		}
		name := e.Name()
		if !showDotfiles && strings.HasPrefix(name, ".") {
			continue
		}
		if partial != "" && !strings.HasPrefix(name, partial) {
			continue
		}

		suggestion := buildOneSuggestion(prefix, name, endsWithSep, tilde)
		if e.IsDir() {
			suggestion += "/"
		}
		suggestions = append(suggestions, suggestion)
	}
	return suggestions
}

// listCurrentDir returns suggestions for the current directory, including
// navigation shortcuts ../ and ~/. Dotfiles are hidden.
func listCurrentDir(pt PathType) []string {
	entries, err := os.ReadDir(".")
	if err != nil {
		return nil
	}
	suggestions := make([]string, 0, len(entries)+2)
	// Navigation shortcuts are directories — include unless filtering to files only
	if pt != PathFile {
		suggestions = append(suggestions, "../", "~/")
	}
	for _, e := range entries {
		if pt == PathDir && !e.IsDir() {
			continue
		}
		if pt == PathFile && e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if e.IsDir() {
			name += "/"
		}
		suggestions = append(suggestions, name)
	}
	return suggestions
}

// buildOneSuggestion constructs the full suggestion path for one entry.
func buildOneSuggestion(prefix, name string, endsWithSep, tilde bool) string {
	if endsWithSep {
		return prefix + name
	}
	suggestion := filepath.Join(filepath.Dir(prefix), name)
	if tilde {
		home, _ := os.UserHomeDir()
		if strings.HasPrefix(suggestion, home) {
			suggestion = "~" + suggestion[len(home):]
		}
	}
	return suggestion
}
