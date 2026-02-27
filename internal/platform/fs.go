package platform

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FileExists reports whether the named file or directory exists.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// CopyFile copies a single file from src to dst, preserving permissions.
func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// WalkFiles walks a directory and calls fn for each regular file,
// passing the relative path from root.
func WalkFiles(root string, fn func(relPath string) error) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		return fn(rel)
	})
}

// SymlinkFile creates a symlink at dst pointing to src.
// Removes any existing file/symlink at dst first.
func SymlinkFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	// Remove existing
	os.Remove(dst)
	return os.Symlink(src, dst)
}

// EnsureGitignoreEntries creates or updates a .gitignore file so that all
// non-empty, non-comment lines in requiredEntries are present. Missing entries
// are appended under a "# Added by claude-workspace" header. Returns true if
// the file was modified.
func EnsureGitignoreEntries(gitignorePath string, requiredEntries string) (modified bool, err error) {
	existing, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}

	have := parseGitignoreEntries(string(existing))
	missing := findMissingEntries(have, requiredEntries)

	if len(missing) == 0 {
		return false, nil
	}

	if err := os.MkdirAll(filepath.Dir(gitignorePath), 0755); err != nil {
		return false, err
	}

	content := appendGitignoreBlock(existing, missing)
	if err := os.WriteFile(gitignorePath, content, 0644); err != nil {
		return false, err
	}
	return true, nil
}

// parseGitignoreEntries parses gitignore content into a set of non-empty, non-comment lines.
func parseGitignoreEntries(content string) map[string]bool {
	have := make(map[string]bool)
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			have[trimmed] = true
		}
	}
	return have
}

// findMissingEntries returns required entries that are not present in the existing set.
func findMissingEntries(existing map[string]bool, required string) []string {
	var missing []string
	for _, line := range strings.Split(required, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !existing[trimmed] {
			missing = append(missing, trimmed)
		}
	}
	return missing
}

// appendGitignoreBlock appends missing entries under a header to the existing file content.
func appendGitignoreBlock(existing []byte, missing []string) []byte {
	var buf strings.Builder
	if len(existing) > 0 && !strings.HasSuffix(string(existing), "\n") {
		buf.WriteString("\n")
	}
	buf.WriteString("\n# Added by claude-workspace\n")
	for _, entry := range missing {
		buf.WriteString(entry + "\n")
	}
	return append(existing, []byte(buf.String())...)
}

// IsExecutable checks if a file has any execute permission bit set.
func IsExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode()&0111 != 0
}
