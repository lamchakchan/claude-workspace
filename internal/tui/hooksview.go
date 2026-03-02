package tui

import (
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/lamchakchan/claude-workspace/internal/hooks"
	"github.com/lamchakchan/claude-workspace/internal/platform"
)

// hookScriptItem implements ListItem for a hook script.
type hookScriptItem struct {
	script hooks.HookScript
}

func (h *hookScriptItem) Title() string {
	desc := h.script.Description
	if len(desc) > 70 {
		desc = desc[:67] + "..."
	}
	if desc != "" {
		return h.script.Name + "  " + desc
	}
	return h.script.Name
}

func (h *hookScriptItem) Detail() string {
	if h.script.Path != "" {
		data, err := os.ReadFile(h.script.Path)
		if err == nil {
			return string(data)
		}
	}
	return unableToReadFile
}

// hookConfigItem implements ListItem for a hook configuration entry.
type hookConfigItem struct {
	config hooks.HookConfig
}

func (h *hookConfigItem) Title() string {
	msg := h.config.StatusMessage
	if len(msg) > 50 {
		msg = msg[:47] + "..."
	}
	return fmt.Sprintf("%s  %s  %s", h.config.Event, h.config.Matcher, msg)
}

func (h *hookConfigItem) Detail() string {
	return fmt.Sprintf("Event:   %s\nMatcher: %s\nCommand: %s\nStatus:  %s",
		h.config.Event, h.config.Matcher, h.config.Command, h.config.StatusMessage)
}

// HooksModel displays discovered hooks in an expandable list.
type HooksModel struct {
	list *ExpandListModel
}

// NewHooks creates a new hooks expandable list.
func NewHooks(theme *Theme) *HooksModel {
	return &HooksModel{
		list: NewExpandList("Hooks", loadHookSections, "Hooks are shell scripts that run before/after tool use or on events.", theme),
	}
}

func (m *HooksModel) Init() tea.Cmd  { return m.list.Init() }
func (m *HooksModel) View() tea.View { return m.list.View() }
func (m *HooksModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := m.list.Update(msg)
	return m, cmd
}

func loadHookSections() ([]ListSection, error) {
	var sections []ListSection

	cwd, err := os.Getwd()
	if err == nil {
		hooksDir := filepath.Join(cwd, ".claude", "hooks")
		if platform.FileExists(hooksDir) {
			scripts := hooks.DiscoverHookScripts(hooksDir)
			if len(scripts) > 0 {
				items := make([]ListItem, 0, len(scripts))
				for i := range scripts {
					items = append(items, &hookScriptItem{script: scripts[i]})
				}
				sections = append(sections, ListSection{Title: "Project Hook Scripts (.claude/hooks/)", Items: items})
			}
		}
	}

	if err == nil {
		settingsPath := filepath.Join(cwd, ".claude", "settings.json")
		if platform.FileExists(settingsPath) {
			configs := hooks.DiscoverHookConfig(settingsPath)
			if len(configs) > 0 {
				items := make([]ListItem, 0, len(configs))
				for i := range configs {
					items = append(items, &hookConfigItem{config: configs[i]})
				}
				sections = append(sections, ListSection{Title: "Hook Configuration (settings.json)", Items: items})
			}
		}
	}

	return sections, nil
}
