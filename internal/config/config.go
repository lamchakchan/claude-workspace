// Package config implements the "config" command, which provides a viewer and
// editor for all Claude Code configuration across every scope layer.
package config

import (
	"flag"
	"fmt"
	"io"
	"os"
)

// Run executes the config command, writing output to os.Stdout.
func Run(args []string) error {
	return RunTo(os.Stdout, args)
}

// RunTo executes the config command, writing all output to w.
// Subcommands:
//
//	(no args)             — signal TUI mode (handled by main.go)
//	view                  — non-interactive formatted output of all config
//	get <key>             — show a single key with layer breakdown
//	set <key> <value>     — set a value (--scope user|project|local)
func RunTo(w io.Writer, args []string) error {
	if len(args) == 0 {
		// No subcommand: TUI mode is launched by main.go; nothing to do here.
		return nil
	}

	reg := GlobalRegistry()

	subcmd := args[0]
	switch subcmd {
	case "view":
		snap, err := ReadAll()
		if err != nil {
			return fmt.Errorf("reading config: %w", err)
		}
		return FormatView(w, snap, reg)

	case "get":
		if len(args) < 2 {
			return fmt.Errorf("usage: config get <key>")
		}
		key := args[1]
		snap, err := ReadAll()
		if err != nil {
			return fmt.Errorf("reading config: %w", err)
		}
		return FormatGet(w, key, snap, reg)

	case "set":
		return runSet(args[1:])

	case "delete", "unset":
		return runDelete(args[1:])

	default:
		return fmt.Errorf("unknown config subcommand %q (available: view, get, set, delete)", subcmd)
	}
}

// runDelete handles "config delete <key> [--scope user|project|local]".
func runDelete(args []string) error {
	fs := flag.NewFlagSet("config delete", flag.ContinueOnError)
	scope := fs.String("scope", "user", "config scope to delete from: user, project, or local")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}

	remaining := fs.Args()
	if len(remaining) < 1 {
		return fmt.Errorf("usage: config delete <key> [--scope user|project|local]")
	}
	key := remaining[0]

	configScope := ConfigScope(*scope)
	switch configScope {
	case ScopeUser, ScopeProject, ScopeLocal:
		// valid
	default:
		return fmt.Errorf("invalid scope %q: must be user, project, or local", *scope)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	if err := DeleteSettingsValue(key, configScope, home, cwd); err != nil {
		return fmt.Errorf("deleting config: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Deleted %s from %s scope\n", key, configScope)
	return nil
}

// runSet handles "config set <key> <value> [--scope user|project|local]".
func runSet(args []string) error {
	fs := flag.NewFlagSet("config set", flag.ContinueOnError)
	scope := fs.String("scope", "user", "config scope to write to: user, project, or local")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}

	remaining := fs.Args()
	if len(remaining) < 2 {
		return fmt.Errorf("usage: config set <key> <value> [--scope user|project|local]")
	}
	key := remaining[0]
	value := remaining[1]

	configScope := ConfigScope(*scope)
	switch configScope {
	case ScopeUser, ScopeProject, ScopeLocal:
		// valid
	default:
		return fmt.Errorf("invalid scope %q: must be user, project, or local", *scope)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	if err := WriteSettingsValue(key, value, configScope, home, cwd); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Set %s = %s (scope: %s)\n", key, value, configScope)
	return nil
}
