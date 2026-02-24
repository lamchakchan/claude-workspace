package cost

import (
	"fmt"
	"os"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

// Run is the entry point for the cost command.
// args is os.Args[2:] (everything after "cost").
func Run(args []string) error {
	runtime, prefix := detectRuntime()
	if runtime == "" {
		fmt.Fprintln(os.Stderr, "  bun or npx is required to run ccusage.")
		fmt.Fprintln(os.Stderr, "  Install Node.js: https://nodejs.org")
		fmt.Fprintln(os.Stderr, "  Install Bun:     https://bun.sh")
		return fmt.Errorf("bun or npx not found")
	}
	ccusageArgs := append(prefix, args...)
	return platform.Run(runtime, ccusageArgs...)
}

func detectRuntime() (string, []string) {
	if platform.Exists("bun") {
		return "bun", []string{"x", "ccusage"}
	}
	if platform.Exists("npx") {
		return "npx", []string{"-y", "ccusage"}
	}
	return "", nil
}
