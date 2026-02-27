// Package cost implements the "cost" command, which delegates to ccusage
// (via bun or npx) to display Claude Code usage and cost reports.
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
	cmdArgs := make([]string, 0, len(prefix)+len(args))
	cmdArgs = append(cmdArgs, prefix...)
	cmdArgs = append(cmdArgs, args...)
	return platform.Run(runtime, cmdArgs...)
}

// RunCapture runs ccusage and returns the output as a string.
func RunCapture(args []string) (string, error) {
	runtime, prefix := detectRuntime()
	if runtime == "" {
		return "", fmt.Errorf("bun or npx not found; install Node.js (https://nodejs.org) or Bun (https://bun.sh)")
	}
	cmdArgs := make([]string, 0, len(prefix)+len(args))
	cmdArgs = append(cmdArgs, prefix...)
	cmdArgs = append(cmdArgs, args...)
	return platform.Output(runtime, cmdArgs...)
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
