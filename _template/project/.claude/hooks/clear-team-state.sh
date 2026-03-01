#!/bin/bash
# Stop hook: removes team state file so the statusline goes dark immediately on session end.
# For abrupt shutdowns (SIGKILL, crash), the statusline's 30-minute TTL handles cleanup.
rm -f ~/.claude/team-state.json
exit 0
