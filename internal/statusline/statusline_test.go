package statusline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteWrapperScript_CreatesExecutableScript(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "statusline.sh")

	if err := writeWrapperScript(path); err != nil {
		t.Fatalf("writeWrapperScript: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat script: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Errorf("script is not executable: mode=%v", info.Mode())
	}
}

func TestWriteWrapperScript_ContainsRequiredSections(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "statusline.sh")

	if err := writeWrapperScript(path); err != nil {
		t.Fatalf("writeWrapperScript: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading script: %v", err)
	}
	body := string(content)

	checks := []struct {
		desc    string
		contain string
	}{
		{"shebang", "#!/usr/bin/env bash"},
		{"bun runtime", "bun x ccusage statusline"},
		{"npx runtime", "npx -y ccusage statusline"},
		{"jq fallback", "jq -r"},
		{"reset countdown", "subscriptionCreatedAt"},
		{"python3 invocation", "python3"},
		{"output combination", "resets"},
	}
	for _, c := range checks {
		if !strings.Contains(body, c.contain) {
			t.Errorf("script missing %s (expected to contain %q)", c.desc, c.contain)
		}
	}
}

func TestConfigure_WritesScriptAndSettings(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := configure(false); err != nil {
		t.Fatalf("configure: %v", err)
	}

	scriptPath := filepath.Join(home, ".claude", "statusline.sh")
	if _, err := os.Stat(scriptPath); err != nil {
		t.Errorf("statusline.sh not created: %v", err)
	}

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("reading settings.json: %v", err)
	}
	if !strings.Contains(string(data), "statusline.sh") {
		t.Errorf("settings.json does not reference statusline.sh: %s", data)
	}
}
