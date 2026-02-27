// Package skills discovers and lists available Claude Code skills and personal commands.
package skills

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

// Skill represents a discovered skill or personal command.
type Skill struct {
	Name        string
	Description string
}

// Run routes the skills subcommand.
func Run(args []string) error {
	subcmd := "list"
	if len(args) > 0 {
		subcmd = args[0]
	}
	switch subcmd {
	case "list":
		return list()
	default:
		fmt.Fprintf(os.Stderr, "Unknown skills subcommand: %s\n", subcmd)
		fmt.Fprintln(os.Stderr, "Usage: claude-workspace skills [list]")
		return fmt.Errorf("unknown subcommand: %s", subcmd)
	}
}

// list discovers skills from project, personal, and platform sources and prints them.
func list() error {
	platform.PrintBanner(os.Stdout, "Skills")
	fmt.Println()

	anyFound := false

	// 1. Project skills
	cwd, err := os.Getwd()
	if err == nil {
		skillsDir := filepath.Join(cwd, ".claude", "skills")
		if platform.FileExists(skillsDir) {
			projectSkills := discoverSkills(skillsDir)
			if len(projectSkills) > 0 {
				anyFound = true
				platform.PrintSection(os.Stdout, "Project Skills (.claude/skills/)")
				printSkillTable(projectSkills)
			}
		}
	}

	// 2. Personal commands
	home, err := os.UserHomeDir()
	if err == nil {
		commandsDir := filepath.Join(home, ".claude", "commands")
		if platform.FileExists(commandsDir) {
			commands := discoverCommands(commandsDir)
			if len(commands) > 0 {
				anyFound = true
				platform.PrintSection(os.Stdout, "Personal Commands (~/.claude/commands/)")
				printSkillTable(commands)
			}
		}
	}

	// 3. Platform built-in skills (from embedded FS)
	if platform.FS != nil {
		builtins := discoverEmbeddedSkills(platform.FS, ".claude/skills")
		if len(builtins) > 0 {
			anyFound = true
			platform.PrintSection(os.Stdout, "Platform Built-in Skills")
			printSkillTable(builtins)
		}
	}

	if !anyFound {
		fmt.Println("  No skills found.")
		fmt.Println()
		fmt.Println("  Create a project skill:   .claude/skills/my-skill/SKILL.md")
		fmt.Println("  Create a personal command: ~/.claude/commands/my-command.md")
		fmt.Println()
		fmt.Println("  See: docs/SKILLS.md for details")
		fmt.Println()
		return nil
	}

	// Tips
	platform.PrintSection(os.Stdout, "Tips")
	fmt.Println("  Invoke with: /skill-name inside Claude Code")
	fmt.Println("  Create new:  .claude/skills/my-skill/SKILL.md (project, shared)")
	fmt.Println("               ~/.claude/commands/my-command.md (personal, local)")
	fmt.Println()

	return nil
}

// discoverSkills walks a directory for SKILL.md files and parses their frontmatter.
func discoverSkills(root string) []Skill {
	var skills []Skill
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.Name() == "SKILL.md" {
			name, desc := parseFrontmatter(path)
			if name == "" {
				// Fall back to directory name
				name = filepath.Base(filepath.Dir(path))
			}
			skills = append(skills, Skill{Name: name, Description: desc})
		}
		return nil
	})
	return skills
}

// discoverCommands walks a directory for .md files and uses filename + first line as description.
func discoverCommands(root string) []Skill {
	var commands []Skill
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		name := strings.TrimSuffix(d.Name(), ".md")
		desc := firstNonEmptyLine(path)
		commands = append(commands, Skill{Name: name, Description: desc})
		return nil
	})
	return commands
}

// discoverEmbeddedSkills walks the embedded FS for SKILL.md files.
func discoverEmbeddedSkills(efs fs.FS, root string) []Skill {
	var skills []Skill
	_ = fs.WalkDir(efs, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.Name() == "SKILL.md" {
			data, readErr := fs.ReadFile(efs, path)
			if readErr != nil {
				return nil
			}
			name, desc := parseFrontmatterBytes(data)
			if name == "" {
				name = filepath.Base(filepath.Dir(path))
			}
			skills = append(skills, Skill{Name: name, Description: desc})
		}
		return nil
	})
	return skills
}

// parseFrontmatter reads a file and extracts name and description from YAML frontmatter.
func parseFrontmatter(path string) (name, description string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", ""
	}
	return parseFrontmatterBytes(data)
}

// parseFrontmatterBytes extracts name and description from YAML frontmatter bytes.
func parseFrontmatterBytes(data []byte) (name, description string) {
	content := string(data)
	if !strings.HasPrefix(strings.TrimSpace(content), "---") {
		return "", ""
	}

	// Find the opening and closing ---
	trimmed := strings.TrimSpace(content)
	firstDelim := strings.Index(trimmed, "---")
	if firstDelim < 0 {
		return "", ""
	}
	rest := trimmed[firstDelim+3:]
	secondDelim := strings.Index(rest, "---")
	if secondDelim < 0 {
		return "", ""
	}

	frontmatter := rest[:secondDelim]
	for _, line := range strings.Split(frontmatter, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		} else if strings.HasPrefix(line, "description:") {
			description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		}
	}
	return name, description
}

// firstNonEmptyLine reads a file and returns its first non-empty line.
func firstNonEmptyLine(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

// printSkillTable prints skills in aligned columns.
func printSkillTable(skills []Skill) {
	if len(skills) == 0 {
		return
	}

	// Find max name length for alignment
	maxName := 0
	for _, s := range skills {
		if len(s.Name) > maxName {
			maxName = len(s.Name)
		}
	}

	for _, s := range skills {
		desc := s.Description
		if len(desc) > 70 {
			desc = desc[:67] + "..."
		}
		fmt.Printf("  %-*s  %s\n", maxName, s.Name, desc)
	}
	fmt.Println()
}
