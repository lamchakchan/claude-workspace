// Package enrich implements the "enrich" command, which generates or regenerates
// a project's .claude/CLAUDE.md file using AI-powered analysis of the codebase.
package enrich

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

// Run executes the enrich command for the given project path. It generates a
// static scaffold if one does not exist, then optionally enriches it with AI
// analysis. Pass --scaffold-only in args to skip AI enrichment.
//
// If .claude/CLAUDE.md already exists, the scaffold and enrichment target
// .claude/rules/platform.md instead (non-destructive).
func Run(projectPath string, args []string) error {
	scaffoldOnly := contains(args, "--scaffold-only")

	// Resolve project dir (default to cwd)
	projectDir := projectPath
	if projectDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}
		projectDir = cwd
	}

	var err error
	projectDir, err = filepath.Abs(projectDir)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	if !platform.FileExists(projectDir) {
		return fmt.Errorf("project directory not found: %s", projectDir)
	}

	claudeDir := filepath.Join(projectDir, ".claude")
	claudeMdPath := filepath.Join(claudeDir, "CLAUDE.md")

	// Ensure .claude/ dir exists
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("creating .claude directory: %w", err)
	}

	// Determine target: if CLAUDE.md exists, write to rules/platform.md instead
	targetPath := claudeMdPath
	scaffoldGenerated := false
	if platform.FileExists(claudeMdPath) {
		rulesDir := filepath.Join(claudeDir, "rules")
		if err := os.MkdirAll(rulesDir, 0755); err != nil {
			return fmt.Errorf("creating .claude/rules directory: %w", err)
		}
		targetPath = filepath.Join(rulesDir, "platform.md")
		platform.PrintWarningLine(os.Stdout, "CLAUDE.md already exists. Targeting .claude/rules/platform.md")
	}

	// Generate scaffold if target doesn't exist
	if !platform.FileExists(targetPath) {
		content := platform.GenerateClaudeMdScaffold(projectDir)
		if err := os.WriteFile(targetPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("writing scaffold: %w", err)
		}
		relTarget, _ := filepath.Rel(projectDir, targetPath)
		platform.PrintSuccess(os.Stdout, fmt.Sprintf("Created %s scaffold", relTarget))
		scaffoldGenerated = true
	}

	if scaffoldOnly {
		if !scaffoldGenerated {
			relTarget, _ := filepath.Rel(projectDir, targetPath)
			platform.PrintWarningLine(os.Stdout, fmt.Sprintf("%s already exists. Skipping scaffold generation.", relTarget))
		}
		return nil
	}

	// Run AI enrichment
	relTarget, _ := filepath.Rel(projectDir, targetPath)
	platform.PrintStep(os.Stdout, 1, 1, fmt.Sprintf("Enriching %s with project context...", relTarget))
	if err := platform.EnrichClaudeMd(projectDir, targetPath); err != nil {
		platform.PrintWarningLine(os.Stdout, fmt.Sprintf("Note: %v", err))
		if scaffoldGenerated {
			fmt.Printf("  Using static scaffold. Edit %s to customize.\n", relTarget)
		}
		return nil
	}

	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
