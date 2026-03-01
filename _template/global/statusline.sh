#!/usr/bin/env bash
# Managed by claude-workspace — re-run "claude-workspace statusline" to regenerate.
# Combines ccusage (or jq fallback) with computed statusline data via claude-workspace.

input=$(cat)

# Base statusline: runtime detected at execution time.
# head -1 guards against ccusage emitting multi-line error messages to stdout.
# Fall back to jq if ccusage returns nothing or an error line (starts with ❌).
base=""
if command -v bun &>/dev/null; then
    base=$(printf '%s' "$input" | bun x ccusage statusline 2>/dev/null | head -1)
elif command -v npx &>/dev/null; then
    base=$(printf '%s' "$input" | npx -y ccusage statusline 2>/dev/null | head -1)
fi
if [[ -z "$base" || "$base" == ❌* ]]; then
    base=$(printf '%s' "$input" | jq -r \
        '"\(.model.display_name) | $\(.cost.total_cost_usd | . * 1000 | round / 1000) | \(.context_window.used_percentage)% ctx"' \
        2>/dev/null)
fi

# Delegate computed parts (reset countdown, service alerts, width compaction)
# to the Go binary. Falls back to printing just the base line if not available.
if command -v claude-workspace &>/dev/null; then
    printf '%s' "$input" | \
        COLS="$(tput cols 2>/dev/null || echo 120)" \
        claude-workspace statusline render --base="$base"
else
    [[ -n "$base" ]] && printf '%s\n' "$base"
fi
