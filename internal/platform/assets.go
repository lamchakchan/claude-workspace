// Package platform provides shared utilities for file operations, shell execution,
// color output, JSON handling, environment detection, CLAUDE.md generation, and
// package manager integration. It serves as the common foundation imported by all
// CLI subcommand packages.
package platform

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// FS is set by main to the embedded project-level filesystem (_template/project).
var FS fs.FS

// GlobalFS is set by main to the embedded global-level filesystem (_template/global).
var GlobalFS fs.FS

// ExtractTo extracts files from the embedded FS srcDir to destDir on disk.
// If force is false, existing files are skipped.
func ExtractTo(efs fs.FS, srcDir, destDir string, force bool) error {
	return fs.WalkDir(efs, srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Compute relative path from srcDir
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(destDir, rel)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		// Skip existing files unless force
		if !force {
			if _, err := os.Stat(destPath); err == nil {
				return nil
			}
		}

		data, err := fs.ReadFile(efs, path)
		if err != nil {
			return fmt.Errorf("reading embedded %s: %w", path, err)
		}

		// Preserve executable bit for .sh files
		perm := os.FileMode(0644)
		if filepath.Ext(path) == ".sh" {
			perm = 0755
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		return os.WriteFile(destPath, data, perm)
	})
}

// ExtractForSymlink extracts embedded assets to ~/.claude-workspace/assets/
// and returns the path. Used by attach --symlink to create a shared cache.
func ExtractForSymlink() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	cacheDir := filepath.Join(home, ".claude-workspace", "assets")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}

	// Always re-extract to update cached assets
	if err := ExtractTo(FS, ".claude", filepath.Join(cacheDir, ".claude"), true); err != nil {
		return "", fmt.Errorf("extracting .claude assets: %w", err)
	}

	// Extract .mcp.json
	data, err := fs.ReadFile(FS, ".mcp.json")
	if err != nil {
		return "", fmt.Errorf("reading embedded .mcp.json: %w", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, ".mcp.json"), data, 0644); err != nil {
		return "", fmt.Errorf("writing .mcp.json: %w", err)
	}

	return cacheDir, nil
}

// ReadAsset reads a file from the embedded FS and returns its contents.
func ReadAsset(path string) ([]byte, error) {
	return fs.ReadFile(FS, path)
}

// ReadGlobalAsset reads a file from the embedded GlobalFS and returns its contents.
func ReadGlobalAsset(path string) ([]byte, error) {
	return fs.ReadFile(GlobalFS, path)
}

// WalkAssets walks the embedded FS directory and calls fn for each file.
func WalkAssets(dir string, fn func(path string, d fs.DirEntry) error) error {
	return fs.WalkDir(FS, dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == dir {
			return nil
		}
		return fn(path, d)
	})
}
