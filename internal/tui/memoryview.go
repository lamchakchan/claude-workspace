package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lamchakchan/claude-workspace/internal/memory"
)

// MemoryModel displays memory layers status in a scrollable viewer.
type MemoryModel struct {
	viewer *ViewerModel
}

// NewMemory creates a new memory layers viewer.
func NewMemory(theme *Theme) *MemoryModel {
	return &MemoryModel{
		viewer: NewLoadingViewer("Memory Layers", loadMemoryLayers, theme),
	}
}

func (m *MemoryModel) Init() tea.Cmd  { return m.viewer.Init() }
func (m *MemoryModel) View() tea.View { return m.viewer.View() }
func (m *MemoryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := m.viewer.Update(msg)
	return m, cmd
}

func loadMemoryLayers() (string, error) {
	layers, err := memory.DiscoverLayers()
	if err != nil {
		return "", fmt.Errorf("discovering memory layers: %w", err)
	}

	var b strings.Builder
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#06B6D4"))
	okStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#10B981"))
	warnStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F59E0B"))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

	for _, layer := range layers {
		b.WriteString("  " + labelStyle.Render(layer.Label) + "\n")

		status := okStyle.Render("[OK]")
		if !layer.Exists {
			status = warnStyle.Render("[NOT FOUND]")
		}
		b.WriteString("    Status: " + status + "\n")

		if layer.Path != "" {
			b.WriteString("    Path:   " + mutedStyle.Render(layer.Path) + "\n")
		}

		if layer.Provider != "" {
			b.WriteString("    Provider: " + mutedStyle.Render(layer.Provider) + "\n")
		}

		if layer.Stats != "" {
			b.WriteString("    Stats:  " + mutedStyle.Render(layer.Stats) + "\n")
		}

		if layer.Lines > 0 {
			b.WriteString(fmt.Sprintf("    Lines:  %s\n", mutedStyle.Render(fmt.Sprintf("%d", layer.Lines))))
		}

		if len(layer.Files) > 0 {
			b.WriteString(fmt.Sprintf("    Files:  %s\n", mutedStyle.Render(fmt.Sprintf("%d", len(layer.Files)))))
			for name := range layer.Files {
				b.WriteString("      - " + mutedStyle.Render(name) + "\n")
			}
		}

		b.WriteString("\n")
	}

	return b.String(), nil
}
