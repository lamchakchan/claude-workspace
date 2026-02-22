package attach

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

func Run(targetPath string, allArgs []string) error {
	if targetPath == "" {
		fmt.Fprintln(os.Stderr, "Usage: claude-workspace attach <project-path> [--symlink] [--force]")
		os.Exit(1)
	}

	projectDir, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	useSymlinks := contains(allArgs, "--symlink")
	force := contains(allArgs, "--force")

	if !platform.FileExists(projectDir) {
		return fmt.Errorf("project directory not found: %s", projectDir)
	}

	fmt.Printf("\n=== Attaching Claude Platform to: %s ===\n\n", projectDir)

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
	fmt.Println("[1/6] Setting up agents...")
	if useSymlinks {
		copyOrLinkFromDisk(filepath.Join(assetBase, ".claude", "agents"), filepath.Join(claudeDir, "agents"), true, force, projectDir)
	} else {
		copyFromEmbed(".claude/agents", filepath.Join(claudeDir, "agents"), force, projectDir)
	}

	// Copy or symlink skills
	fmt.Println("[2/6] Setting up skills...")
	if useSymlinks {
		copyOrLinkFromDisk(filepath.Join(assetBase, ".claude", "skills"), filepath.Join(claudeDir, "skills"), true, force, projectDir)
	} else {
		copyFromEmbed(".claude/skills", filepath.Join(claudeDir, "skills"), force, projectDir)
	}

	// Copy or symlink hooks
	fmt.Println("[3/6] Setting up hooks...")
	if useSymlinks {
		copyOrLinkFromDisk(filepath.Join(assetBase, ".claude", "hooks"), filepath.Join(claudeDir, "hooks"), true, force, projectDir)
	} else {
		copyFromEmbed(".claude/hooks", filepath.Join(claudeDir, "hooks"), force, projectDir)
	}

	// Create or merge settings.json
	fmt.Println("[4/6] Setting up settings...")
	setupProjectSettings(claudeDir, force)

	// Create or merge .mcp.json
	fmt.Println("[5/6] Setting up MCP configuration...")
	setupMcpConfig(projectDir, force)

	// Create project CLAUDE.md
	fmt.Println("[6/6] Setting up CLAUDE.md...")
	setupProjectClaudeMd(projectDir, claudeDir, force)

	// Setup gitignore
	setupGitignore(claudeDir)

	fmt.Println("\n=== Attachment Complete ===")
	fmt.Printf("\nPlatform attached to: %s\n", projectDir)
	fmt.Println("\nTo start Claude Code:")
	fmt.Printf("  cd %s\n", projectDir)
	fmt.Println("  claude")
	fmt.Println("\nTo customize for this project:")
	fmt.Printf("  Edit %s for team instructions\n", filepath.Join(claudeDir, "CLAUDE.md"))
	fmt.Println("  Copy .claude/settings.local.json.example to .claude/settings.local.json for personal overrides")
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
			fmt.Printf("  Skipping (exists): %s\n", relFromCwd)
			return nil
		}

		data, err := platform.FS.ReadFile(path)
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

		fmt.Printf("  Copied: %s\n", rel)
		return nil
	})

	if err != nil {
		fmt.Printf("  Error: %v\n", err)
	}
}

// copyOrLinkFromDisk copies or symlinks files from a disk directory.
func copyOrLinkFromDisk(src, dest string, symlink, force bool, projectDir string) {
	if !platform.FileExists(src) {
		fmt.Printf("  Skipping: %s does not exist\n", src)
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
			fmt.Printf("  Skipping (exists): %s\n", relFromCwd)
			return nil
		}

		if symlink {
			if platform.FileExists(destFile) {
				os.Remove(destFile)
			}
			if err := platform.SymlinkFile(srcFile, destFile); err != nil {
				fmt.Printf("  Error symlinking %s: %v\n", relPath, err)
				return nil
			}
			fmt.Printf("  Linked: %s\n", relPath)
		} else {
			if err := platform.CopyFile(srcFile, destFile); err != nil {
				fmt.Printf("  Error copying %s: %v\n", relPath, err)
				return nil
			}
			fmt.Printf("  Copied: %s\n", relPath)
		}
		return nil
	})
}

func setupProjectSettings(claudeDir string, force bool) {
	settingsPath := filepath.Join(claudeDir, "settings.json")

	if platform.FileExists(settingsPath) && !force {
		fmt.Println("  Project settings already exist. Use --force to overwrite.")
		return
	}

	// Read platform settings from embedded FS
	data, err := platform.ReadAsset(".claude/settings.json")
	if err != nil {
		fmt.Printf("  Error reading embedded settings: %v\n", err)
		return
	}

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		fmt.Printf("  Error writing settings: %v\n", err)
		return
	}
	fmt.Println("  Created .claude/settings.json")

	// Copy the local settings example
	if exampleData, err := platform.ReadAsset(".claude/settings.local.json.example"); err == nil {
		destExample := filepath.Join(claudeDir, "settings.local.json.example")
		if !platform.FileExists(destExample) || force {
			os.WriteFile(destExample, exampleData, 0644)
			fmt.Println("  Created .claude/settings.local.json.example")
		}
	}
}

func setupMcpConfig(projectDir string, force bool) {
	mcpPath := filepath.Join(projectDir, ".mcp.json")

	if platform.FileExists(mcpPath) && !force {
		fmt.Println("  MCP config already exists. Use --force to overwrite.")
		return
	}

	data, err := platform.ReadAsset(".mcp.json")
	if err != nil {
		fmt.Printf("  Error reading embedded .mcp.json: %v\n", err)
		return
	}

	if err := os.WriteFile(mcpPath, data, 0644); err != nil {
		fmt.Printf("  Error writing .mcp.json: %v\n", err)
		return
	}
	fmt.Println("  Created .mcp.json")
}

func setupProjectClaudeMd(projectDir, claudeDir string, force bool) {
	claudeMdPath := filepath.Join(claudeDir, "CLAUDE.md")

	if platform.FileExists(claudeMdPath) && !force {
		fmt.Println("  Project CLAUDE.md already exists. Use --force to overwrite.")
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
		fmt.Printf("  Error writing CLAUDE.md: %v\n", err)
		return
	}
	fmt.Println("  Created .claude/CLAUDE.md (customize for your project)")

	// Copy local example
	if exampleData, err := platform.ReadAsset(".claude/CLAUDE.local.md.example"); err == nil {
		destExample := filepath.Join(claudeDir, "CLAUDE.local.md.example")
		if !platform.FileExists(destExample) || force {
			os.WriteFile(destExample, exampleData, 0644)
			fmt.Println("  Created .claude/CLAUDE.local.md.example")
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
	fmt.Println("  Created .claude/.gitignore")
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
