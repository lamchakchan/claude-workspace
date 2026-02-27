package tui

import (
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
}

// FormResult is sent when the user submits or cancels the form.
type FormResult struct {
	Values    []string // values indexed by field position
	Cancelled bool
}

// FormModel is a generic multi-field text input form component.
type FormModel struct {
	Title  string
	Fields []FormField
	inputs []textinput.Model
	cursor int // focused field index
	done   bool
	theme  Theme
}

// NewForm creates a new form with the given fields.
func NewForm(title string, fields []FormField, theme Theme) FormModel {
	inputs := make([]textinput.Model, len(fields))
	for i, f := range fields {
		ti := textinput.New()
		ti.Placeholder = f.Placeholder
		if f.Password {
			ti.EchoMode = textinput.EchoPassword
		}
		if i == 0 {
			ti.Focus()
		}
		inputs[i] = ti
	}

	return FormModel{
		Title:  title,
		Fields: fields,
		inputs: inputs,
		theme:  theme,
	}
}

// Values returns the current values of all fields.
func (m FormModel) Values() []string {
	vals := make([]string, len(m.inputs))
	for i, inp := range m.inputs {
		vals[i] = inp.Value()
	}
	return vals
}

func (m FormModel) Init() tea.Cmd {
	if len(m.inputs) > 0 {
		return textinput.Blink
	}
	return nil
}

func (m FormModel) Update(msg tea.Msg) (FormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.done = true
			return m, func() tea.Msg { return FormResult{Cancelled: true} }

		case "tab", "down":
			m.inputs[m.cursor].Blur()
			m.cursor = (m.cursor + 1) % len(m.inputs)
			m.inputs[m.cursor].Focus()

		case "shift+tab", "up":
			m.inputs[m.cursor].Blur()
			m.cursor = (m.cursor - 1 + len(m.inputs)) % len(m.inputs)
			m.inputs[m.cursor].Focus()

		case "enter":
			if m.cursor == len(m.inputs)-1 {
				return m.submit()
			}
			m.inputs[m.cursor].Blur()
			m.cursor++
			m.inputs[m.cursor].Focus()
		}
	}

	var cmd tea.Cmd
	m.inputs[m.cursor], cmd = m.inputs[m.cursor].Update(msg)
	return m, cmd
}

func (m FormModel) submit() (FormModel, tea.Cmd) {
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

func (m FormModel) View() string {
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
		b.WriteString("  " + m.inputs[i].View() + "\n\n")
	}

	b.WriteString("\n")
	help := m.theme.HelpKey.Render("tab") + " " + m.theme.HelpDesc.Render("next field") + "  " +
		m.theme.HelpKey.Render("enter") + " " + m.theme.HelpDesc.Render("submit") + "  " +
		m.theme.HelpKey.Render("esc") + " " + m.theme.HelpDesc.Render("cancel")
	b.WriteString(help)

	return b.String()
}
