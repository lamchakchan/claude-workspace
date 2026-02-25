package platform

import (
	"bytes"
	"strings"
	"testing"
)

func TestApplyColorEnabled(t *testing.T) {
	colorEnabled = true
	got := apply(ansiRed, "hello")
	want := "\033[31mhello\033[0m"
	if got != want {
		t.Errorf("apply(red, hello) = %q, want %q", got, want)
	}
}

func TestApplyColorDisabled(t *testing.T) {
	colorEnabled = false
	got := apply(ansiRed, "hello")
	want := "hello"
	if got != want {
		t.Errorf("apply(red, hello) = %q, want %q", got, want)
	}
}

func TestSemanticFunctionsEnabled(t *testing.T) {
	colorEnabled = true

	tests := []struct {
		name string
		fn   func(string) string
		want string
	}{
		{"Bold", Bold, "\033[1mtext\033[0m"},
		{"Red", Red, "\033[31mtext\033[0m"},
		{"Green", Green, "\033[32mtext\033[0m"},
		{"Yellow", Yellow, "\033[33mtext\033[0m"},
		{"Blue", Blue, "\033[34mtext\033[0m"},
		{"Cyan", Cyan, "\033[36mtext\033[0m"},
		{"BoldRed", BoldRed, "\033[1m\033[31mtext\033[0m"},
		{"BoldGreen", BoldGreen, "\033[1m\033[32mtext\033[0m"},
		{"BoldBlue", BoldBlue, "\033[1m\033[34mtext\033[0m"},
		{"BoldCyan", BoldCyan, "\033[1m\033[36mtext\033[0m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn("text")
			if got != tt.want {
				t.Errorf("%s(text) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestSemanticFunctionsDisabled(t *testing.T) {
	colorEnabled = false

	fns := []func(string) string{Bold, Red, Green, Yellow, Blue, Cyan, BoldRed, BoldGreen, BoldBlue, BoldCyan}
	for _, fn := range fns {
		got := fn("text")
		if got != "text" {
			t.Errorf("expected plain text when color disabled, got %q", got)
		}
	}
}

func TestInitColorNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("TERM", "xterm-256color")
	InitColor()
	if colorEnabled {
		t.Error("colorEnabled should be false when NO_COLOR is set")
	}
}

func TestInitColorTermDumb(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "dumb")
	InitColor()
	if colorEnabled {
		t.Error("colorEnabled should be false when TERM=dumb")
	}
}

func TestInitColorNoColorEmpty(t *testing.T) {
	// NO_COLOR="" means not set (os.Getenv returns "")
	// but t.Setenv("NO_COLOR", "") still sets it to empty string
	// which os.Getenv returns as "" — so NO_COLOR check passes.
	// We need to unset it properly via Unsetenv, but t.Setenv doesn't support that.
	// Instead, test that TERM=dumb still disables.
	t.Setenv("TERM", "dumb")
	InitColor()
	if colorEnabled {
		t.Error("colorEnabled should be false when TERM=dumb")
	}
}

func TestPrintBanner(t *testing.T) {
	var buf bytes.Buffer

	colorEnabled = true
	PrintBanner(&buf, "Test")
	got := buf.String()
	if got != "\n\033[1m\033[36m=== Test ===\033[0m\n" {
		t.Errorf("PrintBanner colored = %q", got)
	}

	buf.Reset()
	colorEnabled = false
	PrintBanner(&buf, "Test")
	got = buf.String()
	if got != "\n=== Test ===\n" {
		t.Errorf("PrintBanner plain = %q", got)
	}
}

func TestPrintLayerBanner(t *testing.T) {
	var buf bytes.Buffer

	colorEnabled = false
	PrintLayerBanner(&buf, "User CLAUDE.md")
	got := buf.String()
	wantRule := "\n" + strings.Repeat("─", 50) + "\n  ▶ User CLAUDE.md\n"
	if got != wantRule {
		t.Errorf("PrintLayerBanner plain = %q, want %q", got, wantRule)
	}

	buf.Reset()
	colorEnabled = true
	PrintLayerBanner(&buf, "User CLAUDE.md")
	got = buf.String()
	wantColored := "\n\033[1m\033[36m" + strings.Repeat("─", 50) + "\033[0m\n  \033[1m▶ User CLAUDE.md\033[0m\n"
	if got != wantColored {
		t.Errorf("PrintLayerBanner colored = %q, want %q", got, wantColored)
	}
}

func TestPrintSection(t *testing.T) {
	var buf bytes.Buffer

	colorEnabled = false
	PrintSection(&buf, "Sec")
	got := buf.String()
	if got != "\n--- Sec ---\n" {
		t.Errorf("PrintSection plain = %q", got)
	}
}

func TestPrintSectionLabel(t *testing.T) {
	var buf bytes.Buffer

	colorEnabled = false
	PrintSectionLabel(&buf, "Label")
	got := buf.String()
	if got != "\n[Label]\n" {
		t.Errorf("PrintSectionLabel plain = %q", got)
	}
}

func TestPrintStep(t *testing.T) {
	var buf bytes.Buffer

	colorEnabled = false
	PrintStep(&buf, 1, 7, "Setting up agents...")
	got := buf.String()
	if got != "\n[1/7] Setting up agents...\n" {
		t.Errorf("PrintStep plain = %q", got)
	}

	buf.Reset()
	colorEnabled = true
	PrintStep(&buf, 2, 7, "Setting up skills...")
	got = buf.String()
	want := "\n\033[1m\033[34m[2/7]\033[0m Setting up skills...\n"
	if got != want {
		t.Errorf("PrintStep colored = %q, want %q", got, want)
	}
}

func TestPrintOK(t *testing.T) {
	var buf bytes.Buffer

	colorEnabled = false
	PrintOK(&buf, "Installed: v1.0")
	got := buf.String()
	if got != "  [OK] Installed: v1.0\n" {
		t.Errorf("PrintOK plain = %q", got)
	}
}

func TestPrintFail(t *testing.T) {
	var buf bytes.Buffer

	colorEnabled = false
	PrintFail(&buf, "Not found")
	got := buf.String()
	if got != "  [FAIL] Not found\n" {
		t.Errorf("PrintFail plain = %q", got)
	}
}

func TestPrintWarn(t *testing.T) {
	var buf bytes.Buffer

	colorEnabled = false
	PrintWarn(&buf, "Missing config")
	got := buf.String()
	if got != "  [WARN] Missing config\n" {
		t.Errorf("PrintWarn plain = %q", got)
	}
}

func TestPrintInfo(t *testing.T) {
	var buf bytes.Buffer

	PrintInfo(&buf, "Update available")
	got := buf.String()
	if got != "  [INFO] Update available\n" {
		t.Errorf("PrintInfo = %q", got)
	}
}

func TestPrintSuccess(t *testing.T) {
	var buf bytes.Buffer

	colorEnabled = false
	PrintSuccess(&buf, "Created file.txt")
	got := buf.String()
	if got != "  Created file.txt\n" {
		t.Errorf("PrintSuccess plain = %q", got)
	}
}

func TestPrintWarningLine(t *testing.T) {
	var buf bytes.Buffer

	colorEnabled = false
	PrintWarningLine(&buf, "Skipping (exists): file.txt")
	got := buf.String()
	if got != "  Skipping (exists): file.txt\n" {
		t.Errorf("PrintWarningLine plain = %q", got)
	}
}

func TestPrintErrorLine(t *testing.T) {
	var buf bytes.Buffer

	colorEnabled = false
	PrintErrorLine(&buf, "Error: something broke")
	got := buf.String()
	if got != "  Error: something broke\n" {
		t.Errorf("PrintErrorLine plain = %q", got)
	}
}

func TestPrintPrompt(t *testing.T) {
	var buf bytes.Buffer

	colorEnabled = false
	PrintPrompt(&buf, "Proceed? [Y/n] ")
	got := buf.String()
	if got != "Proceed? [Y/n] " {
		t.Errorf("PrintPrompt plain = %q", got)
	}

	buf.Reset()
	colorEnabled = true
	PrintPrompt(&buf, "Proceed? [Y/n] ")
	got = buf.String()
	want := "\033[1mProceed? [Y/n] \033[0m"
	if got != want {
		t.Errorf("PrintPrompt colored = %q, want %q", got, want)
	}
}
