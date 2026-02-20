#!/bin/bash
set -euo pipefail

# Claude Code Platform Entrypoint
# Handles first-run setup and project attachment before launching Claude Code.

PLATFORM_DIR="/opt/claude-platform"

echo "=== Claude Code Platform ==="

# --- API Key Provisioning ---
# Option 2: Self-provision via environment variable
if [ -n "${ANTHROPIC_API_KEY:-}" ]; then
  echo "[auth] API key provided via environment variable"
elif [ -f "/root/.claude.json" ]; then
  echo "[auth] Using existing authentication from mounted config"
else
  echo "[auth] No API key found."
  echo "  Set ANTHROPIC_API_KEY environment variable, or"
  echo "  mount your ~/.claude.json: -v ~/.claude.json:/root/.claude.json"
  echo ""
  echo "  Starting Claude Code for interactive login..."
fi

# --- Project Attachment ---
# If /workspace has files and no .claude directory, auto-attach
if [ -d "/workspace" ] && [ "$(ls -A /workspace 2>/dev/null)" ] && [ ! -d "/workspace/.claude" ]; then
  echo "[setup] Attaching platform config to /workspace..."
  cd "$PLATFORM_DIR"
  bun run cli/index.ts attach /workspace 2>/dev/null || true
  cd /workspace
elif [ -d "/workspace/.claude" ]; then
  echo "[setup] Project already configured"
  cd /workspace
fi

# --- Global Config ---
# Ensure global CLAUDE.md exists
if [ ! -f "/root/.claude/CLAUDE.md" ]; then
  echo "[setup] Creating global CLAUDE.md..."
  mkdir -p /root/.claude
  cat > /root/.claude/CLAUDE.md << 'GLOBALMD'
# Global Claude Code Instructions

## Defaults
- Always use TodoWrite for multi-step tasks
- Plan before implementing significant changes
- Use subagents for context isolation
- Run tests after making changes
- Never commit secrets or credentials
- Work on feature branches, never main/master
GLOBALMD
fi

# --- Environment Info ---
echo "[env] Working directory: $(pwd)"
echo "[env] Claude Code: $(claude --version 2>/dev/null || echo 'not found')"
echo "[env] Node: $(node --version)"
echo "[env] Bun: $(bun --version)"
echo "[env] Git: $(git --version)"
echo "[env] tmux: $(tmux -V 2>/dev/null || echo 'not found')"
echo ""

# --- Launch ---
exec "$@"
