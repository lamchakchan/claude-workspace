package tui

import (
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/lamchakchan/claude-workspace/internal/platform"
	"github.com/lamchakchan/claude-workspace/internal/skills"
)

// skillItem implements ListItem for a skill.
type skillItem struct {
	skill skills.Skill
}

func (s *skillItem) Title() string {
	desc := s.skill.Description
	if len(desc) > 70 {
		desc = desc[:67] + "..."
	}
	if desc != "" {
		return s.skill.Name + "  " + desc
	}
	return s.skill.Name
}

func (s *skillItem) Detail() string {
	if s.skill.Path != "" {
		data, err := os.ReadFile(s.skill.Path)
		if err == nil {
			return strings.TrimSpace(string(data))
		}
	}
	return unableToReadFile
}

// SkillsModel displays discovered skills in an expandable list.
type SkillsModel struct {
	list *ExpandListModel
}

// NewSkills creates a new skills expandable list.
func NewSkills(theme *Theme) *SkillsModel {
	return &SkillsModel{
		list: NewExpandList("Skills", loadSkillSections, "Invoke with: /skill-name inside Claude Code", theme),
	}
}

func (m *SkillsModel) Init() tea.Cmd  { return m.list.Init() }
func (m *SkillsModel) View() tea.View { return m.list.View() }
func (m *SkillsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := m.list.Update(msg)
	return m, cmd
}

func loadSkillSections() ([]ListSection, error) { //nolint:dupl // structurally similar to loadAgentSections but uses different discover/item types
	var sections []ListSection

	cwd, err := os.Getwd()
	if err == nil {
		skillsDir := filepath.Join(cwd, ".claude", "skills")
		if platform.FileExists(skillsDir) {
			projectSkills := skills.DiscoverSkills(skillsDir)
			if len(projectSkills) > 0 {
				items := make([]ListItem, 0, len(projectSkills))
				for i := range projectSkills {
					items = append(items, &skillItem{skill: projectSkills[i]})
				}
				sections = append(sections, ListSection{Title: "Project Skills (.claude/skills/)", Items: items})
			}
		}
	}

	home, err := os.UserHomeDir()
	if err == nil {
		commandsDir := filepath.Join(home, ".claude", "commands")
		if platform.FileExists(commandsDir) {
			commands := skills.DiscoverCommands(commandsDir)
			if len(commands) > 0 {
				items := make([]ListItem, 0, len(commands))
				for i := range commands {
					items = append(items, &skillItem{skill: commands[i]})
				}
				sections = append(sections, ListSection{Title: "Personal Commands (~/.claude/commands/)", Items: items})
			}
		}
	}

	return sections, nil
}
