package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// scrollbarWidth is the space reserved for the scrollbar column (space + char).
const scrollbarWidth = 2

// renderScrollbar returns a single-column string (one char per row) showing
// a scrollbar track with a proportional thumb. Returns empty string when
// all content fits on screen.
func renderScrollbar(trackHeight, totalItems, visibleItems int, scrollPercent float64, theme *Theme) string {
	if totalItems <= visibleItems || trackHeight < 1 {
		return ""
	}
	thumbSize := trackHeight * visibleItems / totalItems
	if thumbSize < 1 {
		thumbSize = 1
	}
	thumbStart := int(scrollPercent * float64(trackHeight-thumbSize))
	if thumbStart < 0 {
		thumbStart = 0
	}
	if thumbStart+thumbSize > trackHeight {
		thumbStart = trackHeight - thumbSize
	}

	track := lipgloss.NewStyle().Foreground(theme.Muted)
	thumb := lipgloss.NewStyle().Foreground(theme.Secondary)

	lines := make([]string, trackHeight)
	for i := range lines {
		if i >= thumbStart && i < thumbStart+thumbSize {
			lines[i] = thumb.Render("┃")
		} else {
			lines[i] = track.Render("│")
		}
	}
	return strings.Join(lines, "\n")
}

// viewerContentMsg carries loaded content to the viewer.
type viewerContentMsg struct {
	content string
}

// viewerErrorMsg carries an error from the async loader.
type viewerErrorMsg struct {
	err string
}

// viewerCopiedMsg clears the "Copied!" flash after a delay.
type viewerCopiedMsg struct{}

// ViewerModel wraps a bubbles viewport with a title bar, help footer,
// and scroll percentage indicator. It supports an optional async loading state.
type ViewerModel struct {
	title    string
	viewport viewport.Model
	theme    *Theme
	ready    bool
	loading  bool
	err      string
	copied   bool
	loader   func() (string, error) // optional async loader
	width    int
	height   int
}

// NewViewer creates a viewer with content already available.
func NewViewer(title, content string, theme *Theme) *ViewerModel {
	vp := viewport.New()
	vp.SoftWrap = true
	vp.SetContent(content)

	return &ViewerModel{
		title:    title,
		viewport: vp,
		theme:    theme,
		ready:    false, // sized on first WindowSizeMsg
	}
}

// NewLoadingViewer creates a viewer that shows "Loading..." and runs loader
// asynchronously to fetch content.
func NewLoadingViewer(title string, loader func() (string, error), theme *Theme) *ViewerModel {
	return &ViewerModel{
		title:   title,
		theme:   theme,
		loading: true,
		loader:  loader,
	}
}

// SetSize initializes or resizes the viewport to the given dimensions.
// Use this instead of sending a WindowSizeMsg when you need to size
// the viewer synchronously (e.g. right after construction).
func (m *ViewerModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	vpHeight := max(1, height-viewerHeaderLines-viewerFooterLines)
	vpWidth := max(1, width-scrollbarWidth)
	content := m.viewport.GetContent()
	m.viewport = viewport.New(viewport.WithWidth(vpWidth), viewport.WithHeight(vpHeight))
	m.viewport.SoftWrap = true
	if content != "" {
		m.viewport.SetContent(content)
	}
	m.ready = true
}

const viewerHeaderLines = 2 // title + blank line
const viewerFooterLines = 2 // blank line + help text

func (m *ViewerModel) Init() tea.Cmd {
	if m.loader != nil {
		loader := m.loader
		return func() tea.Msg {
			content, err := loader()
			if err != nil {
				return viewerErrorMsg{err: err.Error()}
			}
			return viewerContentMsg{content: content}
		}
	}
	return nil
}

func (m *ViewerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		vpHeight := max(1, msg.Height-viewerHeaderLines-viewerFooterLines)
		vpWidth := max(1, msg.Width-scrollbarWidth)
		if !m.ready {
			m.viewport = viewport.New(viewport.WithWidth(vpWidth), viewport.WithHeight(vpHeight))
			m.viewport.SoftWrap = true
			// Re-apply any existing content (from NewViewer).
			if content := m.viewport.GetContent(); content != "" {
				m.viewport.SetContent(content)
			}
			m.ready = true
		} else {
			m.viewport.SetWidth(vpWidth)
			m.viewport.SetHeight(vpHeight)
		}
		return m, nil

	case viewerContentMsg:
		m.loading = false
		m.viewport = viewport.New(viewport.WithWidth(max(1, m.width-scrollbarWidth)), viewport.WithHeight(max(1, m.height-viewerHeaderLines-viewerFooterLines)))
		m.viewport.SoftWrap = true
		m.viewport.SetContent(msg.content)
		m.ready = true
		return m, nil

	case viewerErrorMsg:
		m.loading = false
		m.err = msg.err
		return m, nil

	case viewerCopiedMsg:
		m.copied = false
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", keyCtrlC, "esc":
			return m, func() tea.Msg { return PopViewMsg{} }
		case "g":
			m.viewport.GotoTop()
			return m, nil
		case "G":
			m.viewport.GotoBottom()
			return m, nil
		case "y":
			m.copied = true
			return m, tea.Batch(
				tea.SetClipboard(m.viewport.GetContent()),
				tea.Tick(2*time.Second, func(time.Time) tea.Msg {
					return viewerCopiedMsg{}
				}),
			)
		}
	}

	if m.ready {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *ViewerModel) View() tea.View {
	var b strings.Builder

	// Title bar
	b.WriteString(m.theme.SectionBanner(m.title))

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

	if m.ready {
		vpContent := m.viewport.View()
		totalLines := strings.Count(m.viewport.GetContent(), "\n") + 1
		vpHeight := m.viewport.Height()
		bar := renderScrollbar(vpHeight, totalLines, vpHeight, m.viewport.ScrollPercent(), m.theme)
		if bar != "" {
			b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, vpContent, " ", bar))
		} else {
			b.WriteString(vpContent)
		}
		b.WriteString("\n")
	}

	// Footer
	pct := int(m.viewport.ScrollPercent() * 100)
	var trail string
	if m.copied {
		trail = lipgloss.NewStyle().Foreground(m.theme.Success).Bold(true).Render("Copied!")
	} else {
		trail = m.theme.HelpKey.Render(fmt.Sprintf("%d", pct)) + "%"
	}
	help := fmt.Sprintf(
		"%s scroll  %s page  %s/%s top/bottom  %s copy  %s back  %s",
		m.theme.HelpKey.Render("j/k"),
		m.theme.HelpKey.Render("pgup/pgdn"),
		m.theme.HelpKey.Render("g"),
		m.theme.HelpKey.Render("G"),
		m.theme.HelpKey.Render("y"),
		m.theme.HelpKey.Render("esc"),
		trail,
	)
	b.WriteString(help)

	return tea.NewView(b.String())
}
