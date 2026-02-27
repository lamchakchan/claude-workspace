---
name: statusline-setup
description: Configures the Claude Code statusline to display live session cost, context usage, model name, weekly reset countdown, and service status alerts (GitHub, Claude, Cloudflare, AWS, Google Cloud, Azure DevOps). Use when the user wants to set up or customize their Claude Code statusline, or when the statusLine setting is missing from ~/.claude/settings.json.
---

# Statusline Setup

## What the Statusline Shows

The Claude Code statusline displays live telemetry after every assistant message:

```
Opus | $0.23 session / $1.23 today / $0.45 block (2h 45m left) | $0.12/hr | 25,000 (12%) | resets in 3d
```

Fields: model name, session cost, daily spend, active billing block spend (with time remaining), hourly burn rate, tokens used, context window percentage, and weekly Pro/Max reset countdown.

The reset countdown (`resets in Xd` / `resets tomorrow` / `resets today`) is derived automatically from the subscription start date in `~/.claude.json` — no manual configuration required. Requires `python3` (standard on macOS and most Linux); silently omitted if unavailable.

## Service Status Alerts

When monitored services experience outages or degraded performance, a colored alert line appears above the normal statusline:

```
🚨 GitHub: Major System Outage  ⚠️  Claude: Degraded Performance
Opus | $0.23 session / $1.23 today / $0.45 block (2h 45m left) | $0.12/hr | 25,000 (12%) | resets in 3d
```

Monitored services: GitHub, Claude, Cloudflare, AWS, Google Cloud, Azure DevOps. Alerts use bold red (🚨) for major/critical and bold yellow (⚠️) for minor/degraded. Responses are cached for 5 minutes with a 2-second HTTP timeout. When all services are healthy, the alert line is hidden entirely.

## Recommended Tool: ccusage

The recommended backend is **ccusage** (`github.com/ryoppippi/ccusage`), which reads `~/.claude/projects/` JSONL files to compute historical cost data and enriches the statusline with block-level tracking.

Runtime options (detected in preference order at execution time):
- `bun x ccusage statusline` — fastest, preferred if bun is installed
- `npx -y ccusage statusline` — standard fallback via Node.js
- Inline `jq` — used when neither bun nor npx is available

## Configuring the Statusline

Use the built-in `statusline-setup` subagent (Tools: Read, Edit) to configure interactively. Invoke it via the Task tool:

```
Task tool: subagent_type=statusline-setup
Prompt: Configure the statusline in ~/.claude/settings.json using the best available runtime
```

The subagent will:
1. Check whether `statusLine` is already present in `~/.claude/settings.json`
2. Write `~/.claude/statusline.sh` — a wrapper script that handles runtime detection and reset countdown
3. Register the script in `~/.claude/settings.json`
4. Confirm the configuration was applied

## Manual Configuration

To configure directly, add the following to `~/.claude/settings.json`:

```json
{
  "statusLine": {
    "type": "command",
    "command": "bash ~/.claude/statusline.sh",
    "padding": 0
  }
}
```

Then write `~/.claude/statusline.sh` with the wrapper script from the automated setup (see below).

## Automated Setup

Run from the terminal to write the wrapper script and configure automatically:

```bash
claude-workspace statusline
```

Use `--force` to overwrite an existing configuration:

```bash
claude-workspace statusline --force
```

This writes `~/.claude/statusline.sh`, which detects the available runtime (bun/npx/jq) each time it runs, appends the weekly reset countdown from `~/.claude.json`, and checks service status APIs for outage alerts.
