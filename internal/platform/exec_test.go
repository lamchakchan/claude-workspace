package platform

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRun_Success(t *testing.T) {
	err := Run("true")
	if err != nil {
		t.Errorf("Run(true) error = %v", err)
	}
}

func TestRun_Failure(t *testing.T) {
	err := Run("false")
	if err == nil {
		t.Error("Run(false) expected error")
	}
}

func TestRunDir_Success(t *testing.T) {
	dir := t.TempDir()
	err := RunDir(dir, "true")
	if err != nil {
		t.Errorf("RunDir() error = %v", err)
	}
}

func TestRunDir_Failure(t *testing.T) {
	dir := t.TempDir()
	err := RunDir(dir, "false")
	if err == nil {
		t.Error("RunDir(false) expected error")
	}
}

func TestRunQuiet_Success(t *testing.T) {
	err := RunQuiet("true")
	if err != nil {
		t.Errorf("RunQuiet(true) error = %v", err)
	}
}

func TestRunQuiet_Failure(t *testing.T) {
	err := RunQuiet("false")
	if err == nil {
		t.Error("RunQuiet(false) expected error")
	}
}

func TestRunQuietWithEnv_Success(t *testing.T) {
	err := RunQuietWithEnv([]string{"TEST_VAR=hello"}, "true")
	if err != nil {
		t.Errorf("RunQuietWithEnv(true) error = %v", err)
	}
}

func TestRunQuietWithEnv_Failure(t *testing.T) {
	err := RunQuietWithEnv([]string{"TEST_VAR=hello"}, "false")
	if err == nil {
		t.Error("RunQuietWithEnv(false) expected error")
	}
}

func TestRunQuietWithEnv_SetsEnvVar(t *testing.T) {
	// Use env command to verify the extra env var is set
	// env prints all environment variables; we use sh -c to check
	out, err := Output("sh", "-c", "TEST_VAR=check_value env | grep TEST_VAR")
	if err != nil {
		t.Skipf("cannot run env check: %v", err)
	}
	_ = out

	// More targeted: run a shell that exits 0 only if the env var matches
	err = RunQuietWithEnv(
		[]string{"MY_CUSTOM_VAR=expected_value"},
		"sh", "-c", `[ "$MY_CUSTOM_VAR" = "expected_value" ]`,
	)
	if err != nil {
		t.Errorf("RunQuietWithEnv did not set env var correctly: %v", err)
	}
}

func TestRunQuietDir_Success(t *testing.T) {
	dir := t.TempDir()
	err := RunQuietDir(dir, "true")
	if err != nil {
		t.Errorf("RunQuietDir() error = %v", err)
	}
}

func TestRunQuietDir_Failure(t *testing.T) {
	dir := t.TempDir()
	err := RunQuietDir(dir, "false")
	if err == nil {
		t.Error("RunQuietDir(false) expected error")
	}
}

func TestRunQuietDir_RunsInDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "marker.txt"), []byte("x"), 0644)

	// ls marker.txt should succeed in the right directory
	err := RunQuietDir(dir, "ls", "marker.txt")
	if err != nil {
		t.Errorf("RunQuietDir() should find marker.txt in dir: %v", err)
	}

	// ls marker.txt should fail in a different directory
	otherDir := t.TempDir()
	err = RunQuietDir(otherDir, "ls", "marker.txt")
	if err == nil {
		t.Error("RunQuietDir() should not find marker.txt in wrong dir")
	}
}

func TestOutput(t *testing.T) {
	out, err := Output("echo", "hello world")
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if out != "hello world" {
		t.Errorf("Output() = %q, want %q", out, "hello world")
	}
}

func TestOutput_TrimsWhitespace(t *testing.T) {
	out, err := Output("printf", "  padded  \n")
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if out != "padded" {
		t.Errorf("Output() = %q, want %q", out, "padded")
	}
}

func TestOutput_CommandNotFound(t *testing.T) {
	_, err := Output("nonexistent_command_xyz_12345")
	if err == nil {
		t.Error("Output() expected error for nonexistent command")
	}
}

func TestOutput_NonZeroExit(t *testing.T) {
	_, err := Output("false")
	if err == nil {
		t.Error("Output(false) expected error")
	}
}

func TestOutputDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "marker.txt"), []byte("x"), 0644)

	out, err := OutputDir(dir, "ls", "marker.txt")
	if err != nil {
		t.Fatalf("OutputDir() error = %v", err)
	}
	if out != "marker.txt" {
		t.Errorf("OutputDir() = %q, want %q", out, "marker.txt")
	}
}

func TestOutputDir_WrongDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "marker.txt"), []byte("x"), 0644)

	otherDir := t.TempDir()
	_, err := OutputDir(otherDir, "ls", "marker.txt")
	if err == nil {
		t.Error("OutputDir() expected error when file not in dir")
	}
}

func TestExists(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{"go exists", "go", true},
		{"echo exists", "echo", true},
		{"nonexistent command", "nonexistent_command_xyz_12345", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Exists(tt.command)
			if got != tt.want {
				t.Errorf("Exists(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

func TestRunSpawn_Success(t *testing.T) {
	code, err := RunSpawn("true")
	if err != nil {
		t.Fatalf("RunSpawn() error = %v", err)
	}
	if code != 0 {
		t.Errorf("RunSpawn(true) exit code = %d, want 0", code)
	}
}

func TestRunSpawn_NonZeroExit(t *testing.T) {
	code, err := RunSpawn("false")
	if err != nil {
		t.Fatalf("RunSpawn(false) unexpected error = %v", err)
	}
	if code == 0 {
		t.Error("RunSpawn(false) exit code = 0, want non-zero")
	}
}

func TestRunSpawn_CommandNotFound(t *testing.T) {
	code, err := RunSpawn("nonexistent_command_xyz_12345")
	if err == nil {
		t.Error("RunSpawn() expected error for nonexistent command")
	}
	if code != -1 {
		t.Errorf("RunSpawn() exit code = %d, want -1", code)
	}
}

func TestRunDirWithStdin(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	out, err := RunDirWithStdin(ctx, dir, "hello from stdin", "cat")
	if err != nil {
		t.Fatalf("RunDirWithStdin() error = %v", err)
	}
	if out != "hello from stdin" {
		t.Errorf("RunDirWithStdin() = %q, want %q", out, "hello from stdin")
	}
}

func TestRunDirWithStdin_TrimsWhitespace(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	out, err := RunDirWithStdin(ctx, dir, "  padded  \n", "cat")
	if err != nil {
		t.Fatalf("RunDirWithStdin() error = %v", err)
	}
	if out != "padded" {
		t.Errorf("RunDirWithStdin() = %q, want %q", out, "padded")
	}
}

func TestRunDirWithStdin_RunsInDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "marker.txt"), []byte("x"), 0644)
	ctx := context.Background()

	out, err := RunDirWithStdin(ctx, dir, "", "ls", "marker.txt")
	if err != nil {
		t.Fatalf("RunDirWithStdin() error = %v", err)
	}
	if out != "marker.txt" {
		t.Errorf("RunDirWithStdin() = %q, want %q", out, "marker.txt")
	}
}

func TestRunDirWithStdin_WrongDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "marker.txt"), []byte("x"), 0644)
	ctx := context.Background()

	otherDir := t.TempDir()
	_, err := RunDirWithStdin(ctx, otherDir, "", "ls", "marker.txt")
	if err == nil {
		t.Error("RunDirWithStdin() expected error when file not in dir")
	}
}

func TestRunDirWithStdin_CommandNotFound(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	_, err := RunDirWithStdin(ctx, dir, "input", "nonexistent_command_xyz_12345")
	if err == nil {
		t.Error("RunDirWithStdin() expected error for nonexistent command")
	}
}

func TestRunDirWithStdin_NonZeroExit(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	_, err := RunDirWithStdin(ctx, dir, "", "false")
	if err == nil {
		t.Error("RunDirWithStdin(false) expected error")
	}
}

func TestRunDirWithStdin_ContextTimeout(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := RunDirWithStdin(ctx, dir, "", "sleep", "10")
	if err == nil {
		t.Error("RunDirWithStdin() expected error on timeout")
	}
	if ctx.Err() != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded, got %v", ctx.Err())
	}
}

func TestRunDirWithStdin_MultilineStdin(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	// wc -l counts lines from stdin
	out, err := RunDirWithStdin(ctx, dir, "line1\nline2\nline3\n", "wc", "-l")
	if err != nil {
		t.Fatalf("RunDirWithStdin() error = %v", err)
	}
	if out != "3" {
		t.Errorf("RunDirWithStdin() = %q, want %q", out, "3")
	}
}
