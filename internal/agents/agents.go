// Package agents discovers and lists available Claude Code agents.
package agents

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

// Agent represents a discovered agent definition.
type Agent struct {
	Name        string
	Description string
	Model       string
	Tools       string
	Path        string
}

// Run routes the agents subcommand.
func Run(args []string) error {
	subcmd := "list"
	if len(args) > 0 {
		subcmd = args[0]
	}
	switch subcmd {
	case "list":
		return list()
	default:
		fmt.Fprintf(os.Stderr, "Unknown agents subcommand: %s\n", subcmd)
		fmt.Fprintln(os.Stderr, "Usage: claude-workspace agents [list]")
		return fmt.Errorf("unknown subcommand: %s", subcmd)
	}
}

// list discovers agents from project, user-global, and platform sources and prints them.
func list() error {
	platform.PrintBanner(os.Stdout, "Agents")
	fmt.Println()

	anyFound := false

	// 1. Project agents
	cwd, err := os.Getwd()
	if err == nil {
		agentsDir := filepath.Join(cwd, ".claude", "agents")
		if platform.FileExists(agentsDir) {
			projectAgents := DiscoverAgents(agentsDir)
			if len(projectAgents) > 0 {
				anyFound = true
				platform.PrintSection(os.Stdout, "Project Agents (.claude/agents/)")
				printAgentTable(projectAgents)
			}
		}
	}

	// 2. User-global agents
	home, err := os.UserHomeDir()
	if err == nil {
		globalDir := filepath.Join(home, ".claude", "agents")
		if platform.FileExists(globalDir) {
			globalAgents := DiscoverAgents(globalDir)
			if len(globalAgents) > 0 {
				anyFound = true
				platform.PrintSection(os.Stdout, "User Agents (~/.claude/agents/)")
				printAgentTable(globalAgents)
			}
		}
	}

	if !anyFound {
		fmt.Println("  No agents found.")
		fmt.Println()
		fmt.Println("  Create a project agent:  .claude/agents/my-agent.md")
		fmt.Println("  Create a personal agent: ~/.claude/agents/my-agent.md")
		fmt.Println()
		return nil
	}

	// Tips
	platform.PrintSection(os.Stdout, "Tips")
	fmt.Println("  Agents are invoked automatically by Claude Code when matching tasks arise.")
	fmt.Println("  Create new:  .claude/agents/my-agent.md")
	fmt.Println()

	return nil
}

// DiscoverAgents walks a directory for .md files and parses their frontmatter.
func DiscoverAgents(root string) []Agent {
	var agents []Agent
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
		agent := parseFrontmatter(path)
		if agent.Name == "" {
			agent.Name = strings.TrimSuffix(d.Name(), ".md")
		}
		agent.Path = path
		agents = append(agents, agent)
		return nil
	})
	return agents
}

// DiscoverEmbeddedAgents walks the embedded FS for .md files and parses their frontmatter.
func DiscoverEmbeddedAgents(efs fs.FS, root string) []Agent {
	var agents []Agent
	_ = fs.WalkDir(efs, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		data, readErr := fs.ReadFile(efs, path)
		if readErr != nil {
			return nil
		}
		agent := parseFrontmatterBytes(data)
		if agent.Name == "" {
			agent.Name = strings.TrimSuffix(d.Name(), ".md")
		}
		agents = append(agents, agent)
		return nil
	})
	return agents
}

// parseFrontmatter reads a file and extracts agent fields from YAML frontmatter.
func parseFrontmatter(path string) Agent {
	data, err := os.ReadFile(path)
	if err != nil {
		return Agent{}
	}
	return parseFrontmatterBytes(data)
}

// parseFrontmatterBytes extracts agent fields from YAML frontmatter bytes.
func parseFrontmatterBytes(data []byte) Agent {
	content := string(data)
	if !strings.HasPrefix(strings.TrimSpace(content), "---") {
		return Agent{}
	}

	// Find the opening and closing ---
	trimmed := strings.TrimSpace(content)
	firstDelim := strings.Index(trimmed, "---")
	if firstDelim < 0 {
		return Agent{}
	}
	rest := trimmed[firstDelim+3:]
	secondDelim := strings.Index(rest, "---")
	if secondDelim < 0 {
		return Agent{}
	}

	var agent Agent
	frontmatter := rest[:secondDelim]
	for _, line := range strings.Split(frontmatter, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			agent.Name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		} else if strings.HasPrefix(line, "description:") {
			agent.Description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		} else if strings.HasPrefix(line, "model:") {
			agent.Model = strings.TrimSpace(strings.TrimPrefix(line, "model:"))
		} else if strings.HasPrefix(line, "tools:") {
			agent.Tools = strings.TrimSpace(strings.TrimPrefix(line, "tools:"))
		}
	}
	return agent
}

// printAgentTable prints agents in aligned columns: Name, Model, Description.
func printAgentTable(agents []Agent) {
	if len(agents) == 0 {
		return
	}

	// Find max lengths for alignment
	maxName := 0
	maxModel := 0
	for _, a := range agents {
		if len(a.Name) > maxName {
			maxName = len(a.Name)
		}
		if len(a.Model) > maxModel {
			maxModel = len(a.Model)
		}
	}

	for _, a := range agents {
		desc := a.Description
		if len(desc) > 60 {
			desc = desc[:57] + "..."
		}
		fmt.Printf("  %-*s  %-*s  %s\n", maxName, a.Name, maxModel, a.Model, desc)
	}
	fmt.Println()
}
