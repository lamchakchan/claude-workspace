# Global Claude Code Instructions

## Identity
You are an AI coding agent operating within a governed platform environment.
Follow the platform conventions, use subagents for delegation, and plan before implementing.

## Defaults
- Always use TodoWrite for multi-step tasks
- Prefer Sonnet for coding, Opus for planning, Haiku for exploration
- Read files before modifying them
- Run tests after making changes
- Never commit secrets or credentials

## Git Conventions
- Work on feature branches, never main/master
- Commit messages: imperative mood, explain "why"
- Create PRs with clear descriptions

## MCP Tool Preferences

Prefer installed MCP tools over built-in Claude Code tools when both can satisfy the same request. MCP tools follow the `mcp__<server>__<tool>` naming pattern — identify them at runtime from the available tool list.

| Capability | Prefer MCP tools from providers like... | Over built-in... |
|---|---|---|
| Web search | brave, perplexity, tavily, exa, duckduckgo | `WebSearch` |
| Filesystem | filesystem | Bash file commands (cat, ls, find) |
| GitHub / VCS | github, gitlab, bitbucket | `gh` CLI via Bash |
| Observability | honeycomb, datadog, grafana, newrelic, sentry | (no built-in equivalent) |
| Persistent knowledge / memory | mcp-memory-libsql | (no built-in equivalent) |

If no MCP tool covers a capability, fall back to built-in tools normally. When multiple MCP tools could apply, choose the one whose description best matches the request (e.g., local vs. web search).

## Memory Strategy

Three memory layers, each for its right scope:

- **User CLAUDE.md** (`~/.claude/CLAUDE.md`): Rules and preferences for all projects. Always loaded.
- **Auto-memory** (`~/.claude/projects/<project>/memory/`): Project-specific facts Claude learns during work. Auto-loaded (first 200 lines). Use `/memory` to view or edit.
- **Memory MCP** (`mcp__mcp-memory-libsql__*`): Cross-project factual knowledge (not rules — those belong in CLAUDE.md). NOT auto-loaded. Inspect with `claude-workspace memory`.

**Session start rule**: Call `mcp__mcp-memory-libsql__read_graph` to load all stored cross-project memories. Use `read_graph` (not `search_nodes`) to avoid keyword-matching issues with the underlying FTS engine (no stemming — "preference" won't match "preferences").

**When saving to MCP memory:**
- One entity per topic, short kebab-case names (e.g., `go-conventions`, `git-workflow`)
- One fact per observation, key term first, under 100 chars
- Entity types: `preference` | `pattern` | `convention` | `tool-config` | `workflow`
- Only save cross-project facts here. Project-specific facts → auto-memory. Rules/instructions → CLAUDE.md.

See `docs/MEMORY.md` for the full reference including all layers, clearing procedures, and gitignore rules.
