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

# Write team state snapshot for statusline (best-effort, does not affect exit code)
_HOOK_INPUT="$INPUT" _HOOK_AGENT="$TEAMMATE_NAME" python3 <<'PYEOF' 2>/dev/null || true
import json, os, pathlib, datetime

teammate = os.environ.get('_HOOK_AGENT', '')
try:
    data = json.loads(os.environ.get('_HOOK_INPUT', '{}'))
except Exception:
    data = {}

state_path = pathlib.Path.home() / '.claude' / 'team-state.json'
existing = {}
try:
    existing = json.loads(state_path.read_text())
except Exception:
    pass

seen = list({*existing.get('agents_seen', []), teammate} - {''})
tasks = data.get('tasks', [])
state = {
    'updated_at': datetime.datetime.utcnow().strftime('%Y-%m-%dT%H:%M:%SZ'),
    'agents_seen': sorted(seen),
    'tasks': {
        'pending':     sum(1 for t in tasks if t.get('status') == 'pending'),
        'in_progress': sum(1 for t in tasks if t.get('status') == 'in_progress'),
        'completed':   sum(1 for t in tasks if t.get('status') == 'completed'),
    },
}
tmp = state_path.with_suffix('.tmp')
tmp.write_text(json.dumps(state))
tmp.rename(state_path)
PYEOF

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
