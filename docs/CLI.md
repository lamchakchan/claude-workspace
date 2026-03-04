# CLI Reference

Complete reference for all `claude-workspace` commands, flags, and options.

## claude-workspace (Interactive Mode)

Launch the interactive TUI when no subcommand is given.

**Synopsis:**

```
claude-workspace
```

**Requirements:** TTY (interactive terminal). Non-TTY environments print help text instead.

**Behavior:** Opens a full-screen launcher menu organized into four groups:

| Group | Commands |
|-------|----------|
| Getting Started | Setup, Attach, Enrich, Sandbox |
| MCP Servers | Add Server, List Servers, Remove Server |
| Inspect & Manage | Doctor, Skills, Agents, Hooks, Sessions, Memory, Cost, Config |
| Maintenance | Upgrade, Statusline |

Select a command to open its dedicated TUI view — form-based views (Attach, Enrich, Sandbox, MCP Add) include path autocomplete with tab completion. Skills, Agents, and Hooks use interactive expandable lists with cursor navigation (j/k), expand/collapse (enter), and scrollbar. Other data views (Doctor, Sessions, Cost, Config) display output inline with scrolling and clipboard copy.

**Environment variables:**

| Variable | Effect |
|----------|--------|
| `NO_COLOR` | Disables color output; falls back to help text instead of TUI |
| `ACCESSIBLE=1` | Same as `NO_COLOR` — skips TUI, prints help text |

**Keyboard shortcuts** (press `?` from any screen to see this reference):

| Context | Key | Action |
|---------|-----|--------|
| Navigation | `↑` / `k` | Move up |
| Navigation | `↓` / `j` | Move down |
| Navigation | `enter` | Select / confirm |
| Navigation | `esc` | Go back |
| Navigation | `q` / `ctrl+c` | Quit |
| Forms | `tab` / `↓` | Next field |
| Forms | `shift+tab` / `↑` | Previous field |
| Forms | `enter` | Next field / submit |
| Forms | `esc` | Cancel |
| Path autocomplete | `↑` / `↓` | Cycle suggestions |
| Path autocomplete | `tab` | Accept suggestion |
| Lists | `j` / `k` | Move up / down |
| Lists | `pgup` / `pgdn` | Page up / down |
| Lists | `g` / `G` | Go to top / bottom |
| Viewers | `j` / `k` | Scroll up / down |
| Viewers | `pgup` / `pgdn` | Page up / down |
| Viewers | `g` / `G` | Go to top / bottom |
| Viewers | `y` | Copy to clipboard |
| Confirmation | `y` / `n` | Confirm yes / no |
| Confirmation | `← / → / tab` | Switch selection |
| Cost tabs | `1`–`5` | Jump to tab |
| Cost tabs | `tab` / `h` / `l` | Cycle tabs |
| Config tabs | `1`–`9` | Jump to category tab |
| Config tabs | `tab` / `shift+tab` / `h` / `l` | Cycle category tabs |
| Config list | `/` | Enter filter mode |
| Config list | `e` | Edit selected key inline |
| Config edit | `tab` | Cycle scope (user / project / local) |
| Config edit | `enter` | Save value |
| Help | `?` | Toggle shortcut reference |

**Example:**

```
$ claude-workspace

claude-workspace  v0.x.x
Claude Code Platform Engineering Kit

Getting Started
> ⚙  Setup         First-time setup & API key provisioning
  📎 Attach        Overlay platform config onto a project
  ✨ Enrich        Re-generate CLAUDE.md with AI analysis
  🔀 Sandbox       Create a sandboxed branch worktree

MCP Servers
  ➕ Add Server    Add a local or remote MCP server
  📋 List Servers  Show all configured servers
  ➖ Remove Server Remove an MCP server

Inspect & Manage
  🩺 Doctor        Check platform configuration health
  🛠  Skills        List available skills and personal commands
  🤖 Agents        List configured agents
  🪝 Hooks         List configured hooks
  💬 Sessions      Browse and review session prompts
  🧠 Memory        Inspect and manage memory layers
  💰 Cost          View usage and costs
  🔧 Config        View and edit configuration

Maintenance
  ⬆  Upgrade       Upgrade claude-workspace and CLI
  📊 Statusline    Configure Claude Code statusline

↑/↓ navigate  enter select  ? help  q quit
```

**See also:** [Getting Started - First-Time Setup](GETTING-STARTED.md#3-first-time-setup)

---

## claude-workspace setup

First-time setup: installs Claude Code CLI, provisions API keys, configures global settings, installs the binary to PATH, installs Node.js if missing, registers MCP servers, and optionally configures the statusline for cost and context display.

**Synopsis:**

```
claude-workspace setup
```

**Flags:** None (interactive wizard).

**Examples:**

```bash
claude-workspace setup
```

**See also:** [Getting Started - Installation](GETTING-STARTED.md#2-installation)

---

## claude-workspace attach

Attach platform configuration (agents, hooks, skills, settings) to a project directory.

**Synopsis:**

```
claude-workspace attach <project-path> [--symlink] [--force] [--no-enrich]
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--symlink` | bool | `false` | Symlink assets from `~/.claude-workspace/assets/` instead of copying. Projects auto-update when the binary is upgraded. |
| `--force` | bool | `false` | Overwrite existing files (default skips files that already exist). |
| `--no-enrich` | bool | `false` | Skip AI-powered CLAUDE.md enrichment. By default, `attach` runs `claude -p` to analyze the project and enrich `.claude/CLAUDE.md` with real project context (directories, conventions, important files). Falls back gracefully to the static scaffold if the Claude CLI is unavailable or errors. |

**Examples:**

```bash
# Copy platform assets into a project (includes AI enrichment)
claude-workspace attach /path/to/my-project

# Use symlinks for automatic updates across projects
claude-workspace attach /path/to/my-project --symlink

# Refresh all platform files (overwrite existing)
claude-workspace attach /path/to/my-project --force

# Skip AI enrichment (use static scaffold only)
claude-workspace attach /path/to/my-project --no-enrich
```

**See also:** [Getting Started - Attaching to a Project](GETTING-STARTED.md)

---

## claude-workspace enrich

Re-generate `.claude/CLAUDE.md` with AI-powered project analysis, without re-running the full `attach` workflow. Useful when a project evolves and the CLAUDE.md falls out of date.

**Synopsis:**

```
claude-workspace enrich [project-path] [--scaffold-only]
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--scaffold-only` | bool | `false` | Generate the static scaffold only (skip AI enrichment). Useful without an API key or for a quick reset. |

**Behavior:**

1. Resolves the project directory (defaults to the current working directory if omitted).
2. Creates `.claude/` if it does not exist.
3. If `.claude/CLAUDE.md` is missing, generates a static scaffold (auto-detects tech stack from `go.mod`, `package.json`, `Cargo.toml`, `pyproject.toml`, `requirements.txt`, `pom.xml`, `build.gradle`, `build.gradle.kts`, `Gemfile`, `*.csproj`, `*.sln`, `mix.exs`, `composer.json`, `Package.swift`, `build.sbt`, `CMakeLists.txt`, `MODULE.bazel`, `WORKSPACE`, `Makefile`).
4. Unless `--scaffold-only`, runs `claude -p` with Opus to analyze the project and overwrite `.claude/CLAUDE.md` with enriched content (directories, conventions, important files). Falls back gracefully if the Claude CLI is unavailable or errors.

**Examples:**

```bash
# Re-enrich the current project's CLAUDE.md
claude-workspace enrich

# Enrich a specific project
claude-workspace enrich /path/to/my-project

# Generate scaffold only (no API key needed)
claude-workspace enrich --scaffold-only

# Generate scaffold for a specific project
claude-workspace enrich /path/to/my-project --scaffold-only
```

**See also:** [`claude-workspace attach --no-enrich`](#claude-workspace-attach)

---

## claude-workspace sandbox

Create a sandboxed git worktree branch for parallel Claude Code sessions on the same repository.

**Synopsis:**

```
claude-workspace sandbox <project-path> <branch-name>
```

**Flags:** None (positional arguments only).

**Examples:**

```bash
# Create a sandboxed worktree for a feature branch
claude-workspace sandbox /path/to/my-project feature-auth

# Multiple sandboxes for parallel work
claude-workspace sandbox /path/to/my-project feature-auth
claude-workspace sandbox /path/to/my-project bugfix-login
```

**See also:** [Architecture - Sandboxing](ARCHITECTURE.md)

---

## claude-workspace mcp add

Add a local or remote MCP server with secure credential handling.

**Synopsis:**

```
claude-workspace mcp add <name> [options] [-- <command> [args...]]
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--scope` | `local\|project\|user` | `local` | Where to save the server configuration. |
| `--transport` | `stdio\|http\|sse` | auto-detected | Transport protocol. Auto-detects `http` if a URL is provided, otherwise `stdio`. |
| `--api-key` | `ENV_VAR_NAME` | — | Prompt for an API key (masked input). Stored as the named environment variable in `~/.claude.json`. |
| `--bearer` | bool | `false` | Prompt for a Bearer token (masked input). Added as an Authorization header. |
| `--oauth` | bool | `false` | Use OAuth 2.0 authentication (complete via `/mcp` in Claude Code). |
| `--client-id` | string | — | OAuth client ID for pre-registered applications. |
| `--client-secret` | bool | `false` | Prompt for OAuth client secret (masked input). |
| `--env` | `KEY=VALUE` | — | Set an environment variable (repeatable, visible in config). |
| `--header` | `'Key: Value'` | — | Add a custom HTTP header (repeatable). |

**Examples:**

```bash
# Local server with API key (prompted securely)
claude-workspace mcp add brave-search --scope user --api-key BRAVE_API_KEY \
  -- npx -y @modelcontextprotocol/server-brave-search

# Database server
claude-workspace mcp add postgres --scope user --api-key DATABASE_URL \
  -- npx -y @bytebase/dbhub

# GitHub (OAuth — authenticate via /mcp in Claude Code)
claude-workspace mcp remote https://api.githubcopilot.com/mcp/ --scope user --name github

# GitHub (PAT — you'll be prompted for your Personal Access Token)
claude-workspace mcp remote https://api.githubcopilot.com/mcp/ --scope user --name github --bearer
```

**See also:** [Getting Started - MCP Servers](GETTING-STARTED.md)

---

## claude-workspace mcp remote

Connect to a remote MCP server or gateway.

**Synopsis:**

```
claude-workspace mcp remote <url> [options]
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--name` | string | derived from URL | Human-readable server name. |
| `--scope` | `local\|project\|user` | `user` | Where to save the server configuration. |
| `--bearer` | bool | `false` | Prompt for a Bearer token (masked input). |
| `--oauth` | bool | `false` | Use OAuth 2.0 authentication. |
| `--client-id` | string | — | OAuth client ID for pre-registered applications. |
| `--client-secret` | bool | `false` | Prompt for OAuth client secret (masked input). |
| `--header` | `'Key: Value'` | — | Add a custom HTTP header (repeatable). |

**Examples:**

```bash
# OAuth servers (authenticate via /mcp in Claude Code)
claude-workspace mcp remote https://mcp.sentry.dev/mcp --scope user --name sentry
claude-workspace mcp remote https://mcp.notion.com/mcp --scope user --name notion

# Bearer token authentication
claude-workspace mcp remote https://mcp.example.com --scope user --bearer

# Organization gateway
claude-workspace mcp remote https://mcp-gateway.company.com --scope user --name company
```

**See also:** [Getting Started - MCP Servers](GETTING-STARTED.md)

---

## claude-workspace mcp list

List all configured MCP servers (user-level and project-level).

**Synopsis:**

```
claude-workspace mcp list
```

**Flags:** None.

**Examples:**

```bash
claude-workspace mcp list
```

---

## claude-workspace mcp remove

Remove an MCP server from your configuration.

**Synopsis:**

```
claude-workspace mcp remove <name> [options]
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--scope` | `local\|project\|user` | `user` | Which config to remove from. |

**Examples:**

```bash
# Remove a server from user config (default)
claude-workspace mcp remove brave-search

# Remove a server from project config
claude-workspace mcp remove sentry --scope project

# Remove a server from local config
claude-workspace mcp remove postgres --scope local
```

---

## claude-workspace upgrade

Check for updates and upgrade both the `claude-workspace` binary and the Claude Code CLI.

**Synopsis:**

```
claude-workspace upgrade [--check] [--yes] [--self-only | --cli-only]
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--check` | bool | `false` | Check for updates and print version comparison. Exits 0 if up-to-date, 1 if an update is available. |
| `--yes`, `-y` | bool | `false` | Non-interactive mode: skip all confirmation prompts. |
| `--self-only` | bool | `false` | Only upgrade `claude-workspace` (skip Claude Code CLI). |
| `--cli-only` | bool | `false` | Only upgrade Claude Code CLI (skip `claude-workspace`). |

`--self-only` and `--cli-only` are mutually exclusive.

**What gets upgraded:**

1. **Binary** — downloads the latest release from GitHub and replaces the installed binary.
2. **Shared assets** — re-extracts `~/.claude-workspace/assets/` so symlinked projects auto-update.
3. **Global settings** — non-destructive merge of new platform defaults into `~/.claude/settings.json`.
4. **Claude Code CLI** — runs the official installer (`claude.ai/install.sh`) to install or upgrade the Claude Code CLI.

Projects using `--symlink` mode pick up new agents, hooks, and skills automatically. Projects using copy mode should re-run `claude-workspace attach --force`.

**Examples:**

```bash
# Upgrade everything (claude-workspace + Claude Code CLI)
claude-workspace upgrade

# Check only (CI-friendly: exit 0 = up-to-date, exit 1 = update available)
claude-workspace upgrade --check

# Non-interactive upgrade (for scripts/CI)
claude-workspace upgrade --yes

# Only upgrade claude-workspace, skip CLI
claude-workspace upgrade --self-only

# Only upgrade Claude Code CLI, skip self
claude-workspace upgrade --cli-only
```

---

## claude-workspace doctor

Run a comprehensive health check on your platform configuration.

**Synopsis:**

```
claude-workspace doctor
```

**Flags:** None.

Checks performed:
- Claude Code CLI installation
- `claude-workspace` in PATH (+ update availability)
- Git installation
- Global configuration (`~/.claude/settings.json`, `~/.claude/CLAUDE.md`)
- Project configuration (settings, agents, skills, hooks, MCP servers)
- Hook executability and configuration
- Authentication status

**Examples:**

```bash
# Run from your project directory
cd /path/to/my-project
claude-workspace doctor
```

**See also:** [Runbook - Troubleshooting](RUNBOOK.md)

---

## claude-workspace skills

List available skills (project-level) and personal commands (`~/.claude/commands/`).

**Synopsis:**

```
claude-workspace skills [list]
```

**Subcommands:**

| Subcommand | Description |
|------------|-------------|
| `list` | List all discovered skills and personal commands (default) |

**Flags:** None.

**Sources scanned:**

1. **Project skills** — `.claude/skills/*/SKILL.md` in the current directory. Parses YAML frontmatter for `name` and `description`.
2. **Personal commands** — `~/.claude/commands/*.md`. Uses filename as name, first non-empty line as description.

**Examples:**

```bash
# List all skills (default subcommand)
claude-workspace skills

# Explicit list subcommand
claude-workspace skills list
```

**See also:** [Skills Reference](SKILLS.md)

---

## claude-workspace agents

List configured agents from project and user-global sources.

**Synopsis:**

```
claude-workspace agents [list]
```

**Subcommands:**

| Subcommand | Description |
|------------|-------------|
| `list` | List all discovered agents (default) |

**Flags:** None.

**Sources scanned:**

1. **Project agents** — `.claude/agents/*.md` in the current directory. Parses YAML frontmatter for `name`, `description`, `model`, and `tools`.
2. **User-global agents** — `~/.claude/agents/*.md`. Same frontmatter parsing.

**Agent frontmatter fields:**

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Agent identifier (kebab-case) |
| `description` | yes | What the agent does |
| `model` | yes | Model to use (`haiku`, `sonnet`, `opus`) |
| `tools` | yes | Comma-separated list of allowed tools |

**Examples:**

```bash
# List all agents (default subcommand)
claude-workspace agents

# Explicit list subcommand
claude-workspace agents list
```

**Example output:**

```
═══════════════════════════════════
  Agents
═══════════════════════════════════

  Project Agents (.claude/agents/)
  code-reviewer         sonnet  Code quality and correctness review...
  explorer              haiku   Fast codebase exploration and context gathering...
  planner               opus    Deep planning agent for complex tasks...
  test-runner           sonnet  Test execution and failure diagnosis...

  Tips
  Agents are invoked automatically by Claude Code when matching tasks arise.
  Create new:  .claude/agents/my-agent.md
```

---

## claude-workspace hooks

List configured hook scripts and hook event configuration from settings.json.

**Synopsis:**

```
claude-workspace hooks [list]
```

**Subcommands:**

| Subcommand | Description |
|------------|-------------|
| `list` | List all discovered hooks (default) |

**Flags:** None.

**Sources scanned:**

1. **Project hook scripts** — `.claude/hooks/*.sh` in the current directory. Extracts the description from the first comment line (after the shebang and `set` directives).
2. **Hook configuration** — `.claude/settings.json` `hooks` key. Shows event bindings with matcher patterns and status messages.

**Examples:**

```bash
# List all hooks (default subcommand)
claude-workspace hooks

# Explicit list subcommand
claude-workspace hooks list
```

**Example output:**

```
═══════════════════════════════════
  Hooks
═══════════════════════════════════

  Project Hook Scripts (.claude/hooks/)
  auto-format.sh              Auto-formats written files using project formatter
  block-dangerous-commands.sh Blocks dangerous shell commands
  enforce-branch-policy.sh    Prevents direct commits to main/master
  validate-secrets.sh         Scans file content being written for potential secrets
  verify-task-completed.sh    Runs project tests before marking task complete

  Hook Configuration (settings.json)
  EVENT          MATCHER      STATUS MESSAGE
  PreToolUse     Bash         Checking command safety...
  PreToolUse     Bash         Checking branch policy...
  PreToolUse     Write|Edit   Scanning for secrets...
  PostToolUse    Write|Edit   Auto-formatting...
  TaskCompleted  (any)        Verifying task completion...

  Tips
  Hooks are shell scripts that run before/after tool use or on events.
  Create new:  .claude/hooks/my-hook.sh (must be executable)
  Configure:   .claude/settings.json under "hooks" key
```

---

## claude-workspace statusline

Configure the Claude Code statusline to display live session cost, context usage, model name, weekly reset countdown, and service status alerts.

**Synopsis:**

```
claude-workspace statusline [--force]
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--force` | bool | `false` | Overwrite existing `statusLine` configuration. |

**Runtime detection** (in preference order):

1. `bun x ccusage statusline` — if `bun` is available (fastest)
2. `npx -y ccusage statusline` — if `npx` is available
3. Inline `jq` fallback — if neither runtime is found (requires `jq`)

**Service status alerts:**

When any monitored service is experiencing issues, a colored alert line appears above the normal statusline. Alerts are cached for 5 minutes with a 2-second HTTP timeout. Monitored services:

| Service | API | Format |
|---------|-----|--------|
| GitHub | `githubstatus.com/api/v2/status.json` | Atlassian Statuspage |
| Claude | `status.claude.com/api/v2/status.json` | Atlassian Statuspage |
| Cloudflare | `cloudflarestatus.com/api/v2/status.json` | Atlassian Statuspage |
| AWS | `health.aws.amazon.com/public/currentevents` | Custom (event array) |
| Google Cloud | `status.cloud.google.com/incidents.json` | Custom (incidents) |
| Azure DevOps | `status.dev.azure.com/_apis/status/health` | Custom (health) |

Requires `python3` (standard on macOS and most Linux); silently omitted if unavailable.

**Behavior:**

- Idempotent by default: skips if `statusLine` is already configured in `~/.claude/settings.json`
- Creates `~/.claude/settings.json` if it does not yet exist
- Restart Claude Code after running to activate the statusline

**Example output** (using ccusage, all services healthy):

```
Opus | $0.23 session / $1.23 today / $0.45 block (2h 45m left) | $0.12/hr | 25,000 (12%) | resets in 3d
```

**Example output** (with service issues — multiline, colored):

```
🚨 GitHub: Major System Outage  ⚠️  Claude: Degraded Performance
Opus | $0.23 session / $1.23 today / $0.45 block (2h 45m left) | $0.12/hr | 25,000 (12%) | resets in 3d
```

**Examples:**

```bash
# Detect runtime and configure statusline
claude-workspace statusline

# Overwrite existing configuration
claude-workspace statusline --force
```

**See also:** [ccusage](https://github.com/ryoppippi/ccusage), [Claude Code statusline docs](https://docs.anthropic.com/en/docs/claude-code/settings#status-line)

---

## claude-workspace sessions

Browse and review prompts from past Claude Code sessions. Reads session data directly from `~/.claude/projects/` — no extra capture step required.

**Synopsis:**

```
claude-workspace sessions [list|show] [options]
```

**Subcommands:**

| Subcommand | Description |
|------------|-------------|
| `list` | List sessions for the current project (default when no subcommand given) |
| `show <id>` | Display all user prompts from a specific session |

**Flags (list):**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--all` | bool | `false` | List sessions across all projects (adds project name prefix to titles). |
| `--limit` | int | `20` | Maximum number of sessions to display. |

**How it works:**

- Session data lives in `~/.claude/projects/<encoded-path>/<uuid>.jsonl`
- Each JSONL file is one conversation session (append-only, one JSON object per line)
- The **title** is derived from the first real user message (slash commands and system messages are filtered out)
- The **session ID** prefix (8 characters) is enough to uniquely identify a session for `show`
- Sessions are sorted newest-first

**Examples:**

```bash
# List recent sessions for the current project
claude-workspace sessions

# List all sessions across every project
claude-workspace sessions list --all

# Show more results
claude-workspace sessions list --limit 50

# View all prompts from a specific session (prefix match)
claude-workspace sessions show 8a3f1b2c
```

**Example output (list):**

```
=== Sessions for /Users/you/my-project ===

  ID          DATE          TITLE
  ----------  ------------  --------------------------------------------------
  8a3f1b2c    2026-02-24    Add authentication middleware to the API gateway
  e13fdc87    2026-02-23    Fix the MCP add command to properly handle env vars
  c7d2a901    2026-02-22    Refactor the upgrade command to support --check flag

  3 session(s) shown. Use 'sessions show <id>' to view prompts.
```

**Example output (show):**

```
=== zesty-sauteeing-avalanche (8a3f1b2c) ===
  Project: /Users/you/my-project
  Prompts: 3

  [1] 15:26:47
  Add authentication middleware to the API gateway

  [2] 15:28:30
  Can you also add rate limiting?

  [3] 15:49:47
  Stage and commit the changes, and push it up.
```

---

## claude-workspace cost

View Claude Code usage and costs by querying local session data via [ccusage](https://github.com/ryoppippi/ccusage). All arguments are forwarded verbatim to ccusage.

**Synopsis:**

```
claude-workspace cost [subcommand] [options]
```

**Subcommands:**

| Subcommand | Description |
|------------|-------------|
| `daily` | Usage grouped by day (default) |
| `weekly` | Usage grouped by week |
| `monthly` | Usage grouped by month |
| `session` | Usage grouped by conversation session |
| `blocks` | Usage grouped by 5-hour billing window |

**Key flags:**

| Flag | Description |
|------|-------------|
| `--breakdown` | Show per-model cost breakdown |
| `--since YYYYMMDD` | Filter results from this date onwards |
| `--until YYYYMMDD` | Filter results up to this date |
| `--json` | Output raw JSON instead of a table |
| `--project <name>` | Filter by project name |
| `--instances` | Show per-instance breakdown |

All ccusage flags pass through verbatim. See `npx ccusage --help` for the full flag reference.

**Runtime detection** (in preference order):

1. `bun x ccusage` — if `bun` is available (fastest)
2. `npx -y ccusage` — if `npx` is available

If neither runtime is found, an error is printed with install instructions.

**Examples:**

```bash
# Show today's cost summary (daily is the default)
claude-workspace cost

# Monthly breakdown by model
claude-workspace cost monthly --breakdown

# Show active 5-hour billing block
claude-workspace cost blocks --active

# Filter daily costs since January 1, 2026
claude-workspace cost daily --since 20260101

# JSON output for scripting
claude-workspace cost --json
```

**See also:** [ccusage](https://github.com/ryoppippi/ccusage)

---

## claude-workspace config

View and edit all Claude Code configuration across every scope layer, with clear attribution showing which layer each value comes from.

**Synopsis:**

```
claude-workspace config [subcommand] [options]
```

**Subcommands:**

| Subcommand | Description |
|------------|-------------|
| *(no args)* | Launch interactive TUI config viewer/editor (TTY required) |
| `view` | Non-interactive formatted output of all config with scope badges |
| `get <key>` | Show a single key with its value at every scope layer |
| `set <key> <value>` | Write a config value to the target scope's `settings.json` |

**Flags (set):**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--scope` | `user\|project\|local` | `user` | Which `settings.json` to write to |

**TUI behavior:**

Launches the full interactive config viewer when no subcommand is given and stdin is a TTY. The TUI displays all known Claude Code configuration keys organized by category with source badges showing where each value originates:

| Badge | Meaning |
|-------|---------|
| `[MGD]` | Enterprise managed settings (highest priority, cannot be overridden) |
| `[USR]` | User-level `~/.claude/settings.json` |
| `[PRJ]` | Project-level `.claude/settings.json` |
| `[LOC]` | Local override `.claude/settings.local.json` |
| `[ENV]` | OS environment variable or `settings.json` `env` block |
| `[DEF]` | Registry default (no active override) |

**Examples:**

```bash
# Interactive TUI (TTY only)
claude-workspace config

# Non-interactive — show all config grouped by category
claude-workspace config view

# Show a single key with all scope layers
claude-workspace config get model
claude-workspace config get CLAUDE_CODE_MAX_OUTPUT_TOKENS

# Write a value to user settings (default scope)
claude-workspace config set model opus

# Write a value to project settings
claude-workspace config set model sonnet --scope project

# Write a value to local settings (gitignored, personal override)
claude-workspace config set model haiku --scope local
```

**Example output (`config view`):**

```
═══════════════════════════════════
  Claude Code Configuration
═══════════════════════════════════

Core
  [DEF] model = (none)  (Override the default model...)
  [PRJ] CLAUDE_CODE_SUBAGENT_MODEL = opus  (Model for all subagents)
  ...

Env: Model & Tokens
  [ENV] CLAUDE_CODE_MAX_OUTPUT_TOKENS = (none)  (Max tokens per response...)
  ...
```

**Example output (`config get model`):**

```
═══════════════════════════════════
  model
═══════════════════════════════════
  Type:        string
  Description: Override the default model (alias or full model ID)

  Effective Value
  [PRJ] sonnet

  Layer Values
  [USR] opus
  [PRJ] sonnet
```

**See also:** [Configuration Reference](CONFIG.md)

---

## Global Options

These options are available on all commands:

| Flag | Description |
|------|-------------|
| `--help`, `-h` | Show help message |
| `--version`, `-v` | Show version |
