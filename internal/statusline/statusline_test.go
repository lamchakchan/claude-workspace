package statusline

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

func TestWriteWrapperScript_CreatesExecutableScript(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "statusline.sh")

	if err := writeWrapperScript(path, []byte("#!/usr/bin/env bash\necho hi\n")); err != nil {
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

func TestStatuslineTemplate_ContainsRequiredSections(t *testing.T) {
	// Read the shell script template from disk to verify its content.
	content, err := os.ReadFile("../../_template/global/statusline.sh")
	if err != nil {
		t.Fatalf("reading statusline template: %v", err)
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
		{"ccusage error guard", "head -1"},
		{"ccusage error fallback", "❌"},
		{"claude-workspace render call", "claude-workspace statusline render"},
		{"base flag passed", "--base="},
		{"tput cols", "tput cols"},
		{"graceful fallback", "command -v claude-workspace"},
	}
	for _, c := range checks {
		if !strings.Contains(body, c.contain) {
			t.Errorf("template missing %s (expected to contain %q)", c.desc, c.contain)
		}
	}
}

func TestConfigure_WritesScriptAndSettings(t *testing.T) {
	// Provide a minimal GlobalFS so configureTo can read the template.
	platform.GlobalFS = fstest.MapFS{
		"statusline.sh": {Data: []byte("#!/usr/bin/env bash\necho test\n"), Mode: 0755},
	}
	t.Cleanup(func() { platform.GlobalFS = nil })

	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := configureTo(io.Discard, false); err != nil {
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
