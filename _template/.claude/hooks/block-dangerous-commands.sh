#!/bin/bash
set -euo pipefail

# Reads JSON input from stdin and blocks dangerous commands
INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // empty')

if [ -z "$COMMAND" ]; then
  exit 0
fi

# Block destructive filesystem commands
if echo "$COMMAND" | grep -qE '^\s*rm\s+(-[a-zA-Z]*f[a-zA-Z]*\s+)?(/|\*|~)'; then
  echo "Blocked: Destructive filesystem operation targeting root, home, or wildcard" >&2
  exit 2
fi

# Block force push to main/master
if echo "$COMMAND" | grep -qiE 'git\s+push\s+.*(-f|--force).*\s+(main|master)\b'; then
  echo "Blocked: Force push to main/master is not allowed" >&2
  exit 2
fi

# Block direct push to main/master without -f (warn via ask)
if echo "$COMMAND" | grep -qiE 'git\s+push\s+.*\s+(main|master)\b' && ! echo "$COMMAND" | grep -qiE -- '--force|^-f'; then
  echo '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"ask","permissionDecisionReason":"This pushes directly to main/master. Are you sure?"}}'
  exit 0
fi

# Block piping curl/wget to shell
if echo "$COMMAND" | grep -qE 'curl\s.*\|\s*(bash|sh|zsh)'; then
  echo "Blocked: Piping remote content to shell is not allowed" >&2
  exit 2
fi

if echo "$COMMAND" | grep -qE 'wget\s.*\|\s*(bash|sh|zsh)'; then
  echo "Blocked: Piping remote content to shell is not allowed" >&2
  exit 2
fi

# Block chmod 777
if echo "$COMMAND" | grep -qE 'chmod\s+777'; then
  echo "Blocked: chmod 777 is a security risk. Use specific permissions instead." >&2
  exit 2
fi

exit 0
