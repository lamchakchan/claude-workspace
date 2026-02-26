package platform

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

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

// Bold wraps s with ANSI bold when color is enabled.
func Bold(s string) string { return apply(ansiBold, s) }

// Red wraps s with ANSI red when color is enabled.
func Red(s string) string { return apply(ansiRed, s) }

// Green wraps s with ANSI green when color is enabled.
func Green(s string) string { return apply(ansiGreen, s) }

// Yellow wraps s with ANSI yellow when color is enabled.
func Yellow(s string) string { return apply(ansiYellow, s) }

// Blue wraps s with ANSI blue when color is enabled.
func Blue(s string) string { return apply(ansiBlue, s) }

// Cyan wraps s with ANSI cyan when color is enabled.
func Cyan(s string) string { return apply(ansiCyan, s) }

// BoldRed wraps s with ANSI bold red when color is enabled.
func BoldRed(s string) string { return apply(ansiBold+ansiRed, s) }

// BoldGreen wraps s with ANSI bold green when color is enabled.
func BoldGreen(s string) string { return apply(ansiBold+ansiGreen, s) }

// BoldBlue wraps s with ANSI bold blue when color is enabled.
func BoldBlue(s string) string { return apply(ansiBold+ansiBlue, s) }

// BoldCyan wraps s with ANSI bold cyan when color is enabled.
func BoldCyan(s string) string { return apply(ansiBold+ansiCyan, s) }

// --- High-level print helpers ---

// PrintBanner prints a bold cyan banner line: "\n=== title ===\n"
func PrintBanner(w io.Writer, title string) {
	fmt.Fprintf(w, "\n%s\n", BoldCyan("=== "+title+" ==="))
}

// PrintLayerBanner prints a prominent layer header with a full-width horizontal rule
// followed by the title, suitable for separating major content sections:
//
//	──────────────────────────────────────────────────
//	▶ title
func PrintLayerBanner(w io.Writer, title string) {
	fmt.Fprintf(w, "\n%s\n  %s\n", BoldCyan(strings.Repeat("─", 50)), Bold("▶ "+title))
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

// PrintCommand prints a command hint: "  $ command" (bold cyan $ + bold command)
func PrintCommand(w io.Writer, cmd string) {
	fmt.Fprintf(w, "  %s %s\n", BoldCyan("$"), Bold(cmd))
}

// PrintManual prints a manual action hint: "  → description" (yellow arrow + message)
func PrintManual(w io.Writer, msg string) {
	fmt.Fprintf(w, "  %s %s\n", Yellow("→"), msg)
}

// --- Spinner ---

// Spinner displays an animated progress indicator on a TTY.
type Spinner struct {
	stop chan struct{}
	done chan struct{}
}

// StartSpinner starts a terminal spinner with the given message.
// On non-TTY (colorEnabled=false), it prints a static line and Stop is a no-op.
func StartSpinner(w io.Writer, msg string) *Spinner {
	s := &Spinner{
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}

	if !colorEnabled {
		fmt.Fprintf(w, "  ... %s\n", msg)
		close(s.done)
		return s
	}

	frames := []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")
	var mu sync.Mutex

	go func() {
		defer close(s.done)
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		i := 0
		for {
			select {
			case <-s.stop:
				mu.Lock()
				// Clear the spinner line
				fmt.Fprintf(w, "\r%s\r", strings.Repeat(" ", len(msg)+6))
				mu.Unlock()
				return
			case <-ticker.C:
				mu.Lock()
				fmt.Fprintf(w, "\r  %s %s", Cyan(string(frames[i%len(frames)])), msg)
				mu.Unlock()
				i++
			}
		}
	}()

	return s
}

// Stop halts the spinner and clears its line.
func (s *Spinner) Stop() {
	select {
	case <-s.stop:
		// Already stopped
	default:
		close(s.stop)
	}
	<-s.done
}
