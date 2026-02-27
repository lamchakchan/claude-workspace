# Project-Level Instructions

This file contains team-shared instructions loaded for every Claude Code session in this project. Customize this for your specific project.

## Project Context

<!-- Describe your project here -->
Project: Claude Code Platform Engineering Kit
Purpose: Preconfigured AI agent platform for teams adopting Claude Code
Tech Stack: Go, Shell scripts, YAML, JSON, Markdown

## Team Conventions

### Code Style
- Shell scripts: Use `set -euo pipefail`, quote variables, use shellcheck
- JSON: 2-space indentation, no trailing commas
- Markdown: ATX headings, fenced code blocks with language tags

### Testing
- All hooks must be tested before deployment
- Scripts should handle edge cases gracefully
- Validate with `shellcheck` for shell scripts

### Documentation
- Every new feature needs a corresponding docs update
- Configuration changes must be reflected in README.md
- Use inline comments for non-obvious logic

## MCP Tool Preferences

Prefer MCP tools over built-in Claude Code tools when both can satisfy the same request. MCP tools follow the `mcp__<server>__<tool>` naming pattern — identify available tools at runtime.

| Capability | Prefer MCP tools from providers like... | Over built-in... |
|---|---|---|
| Web search | brave, perplexity, tavily, exa, duckduckgo | `WebSearch` |
| Filesystem | filesystem | Bash file commands (cat, ls, find) |
| GitHub / VCS | github, gitlab, bitbucket | `gh` CLI via Bash |
| Observability | honeycomb, datadog, grafana, newrelic, sentry | (no built-in equivalent) |
| Persistent memory | engram | (no built-in equivalent) |

If no MCP tool covers a capability, fall back to built-in tools normally.

## Plan Conventions

- Plans live in `./plans/` — use naming: `plan-YYYY-MM-DD-<short-description>.md`
- Include Status (Draft/Approved/In Progress/Complete) and Last Updated fields
- Plans should be self-contained — resumable without the original session context
- Use `/plan-resume` to pick up parked plans in a new session

## Directory Layout

```
.claude/agents/   - Custom subagent definitions (Markdown + YAML frontmatter)
.claude/skills/   - Reusable skill definitions
.claude/hooks/    - Safety and quality gate scripts
main.go           - Go CLI entry point
internal/         - Go command implementations
docs/             - Detailed documentation
```

## Important Files

- `README.md` - Main documentation and quick start guide
- `.claude/settings.json` - Team settings with safe defaults
- `.mcp.json` - MCP server configurations
- `install.sh` - One-liner installer script
- `Makefile` - Build, test, and install targets
