package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/lamchakchan/claude-workspace/internal/attach"
	"github.com/lamchakchan/claude-workspace/internal/cost"
	"github.com/lamchakchan/claude-workspace/internal/doctor"
	"github.com/lamchakchan/claude-workspace/internal/mcp"
	"github.com/lamchakchan/claude-workspace/internal/platform"
	"github.com/lamchakchan/claude-workspace/internal/sandbox"
	"github.com/lamchakchan/claude-workspace/internal/sessions"
	"github.com/lamchakchan/claude-workspace/internal/setup"
	"github.com/lamchakchan/claude-workspace/internal/statusline"
	"github.com/lamchakchan/claude-workspace/internal/upgrade"
)

// version is set via -ldflags at build time
var version = "dev"

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
	// Wire embedded assets to the platform package, stripping the _template prefix
	sub, err := fs.Sub(PlatformFS, "_template")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing embedded assets: %v\n", err)
		os.Exit(1)
	}
	platform.FS = sub
	platform.InitColor()

	args := os.Args[1:]

	if len(args) == 0 {
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
	case "setup":
		if err := setup.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "attach":
		var target string
		if len(args) > 1 {
			target = args[1]
		}
		if err := attach.Run(target, args); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "sandbox":
		var projectPath, branchName string
		if len(args) > 1 {
			projectPath = args[1]
		}
		if len(args) > 2 {
			branchName = args[2]
		}
		if err := sandbox.Run(projectPath, branchName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "mcp":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: claude-workspace mcp <add|remote|list>")
			os.Exit(1)
		}
		subcmd := args[1]
		switch subcmd {
		case "add":
			if err := mcp.Add(args[2:]); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		case "remote":
			var url string
			if len(args) > 2 {
				url = args[2]
			}
			if err := mcp.Remote(url, args[3:]); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		case "list":
			if err := mcp.List(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		default:
			fmt.Fprintf(os.Stderr, "Unknown mcp subcommand: %s\n", subcmd)
			fmt.Println("Available: add, remote, list")
			os.Exit(1)
		}
	case "upgrade":
		if err := upgrade.Run(version, args[1:]); err != nil {
			if errors.Is(err, upgrade.ErrUpdateAvailable) {
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "doctor":
		if err := doctor.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "statusline":
		if err := statusline.Run(args[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "sessions":
		if err := sessions.Run(args[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "cost":
		if err := cost.Run(args[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		fmt.Print(helpText)
		os.Exit(1)
	}
}
