// Package main provides the claude-workspace CLI, a platform engineering kit
// for deploying Claude Code AI agents across organizations. It embeds template
// assets at compile time and routes subcommands to their respective packages.
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/lamchakchan/claude-workspace/internal/agents"
	"github.com/lamchakchan/claude-workspace/internal/attach"
	"github.com/lamchakchan/claude-workspace/internal/config"
	"github.com/lamchakchan/claude-workspace/internal/cost"
	"github.com/lamchakchan/claude-workspace/internal/doctor"
	"github.com/lamchakchan/claude-workspace/internal/enrich"
	"github.com/lamchakchan/claude-workspace/internal/hooks"
	"github.com/lamchakchan/claude-workspace/internal/mcp"
	"github.com/lamchakchan/claude-workspace/internal/memory"
	"github.com/lamchakchan/claude-workspace/internal/platform"
	"github.com/lamchakchan/claude-workspace/internal/plugins"
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
	"config":     runConfig,
	"doctor":     func(_ []string) error { return doctor.Run() },
	"agents":     func(a []string) error { return agents.Run(a[1:]) },
	"hooks":      func(a []string) error { return hooks.Run(a[1:]) },
	"statusline": func(a []string) error { return statusline.Run(a[1:]) },
	"memory":     func(a []string) error { return memory.Run(a[1:]) },
	"sessions":   func(a []string) error { return sessions.Run(a[1:]) },
	"cost":       func(a []string) error { return cost.Run(a[1:]) },
	"plugins":    func(a []string) error { return plugins.Run(a[1:]) },
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
  sandbox create <path> <name>   Create a sandboxed branch worktree
  sandbox list <path>            List sandboxes for a project
  sandbox remove <path> <name>   Remove a sandboxed branch worktree
  mcp add <name> [options]       Add an MCP server (local or remote)
  mcp remote <url>               Connect to a remote MCP server/gateway
  mcp list                       List all configured MCP servers
  mcp remove <name>              Remove an MCP server
  upgrade [--self-only|--cli-only]  Upgrade claude-workspace and Claude Code CLI
  doctor                         Check platform configuration health
  agents [list]                  List configured agents
  hooks [list]                   List configured hooks and hook scripts
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
  plugins [subcommand]           Manage Claude Code plugins
    (no args) / list             List installed plugins
    add <plugin[@marketplace]>   Install a plugin
      [--scope user|project]     Installation scope (default: user)
    remove <plugin>              Remove an installed plugin
      [--scope user|project]     Scope (default: user)
    available                    List available plugins from marketplaces
    marketplace list             List configured plugin marketplaces
    marketplace add <target>     Add a plugin marketplace (owner/repo or local path)
    marketplace remove <name>    Remove a configured marketplace

  Plugin examples:
    claude-workspace plugins list
    claude-workspace plugins add code-review@claude-plugins-official
    claude-workspace plugins remove code-review --scope user
    claude-workspace plugins marketplace list
    claude-workspace plugins marketplace add anthropics/claude-plugins-official
    claude-workspace plugins marketplace add ~/git/myorg/my-plugins
    claude-workspace plugins marketplace add /home/user/git/org/repo
    claude-workspace plugins marketplace remove claude-plugins-official

  config [subcommand]            View and edit all Claude Code configuration
    (no args)                    Launch interactive TUI config viewer/editor
    view                         Non-interactive formatted output of all config
    get <key>                    Show a single key with all scope layers
    set <key> <value>            Set a config value
      [--scope user|project|local]  Which settings.json to write (default: user)

Options:
  --help, -h       Show this help message
  --version, -v    Show version

MCP Authentication:
  --api-key ENV_NAME     Securely prompt for API key (masked input)
  --bearer               Securely prompt for Bearer token (masked input)
  --oauth                Use OAuth 2.0 (authenticate via /mcp in session)
  --client-id <id>       OAuth client ID for pre-registered apps
  --client-secret        Prompt for OAuth client secret (masked input)
  --header 'Key: Value'  Add custom HTTP header (repeatable)

Examples:
  claude-workspace setup
  claude-workspace attach /path/to/my-project
  claude-workspace sandbox create /path/to/my-project feature-auth
  claude-workspace sandbox list /path/to/my-project
  claude-workspace mcp add postgres --scope user --api-key DATABASE_URL -- npx -y @bytebase/dbhub
  claude-workspace mcp add brave --scope user --api-key BRAVE_API_KEY -- npx -y @modelcontextprotocol/server-brave-search
  claude-workspace mcp remote https://mcp.sentry.dev/mcp --scope user --name sentry
  claude-workspace mcp remote https://mcp-gateway.company.com --scope user --bearer
  claude-workspace mcp remote https://mcp.example.com --scope user --header 'X-API-Key: mykey'
  claude-workspace mcp remove brave-search
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

	mcpConfigSub, err := fs.Sub(McpConfigFS, "docs/mcp-configs")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing embedded MCP configs: %v\n", err)
		os.Exit(1)
	}
	platform.McpConfigFS = mcpConfigSub

	marketplaceRegistrySub, err := fs.Sub(MarketplaceRegistryFS, "docs/plugin-marketplaces")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing embedded marketplace registry: %v\n", err)
		os.Exit(1)
	}
	platform.MarketplaceRegistryFS = marketplaceRegistrySub
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
	subcmd := ""
	if len(args) > 1 {
		subcmd = args[1]
	}

	switch subcmd {
	case "create":
		var projectPath, branchName string
		if len(args) > 2 {
			projectPath = args[2]
		}
		if len(args) > 3 {
			branchName = args[3]
		}
		return sandbox.Create(projectPath, branchName)
	case "remove":
		var projectPath, branchName string
		if len(args) > 2 {
			projectPath = args[2]
		}
		if len(args) > 3 {
			branchName = args[3]
		}
		return sandbox.Remove(projectPath, branchName)
	case "list":
		var projectPath string
		if len(args) > 2 {
			projectPath = args[2]
		}
		return sandbox.List(projectPath)
	default:
		// Backward compat: sandbox <path> <branch> defaults to create
		var branchName string
		if len(args) > 2 {
			branchName = args[2]
		}
		return sandbox.Create(subcmd, branchName)
	}
}

func runMCP(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: claude-workspace mcp <add|remote|remove|list>")
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
	case "remove":
		return mcp.Remove(args[2:])
	default:
		return fmt.Errorf("unknown mcp subcommand: %s (available: add, remote, remove, list)", subcmd)
	}
}

func runUpgrade(args []string) error {
	return upgrade.Run(version, args[1:])
}

func runConfig(args []string) error {
	subArgs := args[1:]
	// No subcommand and connected to a TTY: launch config TUI directly
	if len(subArgs) == 0 && platform.IsTTY() {
		return tui.RunConfig(version)
	}
	return config.Run(subArgs)
}
