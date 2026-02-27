package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lamchakchan/claude-workspace/internal/platform"
	"github.com/lamchakchan/claude-workspace/internal/skills"
)

// SkillsModel displays discovered skills in a scrollable viewer.
type SkillsModel struct {
	viewer *ViewerModel
}

// NewSkills creates a new skills viewer.
func NewSkills(theme *Theme) *SkillsModel {
	return &SkillsModel{
		viewer: NewLoadingViewer("Skills", loadSkills, theme),
	}
}

func (m *SkillsModel) Init() tea.Cmd  { return m.viewer.Init() }
func (m *SkillsModel) View() tea.View { return m.viewer.View() }
func (m *SkillsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := m.viewer.Update(msg)
	return m, cmd
}

func loadSkills() (string, error) {
	var b strings.Builder
	anyFound := false

	cwd, err := os.Getwd()
	if err == nil {
		skillsDir := filepath.Join(cwd, ".claude", "skills")
		if platform.FileExists(skillsDir) {
			projectSkills := skills.DiscoverSkills(skillsDir)
			if len(projectSkills) > 0 {
				anyFound = true
				writeSkillSection(&b, "Project Skills (.claude/skills/)", projectSkills)
			}
		}
	}

	home, err := os.UserHomeDir()
	if err == nil {
		commandsDir := filepath.Join(home, ".claude", "commands")
		if platform.FileExists(commandsDir) {
			commands := skills.DiscoverCommands(commandsDir)
			if len(commands) > 0 {
				anyFound = true
				writeSkillSection(&b, "Personal Commands (~/.claude/commands/)", commands)
			}
		}
	}

	if platform.FS != nil {
		builtins := skills.DiscoverEmbeddedSkills(platform.FS, ".claude/skills")
		if len(builtins) > 0 {
			anyFound = true
			writeSkillSection(&b, "Platform Built-in Skills", builtins)
		}
	}

	if !anyFound {
		b.WriteString("  No skills found.\n\n")
		b.WriteString("  Create a project skill:   .claude/skills/my-skill/SKILL.md\n")
		b.WriteString("  Create a personal command: ~/.claude/commands/my-command.md\n")
	}

	b.WriteString("\n  Invoke with: /skill-name inside Claude Code\n")

	return b.String(), nil
}

func writeSkillSection(b *strings.Builder, title string, items []skills.Skill) {
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#06B6D4"))
	b.WriteString("  " + sectionStyle.Render(title) + "\n")

	maxName := 0
	for _, s := range items {
		if len(s.Name) > maxName {
			maxName = len(s.Name)
		}
	}
	for _, s := range items {
		desc := s.Description
		if len(desc) > 70 {
			desc = desc[:67] + "..."
		}
		fmt.Fprintf(b, "    %-*s  %s\n", maxName, s.Name, desc)
	}
	b.WriteString("\n")
}
