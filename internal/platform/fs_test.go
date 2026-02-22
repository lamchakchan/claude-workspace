package platform

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "exists.txt")
	os.WriteFile(f, []byte("hi"), 0644)

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"existing file", f, true},
		{"existing directory", dir, true},
		{"nonexistent file", filepath.Join(dir, "nope.txt"), false},
		{"nonexistent nested", filepath.Join(dir, "a", "b", "nope.txt"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FileExists(tt.path)
			if got != tt.want {
				t.Errorf("FileExists(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "sub", "dst.txt")

	content := []byte("hello copy")
	os.WriteFile(src, content, 0755)

	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile() error = %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("reading dst: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("CopyFile() content = %q, want %q", got, content)
	}

	srcInfo, _ := os.Stat(src)
	dstInfo, _ := os.Stat(dst)
	if srcInfo.Mode().Perm() != dstInfo.Mode().Perm() {
		t.Errorf("CopyFile() mode = %o, want %o", dstInfo.Mode().Perm(), srcInfo.Mode().Perm())
	}
}

func TestCopyFile_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "a", "b", "c", "dst.txt")

	os.WriteFile(src, []byte("deep"), 0644)

	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile() error = %v", err)
	}

	got, _ := os.ReadFile(dst)
	if string(got) != "deep" {
		t.Errorf("CopyFile() content = %q, want %q", got, "deep")
	}
}

func TestCopyFile_NonexistentSrc(t *testing.T) {
	dir := t.TempDir()
	err := CopyFile(filepath.Join(dir, "nope.txt"), filepath.Join(dir, "dst.txt"))
	if err == nil {
		t.Error("CopyFile() expected error for nonexistent source")
	}
}

func TestCopyFile_OverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")

	os.WriteFile(src, []byte("new content"), 0644)
	os.WriteFile(dst, []byte("old content"), 0644)

	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile() error = %v", err)
	}

	got, _ := os.ReadFile(dst)
	if string(got) != "new content" {
		t.Errorf("CopyFile() should overwrite, got %q", got)
	}
}

func TestWalkFiles(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "sub", "deep"), 0755)
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "sub", "b.txt"), []byte("b"), 0644)
	os.WriteFile(filepath.Join(dir, "sub", "deep", "c.txt"), []byte("c"), 0644)

	var files []string
	err := WalkFiles(dir, func(relPath string) error {
		files = append(files, relPath)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkFiles() error = %v", err)
	}

	sort.Strings(files)
	want := []string{
		"a.txt",
		filepath.Join("sub", "b.txt"),
		filepath.Join("sub", "deep", "c.txt"),
	}
	sort.Strings(want)

	if len(files) != len(want) {
		t.Fatalf("WalkFiles() found %d files, want %d: %v", len(files), len(want), files)
	}
	for i := range want {
		if files[i] != want[i] {
			t.Errorf("WalkFiles() file[%d] = %q, want %q", i, files[i], want[i])
		}
	}
}

func TestWalkFiles_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	var files []string
	err := WalkFiles(dir, func(relPath string) error {
		files = append(files, relPath)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkFiles() error = %v", err)
	}
	if len(files) != 0 {
		t.Errorf("WalkFiles() found %d files in empty dir, want 0", len(files))
	}
}

func TestWalkFiles_SkipsDirectories(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "emptydir"), 0755)
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("x"), 0644)

	var files []string
	err := WalkFiles(dir, func(relPath string) error {
		files = append(files, relPath)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkFiles() error = %v", err)
	}
	if len(files) != 1 || files[0] != "file.txt" {
		t.Errorf("WalkFiles() should only return files, got %v", files)
	}
}

func TestSymlinkFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "original.txt")
	dst := filepath.Join(dir, "link.txt")

	os.WriteFile(src, []byte("target"), 0644)

	if err := SymlinkFile(src, dst); err != nil {
		t.Fatalf("SymlinkFile() error = %v", err)
	}

	info, err := os.Lstat(dst)
	if err != nil {
		t.Fatalf("Lstat() error = %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("SymlinkFile() did not create a symlink")
	}

	target, err := os.Readlink(dst)
	if err != nil {
		t.Fatalf("Readlink() error = %v", err)
	}
	if target != src {
		t.Errorf("SymlinkFile() target = %q, want %q", target, src)
	}
}

func TestSymlinkFile_OverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	src1 := filepath.Join(dir, "first.txt")
	src2 := filepath.Join(dir, "second.txt")
	dst := filepath.Join(dir, "link.txt")

	os.WriteFile(src1, []byte("first"), 0644)
	os.WriteFile(src2, []byte("second"), 0644)

	SymlinkFile(src1, dst)

	if err := SymlinkFile(src2, dst); err != nil {
		t.Fatalf("SymlinkFile() error on overwrite = %v", err)
	}

	target, _ := os.Readlink(dst)
	if target != src2 {
		t.Errorf("SymlinkFile() after overwrite target = %q, want %q", target, src2)
	}
}

func TestSymlinkFile_OverwritesRegularFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")

	os.WriteFile(src, []byte("source"), 0644)
	os.WriteFile(dst, []byte("will be replaced"), 0644)

	if err := SymlinkFile(src, dst); err != nil {
		t.Fatalf("SymlinkFile() error = %v", err)
	}

	info, _ := os.Lstat(dst)
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("SymlinkFile() should replace regular file with symlink")
	}
}

func TestSymlinkFile_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "a", "b", "link.txt")

	os.WriteFile(src, []byte("x"), 0644)

	if err := SymlinkFile(src, dst); err != nil {
		t.Fatalf("SymlinkFile() error = %v", err)
	}

	if _, err := os.Lstat(dst); err != nil {
		t.Errorf("symlink not created at nested path: %v", err)
	}
}

func TestIsExecutable(t *testing.T) {
	dir := t.TempDir()

	execFile := filepath.Join(dir, "run.sh")
	os.WriteFile(execFile, []byte("#!/bin/bash"), 0755)

	noExecFile := filepath.Join(dir, "data.txt")
	os.WriteFile(noExecFile, []byte("data"), 0644)

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"executable file", execFile, true},
		{"non-executable file", noExecFile, false},
		{"nonexistent file", filepath.Join(dir, "nope"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsExecutable(tt.path)
			if got != tt.want {
				t.Errorf("IsExecutable(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
