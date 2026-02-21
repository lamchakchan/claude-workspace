# Claude Code Platform Engineering Kit

A preconfigured, batteries-included platform for deploying Claude Code AI agents across your organization. Designed for teams adopting AI-assisted development for the first time, with safe defaults, layered prompt architecture, and multi-project support.

## Architecture Overview

```
claude-platform/
├── .claude/                        # Core Claude Code configuration
│   ├── settings.json               # Team-shared settings (safe defaults)
│   ├── settings.local.json.example # Template for personal overrides
│   ├── CLAUDE.md                   # Team-shared system prompt
│   ├── CLAUDE.local.md.example     # Template for personal context
│   ├── agents/                     # Custom subagent definitions
│   │   ├── planner.md              # Deep planning agent (plan-first workflow)
│   │   ├── code-reviewer.md        # Automated code review agent
│   │   ├── explorer.md             # Codebase exploration & context gathering
│   │   ├── test-runner.md          # Test execution & validation agent
│   │   └── security-scanner.md     # Security analysis agent
│   ├── skills/                     # Reusable skill definitions
│   │   ├── plan-and-execute/       # Plan-first development workflow
│   │   ├── context-manager/        # Large codebase context strategy
│   │   ├── pr-workflow/            # PR creation & review workflow
│   │   └── onboarding/             # New project onboarding skill
│   └── hooks/                      # Safety & quality gate scripts
│       ├── block-dangerous-commands.sh
│       ├── enforce-branch-policy.sh
│       ├── auto-format.sh
│       └── validate-secrets.sh
├── main.go                         # Go CLI entry point
├── assets.go                       # Embedded asset declarations
├── internal/                       # Go command implementations
│   ├── platform/                   # Shared helpers (exec, json, fs, assets)
│   ├── setup/                      # First-time setup & API key provisioning
│   ├── attach/                     # Attach platform to any project repo
│   ├── sandbox/                    # Create parallel branch worktrees
│   ├── mcp/                        # MCP server management
│   └── doctor/                     # Health check & diagnostics
├── .mcp.json                       # Project-scoped MCP servers
├── templates/
│   └── mcp-configs/                # Ready-to-use MCP configurations
│       ├── database.json           # PostgreSQL/MySQL/SQLite MCP
│       ├── observability.json      # Sentry/Grafana MCP
│       └── collaboration.json      # GitHub/Notion/Slack/Linear MCP
├── CLAUDE.md                       # Global platform-level instructions
└── plans/                          # Generated plans directory
```

## Quick Start

### Prerequisites

- **Node.js 18+** and **npm** (for Claude Code CLI)
- **Git** (for version control and worktree sandboxing)

### Install

**One-liner (macOS / Linux):**

```bash
curl -fsSL https://raw.githubusercontent.com/lamchakchan/claude-platform/main/install.sh | bash
```

**Or download manually** from [GitHub Releases](https://github.com/lamchakchan/claude-platform/releases):

```bash
# macOS (Apple Silicon)
curl -fsSL https://github.com/lamchakchan/claude-platform/releases/latest/download/claude-platform_<version>_darwin_arm64.tar.gz | tar xz
sudo mv claude-platform /usr/local/bin/

# Linux (x86_64)
curl -fsSL https://github.com/lamchakchan/claude-platform/releases/latest/download/claude-platform_<version>_linux_amd64.tar.gz | tar xz
sudo mv claude-platform /usr/local/bin/
```

**Or build from source:**

```bash
git clone https://github.com/lamchakchan/claude-platform.git
cd claude-platform
make install   # builds and copies to /usr/local/bin
```

### Setup

```bash
# 1. Run setup (installs Claude Code CLI, provisions API key, configures globals)
claude-platform setup

# 2. Attach to your project and start coding
claude-platform attach /path/to/your/project
cd /path/to/your/project && claude
```

The `setup` command handles everything: installs Claude Code CLI if missing, runs interactive API key provisioning, creates global settings, and installs optional tools.

## Key Features

### 1. Safe Defaults
- Dangerous commands blocked by hooks (rm -rf, force push to main)
- Secrets detection prevents committing .env files or credentials
- Branch policy enforcement (no direct pushes to main/master)
- Default model set to latest Sonnet for cost-effective coding
- Opus available for complex reasoning tasks via subagent config

### 2. Plan-First Workflow
- Every significant task starts with a visible plan
- Plans are written to `./plans/` directory for review
- Planner subagent creates structured, reviewable implementation plans
- Plan-and-execute skill enforces the plan-then-implement pattern
- Extended thinking enabled for architectural decisions
- TodoWrite used throughout for visible progress tracking

### 3. Multi-Project Support
- `claude-platform attach <path>` copies/symlinks platform config into any repo
- Each project gets its own CLAUDE.md layer for project-specific context
- Shared agents and skills work across all attached projects
- `--symlink` flag keeps config in sync across all attached projects (shared via `~/.claude-platform/assets/`)

### 4. Parallel Branch Sandboxing
- `claude-platform sandbox <path> <branch>` creates git worktrees
- Multiple Claude Code instances can work on the same repo simultaneously
- Each sandbox gets its own branch and isolated working directory
- Dependencies auto-installed in each worktree

### 5. Layered Prompt System
```
Priority (highest to lowest):
├── Managed Settings          # Org-wide policy (IT-deployed)
├── ~/.claude/CLAUDE.md       # Personal global instructions
├── .claude/CLAUDE.md         # Team project instructions
├── .claude/CLAUDE.local.md   # Personal project instructions
└── Subagent/Skill prompts    # Task-specific instructions
```

### 6. MCP Integration
- Preconfigured local MCP servers (memory, filesystem, git)
- Ready-to-use templates for databases, Docker, observability, collaboration tools
- Easy CLI to add new local MCP: `claude-platform mcp add <name> -- <cmd>`
- Remote MCP gateway: `claude-platform mcp remote <url>`
- Managed MCP policies for organizational control

### 7. Flexible Model Selection
- Default: Latest Sonnet for all coding work (`model: sonnet` in settings)
- Explorer subagent: Haiku for fast, cheap codebase discovery
- Planner/Reviewer/Security: Sonnet for detailed analysis
- Team agents: Sonnet default, Opus available for complex reasoning
- Override per-session: `ANTHROPIC_MODEL=opus claude`
- Override subagent model: `CLAUDE_CODE_SUBAGENT_MODEL=opus`

### 8. Custom Subagents
| Agent | Model | Purpose |
|-------|-------|---------|
| `planner` | Sonnet | Deep planning with persistent memory |
| `code-reviewer` | Sonnet | Automated code review |
| `explorer` | Haiku | Fast codebase exploration |
| `test-runner` | Sonnet | Test execution & analysis |
| `security-scanner` | Sonnet | Security vulnerability scanning |

## Self-Provisioning API Key (Option 2)

```bash
claude-platform setup
# Follow the interactive flow to provision your key
```

## Adding MCP Servers

### Add a local MCP server
```bash
# Basic local server
claude-platform mcp add postgres -- npx -y @bytebase/dbhub --dsn "postgresql://..."

# With API key (securely prompted, masked input)
claude-platform mcp add brave --api-key BRAVE_API_KEY -- npx -y @modelcontextprotocol/server-brave-search
claude-platform mcp add postgres --api-key DATABASE_URL -- npx -y @bytebase/dbhub

# Via Claude Code directly
claude mcp add --transport http github https://api.githubcopilot.com/mcp/
```

### Connect to a remote MCP gateway
```bash
# OAuth-based (authenticate in Claude Code via /mcp)
claude-platform mcp remote https://api.githubcopilot.com/mcp/ --name github

# Bearer token (securely prompted, masked input)
claude-platform mcp remote https://mcp-gateway.company.com --name gateway --bearer

# OAuth with pre-registered client
claude-platform mcp remote https://mcp.example.com --name example --oauth --client-id my-app-id --client-secret
```

### MCP Authentication Options

| Flag | Description |
|------|-------------|
| `--api-key ENV_NAME` | Securely prompt for an API key, stored as environment variable |
| `--bearer` | Securely prompt for a Bearer token (masked input) |
| `--oauth` | Use OAuth 2.0 (authenticate via `/mcp` in session) |
| `--client-id <id>` | OAuth client ID for pre-registered apps |
| `--client-secret` | Securely prompt for OAuth client secret |

All secrets are entered via masked input (not visible on screen) and stored in local Claude config — **never** in `.mcp.json` (which is committed to git).

### Use a template
Check `templates/mcp-configs/` for ready-to-use configurations for databases, Sentry, GitHub, Notion, Slack, Linear, and more.

## Parallel Development

### Using Git Worktrees (Native)
```bash
claude-platform sandbox /path/to/project feature-auth
claude-platform sandbox /path/to/project feature-api

# Each worktree gets its own branch and Claude Code instance
cd /path/to/project-worktrees/feature-auth && claude
cd /path/to/project-worktrees/feature-api && claude
```

### Using Agent Teams (Experimental)
Agent teams are enabled by default in this platform. **tmux is not required** — the default in-process mode works in any terminal. Ask Claude to create a team:
```
Create an agent team with 3 teammates to implement the auth module,
API endpoints, and frontend components in parallel.
```
For split-pane view (each teammate in its own pane), install tmux and set `"teammateMode": "tmux"` in settings.

## Health Check

```bash
claude-platform doctor
```

This checks: Claude Code CLI, Git, global settings, project config, agents, skills, hooks, MCP servers, and authentication.

## Configuration Reference

| File | Purpose | Shared? |
|------|---------|---------|
| `.claude/settings.json` | Team defaults, permissions, hooks | Yes (git) |
| `.claude/settings.local.json` | Personal overrides | No (gitignored) |
| `.claude/CLAUDE.md` | Team instructions | Yes (git) |
| `.claude/CLAUDE.local.md` | Personal instructions | No (gitignored) |
| `.claude/agents/*.md` | Subagent definitions | Yes (git) |
| `.claude/skills/*/SKILL.md` | Skill definitions | Yes (git) |
| `.claude/hooks/*.sh` | Hook scripts | Yes (git) |
| `.mcp.json` | Project MCP servers | Yes (git) |
| `~/.claude/settings.json` | Global user settings | No (local) |
| `~/.claude/CLAUDE.md` | Global user instructions | No (local) |
| `~/.claude/agents/*.md` | User-level agents | No (local) |

## Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `ANTHROPIC_API_KEY` | API authentication | (required) |
| `ANTHROPIC_MODEL` | Override default model | `sonnet` |
| `CLAUDE_CODE_SUBAGENT_MODEL` | Subagent model | `sonnet` |
| `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS` | Enable agent teams | `1` |
| `CLAUDE_AUTOCOMPACT_PCT_OVERRIDE` | Auto-compact threshold | `80` |
| `CLAUDE_CODE_ENABLE_TASKS` | Enable task tracking | `true` |
