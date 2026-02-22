package setup

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDetectNpmClaude_PathHeuristic(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	// Create a fake claude binary inside a node_modules/.bin directory
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "node_modules", ".bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}

	fakeClaude := filepath.Join(binDir, "claude")
	if err := os.WriteFile(fakeClaude, []byte("#!/bin/sh\necho fake"), 0755); err != nil {
		t.Fatal(err)
	}

	// Prepend the fake bin dir to PATH
	origPath := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", origPath) })
	os.Setenv("PATH", binDir+":"+origPath)

	info := DetectNpmClaude()
	if !info.Detected {
		t.Fatal("expected Detected=true for node_modules path")
	}
	if info.Source != "path-heuristic" {
		t.Fatalf("expected Source='path-heuristic', got %q", info.Source)
	}
	if info.Path != fakeClaude {
		t.Errorf("expected Path=%q, got %q", fakeClaude, info.Path)
	}
}

func TestDetectNpmClaude_PathHeuristicGlobalNpm(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	// Simulate npm global install: /usr/local/lib/node_modules/.bin/claude
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "lib", "node_modules", "@anthropic-ai", "claude-code", "cli")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}

	fakeClaude := filepath.Join(binDir, "claude")
	if err := os.WriteFile(fakeClaude, []byte("#!/bin/sh\necho fake"), 0755); err != nil {
		t.Fatal(err)
	}

	origPath := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", origPath) })
	os.Setenv("PATH", binDir+":"+origPath)

	info := DetectNpmClaude()
	if !info.Detected {
		t.Fatal("expected Detected=true for global npm node_modules path")
	}
	if info.Source != "path-heuristic" {
		t.Fatalf("expected Source='path-heuristic', got %q", info.Source)
	}
}

func TestDetectNpmClaude_NoClaude(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	// Set PATH to an empty temp dir so neither claude nor npm is found
	tmpDir := t.TempDir()
	origPath := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", origPath) })
	os.Setenv("PATH", tmpDir)

	info := DetectNpmClaude()
	if info.Detected {
		t.Fatal("expected Detected=false when claude is not in PATH")
	}
	if info.Path != "" {
		t.Errorf("expected empty Path, got %q", info.Path)
	}
	if info.Source != "" {
		t.Errorf("expected empty Source, got %q", info.Source)
	}
}

func TestDetectNpmClaude_RegularPathNotDetected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	// Claude in a regular directory (e.g. ~/.local/bin) with no npm available
	// should NOT be detected as npm-installed
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, ".local", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}

	fakeClaude := filepath.Join(binDir, "claude")
	if err := os.WriteFile(fakeClaude, []byte("#!/bin/sh\necho fake"), 0755); err != nil {
		t.Fatal(err)
	}

	origPath := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", origPath) })
	// Only include the bin dir — no npm available
	os.Setenv("PATH", binDir)

	info := DetectNpmClaude()
	if info.Detected {
		t.Fatal("expected Detected=false for regular binary path without npm")
	}
}

func TestDetectNpmClaude_AsdfShimWithoutNpm(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	// Claude via asdf shim, but npm not available → should not detect
	tmpDir := t.TempDir()
	shimDir := filepath.Join(tmpDir, ".asdf", "shims")
	if err := os.MkdirAll(shimDir, 0755); err != nil {
		t.Fatal(err)
	}

	fakeClaude := filepath.Join(shimDir, "claude")
	if err := os.WriteFile(fakeClaude, []byte("#!/bin/sh\necho fake"), 0755); err != nil {
		t.Fatal(err)
	}

	origPath := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", origPath) })
	// Only the shim dir — no npm
	os.Setenv("PATH", shimDir)

	info := DetectNpmClaude()
	if info.Detected {
		t.Fatal("expected Detected=false for asdf shim when npm is not available")
	}
}

func TestDetectNpmClaude_NpmExistsButPackageNotInstalled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	// Claude in a regular path, npm exists but reports package not installed
	// (npm list exits non-zero)
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Fake claude binary
	fakeClaude := filepath.Join(binDir, "claude")
	if err := os.WriteFile(fakeClaude, []byte("#!/bin/sh\necho fake"), 0755); err != nil {
		t.Fatal(err)
	}

	// Fake npm that always exits 1 (package not found)
	fakeNpm := filepath.Join(binDir, "npm")
	if err := os.WriteFile(fakeNpm, []byte("#!/bin/sh\nexit 1"), 0755); err != nil {
		t.Fatal(err)
	}

	origPath := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", origPath) })
	os.Setenv("PATH", binDir)

	info := DetectNpmClaude()
	if info.Detected {
		t.Fatal("expected Detected=false when npm list reports package not installed")
	}
}

func TestDetectNpmClaude_NpmListConfirmsInstalled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	// Claude in a regular path (no node_modules), but npm list confirms it's installed
	// This covers the Volta / asdf / custom prefix scenarios
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Fake claude binary (not in node_modules)
	fakeClaude := filepath.Join(binDir, "claude")
	if err := os.WriteFile(fakeClaude, []byte("#!/bin/sh\necho fake"), 0755); err != nil {
		t.Fatal(err)
	}

	// Fake npm that reports the package as installed
	fakeNpm := filepath.Join(binDir, "npm")
	npmScript := `#!/bin/sh
echo "/usr/local/lib"
echo "├── @anthropic-ai/claude-code@1.0.0"
exit 0
`
	if err := os.WriteFile(fakeNpm, []byte(npmScript), 0755); err != nil {
		t.Fatal(err)
	}

	origPath := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", origPath) })
	os.Setenv("PATH", binDir)

	info := DetectNpmClaude()
	if !info.Detected {
		t.Fatal("expected Detected=true when npm list confirms package installed")
	}
	if info.Source != "npm-list" {
		t.Fatalf("expected Source='npm-list', got %q", info.Source)
	}
}

func TestDetectNpmClaude_PathHeuristicTakesPrecedenceOverNpmList(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	// When claude is in node_modules, path-heuristic should be returned
	// even if npm is also available — verifying early return
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "node_modules", ".bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}

	fakeClaude := filepath.Join(binDir, "claude")
	if err := os.WriteFile(fakeClaude, []byte("#!/bin/sh\necho fake"), 0755); err != nil {
		t.Fatal(err)
	}

	// Also provide a fake npm
	fakeNpm := filepath.Join(binDir, "npm")
	if err := os.WriteFile(fakeNpm, []byte("#!/bin/sh\necho installed; exit 0"), 0755); err != nil {
		t.Fatal(err)
	}

	origPath := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", origPath) })
	os.Setenv("PATH", binDir)

	info := DetectNpmClaude()
	if !info.Detected {
		t.Fatal("expected Detected=true")
	}
	if info.Source != "path-heuristic" {
		t.Fatalf("expected Source='path-heuristic' (early return), got %q", info.Source)
	}
}

func TestUninstallNpmClaude_NoNpm(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	// Set PATH to an empty temp dir so npm is not found
	tmpDir := t.TempDir()
	origPath := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", origPath) })
	os.Setenv("PATH", tmpDir)

	err := UninstallNpmClaude()
	if err == nil {
		t.Fatal("expected error when npm is not in PATH")
	}
}

func TestUninstallNpmClaude_NpmFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	// Fake npm that always fails (simulates permission denied without sudo)
	// and no sudo available either
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}

	fakeNpm := filepath.Join(binDir, "npm")
	if err := os.WriteFile(fakeNpm, []byte("#!/bin/sh\nexit 1"), 0755); err != nil {
		t.Fatal(err)
	}

	origPath := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", origPath) })
	os.Setenv("PATH", binDir)

	err := UninstallNpmClaude()
	if err == nil {
		t.Fatal("expected error when npm uninstall fails")
	}
}

func TestUninstallNpmClaude_NpmSucceeds(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	// Fake npm that succeeds on uninstall
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}

	fakeNpm := filepath.Join(binDir, "npm")
	if err := os.WriteFile(fakeNpm, []byte("#!/bin/sh\nexit 0"), 0755); err != nil {
		t.Fatal(err)
	}

	origPath := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", origPath) })
	os.Setenv("PATH", binDir)

	err := UninstallNpmClaude()
	if err != nil {
		t.Fatalf("expected no error when npm uninstall succeeds, got: %v", err)
	}
}

func TestUninstallNpmClaude_RunsAsdfReshim(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	// Fake npm (succeeds) and fake asdf that records it was called
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}

	fakeNpm := filepath.Join(binDir, "npm")
	if err := os.WriteFile(fakeNpm, []byte("#!/bin/sh\nexit 0"), 0755); err != nil {
		t.Fatal(err)
	}

	// Fake asdf that writes a marker file when called with "reshim nodejs"
	// Use shell redirection (>) instead of touch, since touch may not be in the restricted PATH
	marker := filepath.Join(tmpDir, "asdf-reshim-called")
	fakeAsdf := filepath.Join(binDir, "asdf")
	asdfScript := "#!/bin/sh\nif [ \"$1\" = \"reshim\" ] && [ \"$2\" = \"nodejs\" ]; then : > " + marker + "; fi\nexit 0\n"
	if err := os.WriteFile(fakeAsdf, []byte(asdfScript), 0755); err != nil {
		t.Fatal(err)
	}

	origPath := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", origPath) })
	os.Setenv("PATH", binDir)

	err := UninstallNpmClaude()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if _, err := os.Stat(marker); os.IsNotExist(err) {
		t.Fatal("expected asdf reshim nodejs to be called after npm uninstall")
	}
}
