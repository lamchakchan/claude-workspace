#!/bin/bash
set -euo pipefail

# Demo script: shows what the statusline looks like when team agents are active.
# Simulates 3 stages of a team session by writing team-state.json and rendering.

STATE_FILE="$HOME/.claude/team-state.json"
BACKUP_FILE=""

# Save existing state and restore on exit
if [ -f "$STATE_FILE" ]; then
  BACKUP_FILE="$(mktemp)"
  cp "$STATE_FILE" "$BACKUP_FILE"
fi

cleanup() {
  if [ -n "$BACKUP_FILE" ] && [ -f "$BACKUP_FILE" ]; then
    cp "$BACKUP_FILE" "$STATE_FILE"
    rm -f "$BACKUP_FILE"
  elif [ -z "$BACKUP_FILE" ]; then
    rm -f "$STATE_FILE"
  fi
}
trap cleanup EXIT

# Build the binary if not already present
if [ ! -f ./ccw ] && [ ! -L ./ccw ]; then
  make build 2>/dev/null || go build -o ccw . 2>/dev/null
fi

# Use ccw symlink if present, otherwise fall back to claude-workspace
CCW="./ccw"
if [ ! -f "$CCW" ] && [ ! -L "$CCW" ]; then
  CCW="./claude-workspace"
fi

write_state() {
  local pending=$1 in_progress=$2 completed=$3
  python3 -c "
import json, datetime, pathlib
pathlib.Path.home().joinpath('.claude','team-state.json').write_text(json.dumps({
    'updated_at': datetime.datetime.utcnow().strftime('%Y-%m-%dT%H:%M:%SZ'),
    'agents_seen': ['agent-a', 'agent-b'],
    'tasks': {'pending': $pending, 'in_progress': $in_progress, 'completed': $completed}
}))"
}

echo ""
echo "=== Stage 1: Team starting ==="
echo ""
write_state 4 0 0
echo '{}' | $CCW statusline render
echo ""

echo ""
echo "=== Stage 2: Work in progress ==="
echo ""
write_state 1 1 2
echo '{}' | $CCW statusline render
echo ""

echo ""
echo "=== Stage 3: All complete ==="
echo ""
write_state 0 0 4
echo '{}' | $CCW statusline render
echo ""
