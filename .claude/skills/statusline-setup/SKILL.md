---
name: statusline-setup
description: Configures the Claude Code statusline to display live session cost, context usage, and model name. Use when the user wants to set up or customize their Claude Code statusline, or when the statusLine setting is missing from ~/.claude/settings.json.
---

# Statusline Setup

## What the Statusline Shows

The Claude Code statusline displays live telemetry after every assistant message:

```
Opus | $0.23 session / $1.23 today / $0.45 block (2h 45m left) | $0.12/hr | 25,000 (12%)
```

Fields: model name, session cost, daily spend, active billing block spend (with time remaining), hourly burn rate, tokens used, and context window percentage.

## Recommended Tool: ccusage

The recommended backend is **ccusage** (`github.com/ryoppippi/ccusage`), which reads `~/.claude/projects/` JSONL files to compute historical cost data and enriches the statusline with block-level tracking.

Runtime options (detected in preference order):
- `bun x ccusage statusline` — fastest, preferred if bun is installed
- `npx -y ccusage statusline` — standard fallback via Node.js

## Configuring the Statusline

Use the built-in `statusline-setup` subagent (Tools: Read, Edit) to configure interactively. Invoke it via the Task tool:

```
Task tool: subagent_type=statusline-setup
Prompt: Configure the statusline in ~/.claude/settings.json using the best available runtime
```

The subagent will:
1. Check whether `statusLine` is already present in `~/.claude/settings.json`
2. Detect available runtimes (`bun`, `npx`)
3. Write the appropriate `statusLine` block
4. Confirm the configuration was applied

## Manual Configuration

To configure directly, add the following to `~/.claude/settings.json`:

```json
{
  "statusLine": {
    "type": "command",
    "command": "bun x ccusage statusline",
    "padding": 0
  }
}
```

Replace the command with `npx -y ccusage statusline` if bun is not available.

## Fallback (no runtime)

If neither bun nor npx is installed, use an inline jq command:

```json
{
  "statusLine": {
    "type": "command",
    "command": "jq -r '\"\\(.model.display_name) | $\\(.cost.total_cost_usd | . * 1000 | round / 1000) | \\(.context_window.used_percentage)% ctx\"'",
    "padding": 0
  }
}
```

## Automated Setup

Run from the terminal to auto-detect the best runtime and configure automatically:

```bash
claude-workspace statusline
```

Use `--force` to overwrite an existing configuration:

```bash
claude-workspace statusline --force
```
