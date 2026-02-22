#!/bin/bash
set -euo pipefail

# Blocks commits and checkouts to protected branches
INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // empty')

if [ -z "$COMMAND" ]; then
  exit 0
fi

# Only check git commands
if ! echo "$COMMAND" | grep -qE '^\s*git\s+'; then
  exit 0
fi

# Get current branch
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

# Block commits directly to main/master
if echo "$COMMAND" | grep -qE 'git\s+commit' && [[ "$CURRENT_BRANCH" =~ ^(main|master)$ ]]; then
  echo "Blocked: Direct commits to $CURRENT_BRANCH are not allowed. Create a feature branch first." >&2
  exit 2
fi

# Warn about checkout to main/master (allow but inform)
if echo "$COMMAND" | grep -qE 'git\s+checkout\s+(main|master)\s*$'; then
  echo '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"ask","permissionDecisionReason":"Switching to a protected branch. Make sure you create a feature branch before making changes."}}'
  exit 0
fi

exit 0
