package tui

import (
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/lamchakchan/claude-workspace/internal/agents"
	"github.com/lamchakchan/claude-workspace/internal/platform"
)

// agentItem implements ListItem for an agent.
type agentItem struct {
	agent agents.Agent
}

func (a *agentItem) Title() string {
	desc := a.agent.Description
	if len(desc) > 60 {
		desc = desc[:57] + "..."
	}
	if a.agent.Model != "" {
		return fmt.Sprintf("%s  [%s]  %s", a.agent.Name, a.agent.Model, desc)
	}
	return fmt.Sprintf("%s  %s", a.agent.Name, desc)
}

func (a *agentItem) Detail() string {
	if a.agent.Path != "" {
		data, err := os.ReadFile(a.agent.Path)
		if err == nil {
			return string(data)
		}
	}
	return "(unable to read file)"
}

// AgentsModel displays discovered agents in an expandable list.
type AgentsModel struct {
	list *ExpandListModel
}

// NewAgents creates a new agents expandable list.
func NewAgents(theme *Theme) *AgentsModel {
	return &AgentsModel{
		list: NewExpandList("Agents", loadAgentSections, "Agents are invoked automatically by Claude Code when matching tasks arise.", theme),
	}
}

func (m *AgentsModel) Init() tea.Cmd  { return m.list.Init() }
func (m *AgentsModel) View() tea.View { return m.list.View() }
func (m *AgentsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := m.list.Update(msg)
	return m, cmd
}

func loadAgentSections() ([]ListSection, error) {
	var sections []ListSection

	cwd, err := os.Getwd()
	if err == nil {
		agentsDir := filepath.Join(cwd, ".claude", "agents")
		if platform.FileExists(agentsDir) {
			projectAgents := agents.DiscoverAgents(agentsDir)
			if len(projectAgents) > 0 {
				items := make([]ListItem, 0, len(projectAgents))
				for i := range projectAgents {
					items = append(items, &agentItem{agent: projectAgents[i]})
				}
				sections = append(sections, ListSection{Title: "Project Agents (.claude/agents/)", Items: items})
			}
		}
	}

	home, err := os.UserHomeDir()
	if err == nil {
		globalDir := filepath.Join(home, ".claude", "agents")
		if platform.FileExists(globalDir) {
			globalAgents := agents.DiscoverAgents(globalDir)
			if len(globalAgents) > 0 {
				items := make([]ListItem, 0, len(globalAgents))
				for i := range globalAgents {
					items = append(items, &agentItem{agent: globalAgents[i]})
				}
				sections = append(sections, ListSection{Title: "User Agents (~/.claude/agents/)", Items: items})
			}
		}
	}

	return sections, nil
}
