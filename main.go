// Package main provides the claude-workspace CLI, a platform engineering kit
// for deploying Claude Code AI agents across organizations. It embeds template
// assets at compile time and routes subcommands to their respective packages.
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/lamchakchan/claude-workspace/internal/attach"
	"github.com/lamchakchan/claude-workspace/internal/cost"
	"github.com/lamchakchan/claude-workspace/internal/doctor"
	"github.com/lamchakchan/claude-workspace/internal/enrich"
	"github.com/lamchakchan/claude-workspace/internal/mcp"
	"github.com/lamchakchan/claude-workspace/internal/memory"
	"github.com/lamchakchan/claude-workspace/internal/platform"
	"github.com/lamchakchan/claude-workspace/internal/sandbox"
	"github.com/lamchakchan/claude-workspace/internal/sessions"
	"github.com/lamchakchan/claude-workspace/internal/setup"
	"github.com/lamchakchan/claude-workspace/internal/statusline"
	"github.com/lamchakchan/claude-workspace/internal/tui"
	"github.com/lamchakchan/claude-workspace/internal/upgrade"
)

// version is set via -ldflags at build time
var version = "dev"

// commands maps CLI command names to their handler functions.
var commands = map[string]func([]string) error{
	"setup":      runSetup,
	"attach":     runAttach,
	"enrich":     runEnrich,
	"sandbox":    runSandbox,
	"mcp":        runMCP,
	"upgrade":    runUpgrade,
	"doctor":     func(_ []string) error { return doctor.Run() },
	"statusline": func(a []string) error { return statusline.Run(a[1:]) },
	"memory":     func(a []string) error { return memory.Run(a[1:]) },
	"sessions":   func(a []string) error { return sessions.Run(a[1:]) },
	"cost":       func(a []string) error { return cost.Run(a[1:]) },
}

const helpText = `
claude-workspace - Claude Code Platform Engineering Kit CLI

Usage:
  claude-workspace <command> [options]

Commands:
  setup                          First-time setup & API key provisioning
  attach <project-path>          Attach platform config to a project
    [--symlink]                  Use symlinks instead of copying assets
    [--force]                    Overwrite existing files
    [--no-enrich]                Skip AI-powered CLAUDE.md enrichment
  enrich [project-path]          Re-generate .claude/CLAUDE.md with AI analysis
    [--scaffold-only]            Generate static scaffold only (skip AI enrichment)
  sandbox <project-path> <name>  Create a sandboxed branch worktree
  mcp add <name> [options]       Add an MCP server (local or remote)
  mcp remote <url>               Connect to a remote MCP server/gateway
  mcp list                       List all configured MCP servers
  upgrade [--self-only|--cli-only]  Upgrade claude-workspace and Claude Code CLI
  doctor                         Check platform configuration health
  statusline                     Configure Claude Code statusline (cost & context display)
    [--force]                    Overwrite existing statusLine configuration
  sessions [list|show] [options]   Browse and review session prompts
    list                           List sessions for current project (default)
    list --all                     List sessions across all projects
    list --limit N                 Limit results (default: 20)
    show <session-id>              Show all user prompts from a session
  memory [subcommand] [options]  Inspect and manage memory layers
    (no args)                    Overview of all layers
    show [--scope=user|project|local|auto|mcp|all]
    export [--output=path]       Export all layers to structured JSON
    import <file> [--scope=...] [--confirm]
  cost [subcommand] [options]    View Claude Code usage and costs (via ccusage)
    daily|weekly|monthly         Usage by time period (default: daily)
    session                      Usage by conversation session
    blocks                       Usage by 5-hour billing window
    [--breakdown]                Per-model cost breakdown
    [--since YYYYMMDD]           Filter from date
    [--json]                     JSON output

Options:
  --help, -h       Show this help message
  --version, -v    Show version

MCP Authentication:
  --api-key ENV_NAME     Securely prompt for API key (masked input)
  --bearer               Securely prompt for Bearer token (masked input)
  --oauth                Use OAuth 2.0 (authenticate via /mcp in session)
  --client-id <id>       OAuth client ID for pre-registered apps
  --client-secret        Prompt for OAuth client secret (masked input)

Examples:
  claude-workspace setup
  claude-workspace attach /path/to/my-project
  claude-workspace sandbox /path/to/my-project feature-auth
  claude-workspace mcp add postgres --api-key DATABASE_URL -- npx -y @bytebase/dbhub
  claude-workspace mcp add brave --api-key BRAVE_API_KEY -- npx -y @modelcontextprotocol/server-brave-search
  claude-workspace mcp remote https://mcp.sentry.dev/mcp --name sentry
  claude-workspace mcp remote https://mcp-gateway.company.com --bearer
  claude-workspace statusline
  claude-workspace statusline --force
  claude-workspace sessions
  claude-workspace sessions list --all --limit 50
  claude-workspace sessions show 8a3f1b2c
  claude-workspace cost
  claude-workspace cost monthly --breakdown
  claude-workspace cost blocks --active
`

func main() {
	// Wire embedded assets to the platform package
	projectSub, err := fs.Sub(PlatformFS, "_template/project")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing embedded project assets: %v\n", err)
		os.Exit(1)
	}
	platform.FS = projectSub

	globalSub, err := fs.Sub(PlatformFS, "_template/global")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing embedded global assets: %v\n", err)
		os.Exit(1)
	}
	platform.GlobalFS = globalSub
	platform.InitColor()

	args := os.Args[1:]

	if len(args) == 0 {
		if platform.IsTTY() {
			if err := tui.Run(version); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		}
		fmt.Print(helpText)
		os.Exit(0)
	}

	command := args[0]
	switch command {
	case "--help", "-h":
		fmt.Print(helpText)
		os.Exit(0)
	case "--version", "-v":
		fmt.Printf("claude-workspace %s\n", version)
		os.Exit(0)
	}

	cmd, ok := commands[command]
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		fmt.Print(helpText)
		os.Exit(1)
	}

	if err := cmd(args); err != nil {
		if errors.Is(err, upgrade.ErrUpdateAvailable) {
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runSetup(args []string) error {
	return setup.Run(args[1:])
}

func runAttach(args []string) error {
	var target string
	if len(args) > 1 {
		target = args[1]
	}
	return attach.Run(target, args)
}

func runEnrich(args []string) error {
	var target string
	if len(args) > 1 && args[1][0] != '-' {
		target = args[1]
	}
	return enrich.Run(target, args[1:])
}

func runSandbox(args []string) error {
	var projectPath, branchName string
	if len(args) > 1 {
		projectPath = args[1]
	}
	if len(args) > 2 {
		branchName = args[2]
	}
	return sandbox.Run(projectPath, branchName)
}

func runMCP(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: claude-workspace mcp <add|remote|list>")
	}
	subcmd := args[1]
	switch subcmd {
	case "add":
		return mcp.Add(args[2:])
	case "remote":
		var url string
		if len(args) > 2 {
			url = args[2]
		}
		return mcp.Remote(url, args[3:])
	case "list":
		return mcp.List()
	default:
		return fmt.Errorf("unknown mcp subcommand: %s (available: add, remote, list)", subcmd)
	}
}

func runUpgrade(args []string) error {
	return upgrade.Run(version, args[1:])
}
