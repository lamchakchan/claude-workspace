package setup

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const (
	osWindows      = "windows"
	sourcePathHeur = "path-heuristic"
)

func TestDetectNpmClaude_PathHeuristic(t *testing.T) {
	if runtime.GOOS == osWindows {
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
	if info.Source != sourcePathHeur {
		t.Fatalf("expected Source='path-heuristic', got %q", info.Source)
	}
	if info.Path != fakeClaude {
		t.Errorf("expected Path=%q, got %q", fakeClaude, info.Path)
	}
}

func TestDetectNpmClaude_PathHeuristicGlobalNpm(t *testing.T) {
	if runtime.GOOS == osWindows {
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
	if info.Source != sourcePathHeur {
		t.Fatalf("expected Source='path-heuristic', got %q", info.Source)
	}
}

func TestDetectNpmClaude_NoClaude(t *testing.T) {
	if runtime.GOOS == osWindows {
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
	if runtime.GOOS == osWindows {
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
	if runtime.GOOS == osWindows {
		t.Skip("skipping on Windows")
	}

	// Claude via asdf shim, but no npm-installed claude under any node version → should not detect
	tmpDir := t.TempDir()
	shimDir := filepath.Join(tmpDir, ".asdf", "shims")
	if err := os.MkdirAll(shimDir, 0755); err != nil {
		t.Fatal(err)
	}

	fakeClaude := filepath.Join(shimDir, "claude")
	if err := os.WriteFile(fakeClaude, []byte("#!/bin/sh\necho fake"), 0755); err != nil {
		t.Fatal(err)
	}

	// Isolate ASDF_DATA_DIR so the real ~/.asdf is not scanned
	origAsdfEnv := os.Getenv("ASDF_DATA_DIR")
	t.Cleanup(func() { os.Setenv("ASDF_DATA_DIR", origAsdfEnv) })
	os.Setenv("ASDF_DATA_DIR", filepath.Join(tmpDir, ".asdf"))

	origPath := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", origPath) })
	// Only the shim dir — no npm
	os.Setenv("PATH", shimDir)

	info := DetectNpmClaude()
	if info.Detected {
		t.Fatal("expected Detected=false for asdf shim when no npm-installed claude exists")
	}
}

func TestDetectNpmClaude_NpmExistsButPackageNotInstalled(t *testing.T) {
	if runtime.GOOS == osWindows {
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
	if runtime.GOOS == osWindows {
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
	if runtime.GOOS == osWindows {
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
	if info.Source != sourcePathHeur {
		t.Fatalf("expected Source='path-heuristic' (early return), got %q", info.Source)
	}
}

func TestUninstallNpmClaude_NoNpm(t *testing.T) {
	if runtime.GOOS == osWindows {
		t.Skip("skipping on Windows")
	}

	// Set PATH to an empty temp dir so npm is not found
	tmpDir := t.TempDir()
	origPath := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", origPath) })
	os.Setenv("PATH", tmpDir)

	err := UninstallNpmClaude(NpmInstallInfo{})
	if err == nil {
		t.Fatal("expected error when npm is not in PATH")
	}
}

func TestUninstallNpmClaude_NpmFails(t *testing.T) {
	if runtime.GOOS == osWindows {
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

	err := UninstallNpmClaude(NpmInstallInfo{})
	if err == nil {
		t.Fatal("expected error when npm uninstall fails")
	}
}

func TestUninstallNpmClaude_NpmSucceeds(t *testing.T) {
	if runtime.GOOS == osWindows {
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

	err := UninstallNpmClaude(NpmInstallInfo{})
	if err != nil {
		t.Fatalf("expected no error when npm uninstall succeeds, got: %v", err)
	}
}

func TestUninstallNpmClaude_RunsAsdfReshim(t *testing.T) {
	if runtime.GOOS == osWindows {
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

	err := UninstallNpmClaude(NpmInstallInfo{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if _, err := os.Stat(marker); os.IsNotExist(err) {
		t.Fatal("expected asdf reshim nodejs to be called after npm uninstall")
	}
}

func TestFindAsdfNodejsVersionsWithClaude_FindsVersion(t *testing.T) {
	if runtime.GOOS == osWindows {
		t.Skip("skipping on Windows")
	}

	// Set up a fake ASDF_DATA_DIR with a symlink chain through node_modules
	tmpDir := t.TempDir()
	origEnv := os.Getenv("ASDF_DATA_DIR")
	t.Cleanup(func() { os.Setenv("ASDF_DATA_DIR", origEnv) })
	os.Setenv("ASDF_DATA_DIR", tmpDir)

	// Create node_modules target (the real binary location)
	nodeModulesDir := filepath.Join(tmpDir, "installs", "nodejs", "20.5.1", "lib", "node_modules", "@anthropic-ai", "claude-code", "cli")
	if err := os.MkdirAll(nodeModulesDir, 0755); err != nil {
		t.Fatal(err)
	}
	realBinary := filepath.Join(nodeModulesDir, "claude.js")
	if err := os.WriteFile(realBinary, []byte("#!/usr/bin/env node"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create bin/claude as a symlink to the node_modules path
	binDir := filepath.Join(tmpDir, "installs", "nodejs", "20.5.1", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realBinary, filepath.Join(binDir, "claude")); err != nil {
		t.Fatal(err)
	}

	versions := findAsdfNodejsVersionsWithClaude()
	if len(versions) != 1 {
		t.Fatalf("expected 1 version, got %d: %v", len(versions), versions)
	}
	if versions[0] != "20.5.1" {
		t.Errorf("expected version '20.5.1', got %q", versions[0])
	}
}

func TestFindAsdfNodejsVersionsWithClaude_SkipsNonNpmBinary(t *testing.T) {
	if runtime.GOOS == osWindows {
		t.Skip("skipping on Windows")
	}

	// Binary exists but does NOT resolve through node_modules → should not be detected
	tmpDir := t.TempDir()
	origEnv := os.Getenv("ASDF_DATA_DIR")
	t.Cleanup(func() { os.Setenv("ASDF_DATA_DIR", origEnv) })
	os.Setenv("ASDF_DATA_DIR", tmpDir)

	binDir := filepath.Join(tmpDir, "installs", "nodejs", "18.0.0", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Direct binary, not a symlink through node_modules
	if err := os.WriteFile(filepath.Join(binDir, "claude"), []byte("#!/bin/sh\necho fake"), 0755); err != nil {
		t.Fatal(err)
	}

	versions := findAsdfNodejsVersionsWithClaude()
	if len(versions) != 0 {
		t.Fatalf("expected 0 versions for non-npm binary, got %d: %v", len(versions), versions)
	}
}

func TestFindAsdfNodejsVersionsWithClaude_CustomASDF_DATA_DIR(t *testing.T) {
	if runtime.GOOS == osWindows {
		t.Skip("skipping on Windows")
	}

	// Use a custom ASDF_DATA_DIR that's not $HOME/.asdf
	customDir := filepath.Join(t.TempDir(), "custom-asdf")
	origEnv := os.Getenv("ASDF_DATA_DIR")
	t.Cleanup(func() { os.Setenv("ASDF_DATA_DIR", origEnv) })
	os.Setenv("ASDF_DATA_DIR", customDir)

	// Set up the structure
	nodeModulesDir := filepath.Join(customDir, "installs", "nodejs", "22.1.0", "lib", "node_modules", "@anthropic-ai", "claude-code", "cli")
	if err := os.MkdirAll(nodeModulesDir, 0755); err != nil {
		t.Fatal(err)
	}
	realBinary := filepath.Join(nodeModulesDir, "claude.js")
	if err := os.WriteFile(realBinary, []byte("#!/usr/bin/env node"), 0755); err != nil {
		t.Fatal(err)
	}

	binDir := filepath.Join(customDir, "installs", "nodejs", "22.1.0", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realBinary, filepath.Join(binDir, "claude")); err != nil {
		t.Fatal(err)
	}

	versions := findAsdfNodejsVersionsWithClaude()
	if len(versions) != 1 {
		t.Fatalf("expected 1 version, got %d: %v", len(versions), versions)
	}
	if versions[0] != "22.1.0" {
		t.Errorf("expected version '22.1.0', got %q", versions[0])
	}
}

func TestExtractAsdfNodeVersion(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "standard asdf path",
			path: "/home/user/.asdf/installs/nodejs/20.5.1/bin/claude",
			want: "20.5.1",
		},
		{
			name: "custom asdf data dir",
			path: "/opt/asdf/installs/nodejs/18.17.0/bin/claude",
			want: "18.17.0",
		},
		{
			name: "no nodejs in path",
			path: "/usr/local/bin/claude",
			want: "",
		},
		{
			name: "nodejs at end of path",
			path: "/home/user/.asdf/installs/nodejs",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractAsdfNodeVersion(tt.path)
			if got != tt.want {
				t.Errorf("extractAsdfNodeVersion(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestDetectNpmClaude_AsdfShimWithNodejsInstall(t *testing.T) {
	if runtime.GOOS == osWindows {
		t.Skip("skipping on Windows")
	}

	// End-to-end: claude found at .asdf/shims/claude, and a matching npm install exists
	tmpDir := t.TempDir()

	// Set up ASDF_DATA_DIR
	asdfDir := filepath.Join(tmpDir, ".asdf")
	origAsdfEnv := os.Getenv("ASDF_DATA_DIR")
	t.Cleanup(func() { os.Setenv("ASDF_DATA_DIR", origAsdfEnv) })
	os.Setenv("ASDF_DATA_DIR", asdfDir)

	// Create shim directory with fake claude
	shimDir := filepath.Join(asdfDir, "shims")
	if err := os.MkdirAll(shimDir, 0755); err != nil {
		t.Fatal(err)
	}
	fakeClaude := filepath.Join(shimDir, "claude")
	if err := os.WriteFile(fakeClaude, []byte("#!/bin/sh\necho fake"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create the npm-installed binary under a node version
	nodeModulesDir := filepath.Join(asdfDir, "installs", "nodejs", "20.5.1", "lib", "node_modules", "@anthropic-ai", "claude-code", "cli")
	if err := os.MkdirAll(nodeModulesDir, 0755); err != nil {
		t.Fatal(err)
	}
	realBinary := filepath.Join(nodeModulesDir, "claude.js")
	if err := os.WriteFile(realBinary, []byte("#!/usr/bin/env node"), 0755); err != nil {
		t.Fatal(err)
	}
	binDir := filepath.Join(asdfDir, "installs", "nodejs", "20.5.1", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realBinary, filepath.Join(binDir, "claude")); err != nil {
		t.Fatal(err)
	}

	// Set PATH to only include the shim dir
	origPath := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", origPath) })
	os.Setenv("PATH", shimDir)

	info := DetectNpmClaude()
	if !info.Detected {
		t.Fatal("expected Detected=true for asdf shim with npm nodejs install")
	}
	if info.Source != "asdf-shim" {
		t.Fatalf("expected Source='asdf-shim', got %q", info.Source)
	}
	if len(info.AsdfNodeVersions) != 1 || info.AsdfNodeVersions[0] != "20.5.1" {
		t.Errorf("expected AsdfNodeVersions=['20.5.1'], got %v", info.AsdfNodeVersions)
	}
}

func TestUninstallNpmClaude_SetsASDF_NODEJS_VERSION(t *testing.T) {
	if runtime.GOOS == osWindows {
		t.Skip("skipping on Windows")
	}

	// Fake npm that records the ASDF_NODEJS_VERSION env var to a file
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}

	marker := filepath.Join(tmpDir, "npm-env-marker")
	fakeNpm := filepath.Join(binDir, "npm")
	npmScript := `#!/bin/sh
echo "ASDF_NODEJS_VERSION=$ASDF_NODEJS_VERSION" >> ` + marker + `
exit 0
`
	if err := os.WriteFile(fakeNpm, []byte(npmScript), 0755); err != nil {
		t.Fatal(err)
	}

	origPath := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", origPath) })
	os.Setenv("PATH", binDir)

	info := NpmInstallInfo{
		Detected:         true,
		Source:           "asdf-shim",
		AsdfNodeVersions: []string{"20.5.1", "18.17.0"},
	}

	err := UninstallNpmClaude(info)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Read marker file and check both versions were passed
	content, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("expected marker file to exist: %v", err)
	}

	lines := string(content)
	if !strings.Contains(lines, "ASDF_NODEJS_VERSION=20.5.1") {
		t.Errorf("expected ASDF_NODEJS_VERSION=20.5.1 in npm calls, got:\n%s", lines)
	}
	if !strings.Contains(lines, "ASDF_NODEJS_VERSION=18.17.0") {
		t.Errorf("expected ASDF_NODEJS_VERSION=18.17.0 in npm calls, got:\n%s", lines)
	}
}
