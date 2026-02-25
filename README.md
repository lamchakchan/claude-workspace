# Claude Code Platform Engineering Kit

A preconfigured, batteries-included platform for deploying Claude Code AI agents across your organization. Designed for teams adopting AI-assisted development for the first time, with safe defaults, layered prompt architecture, and multi-project support.

## What You Get

`claude-workspace attach` overlays these files into your project:

| Path | Purpose |
|------|---------|
| `.claude/settings.json` | Team settings with safe defaults |
| `.claude/settings.local.json.example` | Template for personal overrides |
| `.claude/CLAUDE.md` | Project instructions (auto-detected stack) |
| `.claude/CLAUDE.local.md.example` | Template for personal context |
| `.claude/agents/*.md` | 5 subagent definitions |
| `.claude/skills/*/` | 4 skill definitions |
| `.claude/hooks/*.sh` | 4 safety hook scripts |
| `.claude/.gitignore` | Ignores local overrides |
| `.mcp.json` | MCP server configurations |
| `plans/` | Directory for implementation plans |

## Quick Start

### Prerequisites

- **Git** (for version control and worktree sandboxing)
- **Node.js 18+** and **npm** are required for MCP servers but installed automatically by `setup` if missing

### Install

```bash
curl -fsSL https://raw.githubusercontent.com/lamchakchan/claude-workspace/main/install.sh | bash
```

For manual download or building from source, see [Getting Started - Installation](docs/GETTING-STARTED.md#2-installation).

### Setup

```bash
# 1. Run setup (installs Claude Code CLI, provisions API key, configures globals)
claude-workspace setup

# 2. Attach to your project
claude-workspace attach /path/to/your/project

# 3. Start coding
cd /path/to/your/project && claude

# 4. Verify everything works
claude-workspace doctor
```

## Key Features

- **Safe defaults** — Hooks block dangerous commands (rm -rf, force push), detect secrets, and enforce branch policies
- **Plan-first workflow** — Every significant task starts with a visible plan written to `./plans/` for review
- **Multi-project support** — `claude-workspace attach` copies or symlinks config into any repo; `--symlink` keeps projects in sync
- **Parallel sandboxing** — `claude-workspace sandbox` creates git worktrees for multiple Claude instances on the same repo
- **Layered prompt system** — Global, team, project, and personal instructions merge automatically by priority
- **MCP integration** — Preconfigured local servers (memory, filesystem, git) plus CLI for adding databases, APIs, and remote gateways
- **Flexible model selection** — Sonnet for coding, Haiku for exploration, Opus for complex reasoning; override per-session or per-subagent
- **Custom subagents** — Five built-in agents (planner, code-reviewer, explorer, test-runner, security-scanner); add your own in `.claude/agents/`

## Documentation

| Guide | Contents |
|-------|----------|
| [Getting Started](docs/GETTING-STARTED.md) | Installation, setup, first session, subagents, skills, MCP servers, parallel dev, configuration reference, environment variables |
| [CLI Reference](docs/CLI.md) | Every command, flag, and option with examples |
| [Architecture](docs/ARCHITECTURE.md) | Design philosophy, prompt layering, hook system, model strategy, sandboxing |
| [MCP Configs](docs/MCP-CONFIGS.md) | Ready-to-use MCP server configurations by category (collaboration, databases, APIs, and more) |
| [Runbook](docs/RUNBOOK.md) | Maintenance, troubleshooting, onboarding, security, rollback procedures |
| [Memory](docs/MEMORY.md) | Six memory layers, auto-memory, CLAUDE.md files, Memory MCP, clearing procedures, and gitignore rules |
