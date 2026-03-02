package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestParseFrontmatterBytes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Agent
	}{
		{
			name: "valid frontmatter with all fields",
			input: `---
name: code-reviewer
description: Code quality and correctness review
model: sonnet
tools: Read, Grep, Glob, Bash
---

# Code Reviewer
`,
			want: Agent{
				Name:        "code-reviewer",
				Description: "Code quality and correctness review",
				Model:       "sonnet",
				Tools:       "Read, Grep, Glob, Bash",
			},
		},
		{
			name:  "missing frontmatter",
			input: "# Just a heading\nSome content",
			want:  Agent{},
		},
		{
			name:  "empty input",
			input: "",
			want:  Agent{},
		},
		{
			name: "no closing delimiter",
			input: `---
name: broken
description: Missing closing

# No closing delimiter
`,
			want: Agent{},
		},
		{
			name: "partial fields",
			input: `---
name: partial-agent
model: haiku
---
`,
			want: Agent{
				Name:  "partial-agent",
				Model: "haiku",
			},
		},
		{
			name: "only description",
			input: `---
description: Only a description here
---
`,
			want: Agent{
				Description: "Only a description here",
			},
		},
		{
			name: "extra whitespace",
			input: `---
  name:   spaced-agent
  description:   Spaced description
  model:   opus
  tools:   Read, Bash
---
`,
			want: Agent{
				Name:        "spaced-agent",
				Description: "Spaced description",
				Model:       "opus",
				Tools:       "Read, Bash",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseFrontmatterBytes([]byte(tt.input))
			if got.Name != tt.want.Name {
				t.Errorf("Name = %q, want %q", got.Name, tt.want.Name)
			}
			if got.Description != tt.want.Description {
				t.Errorf("Description = %q, want %q", got.Description, tt.want.Description)
			}
			if got.Model != tt.want.Model {
				t.Errorf("Model = %q, want %q", got.Model, tt.want.Model)
			}
			if got.Tools != tt.want.Tools {
				t.Errorf("Tools = %q, want %q", got.Tools, tt.want.Tools)
			}
		})
	}
}

func TestDiscoverAgents(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, root string)
		want  []Agent
	}{
		{
			name: "multiple agents",
			setup: func(t *testing.T, root string) {
				mkAgent(t, root, "alpha", "Alpha does things", "sonnet", "Read, Grep")
				mkAgent(t, root, "beta", "Beta does other things", "haiku", "Bash")
			},
			want: []Agent{
				{Name: "alpha", Description: "Alpha does things", Model: "sonnet", Tools: "Read, Grep"},
				{Name: "beta", Description: "Beta does other things", Model: "haiku", Tools: "Bash"},
			},
		},
		{
			name:  "empty directory",
			setup: func(_ *testing.T, _ string) {},
			want:  nil,
		},
		{
			name: "non-md files ignored",
			setup: func(_ *testing.T, root string) {
				_ = os.WriteFile(filepath.Join(root, "README.txt"), []byte("not an agent"), 0644)
				_ = os.WriteFile(filepath.Join(root, "notes.yaml"), []byte("key: value"), 0644)
			},
			want: nil,
		},
		{
			name: "filename fallback when no name in frontmatter",
			setup: func(_ *testing.T, root string) {
				content := "# Just content, no frontmatter\n"
				_ = os.WriteFile(filepath.Join(root, "my-agent.md"), []byte(content), 0644)
			},
			want: []Agent{
				{Name: "my-agent"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			tt.setup(t, root)
			got := DiscoverAgents(root)
			assertAgents(t, got, tt.want)
		})
	}
}

// assertAgents compares two Agent slices in an order-independent manner.
func assertAgents(t *testing.T, got, want []Agent) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d agents, want %d", len(got), len(want))
	}
	wantMap := make(map[string]Agent, len(want))
	for _, a := range want {
		wantMap[a.Name] = a
	}
	for _, g := range got {
		w, ok := wantMap[g.Name]
		if !ok {
			t.Errorf("unexpected agent: %q", g.Name)
			continue
		}
		if g.Description != w.Description {
			t.Errorf("agent %q Description = %q, want %q", g.Name, g.Description, w.Description)
		}
		if g.Model != w.Model {
			t.Errorf("agent %q Model = %q, want %q", g.Name, g.Model, w.Model)
		}
		if g.Tools != w.Tools {
			t.Errorf("agent %q Tools = %q, want %q", g.Name, g.Tools, w.Tools)
		}
	}
}

// mkAgent creates an agent .md file with frontmatter in the given directory.
func mkAgent(t *testing.T, root, name, desc, model, tools string) {
	t.Helper()
	content := fmt.Sprintf("---\nname: %s\ndescription: %s\nmodel: %s\ntools: %s\n---\n\n# %s\n", name, desc, model, tools, name)
	if err := os.WriteFile(filepath.Join(root, name+".md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
