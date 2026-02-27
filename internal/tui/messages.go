package tui

import (
	tea "charm.land/bubbletea/v2"
)

// PopViewMsg is sent when a view wants to pop itself from the navigation stack.
type PopViewMsg struct{}

// PushViewMsg is sent when a view wants to push a new view onto the navigation stack.
type PushViewMsg struct {
	Model tea.Model
}

// ExecAndReturnMsg suspends the TUI, runs a CLI command, then resumes the
// current view (typically the launcher). Use for data-display commands.
type ExecAndReturnMsg struct {
	Command string
	Args    []string
}
