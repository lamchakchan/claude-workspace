#!/bin/bash
set -euo pipefail

# TeammateIdle hook: nudges idle teammates that still have in-progress tasks.
# Exit 0 = allow idle, Exit 2 = send feedback to keep working.
# Fails open: if expected fields are missing, allows idle.

INPUT=$(cat)
TEAMMATE_NAME=$(echo "$INPUT" | jq -r '.teammate_name // empty' 2>/dev/null || true)

# If we can't read input, fail open
if [ -z "$TEAMMATE_NAME" ]; then
  exit 0
fi

# Check if there are in-progress tasks in the input
IN_PROGRESS=$(echo "$INPUT" | jq -r '.tasks[]? | select(.status == "in_progress") | .subject' 2>/dev/null || true)

if [ -n "$IN_PROGRESS" ]; then
  echo "Teammate $TEAMMATE_NAME still has in-progress tasks:" >&2
  echo "$IN_PROGRESS" | while IFS= read -r task; do
    echo "  - $task" >&2
  done
  echo "Please continue working on your assigned tasks before going idle." >&2
  exit 2
fi

exit 0
