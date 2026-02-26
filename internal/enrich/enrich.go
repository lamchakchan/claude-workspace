package enrich

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

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

	// If CLAUDE.md doesn't exist, generate scaffold first
	scaffoldGenerated := false
	if !platform.FileExists(claudeMdPath) {
		content := platform.GenerateClaudeMdScaffold(projectDir)
		if err := os.WriteFile(claudeMdPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("writing scaffold CLAUDE.md: %w", err)
		}
		platform.PrintSuccess(os.Stdout, "Created .claude/CLAUDE.md scaffold")
		scaffoldGenerated = true
	}

	if scaffoldOnly {
		if !scaffoldGenerated {
			platform.PrintWarningLine(os.Stdout, "CLAUDE.md already exists. Skipping scaffold generation.")
		}
		return nil
	}

	// Run AI enrichment
	platform.PrintStep(os.Stdout, 1, 1, "Enriching CLAUDE.md with project context...")
	if err := platform.EnrichClaudeMd(projectDir, claudeDir); err != nil {
		platform.PrintWarningLine(os.Stdout, fmt.Sprintf("Note: %v", err))
		if scaffoldGenerated {
			fmt.Println("  Using static scaffold. Edit .claude/CLAUDE.md to customize.")
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
