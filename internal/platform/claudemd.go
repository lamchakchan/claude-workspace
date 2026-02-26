package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DetectJSTechStack scans package.json deps for known frameworks.
func DetectJSTechStack(pkg map[string]json.RawMessage) string {
	allDeps := make(map[string]bool)

	for _, key := range []string{"dependencies", "devDependencies"} {
		if raw, ok := pkg[key]; ok {
			var deps map[string]string
			if json.Unmarshal(raw, &deps) == nil {
				for k := range deps {
					allDeps[k] = true
				}
			}
		}
	}

	var stack []string

	if allDeps["next"] {
		stack = append(stack, "Next.js")
	} else if allDeps["react"] {
		stack = append(stack, "React")
	} else if allDeps["vue"] {
		stack = append(stack, "Vue")
	} else if allDeps["@angular/core"] {
		stack = append(stack, "Angular")
	} else if allDeps["svelte"] {
		stack = append(stack, "Svelte")
	}

	if allDeps["express"] {
		stack = append(stack, "Express")
	}
	if allDeps["fastify"] {
		stack = append(stack, "Fastify")
	}
	if allDeps["hono"] {
		stack = append(stack, "Hono")
	}
	if allDeps["typescript"] {
		stack = append(stack, "TypeScript")
	}
	if allDeps["prisma"] || allDeps["@prisma/client"] {
		stack = append(stack, "Prisma")
	}
	if allDeps["drizzle"] {
		stack = append(stack, "Drizzle")
	}

	if len(stack) > 0 {
		return strings.Join(stack, ", ")
	}
	return "Node.js"
}

// GenerateClaudeMdScaffold builds the static scaffold content for a project.
// Returns the markdown string (caller handles file I/O and force logic).
func GenerateClaudeMdScaffold(projectDir string) string {
	projectName := filepath.Base(projectDir)
	techStack := "Unknown"
	buildCmd := ""
	testCmd := ""

	// Try to detect from package.json
	pkgPath := filepath.Join(projectDir, "package.json")
	if FileExists(pkgPath) {
		var pkg map[string]json.RawMessage
		if err := ReadJSONFile(pkgPath, &pkg); err == nil {
			techStack = DetectJSTechStack(pkg)
			var scripts map[string]string
			if raw, ok := pkg["scripts"]; ok {
				json.Unmarshal(raw, &scripts)
			}
			if _, ok := scripts["build"]; ok {
				buildCmd = "npm run build"
			}
			if _, ok := scripts["test"]; ok {
				testCmd = "npm test"
			}
		}
	}

	// Try Cargo.toml
	if FileExists(filepath.Join(projectDir, "Cargo.toml")) {
		techStack = "Rust"
		buildCmd = "cargo build"
		testCmd = "cargo test"
	}

	// Try go.mod
	if FileExists(filepath.Join(projectDir, "go.mod")) {
		techStack = "Go"
		buildCmd = "go build ./..."
		testCmd = "go test ./..."
	}

	// Try pyproject.toml
	if FileExists(filepath.Join(projectDir, "pyproject.toml")) {
		techStack = "Python"
		testCmd = "pytest"
	}

	var sb strings.Builder
	sb.WriteString("# Project Instructions\n\n")
	sb.WriteString("## Project\n")
	sb.WriteString(fmt.Sprintf("Name: %s\n", projectName))
	sb.WriteString(fmt.Sprintf("Tech Stack: %s\n", techStack))
	if buildCmd != "" {
		sb.WriteString(fmt.Sprintf("Build: `%s`\n", buildCmd))
	}
	if testCmd != "" {
		sb.WriteString(fmt.Sprintf("Test: `%s`\n", testCmd))
	}
	sb.WriteString(`
## Conventions
<!-- Add your team's coding conventions here -->

## Key Directories
<!-- Map your project's important directories -->
<!-- Example:
- src/          - Application source code
- tests/        - Test files
- docs/         - Documentation
-->

## Important Notes
<!-- Add project-specific notes for Claude -->
`)

	return sb.String()
}

// BuildEnrichmentPrompt constructs the LLM prompt for AI enrichment.
func BuildEnrichmentPrompt(projectDir, claudeMdPath string) string {
	return fmt.Sprintf(`You are analyzing a software project to generate a CLAUDE.md file with real project context.

The project is located at: %s
There is an existing scaffold at: %s

Your task:
1. Read the existing scaffold at the path above
2. Explore the project: README, dependency files, directory layout, config files, and source files
3. Output ONLY raw markdown (no code fences, no explanations, no preamble) following this exact structure:

# Project Instructions

## Project
Name: <project name>
Purpose: <one-line description from README or package metadata>
Tech Stack: <detected languages/frameworks>
Build: `+"`<build command>`"+`
Test: `+"`<test command>`"+`
Lint: `+"`<lint command if found>`"+`

## Key Directories
- <dir>/ - <description>
(list actual directories found in the project)

## Conventions
- <convention discovered from code>
(e.g., naming patterns, file organization, import style, error handling patterns)

## Important Files
- <file path> - <why it matters>
(list 5-10 files a new developer should read first)

## Important Notes
- <project-specific gotcha or important detail>

Rules:
- Only include information you can verify from the project files
- Do not hallucinate or guess — if unsure, omit the section content
- Keep the total output under 150 lines
- Output raw markdown only — no wrapping code fences, no commentary`, projectDir, claudeMdPath)
}

// EnrichClaudeMd runs claude opus to analyze the project and overwrite CLAUDE.md.
func EnrichClaudeMd(projectDir, claudeDir string) error {
	if !Exists("claude") {
		return fmt.Errorf("claude CLI not found. Install with `claude-workspace setup`")
	}

	claudeMdPath := filepath.Join(claudeDir, "CLAUDE.md")
	prompt := BuildEnrichmentPrompt(projectDir, claudeMdPath)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	spinner := StartSpinner(os.Stderr, "Analyzing project with claude opus (up to 180s)...")
	stdout, stderr, err := RunDirWithStdinCapture(ctx, projectDir, prompt, []string{"CLAUDECODE"}, "claude", "-p",
		"--strict-mcp-config", "--mcp-config", `{"mcpServers":{}}`,
		"--output-format", "text",
		"--model", "opus")
	spinner.Stop()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("enrichment timed out after 180s")
		}
		detail := ""
		if stderr != "" {
			detail = fmt.Sprintf(": %s", stderr)
		}
		return fmt.Errorf("claude exited with error (check API key with `claude-workspace setup`)%s", detail)
	}

	// Find the start of markdown content (skip any preamble lines)
	idx := strings.Index(stdout, "#")
	if idx < 0 {
		if stderr != "" {
			return fmt.Errorf("enrichment produced no markdown output (stderr: %s)", stderr)
		}
		return fmt.Errorf("enrichment produced no markdown output")
	}
	content := stdout[idx:]

	if err := os.WriteFile(claudeMdPath, []byte(content+"\n"), 0644); err != nil {
		return fmt.Errorf("writing enriched CLAUDE.md: %w", err)
	}

	PrintSuccess(os.Stdout, "Enriched .claude/CLAUDE.md with project context")
	return nil
}
