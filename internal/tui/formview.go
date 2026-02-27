package tui

import tea "charm.land/bubbletea/v2"

// formViewUpdate handles the common Update logic for form-based views.
// onSubmit is called with form values when the form is submitted (not cancelled).
// It returns the updated form, and a tea.Cmd. When the cmd is non-nil, the caller
// should return itself as the model along with the cmd.
func formViewUpdate(form *FormModel, msg tea.Msg, onSubmit func([]string) tea.Cmd) (*FormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case FormResult:
		if msg.Cancelled {
			return form, func() tea.Msg { return PopViewMsg{} }
		}
		return form, onSubmit(msg.Values)

	case tea.KeyPressMsg:
		if IsQuit(msg) || IsBack(msg) {
			return form, func() tea.Msg { return PopViewMsg{} }
		}
	}

	return form.Update(msg)
}
