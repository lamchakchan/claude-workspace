package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestParseFrontmatterBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
		wantDesc string
	}{
		{
			name: "valid frontmatter",
			input: `---
name: my-skill
description: Does something useful
---

# My Skill
`,
			wantName: "my-skill",
			wantDesc: "Does something useful",
		},
		{
			name:     "missing frontmatter",
			input:    "# Just a heading\nSome content",
			wantName: "",
			wantDesc: "",
		},
		{
			name: "partial frontmatter missing closing",
			input: `---
name: my-skill
description: Does something

# No closing delimiter
`,
			wantName: "",
			wantDesc: "",
		},
		{
			name:     "empty file",
			input:    "",
			wantName: "",
			wantDesc: "",
		},
		{
			name: "frontmatter with only name",
			input: `---
name: just-name
---
`,
			wantName: "just-name",
			wantDesc: "",
		},
		{
			name: "frontmatter with only description",
			input: `---
description: Only a description
---
`,
			wantName: "",
			wantDesc: "Only a description",
		},
		{
			name: "frontmatter with extra whitespace",
			input: `---
  name:   spaced-name
  description:   Spaced description
---
`,
			wantName: "spaced-name",
			wantDesc: "Spaced description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotDesc := parseFrontmatterBytes([]byte(tt.input))
			if gotName != tt.wantName {
				t.Errorf("name = %q, want %q", gotName, tt.wantName)
			}
			if gotDesc != tt.wantDesc {
				t.Errorf("description = %q, want %q", gotDesc, tt.wantDesc)
			}
		})
	}
}

func TestDiscoverSkills(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, root string)
		want  []Skill
	}{
		{
			name: "multiple skills",
			setup: func(t *testing.T, root string) {
				mkSkill(t, root, "alpha", "alpha-skill", "Alpha does things")
				mkSkill(t, root, "beta", "beta-skill", "Beta does other things")
			},
			want: []Skill{
				{Name: "alpha-skill", Description: "Alpha does things"},
				{Name: "beta-skill", Description: "Beta does other things"},
			},
		},
		{
			name:  "empty directory",
			setup: func(t *testing.T, root string) {},
			want:  nil,
		},
		{
			name: "non-SKILL.md files ignored",
			setup: func(t *testing.T, root string) {
				dir := filepath.Join(root, "my-skill")
				os.MkdirAll(dir, 0755)
				os.WriteFile(filepath.Join(dir, "README.md"), []byte("not a skill"), 0644)
				os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("just notes"), 0644)
			},
			want: nil,
		},
		{
			name: "fallback to directory name when no name in frontmatter",
			setup: func(t *testing.T, root string) {
				dir := filepath.Join(root, "dir-name")
				os.MkdirAll(dir, 0755)
				os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# Just content\n"), 0644)
			},
			want: []Skill{
				{Name: "dir-name", Description: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			tt.setup(t, root)

			got := discoverSkills(root)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d skills, want %d", len(got), len(tt.want))
			}
			// Build a map for order-independent comparison
			wantMap := make(map[string]string, len(tt.want))
			for _, s := range tt.want {
				wantMap[s.Name] = s.Description
			}
			for _, s := range got {
				wantDesc, ok := wantMap[s.Name]
				if !ok {
					t.Errorf("unexpected skill: %q", s.Name)
					continue
				}
				if s.Description != wantDesc {
					t.Errorf("skill %q description = %q, want %q", s.Name, s.Description, wantDesc)
				}
			}
		})
	}
}

func TestDiscoverCommands(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, root string)
		want  []Skill
	}{
		{
			name: "md files discovered",
			setup: func(t *testing.T, root string) {
				os.WriteFile(filepath.Join(root, "deploy.md"), []byte("Deploy to production\nMore details"), 0644)
				os.WriteFile(filepath.Join(root, "review.md"), []byte("Review the PR changes"), 0644)
			},
			want: []Skill{
				{Name: "deploy", Description: "Deploy to production"},
				{Name: "review", Description: "Review the PR changes"},
			},
		},
		{
			name:  "empty directory",
			setup: func(t *testing.T, root string) {},
			want:  nil,
		},
		{
			name: "non-md files ignored",
			setup: func(t *testing.T, root string) {
				os.WriteFile(filepath.Join(root, "script.sh"), []byte("#!/bin/bash"), 0644)
				os.WriteFile(filepath.Join(root, "notes.txt"), []byte("some notes"), 0644)
				os.WriteFile(filepath.Join(root, "valid.md"), []byte("A valid command"), 0644)
			},
			want: []Skill{
				{Name: "valid", Description: "A valid command"},
			},
		},
		{
			name: "file with leading blank lines",
			setup: func(t *testing.T, root string) {
				os.WriteFile(filepath.Join(root, "blank.md"), []byte("\n\n  \nActual content"), 0644)
			},
			want: []Skill{
				{Name: "blank", Description: "Actual content"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			tt.setup(t, root)

			got := discoverCommands(root)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d commands, want %d", len(got), len(tt.want))
			}
			wantMap := make(map[string]string, len(tt.want))
			for _, s := range tt.want {
				wantMap[s.Name] = s.Description
			}
			for _, s := range got {
				wantDesc, ok := wantMap[s.Name]
				if !ok {
					t.Errorf("unexpected command: %q", s.Name)
					continue
				}
				if s.Description != wantDesc {
					t.Errorf("command %q description = %q, want %q", s.Name, s.Description, wantDesc)
				}
			}
		})
	}
}

// mkSkill creates a skill directory with a SKILL.md containing frontmatter.
func mkSkill(t *testing.T, root, dirName, name, desc string) {
	t.Helper()
	dir := filepath.Join(root, dirName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	content := fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n# %s\n", name, desc, name)
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
