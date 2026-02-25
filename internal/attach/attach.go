package attach

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

func Run(targetPath string, allArgs []string) error {
	if targetPath == "" {
		fmt.Fprintln(os.Stderr, "Usage: claude-workspace attach <project-path> [--symlink] [--force] [--no-enrich]")
		os.Exit(1)
	}

	projectDir, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	useSymlinks := contains(allArgs, "--symlink")
	force := contains(allArgs, "--force")
	noEnrich := contains(allArgs, "--no-enrich")

	if !platform.FileExists(projectDir) {
		return fmt.Errorf("project directory not found: %s", projectDir)
	}

	platform.PrintBanner(os.Stdout, fmt.Sprintf("Attaching Claude Platform to: %s", projectDir))
	fmt.Println()

	claudeDir := filepath.Join(projectDir, ".claude")

	// Create directories
	for _, dir := range []string{
		claudeDir,
		filepath.Join(claudeDir, "agents"),
		filepath.Join(claudeDir, "skills"),
		filepath.Join(claudeDir, "hooks"),
		filepath.Join(projectDir, "plans"),
	} {
		os.MkdirAll(dir, 0755)
	}

	// For symlink mode, extract assets first
	var assetBase string
	if useSymlinks {
		assetBase, err = platform.ExtractForSymlink()
		if err != nil {
			return fmt.Errorf("extracting assets for symlink: %w", err)
		}
	}

	// Copy or symlink agents
	platform.PrintStep(os.Stdout, 1, 7, "Setting up agents...")
	if useSymlinks {
		copyOrLinkFromDisk(filepath.Join(assetBase, ".claude", "agents"), filepath.Join(claudeDir, "agents"), true, force, projectDir)
	} else {
		copyFromEmbed(".claude/agents", filepath.Join(claudeDir, "agents"), force, projectDir)
	}

	// Copy or symlink skills
	platform.PrintStep(os.Stdout, 2, 7, "Setting up skills...")
	if useSymlinks {
		copyOrLinkFromDisk(filepath.Join(assetBase, ".claude", "skills"), filepath.Join(claudeDir, "skills"), true, force, projectDir)
	} else {
		copyFromEmbed(".claude/skills", filepath.Join(claudeDir, "skills"), force, projectDir)
	}

	// Copy or symlink hooks
	platform.PrintStep(os.Stdout, 3, 7, "Setting up hooks...")
	if useSymlinks {
		copyOrLinkFromDisk(filepath.Join(assetBase, ".claude", "hooks"), filepath.Join(claudeDir, "hooks"), true, force, projectDir)
	} else {
		copyFromEmbed(".claude/hooks", filepath.Join(claudeDir, "hooks"), force, projectDir)
	}

	// Create or merge settings.json
	platform.PrintStep(os.Stdout, 4, 7, "Setting up settings...")
	setupProjectSettings(claudeDir, force)

	// Create or merge .mcp.json
	platform.PrintStep(os.Stdout, 5, 7, "Setting up MCP configuration...")
	setupMcpConfig(projectDir, force)

	// Create project CLAUDE.md
	platform.PrintStep(os.Stdout, 6, 7, "Setting up CLAUDE.md...")
	setupProjectClaudeMd(projectDir, claudeDir, force)

	// Enrich CLAUDE.md with AI-powered project analysis
	if !noEnrich {
		platform.PrintStep(os.Stdout, 7, 7, "Enriching CLAUDE.md with project context...")
		if err := enrichClaudeMd(projectDir, claudeDir); err != nil {
			platform.PrintWarningLine(os.Stdout, fmt.Sprintf("Note: %v", err))
			fmt.Println("  Using static scaffold. Edit .claude/CLAUDE.md to customize.")
		}
	} else {
		platform.PrintStep(os.Stdout, 7, 7, "Skipping CLAUDE.md enrichment (--no-enrich)")
	}

	// Setup gitignore
	setupGitignore(claudeDir)

	platform.PrintBanner(os.Stdout, "Attachment Complete")
	fmt.Printf("\n%s %s\n", platform.Bold("Platform attached to:"), projectDir)

	platform.PrintSection(os.Stdout, "Start Claude Code")
	platform.PrintCommand(os.Stdout, fmt.Sprintf("cd %s && claude", projectDir))

	platform.PrintSection(os.Stdout, "Customize for this project")
	platform.PrintManual(os.Stdout, fmt.Sprintf("Edit %s for team instructions", filepath.Join(claudeDir, "CLAUDE.md")))
	platform.PrintManual(os.Stdout, "Copy .claude/settings.local.json.example to .claude/settings.local.json for personal overrides")
	fmt.Println()

	return nil
}

// copyFromEmbed copies files from the embedded FS to disk.
func copyFromEmbed(srcDir, destDir string, force bool, projectDir string) {
	cwd, _ := os.Getwd()

	err := fs.WalkDir(platform.FS, srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || path == srcDir {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, _ := filepath.Rel(srcDir, path)
		destFile := filepath.Join(destDir, rel)

		if platform.FileExists(destFile) && !force {
			relFromCwd, _ := filepath.Rel(cwd, destFile)
			if relFromCwd == "" {
				relFromCwd = destFile
			}
			platform.PrintWarningLine(os.Stdout, fmt.Sprintf("Skipping (exists): %s", relFromCwd))
			return nil
		}

		data, err := fs.ReadFile(platform.FS, path)
		if err != nil {
			return err
		}

		os.MkdirAll(filepath.Dir(destFile), 0755)

		perm := os.FileMode(0644)
		if filepath.Ext(path) == ".sh" {
			perm = 0755
		}

		if err := os.WriteFile(destFile, data, perm); err != nil {
			return err
		}

		platform.PrintSuccess(os.Stdout, fmt.Sprintf("Copied: %s", rel))
		return nil
	})

	if err != nil {
		platform.PrintErrorLine(os.Stdout, fmt.Sprintf("Error: %v", err))
	}
}

// copyOrLinkFromDisk copies or symlinks files from a disk directory.
func copyOrLinkFromDisk(src, dest string, symlink, force bool, projectDir string) {
	if !platform.FileExists(src) {
		platform.PrintWarningLine(os.Stdout, fmt.Sprintf("Skipping: %s does not exist", src))
		return
	}

	cwd, _ := os.Getwd()

	platform.WalkFiles(src, func(relPath string) error {
		srcFile := filepath.Join(src, relPath)
		destFile := filepath.Join(dest, relPath)

		if platform.FileExists(destFile) && !force {
			relFromCwd, _ := filepath.Rel(cwd, destFile)
			if relFromCwd == "" {
				relFromCwd = destFile
			}
			platform.PrintWarningLine(os.Stdout, fmt.Sprintf("Skipping (exists): %s", relFromCwd))
			return nil
		}

		if symlink {
			if platform.FileExists(destFile) {
				os.Remove(destFile)
			}
			if err := platform.SymlinkFile(srcFile, destFile); err != nil {
				platform.PrintErrorLine(os.Stdout, fmt.Sprintf("Error symlinking %s: %v", relPath, err))
				return nil
			}
			platform.PrintSuccess(os.Stdout, fmt.Sprintf("Linked: %s", relPath))
		} else {
			if err := platform.CopyFile(srcFile, destFile); err != nil {
				platform.PrintErrorLine(os.Stdout, fmt.Sprintf("Error copying %s: %v", relPath, err))
				return nil
			}
			platform.PrintSuccess(os.Stdout, fmt.Sprintf("Copied: %s", relPath))
		}
		return nil
	})
}

func setupProjectSettings(claudeDir string, force bool) {
	settingsPath := filepath.Join(claudeDir, "settings.json")

	if platform.FileExists(settingsPath) && !force {
		platform.PrintWarningLine(os.Stdout, "Project settings already exist. Use --force to overwrite.")
		return
	}

	// Read platform settings from embedded FS
	data, err := platform.ReadAsset(".claude/settings.json")
	if err != nil {
		platform.PrintErrorLine(os.Stdout, fmt.Sprintf("Error reading embedded settings: %v", err))
		return
	}

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		platform.PrintErrorLine(os.Stdout, fmt.Sprintf("Error writing settings: %v", err))
		return
	}
	platform.PrintSuccess(os.Stdout, "Created .claude/settings.json")

	// Copy the local settings example
	if exampleData, err := platform.ReadAsset(".claude/settings.local.json.example"); err == nil {
		destExample := filepath.Join(claudeDir, "settings.local.json.example")
		if !platform.FileExists(destExample) || force {
			os.WriteFile(destExample, exampleData, 0644)
			platform.PrintSuccess(os.Stdout, "Created .claude/settings.local.json.example")
		}
	}
}

func setupMcpConfig(projectDir string, force bool) {
	mcpPath := filepath.Join(projectDir, ".mcp.json")

	if platform.FileExists(mcpPath) && !force {
		platform.PrintWarningLine(os.Stdout, "MCP config already exists. Use --force to overwrite.")
		return
	}

	data, err := platform.ReadAsset(".mcp.json")
	if err != nil {
		platform.PrintErrorLine(os.Stdout, fmt.Sprintf("Error reading embedded .mcp.json: %v", err))
		return
	}

	if err := os.WriteFile(mcpPath, data, 0644); err != nil {
		platform.PrintErrorLine(os.Stdout, fmt.Sprintf("Error writing .mcp.json: %v", err))
		return
	}
	platform.PrintSuccess(os.Stdout, "Created .mcp.json")
}

func setupProjectClaudeMd(projectDir, claudeDir string, force bool) {
	claudeMdPath := filepath.Join(claudeDir, "CLAUDE.md")

	if platform.FileExists(claudeMdPath) && !force {
		platform.PrintWarningLine(os.Stdout, "Project CLAUDE.md already exists. Use --force to overwrite.")
		return
	}

	projectName := filepath.Base(projectDir)
	techStack := "Unknown"
	buildCmd := ""
	testCmd := ""

	// Try to detect from package.json
	pkgPath := filepath.Join(projectDir, "package.json")
	if platform.FileExists(pkgPath) {
		var pkg map[string]json.RawMessage
		if err := platform.ReadJSONFile(pkgPath, &pkg); err == nil {
			techStack = detectTechStack(pkg)
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
	if platform.FileExists(filepath.Join(projectDir, "Cargo.toml")) {
		techStack = "Rust"
		buildCmd = "cargo build"
		testCmd = "cargo test"
	}

	// Try go.mod
	if platform.FileExists(filepath.Join(projectDir, "go.mod")) {
		techStack = "Go"
		buildCmd = "go build ./..."
		testCmd = "go test ./..."
	}

	// Try pyproject.toml
	if platform.FileExists(filepath.Join(projectDir, "pyproject.toml")) {
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

	if err := os.WriteFile(claudeMdPath, []byte(sb.String()), 0644); err != nil {
		platform.PrintErrorLine(os.Stdout, fmt.Sprintf("Error writing CLAUDE.md: %v", err))
		return
	}
	platform.PrintSuccess(os.Stdout, "Created .claude/CLAUDE.md (customize for your project)")

	// Copy local example
	if exampleData, err := platform.ReadAsset(".claude/CLAUDE.local.md.example"); err == nil {
		destExample := filepath.Join(claudeDir, "CLAUDE.local.md.example")
		if !platform.FileExists(destExample) || force {
			os.WriteFile(destExample, exampleData, 0644)
			platform.PrintSuccess(os.Stdout, "Created .claude/CLAUDE.local.md.example")
		}
	}
}

func detectTechStack(pkg map[string]json.RawMessage) string {
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

func setupGitignore(claudeDir string) {
	gitignorePath := filepath.Join(claudeDir, ".gitignore")

	if platform.FileExists(gitignorePath) {
		return
	}

	content := `# Personal local settings (not shared)
settings.local.json
CLAUDE.local.md

# Agent memory (personal)
agent-memory-local/

# Example files are tracked
!*.example
`

	os.WriteFile(gitignorePath, []byte(content), 0644)
	platform.PrintSuccess(os.Stdout, "Created .claude/.gitignore")
}

func enrichClaudeMd(projectDir, claudeDir string) error {
	if !platform.Exists("claude") {
		return fmt.Errorf("claude CLI not found. Install with `claude-workspace setup`")
	}

	claudeMdPath := filepath.Join(claudeDir, "CLAUDE.md")
	prompt := buildEnrichmentPrompt(projectDir, claudeMdPath)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	fmt.Println("  Running claude opus to analyze project (up to 180s)...")
	stdout, stderr, err := platform.RunDirWithStdinCapture(ctx, projectDir, prompt, []string{"CLAUDECODE"}, "claude", "-p",
		"--strict-mcp-config", "--mcp-config", `{"mcpServers":{}}`,
		"--output-format", "text",
		"--model", "opus")
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

	platform.PrintSuccess(os.Stdout, "Enriched .claude/CLAUDE.md with project context")
	return nil
}

func buildEnrichmentPrompt(projectDir, claudeMdPath string) string {
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

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
