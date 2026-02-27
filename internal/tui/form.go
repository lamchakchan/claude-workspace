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
		switch msg.String() {
		case keyCtrlC, "esc":
			m.done = true
			return m, func() tea.Msg { return FormResult{Cancelled: true} }

		case "tab":
			// For path fields: if there's a matched suggestion, let textinput
			// handle tab (accepts the suggestion). Otherwise, advance to next field.
			if m.Fields[m.cursor].IsPath && m.inputs[m.cursor].CurrentSuggestion() != "" {
				var cmd tea.Cmd
				m.inputs[m.cursor], cmd = m.inputs[m.cursor].Update(msg)
				// After accepting a suggestion, refresh suggestions
				return m, tea.Batch(cmd, readDirSuggestions(m.cursor, m.inputs[m.cursor].Value()))
			}
			m.inputs[m.cursor].Blur()
			m.cursor = (m.cursor + 1) % len(m.inputs)
			m.inputs[m.cursor].Focus()

		case "down":
			// If path field has suggestions, cycle through them
			if m.Fields[m.cursor].IsPath && len(m.inputs[m.cursor].MatchedSuggestions()) > 0 {
				var cmd tea.Cmd
				m.inputs[m.cursor], cmd = m.inputs[m.cursor].Update(msg)
				return m, cmd
			}
			m.inputs[m.cursor].Blur()
			m.cursor = (m.cursor + 1) % len(m.inputs)
			m.inputs[m.cursor].Focus()

		case "up":
			// If path field has suggestions, cycle through them
			if m.Fields[m.cursor].IsPath && len(m.inputs[m.cursor].MatchedSuggestions()) > 0 {
				var cmd tea.Cmd
				m.inputs[m.cursor], cmd = m.inputs[m.cursor].Update(msg)
				return m, cmd
			}
			m.inputs[m.cursor].Blur()
			m.cursor = (m.cursor - 1 + len(m.inputs)) % len(m.inputs)
			m.inputs[m.cursor].Focus()

		case "shift+tab":
			m.inputs[m.cursor].Blur()
			m.cursor = (m.cursor - 1 + len(m.inputs)) % len(m.inputs)
			m.inputs[m.cursor].Focus()

		case keyEnter:
			if m.cursor == len(m.inputs)-1 {
				return m.submit()
			}
			m.inputs[m.cursor].Blur()
			m.cursor++
			m.inputs[m.cursor].Focus()
		}
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

	// Expand ~ to home directory
	expanded := prefix
	if strings.HasPrefix(expanded, "~/") || expanded == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		expanded = filepath.Join(home, expanded[1:])
	}

	// Determine the directory to read and the partial name to match
	dir := filepath.Dir(expanded)
	partial := filepath.Base(expanded)

	// If the prefix ends with /, we're listing inside that directory
	if strings.HasSuffix(prefix, "/") || strings.HasSuffix(prefix, string(filepath.Separator)) {
		dir = expanded
		partial = ""
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	showDotfiles := strings.HasPrefix(partial, ".")

	suggestions := make([]string, 0, len(entries))
	for _, e := range entries {
		name := e.Name()

		// Hide dotfiles unless explicitly typing a dot
		if !showDotfiles && strings.HasPrefix(name, ".") {
			continue
		}

		// Filter by partial match
		if partial != "" && !strings.HasPrefix(name, partial) {
			continue
		}

		// Build the full suggestion path matching the original prefix style
		var suggestion string
		if strings.HasSuffix(prefix, "/") || strings.HasSuffix(prefix, string(filepath.Separator)) {
			suggestion = prefix + name
		} else {
			suggestion = filepath.Join(filepath.Dir(prefix), name)
			// Preserve ~ prefix
			if strings.HasPrefix(prefix, "~/") {
				home, _ := os.UserHomeDir()
				if strings.HasPrefix(suggestion, home) {
					suggestion = "~" + suggestion[len(home):]
				}
			}
		}

		// Append / to directories to make it easy to drill deeper
		if e.IsDir() {
			suggestion += "/"
		}

		suggestions = append(suggestions, suggestion)
	}

	return suggestions
}
