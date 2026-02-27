package tui

import (
	"fmt"
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"
)

// appModel is the root model that manages the navigation stack.
type appModel struct {
	stack       []tea.Model // view navigation stack
	width       int
	height      int
	theme       Theme
	version     string
	pendingCmd  string   // command to run after TUI exits
	pendingArgs []string // args for pending command
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
	result, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	// If a command was selected, execute it after the TUI exits
	if m, ok := result.(*appModel); ok && m.pendingCmd != "" {
		return execCommand(m.pendingCmd, m.pendingArgs)
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

	case ExecAndReturnMsg:
		// Run the command inline; TUI resumes current view when done.
		cmd := m.execInline(msg.Command, msg.Args)
		return m, cmd

	case commandMsg:
		// User selected a plain CLI command — store it and quit TUI
		m.pendingCmd = msg.command
		m.pendingArgs = msg.args
		return m, tea.Quit
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

// execInline runs a CLI subcommand inline via ExecProcess (TUI resumes after).
func (m *appModel) execInline(command string, args []string) tea.Cmd {
	exe, err := os.Executable()
	if err != nil {
		return nil
	}
	cmdArgs := append([]string{command}, args...)
	cmd := exec.Command(exe, cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return tea.ExecProcess(cmd, func(_ error) tea.Msg {
		return nil // resume current view
	})
}

// execCommand runs a claude-workspace subcommand in the user's terminal.
func execCommand(command string, args []string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable: %w", err)
	}

	cmdArgs := append([]string{command}, args...)
	cmd := exec.Command(exe, cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
