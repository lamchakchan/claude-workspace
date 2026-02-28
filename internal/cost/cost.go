// Package cost implements the "cost" command, which delegates to ccusage
// (via bun or npx) to display Claude Code usage and cost reports.
package cost

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

// ChartEntry is a generic label+value pair extracted from any ccusage JSON output.
type ChartEntry struct {
	Label string
	Value float64
}

// costRecord is the superset of fields across all ccusage subcommand JSON entries.
// Only the fields relevant to the subcommand will be populated.
type costRecord struct {
	Date      string  `json:"date"`
	Month     string  `json:"month"`
	Week      string  `json:"week"`
	Title     string  `json:"title"`
	Name      string  `json:"name"`
	ID        string  `json:"id"`
	TotalCost float64 `json:"totalCost"`
}

// label returns the best available label from the record's fields.
func (r *costRecord) label() string {
	switch {
	case r.Date != "":
		return r.Date
	case r.Month != "":
		return r.Month
	case r.Week != "":
		return r.Week
	case r.Title != "":
		return r.Title
	case r.Name != "":
		return r.Name
	case r.ID != "":
		return r.ID
	default:
		return "?"
	}
}

// ParseCostJSON parses the JSON output from any ccusage subcommand.
// The subcommand name (e.g., "daily", "weekly") is used as the JSON envelope key.
func ParseCostJSON(subcommand string, data string) ([]ChartEntry, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(data), &raw); err != nil {
		return nil, fmt.Errorf("parsing %s JSON: %w", subcommand, err)
	}

	arrayData, ok := raw[subcommand]
	if !ok {
		return nil, fmt.Errorf("missing %q key in JSON", subcommand)
	}

	var records []costRecord
	if err := json.Unmarshal(arrayData, &records); err != nil {
		return nil, fmt.Errorf("parsing %s entries: %w", subcommand, err)
	}

	result := make([]ChartEntry, 0, len(records))
	for _, r := range records {
		result = append(result, ChartEntry{Label: r.label(), Value: r.TotalCost})
	}
	return result, nil
}

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
	return RunCaptureContext(context.Background(), args)
}

// RunCaptureContext is like RunCapture but accepts a context for cancellation.
func RunCaptureContext(ctx context.Context, args []string) (string, error) {
	runtime, prefix := detectRuntime()
	if runtime == "" {
		return "", fmt.Errorf("bun or npx not found; install Node.js (https://nodejs.org) or Bun (https://bun.sh)")
	}
	cmdArgs := make([]string, 0, len(prefix)+len(args))
	cmdArgs = append(cmdArgs, prefix...)
	cmdArgs = append(cmdArgs, args...)
	return platform.OutputContext(ctx, runtime, cmdArgs...)
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
