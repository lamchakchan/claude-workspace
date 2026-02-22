# MCP Server Configuration Reference

Ready-to-use MCP server configurations organized by category. Each JSON file in `docs/mcp-configs/` contains server definitions you can add to your project with a single CLI command.

---

## How to Use

Each configuration file contains three sections:

| Section | Purpose |
|---------|---------|
| `examples` | JSON snippets you can paste directly into `.mcp.json` |
| `setup_commands` | One-liner CLI commands to add the server via `claude-workspace mcp` |
| `notes` | Authentication method and setup instructions per server |

**Quickest path:** copy the `setup_commands` value for the server you want and run it in your terminal.

```bash
# Example: add PostgreSQL
claude-workspace mcp add postgres --api-key DATABASE_URL -- npx -y @bytebase/dbhub
```

Servers added via the CLI are stored in your **local** Claude config (not `.mcp.json`), so credentials stay out of version control.

---

## Collaboration

**File:** [`docs/mcp-configs/collaboration.json`](mcp-configs/collaboration.json)

Project management, issue tracking, and team communication servers.

| Server | Type | Auth Method | Setup Command |
|--------|------|-------------|---------------|
| **GitHub** | Remote (HTTP) | OAuth | `claude-workspace mcp remote https://api.githubcopilot.com/mcp/ --name github` |
| **Notion** | Remote (HTTP) | OAuth | `claude-workspace mcp remote https://mcp.notion.com/mcp --name notion` |
| **Linear** | Remote (HTTP) | OAuth | `claude-workspace mcp remote https://mcp.linear.app/sse --name linear` |
| **Slack** | Remote (HTTP) | Env var | Requires `SLACK_MCP_URL` — see your Slack admin for the MCP endpoint |
| **Jira** | Local (npx) | API key | `claude-workspace mcp add jira --api-key JIRA_API_TOKEN -- npx -y @anthropic/claude-code-jira-server` |

**OAuth servers** (GitHub, Notion, Linear): after adding, run `/mcp` inside Claude Code and authenticate via the browser flow.

**Jira**: you'll be prompted to enter `JIRA_API_TOKEN` securely. Also set `JIRA_URL` and `JIRA_EMAIL` as environment variables.

---

## Database

**File:** [`docs/mcp-configs/database.json`](mcp-configs/database.json)

Database introspection and query servers.

| Server | Type | Auth Method | Setup Command |
|--------|------|-------------|---------------|
| **PostgreSQL** | Local (npx) | API key (`DATABASE_URL`) | `claude-workspace mcp add postgres --api-key DATABASE_URL -- npx -y @bytebase/dbhub` |
| **MySQL** | Local (npx) | API key (`DATABASE_URL`) | `claude-workspace mcp add mysql --api-key DATABASE_URL -- npx -y @bytebase/dbhub` |
| **SQLite** | Local (npx) | None | `claude-workspace mcp add sqlite -- npx -y @anthropic/claude-code-sqlite-server ./db.sqlite` |

**PostgreSQL / MySQL**: you'll be prompted to enter the `DATABASE_URL` securely (e.g., `postgresql://user:pass@host:5432/db`).

**SQLite**: no API key needed — just provide the path to your `.sqlite` file.

---

## Observability

**File:** [`docs/mcp-configs/observability.json`](mcp-configs/observability.json)

Error tracking and monitoring servers.

| Server | Type | Auth Method | Setup Command |
|--------|------|-------------|---------------|
| **Sentry** | Remote (HTTP) | OAuth | `claude-workspace mcp remote https://mcp.sentry.dev/mcp --name sentry` |
| **Grafana** | Remote (HTTP) | Bearer token | `claude-workspace mcp remote $GRAFANA_MCP_URL --name grafana --bearer` |

**Sentry**: uses OAuth — authenticate via `/mcp` in Claude Code after adding.

**Grafana**: you'll be prompted for a Bearer token. Generate one in your Grafana instance settings.

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
2. Add a section to this file with a table of servers, auth methods, and setup commands.
3. Update `docs/GETTING-STARTED.md` if the new category is broadly useful.
