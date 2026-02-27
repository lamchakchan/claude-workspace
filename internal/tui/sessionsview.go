package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lamchakchan/claude-workspace/internal/platform"
	"github.com/lamchakchan/claude-workspace/internal/sessions"
)

// sessionsLoadedMsg carries the loaded session list.
type sessionsLoadedMsg struct {
	sessions []sessions.Session
	err      string
}

// sessionPromptsMsg carries prompts for a selected session.
type sessionPromptsMsg struct {
	session sessions.Session
	prompts []sessions.Prompt
	slug    string
	err     string
}

// SessionsModel shows a selectable session list with drill-down to prompts.
type SessionsModel struct {
	theme    *Theme
	sessions []sessions.Session
	cursor   int
	scroll   int // index of first visible session
	loading  bool
	err      string
	width    int
	height   int

	// Drill-down state: when non-nil, we're viewing a session's prompts.
	promptViewer *ViewerModel
}

// NewSessions creates a new sessions list/viewer.
func NewSessions(theme *Theme) *SessionsModel {
	return &SessionsModel{
		theme:   theme,
		loading: true,
	}
}

func (m *SessionsModel) Init() tea.Cmd {
	return loadSessions
}

func loadSessions() tea.Msg {
	home, err := os.UserHomeDir()
	if err != nil {
		return sessionsLoadedMsg{err: fmt.Sprintf("cannot determine home directory: %v", err)}
	}

	projectsDir := filepath.Join(home, ".claude", "projects")
	if !platform.FileExists(projectsDir) {
		return sessionsLoadedMsg{err: "no Claude Code session data found"}
	}

	projectDirs, err := sessions.ResolveProjectDirs(projectsDir, true)
	if err != nil {
		return sessionsLoadedMsg{err: err.Error()}
	}

	var allSessions []sessions.Session
	for _, dir := range projectDirs {
		projectName := sessions.DecodeProjectPath(filepath.Base(dir))
		s, err := sessions.ScanProjectSessions(dir, projectName)
		if err != nil {
			continue
		}
		allSessions = append(allSessions, s...)
	}

	sort.Slice(allSessions, func(i, j int) bool {
		return allSessions[i].StartTime.After(allSessions[j].StartTime)
	})

	if len(allSessions) > 50 {
		allSessions = allSessions[:50]
	}

	return sessionsLoadedMsg{sessions: allSessions}
}

func loadSessionPrompts(s sessions.Session) tea.Cmd {
	return func() tea.Msg {
		home, err := os.UserHomeDir()
		if err != nil {
			return sessionPromptsMsg{session: s, err: err.Error()}
		}
		projectsDir := filepath.Join(home, ".claude", "projects")

		// Search all project dirs for this session file
		projectEntries, err := os.ReadDir(projectsDir)
		if err != nil {
			return sessionPromptsMsg{session: s, err: err.Error()}
		}

		for _, pe := range projectEntries {
			if !pe.IsDir() {
				continue
			}
			dir := filepath.Join(projectsDir, pe.Name())
			path := filepath.Join(dir, s.ID+".jsonl")
			if platform.FileExists(path) {
				prompts, slug, err := sessions.ParseSessionPrompts(path)
				if err != nil {
					return sessionPromptsMsg{session: s, err: err.Error()}
				}
				return sessionPromptsMsg{session: s, prompts: prompts, slug: slug}
			}
		}

		return sessionPromptsMsg{session: s, err: "session file not found"}
	}
}

// visibleRows returns how many session rows fit in the current terminal height.
func (m *SessionsModel) visibleRows() int {
	// SectionBanner: \n + rule + \n heading\n = 3 lines
	// table: \n + header\n = 2 lines, separator\n = 1 line
	// footer: \n + help = 2 lines → total overhead = 8
	const overhead = 8
	rows := m.height - overhead
	if rows < 1 {
		return 1
	}
	return rows
}

// clampScroll ensures the scroll offset keeps the cursor visible.
func (m *SessionsModel) clampScroll() {
	visible := m.visibleRows()
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+visible {
		m.scroll = m.cursor - visible + 1
	}
	maxScroll := len(m.sessions) - visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scroll > maxScroll {
		m.scroll = maxScroll
	}
}

func (m *SessionsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// If viewing prompts, delegate to the prompt viewer
	if m.promptViewer != nil {
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			if IsBack(msg) || IsQuit(msg) {
				m.promptViewer = nil
				return m, nil
			}
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
		}
		if m.promptViewer != nil {
			_, cmd := m.promptViewer.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case sessionsLoadedMsg:
		m.loading = false
		if msg.err != "" {
			m.err = msg.err
			return m, nil
		}
		m.sessions = msg.sessions
		return m, nil

	case sessionPromptsMsg:
		if msg.err != "" {
			m.err = msg.err
			return m, nil
		}
		content := formatSessionPrompts(msg.session, msg.prompts, msg.slug)
		m.promptViewer = NewViewer("Session: "+msg.session.Title, content, m.theme)
		// Forward window size so the viewport initializes correctly
		if m.width > 0 && m.height > 0 {
			m.promptViewer.SetSize(m.width, m.height)
		}
		return m, nil

	case tea.KeyPressMsg:
		if IsQuit(msg) || IsBack(msg) {
			return m, func() tea.Msg { return PopViewMsg{} }
		}

		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			m.clampScroll()
		case "down", "j":
			if m.cursor < len(m.sessions)-1 {
				m.cursor++
			}
			m.clampScroll()
		case "pgup", "b":
			m.cursor -= m.visibleRows()
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.clampScroll()
		case "pgdown", "f":
			m.cursor += m.visibleRows()
			if m.cursor > len(m.sessions)-1 {
				m.cursor = len(m.sessions) - 1
			}
			m.clampScroll()
		case "g":
			m.cursor = 0
			m.clampScroll()
		case "G":
			if len(m.sessions) > 0 {
				m.cursor = len(m.sessions) - 1
			}
			m.clampScroll()
		case keyEnter:
			if len(m.sessions) > 0 {
				return m, loadSessionPrompts(m.sessions[m.cursor])
			}
		}
	}

	return m, nil
}

func (m *SessionsModel) View() tea.View {
	if m.promptViewer != nil {
		return m.promptViewer.View()
	}

	var b strings.Builder
	b.WriteString(m.theme.SectionBanner("Sessions"))

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

	if len(m.sessions) == 0 {
		b.WriteString("\n  No sessions found.\n")
		return tea.NewView(b.String())
	}

	// Table header
	b.WriteString(fmt.Sprintf("\n  %-10s  %-12s  %s\n", "ID", "DATE", "TITLE"))
	mutedLine := lipgloss.NewStyle().Foreground(m.theme.Muted)
	b.WriteString(mutedLine.Render(fmt.Sprintf("  %-10s  %-12s  %s", "──────────", "────────────", strings.Repeat("─", 50))))
	b.WriteString("\n")

	visible := m.visibleRows()
	end := m.scroll + visible
	if end > len(m.sessions) {
		end = len(m.sessions)
	}

	var rows strings.Builder
	for i := m.scroll; i < end; i++ {
		s := m.sessions[i]
		shortID := s.ID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		date := s.StartTime.Local().Format("2006-01-02")
		title := s.Title
		if len(title) > 60 {
			title = title[:57] + "..."
		}

		line := fmt.Sprintf("%-10s  %-12s  %s", shortID, date, title)

		if i == m.cursor {
			cursor := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render("> ")
			line = lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render(line)
			rows.WriteString("  " + cursor + line)
		} else {
			rows.WriteString("    " + line)
		}
		if i < end-1 {
			rows.WriteString("\n")
		}
	}

	total := len(m.sessions)
	var scrollPct float64
	if total > visible {
		scrollPct = float64(m.scroll) / float64(total-visible)
	}
	bar := renderScrollbar(end-m.scroll, total, visible, scrollPct, m.theme)
	if bar != "" {
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, rows.String(), " ", bar))
	} else {
		b.WriteString(rows.String())
	}
	b.WriteString("\n")

	b.WriteString("\n")
	help := fmt.Sprintf(
		"%s navigate  %s page  %s/%s top/bottom  %s view prompts  %s back  %d/%d",
		m.theme.HelpKey.Render("j/k"),
		m.theme.HelpKey.Render("pgup/pgdn"),
		m.theme.HelpKey.Render("g"),
		m.theme.HelpKey.Render("G"),
		m.theme.HelpKey.Render(keyEnter),
		m.theme.HelpKey.Render("esc"),
		m.cursor+1, len(m.sessions),
	)
	b.WriteString(help)

	return tea.NewView(b.String())
}

func formatSessionPrompts(s sessions.Session, prompts []sessions.Prompt, slug string) string {
	var b strings.Builder
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#06B6D4"))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

	title := s.ID
	if slug != "" {
		title = fmt.Sprintf("%s (%s)", slug, s.ID[:min(8, len(s.ID))])
	}

	b.WriteString("  " + titleStyle.Render(title) + "\n")
	b.WriteString("  Project: " + mutedStyle.Render(s.Project) + "\n")
	b.WriteString(fmt.Sprintf("  Prompts: %s\n\n", mutedStyle.Render(fmt.Sprintf("%d", len(prompts)))))

	for i, p := range prompts {
		ts := p.Timestamp.Local().Format("15:04:05")
		b.WriteString(fmt.Sprintf("  %s\n", titleStyle.Render(fmt.Sprintf("[%d] %s", i+1, ts))))
		for _, line := range strings.Split(p.Content, "\n") {
			b.WriteString("  " + line + "\n")
		}
		b.WriteString("\n")
	}

	return b.String()
}
