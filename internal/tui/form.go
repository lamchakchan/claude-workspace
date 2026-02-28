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

// FormField defines a single field in a form.
type FormField struct {
	Label       string
	Placeholder string
	Password    bool
	Required    bool
	IsPath      bool // enables path autocomplete with tab completion
}

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
	Title  string
	Fields []FormField
	inputs []textinput.Model
	cursor int // focused field index
	done   bool
	theme  *Theme
}

// NewForm creates a new form with the given fields.
func NewForm(title string, fields []FormField, theme *Theme) *FormModel {
	inputs := make([]textinput.Model, len(fields))
	for i, f := range fields {
		ti := textinput.New()
		ti.Placeholder = f.Placeholder
		if f.Password {
			ti.EchoMode = textinput.EchoPassword
		}
		if f.IsPath {
			ti.ShowSuggestions = true
		}
		if i == 0 {
			ti.Focus()
		}
		inputs[i] = ti
	}

	return &FormModel{
		Title:  title,
		Fields: fields,
		inputs: inputs,
		theme:  theme,
	}
}

// Values returns the current values of all fields.
func (m *FormModel) Values() []string {
	vals := make([]string, len(m.inputs))
	for i := range m.inputs {
		vals[i] = m.inputs[i].Value()
	}
	return vals
}

func (m *FormModel) Init() tea.Cmd {
	if len(m.inputs) > 0 {
		return textinput.Blink
	}
	return nil
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

	var cmd tea.Cmd
	prevValue := m.inputs[m.cursor].Value()
	m.inputs[m.cursor], cmd = m.inputs[m.cursor].Update(msg)

	// If this is a path field and the value changed, refresh suggestions
	if m.Fields[m.cursor].IsPath && m.inputs[m.cursor].Value() != prevValue {
		sugCmd := readDirSuggestions(m.cursor, m.inputs[m.cursor].Value())
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

	case keyTab:
		return m.handleTabKey(msg)

	case keyDown:
		return m.handleDownKey(msg)

	case keyUp:
		return m.handleUpKey(msg)

	case keyShiftTab:
		m.focusPrev()

	case keyEnter:
		if m.cursor == len(m.inputs)-1 {
			return m.submit()
		}
		m.inputs[m.cursor].Blur()
		m.cursor++
		m.inputs[m.cursor].Focus()
	}
	return m, nil
}

// handleTabKey handles tab in the form (path suggestion accept or field advance).
func (m *FormModel) handleTabKey(msg tea.KeyPressMsg) (*FormModel, tea.Cmd) {
	if m.Fields[m.cursor].IsPath && m.inputs[m.cursor].CurrentSuggestion() != "" {
		var cmd tea.Cmd
		m.inputs[m.cursor], cmd = m.inputs[m.cursor].Update(msg)
		return m, tea.Batch(cmd, readDirSuggestions(m.cursor, m.inputs[m.cursor].Value()))
	}
	m.focusNext()
	return m, nil
}

// handleDownKey handles down arrow in the form (suggestion cycling or field advance).
func (m *FormModel) handleDownKey(msg tea.KeyPressMsg) (*FormModel, tea.Cmd) {
	if m.Fields[m.cursor].IsPath && len(m.inputs[m.cursor].MatchedSuggestions()) > 0 {
		var cmd tea.Cmd
		m.inputs[m.cursor], cmd = m.inputs[m.cursor].Update(msg)
		return m, cmd
	}
	m.focusNext()
	return m, nil
}

// handleUpKey handles up arrow in the form (suggestion cycling or field retreat).
func (m *FormModel) handleUpKey(msg tea.KeyPressMsg) (*FormModel, tea.Cmd) {
	if m.Fields[m.cursor].IsPath && len(m.inputs[m.cursor].MatchedSuggestions()) > 0 {
		var cmd tea.Cmd
		m.inputs[m.cursor], cmd = m.inputs[m.cursor].Update(msg)
		return m, cmd
	}
	m.focusPrev()
	return m, nil
}

// focusNext moves focus to the next input field.
func (m *FormModel) focusNext() {
	m.inputs[m.cursor].Blur()
	m.cursor = (m.cursor + 1) % len(m.inputs)
	m.inputs[m.cursor].Focus()
}

// focusPrev moves focus to the previous input field.
func (m *FormModel) focusPrev() {
	m.inputs[m.cursor].Blur()
	m.cursor = (m.cursor - 1 + len(m.inputs)) % len(m.inputs)
	m.inputs[m.cursor].Focus()
}

func (m *FormModel) submit() (*FormModel, tea.Cmd) {
	for i, f := range m.Fields {
		if f.Required && strings.TrimSpace(m.inputs[i].Value()) == "" {
			m.inputs[m.cursor].Blur()
			m.cursor = i
			m.inputs[i].Focus()
			return m, nil
		}
	}

	vals := m.Values()
	m.done = true
	return m, func() tea.Msg { return FormResult{Values: vals} }
}

func (m *FormModel) View() string {
	var b strings.Builder

	b.WriteString(m.theme.Title.Render(m.Title))
	b.WriteString("\n\n")

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Secondary)

	for i, f := range m.Fields {
		label := labelStyle.Render(f.Label)
		if f.Required {
			label += lipgloss.NewStyle().Foreground(m.theme.Error).Render(" *")
		}
		b.WriteString(label + "\n")
		b.WriteString("  " + m.inputs[i].View() + "\n")

		// Render suggestion dropdown for focused path fields
		if i == m.cursor && f.IsPath {
			matches := m.inputs[i].MatchedSuggestions()
			if len(matches) > 0 {
				selectedIdx := m.inputs[i].CurrentSuggestionIndex()
				b.WriteString(m.renderSuggestionList(matches, selectedIdx))
			}
		}

		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Build help text with path-aware hints
	if m.Fields[m.cursor].IsPath {
		help := m.theme.HelpKey.Render("↑/↓") + " " + m.theme.HelpDesc.Render("select") + "  " +
			m.theme.HelpKey.Render("tab") + " " + m.theme.HelpDesc.Render("accept") + "  " +
			m.theme.HelpKey.Render(keyEnter) + " " + m.theme.HelpDesc.Render("submit") + "  " +
			m.theme.HelpKey.Render("esc") + " " + m.theme.HelpDesc.Render("cancel")
		b.WriteString(help)
	} else {
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

// readDirSuggestions returns a Cmd that reads directory entries and produces
// path suggestions matching the current input value.
func readDirSuggestions(fieldIndex int, value string) tea.Cmd {
	return func() tea.Msg {
		suggestions := listPathSuggestions(value)
		return pathSuggestionsMsg{
			fieldIndex:  fieldIndex,
			suggestions: suggestions,
		}
	}
}

// listPathSuggestions generates file/directory completion suggestions for the
// given path prefix. It expands ~ to the home directory, hides dotfiles unless
// the user explicitly types a dot, and appends / to directories.
func listPathSuggestions(prefix string) []string {
	if prefix == "" {
		return nil
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

	return buildSuggestions(entries, prefix, partial, tilde)
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
func buildSuggestions(entries []os.DirEntry, prefix, partial string, tilde bool) []string {
	showDotfiles := strings.HasPrefix(partial, ".")
	endsWithSep := strings.HasSuffix(prefix, "/") || strings.HasSuffix(prefix, string(filepath.Separator))

	suggestions := make([]string, 0, len(entries))
	for _, e := range entries {
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
