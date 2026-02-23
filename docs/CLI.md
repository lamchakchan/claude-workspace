# CLI Reference

Complete reference for all `claude-workspace` commands, flags, and options.

## claude-workspace setup

First-time setup: installs Claude Code CLI, provisions API keys, configures global settings, installs the binary to PATH, installs Node.js if missing, and registers MCP servers.

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
claude-workspace mcp add brave-search --api-key BRAVE_API_KEY \
  -- npx -y @modelcontextprotocol/server-brave-search

# Database server
claude-workspace mcp add postgres --api-key DATABASE_URL \
  -- npx -y @bytebase/dbhub

# Remote server with auto-detected OAuth
claude-workspace mcp add github --transport http \
  https://api.githubcopilot.com/mcp/
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
claude-workspace mcp remote https://mcp.sentry.dev/mcp --name sentry
claude-workspace mcp remote https://mcp.notion.com/mcp --name notion

# Bearer token authentication
claude-workspace mcp remote https://mcp.example.com --bearer

# Organization gateway
claude-workspace mcp remote https://mcp-gateway.company.com --name company
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

## Global Options

These options are available on all commands:

| Flag | Description |
|------|-------------|
| `--help`, `-h` | Show help message |
| `--version`, `-v` | Show version |
