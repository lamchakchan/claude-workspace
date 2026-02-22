package platform

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

// colorEnabled controls whether ANSI escape codes are emitted.
// Set once by InitColor().
var colorEnabled bool

// InitColor determines whether color output should be enabled.
// It respects NO_COLOR (https://no-color.org/), TERM=dumb, and non-TTY stdout.
func InitColor() {
	if os.Getenv("NO_COLOR") != "" {
		colorEnabled = false
		return
	}
	if os.Getenv("TERM") == "dumb" {
		colorEnabled = false
		return
	}
	colorEnabled = term.IsTerminal(int(os.Stdout.Fd()))
}

// ANSI escape codes
const (
	ansiReset  = "\033[0m"
	ansiBold   = "\033[1m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiBlue   = "\033[34m"
	ansiCyan   = "\033[36m"
)

// apply wraps s with the given ANSI code when color is enabled.
func apply(code, s string) string {
	if !colorEnabled {
		return s
	}
	return code + s + ansiReset
}

// --- Semantic color functions ---

func Bold(s string) string      { return apply(ansiBold, s) }
func Red(s string) string       { return apply(ansiRed, s) }
func Green(s string) string     { return apply(ansiGreen, s) }
func Yellow(s string) string    { return apply(ansiYellow, s) }
func Blue(s string) string      { return apply(ansiBlue, s) }
func Cyan(s string) string      { return apply(ansiCyan, s) }
func BoldRed(s string) string   { return apply(ansiBold+ansiRed, s) }
func BoldGreen(s string) string { return apply(ansiBold+ansiGreen, s) }
func BoldBlue(s string) string  { return apply(ansiBold+ansiBlue, s) }
func BoldCyan(s string) string  { return apply(ansiBold+ansiCyan, s) }

// --- High-level print helpers ---

// PrintBanner prints a bold cyan banner line: "\n=== title ===\n"
func PrintBanner(w io.Writer, title string) {
	fmt.Fprintf(w, "\n%s\n", BoldCyan("=== "+title+" ==="))
}

// PrintSection prints a cyan section header: "\n--- title ---\n"
func PrintSection(w io.Writer, title string) {
	fmt.Fprintf(w, "\n%s\n", Cyan("--- "+title+" ---"))
}

// PrintSectionLabel prints a bold section label: "\n[label]\n"
func PrintSectionLabel(w io.Writer, label string) {
	fmt.Fprintf(w, "\n%s\n", Bold("["+label+"]"))
}

// PrintStep prints a bold blue step label: "\n[n/total] label\n"
func PrintStep(w io.Writer, n, total int, label string) {
	fmt.Fprintf(w, "\n%s %s\n", BoldBlue(fmt.Sprintf("[%d/%d]", n, total)), label)
}

// PrintOK prints a bold green OK status: "  [OK] msg\n"
func PrintOK(w io.Writer, msg string) {
	fmt.Fprintf(w, "  %s %s\n", BoldGreen("[OK]"), msg)
}

// PrintFail prints a bold red FAIL status: "  [FAIL] msg\n"
func PrintFail(w io.Writer, msg string) {
	fmt.Fprintf(w, "  %s %s\n", BoldRed("[FAIL]"), msg)
}

// PrintWarn prints a yellow WARN status: "  [WARN] msg\n"
func PrintWarn(w io.Writer, msg string) {
	fmt.Fprintf(w, "  %s %s\n", Yellow("[WARN]"), msg)
}

// PrintInfo prints a plain INFO status: "  [INFO] msg\n"
func PrintInfo(w io.Writer, msg string) {
	fmt.Fprintf(w, "  [INFO] %s\n", msg)
}

// PrintSuccess prints a green message: "  msg\n"
func PrintSuccess(w io.Writer, msg string) {
	fmt.Fprintf(w, "  %s\n", Green(msg))
}

// PrintWarningLine prints a yellow message: "  msg\n"
func PrintWarningLine(w io.Writer, msg string) {
	fmt.Fprintf(w, "  %s\n", Yellow(msg))
}

// PrintErrorLine prints a red message: "  msg\n"
func PrintErrorLine(w io.Writer, msg string) {
	fmt.Fprintf(w, "  %s\n", Red(msg))
}

// PrintPrompt prints a bold prompt without a trailing newline.
func PrintPrompt(w io.Writer, prompt string) {
	fmt.Fprint(w, Bold(prompt))
}
