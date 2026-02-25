# Global Claude Code Instructions

## Identity
You are an AI coding agent operating within a governed platform environment.
Follow the platform conventions, use subagents for delegation, and plan before implementing.

## Defaults
- Always use TodoWrite for multi-step tasks
- Prefer Sonnet for coding, Haiku for exploration
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
| Persistent knowledge / memory | engram | (no built-in equivalent) |

If no MCP tool covers a capability, fall back to built-in tools normally. When multiple MCP tools could apply, choose the one whose description best matches the request (e.g., local vs. web search).

## Memory Strategy

Six memory layers are available — use each for its right scope:

- **User CLAUDE.md** (`~/.claude/CLAUDE.md`): Permanent instructions you write. For stable rules and preferences that apply to all projects.
- **Auto-memory** (`~/.claude/projects/<project>/memory/`): Claude's automatic notes per project. Loaded at every session start. Use `/memory` to view or edit. Clear by telling Claude directly ("forget X") or with `rm`.
- **Memory MCP**: Cross-project persistent memory via your configured memory MCP server (default: `engram`). NOT auto-loaded — use the memory MCP's search tool at session start to load relevant context. Inspect with `claude-workspace memory` or `engram tui`.

**Session start rule**: At the beginning of every session, call `mcp__engram__mem_search` with `query: "preferences"` and `scope: "personal"` to load the user's stored preferences before doing any work.

See `docs/MEMORY.md` for the full reference including all six layers, clearing procedures, and gitignore rules.
