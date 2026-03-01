# MCP Server Configuration Reference

Ready-to-use MCP server configurations organized by category. Each JSON file in `docs/mcp-configs/` contains server definitions you can add to your project with a single CLI command.

---

## Understanding Scopes

Every MCP server is registered at one of three scopes. The scope controls where the configuration is stored and who can see it.

| Scope | Config file | Shared? | Use when… |
|-------|------------|---------|-----------|
| `user` | `~/.claude.json` (global section) | No — personal to your machine | You use the server across **all** projects (e.g. Brave Search, GitHub) |
| `project` | `.mcp.json` in the repo root | **Yes — committed to git** | The whole team needs the server; no credentials in the file |
| `local` | `~/.claude.json` (per-project section) | No — personal to your machine | You need a server for **one specific project** but don't want it in git (e.g. a database connection) |

All three scopes are **additive** — Claude sees every server registered at every scope simultaneously. Scope only controls storage and sharing.

> **`.claude/settings.json` is not an MCP store.** It contains `enableAllProjectMcpServers` — a boolean that controls whether project-scoped servers auto-connect without prompting. Set this to `true` in your personal `.claude/settings.local.json` to skip the confirmation dialog.

```bash
# Explicitly pick a scope when adding any server:
claude-workspace mcp add <name> --scope user   ...   # available in all your projects
claude-workspace mcp add <name> --scope local  ...   # this project only, stays out of git
claude-workspace mcp remote <url> --scope user ...   # remote server, all projects
```

---

## How to Use

Each configuration file contains three sections:

| Section | Purpose |
|---------|---------|
| `examples` | JSON snippets you can paste directly into `.mcp.json` |
| `setup_commands` | One-liner CLI commands to add the server via `claude-workspace mcp` |
| `notes` | Recommended scope, auth method, and setup instructions per server |

**Quickest path:** copy the `setup_commands` value for the server you want and run it in your terminal.

```bash
# Example: add Brave Search at user scope
claude-workspace mcp add brave-search --scope user --api-key BRAVE_API_KEY -- npx -y @modelcontextprotocol/server-brave-search
```

Servers added via the CLI with `--scope user` or `--scope local` are stored in your **local Claude config** (`~/.claude.json`), so credentials never enter version control.

---

## Collaboration

**File:** [`docs/mcp-configs/collaboration.json`](mcp-configs/collaboration.json)

Project management, issue tracking, and team communication servers.

| Server | Scope | Type | Auth Method | Setup Command |
|--------|-------|------|-------------|---------------|
| **GitHub** | user | Remote (HTTP) | OAuth | `claude-workspace mcp remote https://api.githubcopilot.com/mcp/ --name github --scope user` |
| **GitHub (PAT)** | user | Remote (HTTP) | Bearer token | `claude-workspace mcp remote https://api.githubcopilot.com/mcp/ --name github --scope user --bearer` |
| **Notion** | user | Remote (HTTP) | OAuth | `claude-workspace mcp remote https://mcp.notion.com/mcp --name notion --scope user` |
| **Linear** | user | Remote (HTTP) | OAuth | `claude-workspace mcp remote https://mcp.linear.app/sse --name linear --scope user` |
| **Slack** | user | Remote (HTTP) | Env var | Requires `SLACK_MCP_URL` — see your Slack admin for the MCP endpoint |
| **Jira** | user | Local (npx) | API key | `claude-workspace mcp add jira --scope user --api-key JIRA_API_TOKEN -- npx -y @anthropic/claude-code-jira-server` |

**GitHub (OAuth)**: after adding, run `/mcp` inside Claude Code and authenticate via the browser flow. Alternatively, use the **GitHub (PAT)** option if you prefer token-based auth — you'll be prompted to enter your Personal Access Token securely (stored in `~/.claude.json`, never in `.mcp.json`).

**Notion, Linear (OAuth)**: after adding, run `/mcp` inside Claude Code and authenticate via the browser flow.

**Jira**: you'll be prompted to enter `JIRA_API_TOKEN` securely. Also set `JIRA_URL` and `JIRA_EMAIL` as environment variables.

---

## Database

**File:** [`docs/mcp-configs/database.json`](mcp-configs/database.json)

Database introspection and query servers.

| Server | Scope | Type | Auth Method | Setup Command |
|--------|-------|------|-------------|---------------|
| **PostgreSQL** | local | Local (npx) | API key (`DATABASE_URL`) | `claude-workspace mcp add postgres --scope local --api-key DATABASE_URL -- npx -y @bytebase/dbhub` |
| **MySQL** | local | Local (npx) | API key (`DATABASE_URL`) | `claude-workspace mcp add mysql --scope local --api-key DATABASE_URL -- npx -y @bytebase/dbhub` |
| **SQLite** | local | Local (npx) | None | `claude-workspace mcp add sqlite --scope local -- npx -y @anthropic/claude-code-sqlite-server ./db.sqlite` |

Database servers use `--scope local` because connection strings contain credentials and are environment-specific. They are stored in `~/.claude.json` under the current project and never committed to git.

**PostgreSQL / MySQL**: you'll be prompted to enter the `DATABASE_URL` securely (e.g., `postgresql://user:pass@host:5432/db`).

**SQLite**: no API key needed — just provide the path to your `.sqlite` file.

---

## Observability

**File:** [`docs/mcp-configs/observability.json`](mcp-configs/observability.json)

Error tracking and monitoring servers.

| Server | Scope | Type | Auth Method | Setup Command |
|--------|-------|------|-------------|---------------|
| **Sentry** | user | Remote (HTTP) | OAuth | `claude-workspace mcp remote https://mcp.sentry.dev/mcp --name sentry --scope user` |
| **Grafana** | user | Remote (HTTP) | Bearer token | `claude-workspace mcp remote $GRAFANA_MCP_URL --name grafana --scope user --bearer` |
| **Honeycomb** | user | Remote (HTTP) | OAuth | `claude-workspace mcp remote https://mcp.honeycomb.io/mcp --name honeycomb --scope user` |
| **Honeycomb (API key)** | user | Remote (HTTP) | Bearer token | `claude-workspace mcp remote https://mcp.honeycomb.io/mcp --name honeycomb --scope user --bearer` |
| **Dynatrace** | user | Local (npx) | OAuth / Platform token | `claude-workspace mcp add dynatrace --scope user --api-key DT_ENVIRONMENT -- npx -y @dynatrace-oss/dynatrace-mcp-server@latest` |

**Sentry**: uses OAuth — authenticate via `/mcp` in Claude Code after adding.

**Grafana**: you'll be prompted for a Bearer token. Generate one in your Grafana instance settings.

**Honeycomb (OAuth)**: after adding, run `/mcp` in Claude Code to authenticate via browser. Provides `run_query`, `run_bubbleup`, `find_columns`, and `get_trace` tools for production debugging. Requires Honeycomb Intelligence (enabled on all plans). EU teams: replace the URL with `https://mcp.eu1.honeycomb.io/mcp`.

**Honeycomb (API key)**: you'll be prompted for a Bearer token in format `KEY_ID:SECRET_KEY` from your Honeycomb API settings. Use this for unattended agents or CI pipelines. EU teams: replace the URL with `https://mcp.eu1.honeycomb.io/mcp`.

**Dynatrace**: you'll be prompted for `DT_ENVIRONMENT` — your tenant URL (e.g., `https://abc12345.apps.dynatrace.com`). Uses browser OAuth by default. For headless/CI use, also set `DT_PLATFORM_TOKEN` (a Platform Token from your Dynatrace settings). Required scopes: `app-engine:apps:run`, `storage:buckets:read`, `davis-copilot:conversations:execute`.

---

## Search

**File:** [`docs/mcp-configs/search.json`](mcp-configs/search.json)

Web search servers for grounding Claude responses in real-time information.

| Server | Scope | Type | Auth Method | Setup Command |
|--------|-------|------|-------------|---------------|
| **Brave Search** | user | Local (npx) | API key (`BRAVE_API_KEY`) | `claude-workspace mcp add brave-search --scope user --api-key BRAVE_API_KEY -- npx -y @modelcontextprotocol/server-brave-search` |

**Brave Search**: you'll be prompted to enter your `BRAVE_API_KEY` securely. Get a free API key (up to 2,000 queries/month) at [brave.com/search/api](https://brave.com/search/api/).

### Migrating Brave Search to user scope

If you added Brave Search previously without `--scope user`, it was registered at `local` scope (the default for `mcp add`). To move it:

```bash
# 1. Remove the local-scoped registration
claude mcp remove brave-search

# 2. Re-add at user scope (you'll be prompted for the API key again)
claude-workspace mcp add brave-search --scope user --api-key BRAVE_API_KEY -- npx -y @modelcontextprotocol/server-brave-search
```

---

## Memory

**File:** [`docs/mcp-configs/memory.json`](mcp-configs/memory.json)

Cross-project persistent memory servers.

| Server | Scope | Type | Auth Method | Setup Command |
|--------|-------|------|-------------|---------------|
| **Engram** (default) | user | Local (binary) | None | `claude-workspace mcp add --scope user engram -- engram mcp` |
| **Memory** (legacy) | user | Local (npx) | None | `claude mcp add --scope user memory -- npx -y @modelcontextprotocol/server-memory` |

**Engram** is auto-registered at user scope by `claude-workspace setup`. Single Go binary with FTS5 full-text search and SQLite persistence. Install via `brew install gentleman-programming/tap/engram`. Data stored at `~/.engram/engram.db`. See [Gentleman-Programming/engram](https://github.com/Gentleman-Programming/engram).

**Memory** (legacy): the official MCP reference server. JSONL-based knowledge graph with substring-only search. No additional install needed if Node.js is available. Use this if you prefer the entity/relation data model or cannot install the engram binary.

---

## Adding a New Configuration

To contribute a new MCP config category:

1. Create `docs/mcp-configs/my-category.json` following the existing schema:
   ```json
   {
     "_description": "Category description",
     "examples": { ... },
     "setup_commands": { ... },
     "notes": { ... }
   }
   ```
   Include `--scope <value>` in every `setup_commands` entry and a `"Recommended scope: <scope>"` prefix in every `notes` entry.
2. Add a section to this file with a table of servers, recommended scope, auth methods, and setup commands.
3. Update `docs/GETTING-STARTED.md` if the new category is broadly useful.
