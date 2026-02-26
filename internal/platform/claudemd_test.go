package platform

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makePkg(deps map[string]string, devDeps map[string]string) map[string]json.RawMessage {
	pkg := make(map[string]json.RawMessage)
	if deps != nil {
		b, _ := json.Marshal(deps)
		pkg["dependencies"] = b
	}
	if devDeps != nil {
		b, _ := json.Marshal(devDeps)
		pkg["devDependencies"] = b
	}
	return pkg
}

func TestDetectJSTechStack(t *testing.T) {
	tests := []struct {
		name    string
		deps    map[string]string
		devDeps map[string]string
		want    string
	}{
		{
			name: "empty deps returns Node.js",
			deps: map[string]string{},
			want: "Node.js",
		},
		{
			name: "nil deps returns Node.js",
			want: "Node.js",
		},
		{
			name: "react project",
			deps: map[string]string{"react": "^18.0.0"},
			want: "React",
		},
		{
			name: "next.js project (prefers Next.js over React)",
			deps: map[string]string{"next": "^14.0.0", "react": "^18.0.0"},
			want: "Next.js",
		},
		{
			name: "vue project",
			deps: map[string]string{"vue": "^3.0.0"},
			want: "Vue",
		},
		{
			name: "angular project",
			deps: map[string]string{"@angular/core": "^17.0.0"},
			want: "Angular",
		},
		{
			name: "svelte project",
			deps: map[string]string{"svelte": "^4.0.0"},
			want: "Svelte",
		},
		{
			name:    "react with typescript",
			deps:    map[string]string{"react": "^18.0.0"},
			devDeps: map[string]string{"typescript": "^5.0.0"},
			want:    "React, TypeScript",
		},
		{
			name: "express backend",
			deps: map[string]string{"express": "^4.0.0"},
			want: "Express",
		},
		{
			name: "fastify backend",
			deps: map[string]string{"fastify": "^4.0.0"},
			want: "Fastify",
		},
		{
			name: "hono backend",
			deps: map[string]string{"hono": "^4.0.0"},
			want: "Hono",
		},
		{
			name: "next.js + express + typescript + prisma",
			deps: map[string]string{
				"next":           "^14.0.0",
				"react":          "^18.0.0",
				"express":        "^4.0.0",
				"@prisma/client": "^5.0.0",
			},
			devDeps: map[string]string{"typescript": "^5.0.0"},
			want:    "Next.js, Express, TypeScript, Prisma",
		},
		{
			name:    "prisma via prisma package",
			devDeps: map[string]string{"prisma": "^5.0.0"},
			want:    "Prisma",
		},
		{
			name: "drizzle ORM",
			deps: map[string]string{"drizzle": "^0.30.0"},
			want: "Drizzle",
		},
		{
			name:    "typescript only in devDeps",
			devDeps: map[string]string{"typescript": "^5.0.0"},
			want:    "TypeScript",
		},
		{
			name: "multiple backend frameworks",
			deps: map[string]string{"express": "^4.0.0", "fastify": "^4.0.0"},
			want: "Express, Fastify",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := makePkg(tt.deps, tt.devDeps)
			got := DetectJSTechStack(pkg)
			if got != tt.want {
				t.Errorf("DetectJSTechStack() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildEnrichmentPrompt(t *testing.T) {
	prompt := BuildEnrichmentPrompt("/tmp/my-project", "/tmp/my-project/.claude/CLAUDE.md")

	// Verify it includes the project path and CLAUDE.md path
	if !strings.Contains(prompt, "/tmp/my-project") {
		t.Error("prompt should contain project directory path")
	}
	if !strings.Contains(prompt, "/tmp/my-project/.claude/CLAUDE.md") {
		t.Error("prompt should contain CLAUDE.md path")
	}

	// Verify it includes the expected section headers
	for _, section := range []string{
		"# Project Instructions",
		"## Project",
		"## Key Directories",
		"## Conventions",
		"## Important Files",
		"## Important Notes",
	} {
		if !strings.Contains(prompt, section) {
			t.Errorf("prompt should contain section %q", section)
		}
	}

	// Verify it includes key instructions
	if !strings.Contains(prompt, "under 150 lines") {
		t.Error("prompt should mention the 150 line limit")
	}
	if !strings.Contains(prompt, "no code fences") {
		t.Error("prompt should instruct no code fences")
	}
}

func TestEnrichClaudeMd_MissingCLI(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	os.MkdirAll(claudeDir, 0755)
	os.WriteFile(filepath.Join(claudeDir, "CLAUDE.md"), []byte("# scaffold"), 0644)

	// Save and clear PATH so claude won't be found
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", origPath)

	err := EnrichClaudeMd(dir, claudeDir)
	if err == nil {
		t.Fatal("EnrichClaudeMd() expected error when claude CLI missing")
	}
	if !strings.Contains(err.Error(), "claude CLI not found") {
		t.Errorf("error = %q, want to contain 'claude CLI not found'", err.Error())
	}

	// Verify scaffold is preserved
	data, _ := os.ReadFile(filepath.Join(claudeDir, "CLAUDE.md"))
	if string(data) != "# scaffold" {
		t.Errorf("scaffold was modified: got %q", string(data))
	}
}

func TestEnrichClaudeMd_ScriptReturnsInvalidOutput(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	os.MkdirAll(claudeDir, 0755)
	os.WriteFile(filepath.Join(claudeDir, "CLAUDE.md"), []byte("# scaffold"), 0644)

	// Create a fake "claude" script that outputs non-markdown text
	binDir := filepath.Join(dir, "bin")
	os.MkdirAll(binDir, 0755)
	fakeScript := filepath.Join(binDir, "claude")
	os.WriteFile(fakeScript, []byte("#!/bin/sh\necho 'not markdown'\n"), 0755)

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", binDir)
	defer os.Setenv("PATH", origPath)

	err := EnrichClaudeMd(dir, claudeDir)
	if err == nil {
		t.Fatal("EnrichClaudeMd() expected error for invalid output")
	}
	if !strings.Contains(err.Error(), "enrichment produced no markdown output") {
		t.Errorf("error = %q, want to contain 'enrichment produced no markdown output'", err.Error())
	}

	// Verify scaffold is preserved
	data, _ := os.ReadFile(filepath.Join(claudeDir, "CLAUDE.md"))
	if string(data) != "# scaffold" {
		t.Errorf("scaffold was modified: got %q", string(data))
	}
}

func TestEnrichClaudeMd_ScriptReturnsEmptyOutput(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	os.MkdirAll(claudeDir, 0755)
	os.WriteFile(filepath.Join(claudeDir, "CLAUDE.md"), []byte("# scaffold"), 0644)

	// Create a fake "claude" script that outputs nothing
	binDir := filepath.Join(dir, "bin")
	os.MkdirAll(binDir, 0755)
	fakeScript := filepath.Join(binDir, "claude")
	os.WriteFile(fakeScript, []byte("#!/bin/sh\n"), 0755)

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", binDir)
	defer os.Setenv("PATH", origPath)

	err := EnrichClaudeMd(dir, claudeDir)
	if err == nil {
		t.Fatal("EnrichClaudeMd() expected error for empty output")
	}
	if !strings.Contains(err.Error(), "enrichment produced no markdown output") {
		t.Errorf("error = %q, want to contain 'enrichment produced no markdown output'", err.Error())
	}
}

func TestEnrichClaudeMd_ScriptReturnsValidOutput(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	os.MkdirAll(claudeDir, 0755)
	os.WriteFile(filepath.Join(claudeDir, "CLAUDE.md"), []byte("# scaffold"), 0644)

	// Create a fake "claude" script that outputs valid enriched markdown
	binDir := filepath.Join(dir, "bin")
	os.MkdirAll(binDir, 0755)
	enrichedContent := "# Project Instructions\n\n## Project\nName: test-project"
	fakeScript := filepath.Join(binDir, "claude")
	os.WriteFile(fakeScript, []byte("#!/bin/sh\nprintf '"+enrichedContent+"'\n"), 0755)

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", binDir)
	defer os.Setenv("PATH", origPath)

	err := EnrichClaudeMd(dir, claudeDir)
	if err != nil {
		t.Fatalf("EnrichClaudeMd() unexpected error: %v", err)
	}

	// Verify file was overwritten with enriched content
	data, _ := os.ReadFile(filepath.Join(claudeDir, "CLAUDE.md"))
	if !strings.HasPrefix(string(data), "# Project Instructions") {
		t.Errorf("expected enriched content, got %q", string(data))
	}
}

func TestEnrichClaudeMd_ScriptFailsNonZero(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	os.MkdirAll(claudeDir, 0755)
	os.WriteFile(filepath.Join(claudeDir, "CLAUDE.md"), []byte("# scaffold"), 0644)

	// Create a fake "claude" script that exits non-zero
	binDir := filepath.Join(dir, "bin")
	os.MkdirAll(binDir, 0755)
	fakeScript := filepath.Join(binDir, "claude")
	os.WriteFile(fakeScript, []byte("#!/bin/sh\nexit 1\n"), 0755)

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", binDir)
	defer os.Setenv("PATH", origPath)

	err := EnrichClaudeMd(dir, claudeDir)
	if err == nil {
		t.Fatal("EnrichClaudeMd() expected error for non-zero exit")
	}
	if !strings.Contains(err.Error(), "claude exited with error") {
		t.Errorf("error = %q, want to contain 'claude exited with error'", err.Error())
	}

	// Verify scaffold is preserved
	data, _ := os.ReadFile(filepath.Join(claudeDir, "CLAUDE.md"))
	if string(data) != "# scaffold" {
		t.Errorf("scaffold was modified: got %q", string(data))
	}
}

func TestGenerateClaudeMdScaffold(t *testing.T) {
	dir := t.TempDir()

	// Test with a Go project
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)

	scaffold := GenerateClaudeMdScaffold(dir)

	if !strings.Contains(scaffold, "# Project Instructions") {
		t.Error("scaffold should contain header")
	}
	if !strings.Contains(scaffold, "Tech Stack: Go") {
		t.Error("scaffold should detect Go tech stack")
	}
	if !strings.Contains(scaffold, "Build: `go build ./...`") {
		t.Error("scaffold should include Go build command")
	}
	if !strings.Contains(scaffold, "Test: `go test ./...`") {
		t.Error("scaffold should include Go test command")
	}
}
