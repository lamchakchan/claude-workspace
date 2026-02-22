package platform

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

//go:embed testdata
var testFS embed.FS

func TestExtractTo(t *testing.T) {
	dir := t.TempDir()

	err := ExtractTo(testFS, "testdata", dir, false)
	if err != nil {
		t.Fatalf("ExtractTo() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "hello.txt"))
	if err != nil {
		t.Fatalf("reading hello.txt: %v", err)
	}
	if string(content) != "hello world\n" {
		t.Errorf("hello.txt = %q, want %q", content, "hello world\n")
	}

	content, err = os.ReadFile(filepath.Join(dir, "subdir", "nested.txt"))
	if err != nil {
		t.Fatalf("reading nested.txt: %v", err)
	}
	if string(content) != "nested content\n" {
		t.Errorf("nested.txt = %q, want %q", content, "nested content\n")
	}

	content, err = os.ReadFile(filepath.Join(dir, "script.sh"))
	if err != nil {
		t.Fatalf("reading script.sh: %v", err)
	}
	if string(content) != "#!/bin/bash\necho hi\n" {
		t.Errorf("script.sh = %q, want %q", content, "#!/bin/bash\necho hi\n")
	}
}

func TestExtractTo_ShFilePermissions(t *testing.T) {
	dir := t.TempDir()
	ExtractTo(testFS, "testdata", dir, false)

	info, err := os.Stat(filepath.Join(dir, "script.sh"))
	if err != nil {
		t.Fatalf("Stat script.sh: %v", err)
	}
	if info.Mode().Perm() != 0755 {
		t.Errorf("script.sh permissions = %o, want 0755", info.Mode().Perm())
	}

	info, err = os.Stat(filepath.Join(dir, "hello.txt"))
	if err != nil {
		t.Fatalf("Stat hello.txt: %v", err)
	}
	if info.Mode().Perm() != 0644 {
		t.Errorf("hello.txt permissions = %o, want 0644", info.Mode().Perm())
	}
}

func TestExtractTo_NoForceSkipsExisting(t *testing.T) {
	dir := t.TempDir()
	ExtractTo(testFS, "testdata", dir, false)

	// Modify a file
	os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("modified"), 0644)

	// Re-extract without force
	ExtractTo(testFS, "testdata", dir, false)

	content, _ := os.ReadFile(filepath.Join(dir, "hello.txt"))
	if string(content) != "modified" {
		t.Error("ExtractTo(force=false) should not overwrite existing files")
	}
}

func TestExtractTo_ForceOverwrites(t *testing.T) {
	dir := t.TempDir()
	ExtractTo(testFS, "testdata", dir, false)

	os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("modified"), 0644)

	ExtractTo(testFS, "testdata", dir, true)

	content, _ := os.ReadFile(filepath.Join(dir, "hello.txt"))
	if string(content) != "hello world\n" {
		t.Errorf("ExtractTo(force=true) should overwrite, got %q", content)
	}
}

func TestExtractTo_CreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	destDir := filepath.Join(dir, "deep", "nested", "output")

	err := ExtractTo(testFS, "testdata", destDir, false)
	if err != nil {
		t.Fatalf("ExtractTo() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(destDir, "hello.txt")); err != nil {
		t.Error("ExtractTo() should create parent directories")
	}
}

func TestReadAsset(t *testing.T) {
	oldFS := FS
	FS = testFS
	defer func() { FS = oldFS }()

	data, err := ReadAsset("testdata/hello.txt")
	if err != nil {
		t.Fatalf("ReadAsset() error = %v", err)
	}
	if string(data) != "hello world\n" {
		t.Errorf("ReadAsset() = %q, want %q", data, "hello world\n")
	}
}

func TestReadAsset_NotFound(t *testing.T) {
	oldFS := FS
	FS = testFS
	defer func() { FS = oldFS }()

	_, err := ReadAsset("testdata/nonexistent.txt")
	if err == nil {
		t.Error("ReadAsset() expected error for nonexistent file")
	}
}

func TestWalkAssets(t *testing.T) {
	oldFS := FS
	FS = testFS
	defer func() { FS = oldFS }()

	var paths []string
	err := WalkAssets("testdata", func(path string, d fs.DirEntry) error {
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkAssets() error = %v", err)
	}

	sort.Strings(paths)

	// Should include the subdir entry, hello.txt, script.sh, and nested.txt
	wantFiles := []string{
		"testdata/hello.txt",
		"testdata/script.sh",
		"testdata/subdir",
		"testdata/subdir/nested.txt",
	}
	sort.Strings(wantFiles)

	if len(paths) != len(wantFiles) {
		t.Fatalf("WalkAssets() returned %d entries, want %d: %v", len(paths), len(wantFiles), paths)
	}
	for i := range wantFiles {
		if paths[i] != wantFiles[i] {
			t.Errorf("WalkAssets() path[%d] = %q, want %q", i, paths[i], wantFiles[i])
		}
	}
}

func TestWalkAssets_SkipsRoot(t *testing.T) {
	oldFS := FS
	FS = testFS
	defer func() { FS = oldFS }()

	var paths []string
	WalkAssets("testdata", func(path string, d fs.DirEntry) error {
		paths = append(paths, path)
		return nil
	})

	for _, p := range paths {
		if p == "testdata" {
			t.Error("WalkAssets() should skip the root directory itself")
		}
	}
}
