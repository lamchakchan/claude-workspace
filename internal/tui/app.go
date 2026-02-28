package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
)

// appModel is the root model that manages the navigation stack.
type appModel struct {
	stack   []tea.Model // view navigation stack
	width   int
	height  int
	theme   Theme
	version string
}

// Run starts the interactive TUI. It is called when claude-workspace is invoked
// with no arguments on a TTY. Respects NO_COLOR and ACCESSIBLE env vars.
func Run(version string) error {
	// Skip TUI in accessible/no-color mode — fall back to help text output.
	if IsAccessible() {
		return nil
	}

	theme := DefaultTheme()
	launcher := newLauncher(version, &theme)

	app := &appModel{
		stack:   []tea.Model{launcher},
		theme:   theme,
		version: version,
	}

	p := tea.NewProgram(app)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

func (m *appModel) Init() tea.Cmd {
	if len(m.stack) > 0 {
		return m.stack[len(m.stack)-1].Init()
	}
	return nil
}

func (m *appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if len(m.stack) > 0 {
			current := m.stack[len(m.stack)-1]
			updated, cmd := current.Update(msg)
			m.stack[len(m.stack)-1] = updated
			return m, cmd
		}
		return m, nil

	case PushViewMsg:
		m.stack = append(m.stack, msg.Model)
		initCmd := msg.Model.Init()
		// Forward current window size to newly pushed view.
		var sizeCmd tea.Cmd
		if m.width > 0 && m.height > 0 {
			size := tea.WindowSizeMsg{Width: m.width, Height: m.height}
			updated, cmd := msg.Model.Update(size)
			m.stack[len(m.stack)-1] = updated
			sizeCmd = cmd
		}
		return m, tea.Batch(initCmd, sizeCmd)

	case PopViewMsg:
		if len(m.stack) > 1 {
			m.stack = m.stack[:len(m.stack)-1]
		} else {
			return m, tea.Quit
		}
		return m, nil

	}

	// Forward all other messages to the current view
	if len(m.stack) > 0 {
		current := m.stack[len(m.stack)-1]
		updated, cmd := current.Update(msg)
		m.stack[len(m.stack)-1] = updated
		return m, cmd
	}

	return m, nil
}

func (m *appModel) View() tea.View {
	var v tea.View
	v.AltScreen = true

	if len(m.stack) > 0 {
		inner := m.stack[len(m.stack)-1].View()
		v.Content = inner.Content
	}
	return v
}
