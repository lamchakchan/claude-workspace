# Getting Started with Claude Code Platform

This guide walks you through setting up the platform, attaching it to your first project, and using it day-to-day. Written for developers who may be using AI coding agents for the first time.

---

## Table of Contents

1. [Prerequisites](#1-prerequisites)
2. [Installation](#2-installation)
3. [First-Time Setup](#3-first-time-setup)
4. [Attaching to a Project](#4-attaching-to-a-project)
5. [Your First Session](#5-your-first-session)
6. [Using Subagents](#6-using-subagents)
7. [Using Skills](#7-using-skills)
8. [Working with MCP Servers](#8-working-with-mcp-servers)
9. [Parallel Development](#9-parallel-development)
10. [Agent Teams](#10-agent-teams)
11. [Day-to-Day Workflow](#11-day-to-day-workflow)
12. [Tips and Tricks](#12-tips-and-tricks)

---

## 1. Prerequisites

You need:
- **Node.js 18+** and **npm** — for MCP servers (optional)
- **Git** — for version control and worktree sandboxing

The setup wizard checks for these and will guide you if anything is missing. It also checks for optional tools like shellcheck, jq, prettier, and tmux.

---

## 2. Installation

**One-liner (macOS / Linux):**

```bash
curl -fsSL https://raw.githubusercontent.com/lamchakchan/claude-platform/main/install.sh | bash
```

**Or build from source:**

```bash
git clone <platform-repo-url> ~/claude-platform
cd ~/claude-platform
make install   # builds and copies to /usr/local/bin
```

Then run the setup wizard:

```bash
claude-platform setup
```

---

## 3. First-Time Setup

### API Key Provisioning (Option 2: Self-Service)

The platform uses "Option 2" self-provisioning. When you run setup:

```bash
claude-platform setup
# The script will:
# 1. Check if Claude Code CLI is installed (installs if missing)
# 2. Launch the interactive API key provisioning flow
# 3. Create global settings at ~/.claude/settings.json
# 4. Create global instructions at ~/.claude/CLAUDE.md
# 5. Check for optional system tools (shellcheck, jq, prettier, tmux)
```

**Alternative: Environment variable**
```bash
# Add to your shell profile (~/.bashrc, ~/.zshrc)
export ANTHROPIC_API_KEY=sk-ant-your-key-here
```

### Verify Setup

```bash
# Run the health check
claude-platform doctor

# Expected output:
# [OK] Claude Code CLI: 2.x.x
# [OK] Git: git version 2.x.x
# [OK] ~/.claude/settings.json exists
# [OK] ~/.claude/CLAUDE.md exists
# [OK] API key or OAuth configured
```

---

## 4. Attaching to a Project

"Attaching" copies the platform's agents, skills, hooks, settings, and MCP config into your project repository so Claude Code picks them up automatically.

### Basic Attach

```bash
# Copy platform config into your project
claude-platform attach /path/to/your/project

# What this creates in your project:
# .claude/settings.json     - Team settings with safe defaults
# .claude/CLAUDE.md         - Project instructions (auto-detected tech stack)
# .claude/agents/            - All 5 subagent definitions
# .claude/skills/            - All 4 skill definitions
# .claude/hooks/             - All 4 safety hooks
# .mcp.json                  - MCP server configurations
# plans/                     - Directory for implementation plans
```

### Symlink Attach (Keep in Sync)

If you want your project to always use the latest platform config:

```bash
claude-platform attach /path/to/your/project --symlink
```

With symlinks, updating the platform repo automatically updates all attached projects.

### Force Overwrite

If a project already has Claude config and you want to replace it:

```bash
claude-platform attach /path/to/your/project --force
```

### Post-Attach: Customize for Your Project

After attaching, edit `.claude/CLAUDE.md` in your project to add project-specific context:

```markdown
# Project Instructions

## Project
Name: my-web-app
Tech Stack: Next.js, TypeScript, Prisma, PostgreSQL
Build: `npm run build`
Test: `npm test`
Lint: `npm run lint`

## Conventions
- Use functional components with hooks
- All API routes in src/app/api/
- Database queries via Prisma client
- Tests co-located with source files (*.test.ts)

## Key Directories
- src/app/       - Next.js app router pages
- src/lib/       - Shared utilities and helpers
- src/components/ - React components
- prisma/        - Database schema and migrations

## Important Notes
- Always run `npx prisma generate` after schema changes
- Environment variables in .env.local (never commit)
- Feature flags managed via LaunchDarkly
```

### Personal Overrides

For settings you don't want to share with the team:

```bash
# Copy the example to create your local settings
cp .claude/settings.local.json.example .claude/settings.local.json
cp .claude/CLAUDE.local.md.example .claude/CLAUDE.local.md
# Edit these files - they're gitignored
```

---

## 5. Your First Session

### Starting Claude Code

```bash
cd /path/to/your/project
claude
```

What happens at startup:
1. Claude loads `~/.claude/CLAUDE.md` (your global instructions)
2. Claude loads `.claude/CLAUDE.md` (project instructions)
3. Claude loads `.claude/CLAUDE.local.md` (your personal notes, if exists)
4. Hooks are registered (safety checks activate)
5. MCP servers start (memory, filesystem, git)
6. Claude is ready for your prompt

### What You'll See

```
╭──────────────────────────────────────────────╮
│ Claude Code                                  │
│ Model: claude-sonnet-4-6                     │
│ Project: /path/to/your/project               │
╰──────────────────────────────────────────────╯

Tips:
 - Use /help for available commands
 - Use /agents to see available subagents
 - Use /mcp to check MCP server status
 - Use @ to reference files

>
```

### Key Commands Inside Claude Code

| Command | Purpose |
|---------|---------|
| `/help` | Show all available commands |
| `/model` | Switch models (sonnet, opus, haiku) |
| `/agents` | List available subagents |
| `/mcp` | Check MCP server status and authenticate |
| `/compact` | Compress context (do this when sessions get long) |
| `/context` | See what's using your context window |
| `/hooks` | View and manage hooks |
| `/clear` | Clear conversation and start fresh |
| `@filename` | Reference a file in your prompt |
| `Ctrl+C` | Interrupt Claude mid-response |
| `Ctrl+O` | Toggle verbose mode (see hook output) |

### Your First Task

Try a simple task to see the plan-first workflow in action:

```
> Add input validation to the user registration endpoint.
  Make sure to handle edge cases and add tests.
```

Claude will:
1. Create a todo list breaking the work into steps
2. Explore your codebase to find the registration endpoint
3. Plan the changes before implementing
4. Implement validation
5. Add tests
6. Run tests to verify
7. Report results

---

## 6. Using Subagents

Subagents are specialized AI agents that run in isolated context windows. They're invoked automatically by Claude when appropriate, but you can also request them explicitly.

### Available Subagents

| Agent | When to Use | How to Invoke |
|-------|-------------|---------------|
| **planner** | Complex multi-step tasks | "Use the planner to design the approach" |
| **explorer** | Understanding unfamiliar code | "Use the explorer to map the auth module" |
| **code-reviewer** | After making changes | "Run a code review on my changes" |
| **test-runner** | After implementation | "Run the test suite and analyze results" |
| **security-scanner** | Before shipping | "Scan for security vulnerabilities" |

### Example: Planning a Feature

```
> I need to add OAuth2 login with Google and GitHub.
  Use the planner to create a detailed implementation plan first.
```

The planner will:
1. Explore your auth setup
2. Research relevant files
3. Write a plan to `./plans/plan-2026-02-20-oauth2-login.md`
4. Present it for your approval

You can review the plan file directly and ask for changes before Claude proceeds.

### Example: Security Review

```
> Use the security scanner to check the payment processing module
```

### Custom Subagents

Create your own by adding a `.md` file to `.claude/agents/`:

```markdown
---
name: my-agent
description: What this agent does
tools: Read, Grep, Glob, Bash
model: sonnet
---

Your agent's system prompt and instructions here.
```

---

## 7. Using Skills

Skills are reusable workflows that Claude follows when triggered. They're invoked as slash commands.

### Available Skills

| Skill | Trigger | What It Does |
|-------|---------|--------------|
| `/plan-and-execute` | Automatically on complex tasks | Enforces plan-first workflow |
| `/context-manager` | When context is getting full | Strategies for large codebases |
| `/pr-workflow` | "Create a PR for these changes" | Guides the full PR process |
| `/onboarding` | First time in a new project | Maps the codebase and generates CLAUDE.md |

### Example: Onboarding to a New Project

```
> /onboarding
```

Claude will explore the project, detect the tech stack, map the directory structure, and generate/update the project's CLAUDE.md with useful context.

---

## 8. Working with MCP Servers

MCP (Model Context Protocol) servers give Claude access to external tools and data sources.

### Pre-Configured Servers

The platform ships with three MCP servers in `.mcp.json`:
- **memory** - Persistent knowledge graph (remembers across sessions)
- **filesystem** - Secure file operations
- **git** - Git repository operations

### Adding More Servers

**Add a database (with API key):**
```bash
# The --api-key flag prompts securely for the value (masked input)
claude-platform mcp add postgres --api-key DATABASE_URL -- npx -y @bytebase/dbhub
# You'll be prompted: Enter value for DATABASE_URL: ****
# The key is stored as an env var in local Claude config (NOT in .mcp.json)
```

**Add a search API:**
```bash
claude-platform mcp add brave --api-key BRAVE_API_KEY -- npx -y @modelcontextprotocol/server-brave-search
```

**Add GitHub (remote, with OAuth):**
```bash
claude-platform mcp remote https://api.githubcopilot.com/mcp/ --name github
# Then in Claude Code: /mcp → Authenticate
```

**Add a remote server with Bearer token:**
```bash
claude-platform mcp remote https://mcp-gateway.company.com --name gateway --bearer
# You'll be prompted: Enter Bearer token: ****
```

**Add Sentry for error monitoring:**
```bash
claude-platform mcp remote https://mcp.sentry.dev/mcp --name sentry
```

**Add Notion:**
```bash
claude-platform mcp remote https://mcp.notion.com/mcp --name notion
```

### MCP Authentication Methods

The CLI supports three authentication methods for MCP servers:

| Method | Flag | Best For |
|--------|------|----------|
| **API Key** | `--api-key ENV_NAME` | Local servers needing credentials (databases, search APIs) |
| **Bearer Token** | `--bearer` | Remote servers with pre-generated tokens |
| **OAuth 2.0** | `--oauth` | Remote servers with browser-based login (GitHub, Notion) |

All secrets are entered via **masked input** (not visible on screen or in shell history) and stored in your **local Claude config** — never in `.mcp.json` (which is committed to git).

**OAuth with pre-registered app:**
```bash
claude-platform mcp remote https://mcp.example.com --name example \
  --oauth --client-id my-app-id --client-secret
# Prompts for client secret (masked), then you authenticate via /mcp in Claude Code
```

### Using MCP Templates

Check `templates/mcp-configs/` for ready-to-use configurations:

```bash
# See what's available
ls templates/mcp-configs/
# database.json       - PostgreSQL, MySQL, SQLite
# observability.json  - Sentry, Grafana
# collaboration.json  - GitHub, Notion, Slack, Linear, Jira
```

Each template file includes the setup command to copy and run.

### Connecting to a Remote MCP Gateway

If your organization runs a centralized MCP gateway:

```bash
# Without auth (gateway handles auth internally)
claude-platform mcp remote https://mcp-gateway.company.com --name company-gateway

# With Bearer token auth
claude-platform mcp remote https://mcp-gateway.company.com --name company-gateway --bearer

# With OAuth
claude-platform mcp remote https://mcp-gateway.company.com --name company-gateway --oauth
```

### Checking MCP Status

Inside Claude Code:
```
> /mcp
```

This shows all connected servers, their status, and authentication state.

---

## 9. Parallel Development

### Git Worktree Sandboxing

Work on multiple features simultaneously on the same repo:

```bash
# Create sandboxed worktrees
claude-platform sandbox /path/to/project feature-auth
claude-platform sandbox /path/to/project feature-api
claude-platform sandbox /path/to/project bugfix-payments

# Each creates:
# /path/to/project-worktrees/feature-auth/  (branch: feature-auth)
# /path/to/project-worktrees/feature-api/   (branch: feature-api)
# /path/to/project-worktrees/bugfix-payments/ (branch: bugfix-payments)
```

Then open separate terminals for each:

```bash
# Terminal 1
cd /path/to/project-worktrees/feature-auth && claude

# Terminal 2
cd /path/to/project-worktrees/feature-api && claude

# Terminal 3
cd /path/to/project-worktrees/bugfix-payments && claude
```

Each instance works independently with its own branch and files, but shares git history.

### Cleanup

```bash
# List all worktrees
git -C /path/to/project worktree list

# Remove a worktree when done
git -C /path/to/project worktree remove /path/to/project-worktrees/feature-auth
```

---

## 10. Agent Teams

Agent teams let multiple Claude instances collaborate on a single task. This is experimental but enabled by default in the platform.

### Display Modes

Agent teams support two display modes:

| Mode | Requires | Description |
|------|----------|-------------|
| **In-process** (default) | Nothing extra | All teammates run inside your main terminal. Use `Shift+Down` to cycle. Works in any terminal. |
| **Split panes** | tmux or iTerm2 | Each teammate gets its own visible pane. Better visibility but requires extra tooling. |

The platform defaults to **in-process mode** (`"teammateMode": "in-process"` in settings.json), so **tmux is not required**. If you want split-pane mode, install tmux and change the setting:

```json
// .claude/settings.json or .claude/settings.local.json
{ "teammateMode": "tmux" }
```

Or per-session: `claude --teammate-mode tmux`

### When to Use Teams

- Research tasks requiring multiple perspectives
- Large features that can be split into independent modules
- Debugging with competing hypotheses
- Code review from multiple angles

### How to Use

Simply ask Claude to create a team:

```
> Create a team of 3 agents to implement the checkout flow:
  - Agent 1: Backend API endpoints
  - Agent 2: Frontend React components
  - Agent 3: Integration tests
```

### Keyboard Shortcuts (Agent Teams)

| Key | Action |
|-----|--------|
| `Shift+Up/Down` | Select teammates |
| `Ctrl+T` | View shared task list |
| `Enter` | View selected teammate's session |
| `Escape` | Interrupt a teammate |

### When NOT to Use Teams

- Sequential tasks (step A must complete before step B)
- Tasks that modify the same files (merge conflicts)
- Simple, single-file changes

---

## 11. Day-to-Day Workflow

### Starting Your Day

```bash
cd /path/to/project
claude
# Claude loads all your context automatically
```

### Typical Workflow

1. **Describe what you want**: Be specific about the goal, not the implementation
2. **Review the plan**: Claude will create a plan for non-trivial tasks
3. **Approve and monitor**: Watch the todo list as Claude works through steps
4. **Review changes**: Ask Claude to review its own work, or use the code-reviewer agent
5. **Run tests**: Claude should run tests automatically, but you can ask explicitly
6. **Create a PR**: "Create a pull request for these changes"

### Context Management

For large projects, context can fill up. Watch for these signs:
- Claude starts forgetting earlier context
- Responses become slower
- You see "auto-compacting" messages

**Proactive steps:**
```
> /compact          # Manually compress context
> /context          # Check what's using space
```

The platform sets auto-compact at 80% threshold, which is more aggressive than the default to help with large codebases.

### Resuming Work

```bash
# Continue your last session
claude --continue

# Resume a specific session
claude --resume
```

### Ending Your Day

Just close the terminal. Sessions are persisted automatically. Your next `claude --continue` picks up where you left off.

---

## 12. Tips and Tricks

### Use @ References
```
> Look at @src/auth/login.ts and fix the error handling
```

### Be Specific About Scope
```
# Bad: vague, Claude might over-scope
> Fix the bugs

# Good: specific, Claude knows what to do
> Fix the null pointer exception in src/services/user.ts:42
  when the email field is missing from the request body
```

### Use Plan Mode for Risky Changes
```
> /model opusplan
> Refactor the database layer to use connection pooling
# Claude plans with Opus, executes with Sonnet
```

### Override Models When Needed
```
> /model opus
> Design the architecture for the new microservice
# Then switch back
> /model sonnet
```

### Check What Hooks Are Active
```
> /hooks
# Shows all active hooks and their sources
```

### Run the Doctor When Things Feel Off
```bash
claude-platform doctor
# Checks everything: CLI, settings, hooks, MCP, auth
```

### Use the Explorer for Large Codebases
```
> Use the explorer agent to map all the API endpoints in this project
```

This runs in an isolated context and returns a concise summary, keeping your main conversation clean.
