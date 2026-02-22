# Operations Runbook

Procedures for maintaining the Claude Code Platform, troubleshooting issues, and managing the platform lifecycle.

---

## Table of Contents

1. [Routine Maintenance](#1-routine-maintenance)
2. [Updating the Platform](#2-updating-the-platform)
3. [Development Workflow](#3-development-workflow)
4. [Sandboxed Branches](#4-sandboxed-branches)
5. [Updating Claude Code CLI](#5-updating-claude-code-cli)
6. [Managing API Keys](#6-managing-api-keys)
7. [Managing MCP Servers](#7-managing-mcp-servers)
8. [Managing Hooks](#8-managing-hooks)
9. [Managing Agents and Skills](#9-managing-agents-and-skills)
10. [Troubleshooting](#10-troubleshooting)
11. [Rollback Procedures](#11-rollback-procedures)
12. [Onboarding New Team Members](#12-onboarding-new-team-members)
13. [Offboarding](#13-offboarding)
14. [Security Incident Response](#14-security-incident-response)
15. [Monitoring and Observability](#15-monitoring-and-observability)

---

## 1. Routine Maintenance

### Weekly

| Task | Command | Purpose |
|------|---------|---------|
| Run health check | `claude-workspace doctor` | Catch config drift |
| Check Claude Code version | `claude --version` | Stay current |
| Review hook scripts | `ls -la .claude/hooks/` | Verify still executable |
| Check MCP server status | Run `/mcp` in Claude Code | Verify connections |

### Monthly

| Task | Command | Purpose |
|------|---------|---------|
| Update Claude Code CLI | `curl -fsSL https://claude.ai/install.sh \| bash` | Get latest features/fixes |
| Update MCP servers | `npm update` in project | Update MCP dependencies |
| Review permission rules | Read `.claude/settings.json` | Adjust as needed |
| Review CLAUDE.md files | Read all CLAUDE.md layers | Keep context current |
| Clean old worktrees | `git worktree list` → remove stale ones | Free disk space |
| Clean old plans | `ls plans/` → archive old ones | Keep directory manageable |

### Quarterly

| Task | Purpose |
|------|---------|
| Review subagent definitions | Are they still effective? Adjust prompts |
| Review model choices | Are costs acceptable? Optimize per-agent models |
| Audit MCP server list | Remove unused servers, add needed ones |
| Review security scanner rules | Update for new vulnerability patterns |
| Review hook effectiveness | Are hooks catching real issues? Remove noise |

---

## 2. Updating the Platform

### Updating All Attached Projects

When you update the platform repo, projects using `--symlink` get changes automatically. For projects using copies:

```bash
# Re-attach with force to overwrite old config
claude-workspace attach /path/to/project --force
```

### Updating Specific Components

```bash
# Update only agents
cp -r .claude/agents/* /path/to/project/.claude/agents/

# Update only hooks
cp .claude/hooks/*.sh /path/to/project/.claude/hooks/
chmod +x /path/to/project/.claude/hooks/*.sh

# Update only settings (careful - may overwrite project customizations)
# Better: manually merge changes
diff .claude/settings.json /path/to/project/.claude/settings.json
```

### Version Pinning

Tag platform releases for reproducibility:

```bash
git tag v1.0.0
git push origin v1.0.0

# Projects can pin to a version
git clone --branch v1.0.0 <platform-repo> ~/claude-workspace
```

---

## 3. Development Workflow

### Makefile Targets

| Target | Command | Purpose |
|--------|---------|---------|
| `build` | `make build` | Compile the binary for the current platform |
| `install` | `make install` | Build and copy to `/usr/local/bin` (requires sudo) |
| `test` | `make test` | Run `go test ./...` |
| `vet` | `make vet` | Run `go vet ./...` for static analysis |
| `clean` | `make clean` | Remove compiled binaries |
| `build-all` | `make build-all` | Cross-compile for darwin/linux × amd64/arm64 |
| `smoke-test` | `make smoke-test` | Full end-to-end smoke test in a Multipass VM |
| `smoke-test-keep` | `make smoke-test-keep` | Smoke test, keep VM for debugging |
| `smoke-test-fast` | `make smoke-test-fast` | Smoke test with stubbed Claude CLI (~1-2 min) |
| `smoke-test-docker` | `make smoke-test-docker` | Full smoke test using Docker (no VM required) |
| `smoke-test-docker-fast` | `make smoke-test-docker-fast` | Docker smoke test with stubbed Claude CLI (used in CI) |

### Typical Development Cycle

```bash
make vet               # static analysis
make test              # unit tests
make build             # compile
make smoke-test-fast   # end-to-end via Multipass (fast mode)
# or
make smoke-test-docker-fast  # end-to-end via Docker (no VM needed)
```

### Smoke Tests

The smoke test (`scripts/smoke-test.sh`) launches a fresh Ubuntu 24.04 environment and exercises `setup` → `attach` → `doctor` end-to-end. It supports two backends:

- **Multipass** (default) — launches a full VM. Best for local development on macOS.
- **Docker** (`--docker`) — uses a container. Works in CI and anywhere Docker is available (no nested virtualization required).

The CI pipeline runs `make smoke-test-docker-fast` automatically on every push.

**Prerequisites:**

```bash
# Multipass mode (default)
# macOS (Homebrew)
brew install multipass
# Linux (snap)
sudo snap install multipass
# Linux (apt — Ubuntu/Debian)
sudo apt update && sudo apt install multipass

# Docker mode
# Install Docker: https://docs.docker.com/get-docker/
```

**Flags:**
- `--docker` — use Docker instead of Multipass
- `--keep` — preserve VM/container after test for manual inspection
- `--reuse` — reuse an existing VM/container instead of recreating
- `--skip-claude-cli` — stub the `claude` binary (faster, no network required)
- `--name <vm>` — override VM/container name (default: `claude-workspace-smoke`)

### Cross-Compiling for Release

```bash
make build-all
# Produces: claude-workspace-{darwin,linux}-{arm64,amd64}
```

---

## 4. Sandboxed Branches

The `sandbox` command creates an isolated git worktree with a new branch, copies local configuration (`.claude/settings.local.json`, `.claude/CLAUDE.local.md`, `.mcp.json`) into it, and installs dependencies automatically.

### Create a Sandbox

```bash
claude-workspace sandbox <project-path> <branch-name>
```

Examples:
```bash
claude-workspace sandbox ./my-project feature-auth
claude-workspace sandbox ./my-project bugfix-login
```

This will:
1. Create a git worktree at `<project>-worktrees/<branch-name>/`
2. Copy local Claude settings and CLAUDE.local.md into the worktree
3. Copy `.mcp.json` if not already tracked by git
4. Auto-install dependencies (detects bun, npm, yarn, pnpm)

### Work in the Sandbox

```bash
cd /path/to/my-project-worktrees/feature-auth
claude
```

### List and Clean Up Sandboxes

```bash
# List all worktrees
git worktree list

# Remove a sandbox when done
git worktree remove /path/to/my-project-worktrees/feature-auth
```

---

## 5. Updating Claude Code CLI

### Check Current Version

```bash
claude --version
```

### Update

```bash
# Via official installer
curl -fsSL https://claude.ai/install.sh | bash
```

### Breaking Changes

Claude Code releases can change:
- Settings schema (new/renamed fields)
- Hook event format (new fields in JSON input)
- MCP configuration format
- Agent/skill frontmatter options

After updating, always:
1. Run `claude-workspace doctor`
2. Test hooks with `claude --debug`
3. Check `/mcp` status
4. Verify agents appear in `/agents`

---

## 6. Managing API Keys

### Self-Provisioning (Option 2)

```bash
# Initial provisioning via setup
claude-workspace setup

# Re-provisioning (if key expires)
claude  # Launches interactive login flow
```

### Environment Variable

```bash
# Set in shell profile
export ANTHROPIC_API_KEY=sk-ant-...

# Verify
echo $ANTHROPIC_API_KEY | head -c 10
```

### API Key Helper Script

For dynamic key generation (e.g., from a vault):

```json
// In settings.json
{
  "apiKeyHelper": "/path/to/generate-api-key.sh"
}
```

The script should output the API key to stdout. Claude Code calls it before each API request.

### Key Rotation

1. Generate new key in Anthropic Console
2. Update `ANTHROPIC_API_KEY` in your environment
3. Or update key in your vault (if using `apiKeyHelper`)
4. Restart Claude Code sessions

---

## 7. Managing MCP Servers

### List All Servers

```bash
# Via CLI
claude-workspace mcp list

# Via Claude Code (includes status)
# Inside a session: /mcp
```

### Add a Server

```bash
# Local stdio server (no auth)
claude-workspace mcp add <name> -- <command> [args...]

# Local server with API key (securely prompted, masked input)
claude-workspace mcp add <name> --api-key ENV_VAR_NAME -- <command> [args...]

# Remote HTTP server (no auth or OAuth via /mcp)
claude-workspace mcp remote <url> --name <name>

# Remote server with Bearer token (securely prompted)
claude-workspace mcp remote <url> --name <name> --bearer

# Remote server with OAuth + pre-registered client
claude-workspace mcp remote <url> --name <name> --oauth --client-id <id> --client-secret

# Directly via Claude Code CLI
claude mcp add --transport http <name> <url>
```

### Manage MCP Secrets

Secrets added via `--api-key`, `--bearer`, or `--client-secret` are stored in your **local Claude config** (never in `.mcp.json`).

```bash
# Re-run the add command to update a secret (will overwrite)
claude-workspace mcp add postgres --api-key DATABASE_URL -- npx -y @bytebase/dbhub

# Check where secrets are stored
# Local servers: env vars in Claude's local MCP config
# Remote servers: headers/tokens in Claude's local config
# Neither is committed to git
```

### Remove a Server

```bash
claude mcp remove <name>
```

### Troubleshoot a Server

```bash
# Check server status
# In Claude Code: /mcp

# Debug startup
MCP_TIMEOUT=30000 claude --debug

# Check server logs
claude --debug 2>&1 | grep -i mcp

# Increase output limit for large-output servers
MAX_MCP_OUTPUT_TOKENS=50000 claude
```

### Reset MCP Approvals

If you've accidentally denied a project MCP server:

```bash
claude mcp reset-project-choices
```

### Update Project MCP Config

Edit `.mcp.json` directly:

```json
{
  "mcpServers": {
    "new-server": {
      "type": "http",
      "url": "https://mcp.example.com/mcp"
    }
  }
}
```

---

## 8. Managing Hooks

### View Active Hooks

```bash
# In Claude Code
# /hooks

# Or read settings directly
cat .claude/settings.json | jq '.hooks'
```

### Test a Hook Manually

```bash
# Feed test input to a hook
echo '{"tool_name":"Bash","tool_input":{"command":"rm -rf /"}}' | .claude/hooks/block-dangerous-commands.sh
# Expected: exit code 2 (blocked)

echo '{"tool_name":"Bash","tool_input":{"command":"npm test"}}' | .claude/hooks/block-dangerous-commands.sh
# Expected: exit code 0 (allowed)
```

### Debug Hooks in Real Time

```bash
claude --debug
# Or toggle verbose mode: Ctrl+O during a session
```

### Disable All Hooks Temporarily

```json
// In .claude/settings.local.json
{
  "disableAllHooks": true
}
```

Remove the setting when done.

### Add a New Hook

1. Write the script:
```bash
#!/bin/bash
set -euo pipefail
INPUT=$(cat)
# Your logic here
exit 0  # allow, or exit 2 to block
```

2. Make executable: `chmod +x .claude/hooks/my-hook.sh`

3. Register in `.claude/settings.json`:
```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/my-hook.sh"
          }
        ]
      }
    ]
  }
}
```

### Common Hook Issues

| Issue | Cause | Fix |
|-------|-------|-----|
| Hook not firing | Wrong matcher or event | Check matcher regex against tool name |
| Hook blocks everything | Missing exit 0 for allowed cases | Add `exit 0` as the default case |
| "Permission denied" | Script not executable | `chmod +x .claude/hooks/script.sh` |
| JSON parse error | Shell profile prints text | Check for echo/motd in bash profile |
| Hook too slow | Complex logic | Move to async hook or optimize script |

---

## 9. Managing Agents and Skills

### List Available Agents

```bash
# In Claude Code
# /agents

# Or check the directory
ls .claude/agents/
```

### Modify an Agent

Edit the `.md` file directly. Changes take effect on the next Claude Code session.

Key frontmatter fields to adjust:
- `model`: Change the model (sonnet, haiku, opus)
- `tools`: Restrict or expand tool access
- `maxTurns`: Limit how long the agent can run
- `permissionMode`: `plan` (read-only), `default`, `acceptEdits`, `dontAsk`, or `bypassPermissions`
- `memory`: `project`, `user`, or `local` for persistent memory, omit for none

### Create a Project-Specific Agent

```bash
# Create in your project's .claude/agents/ directory
cat > /path/to/project/.claude/agents/deploy-checker.md << 'EOF'
---
name: deploy-checker
description: Validates deployment readiness. Checks build, tests, lint, and env config before deploying.
tools: Read, Grep, Glob, Bash
model: sonnet
---

You verify that a project is ready to deploy. Check:
1. Build succeeds
2. All tests pass
3. No lint errors
4. Environment variables are configured
5. Database migrations are up to date
EOF
```

### List Available Skills

```bash
# In Claude Code: type / to see all commands
# Skills appear as /skill-name

ls .claude/skills/*/SKILL.md
```

---

## 10. Troubleshooting

### "Claude Code not found"

```bash
# Check if installed
which claude

# Install
curl -fsSL https://claude.ai/install.sh | bash
```

### "API key invalid" or "Authentication failed"

```bash
# Check if key is set
echo $ANTHROPIC_API_KEY | head -c 10

# Re-provision
claude  # Follow interactive login

# Check config
cat ~/.claude.json | jq '.oauthAccount'
```

### "Hook script failed"

```bash
# Check if executable
ls -la .claude/hooks/

# Make executable
chmod +x .claude/hooks/*.sh

# Test manually
echo '{}' | .claude/hooks/block-dangerous-commands.sh; echo "Exit: $?"

# Check for syntax errors
shellcheck .claude/hooks/*.sh

# Run with debug
claude --debug
```

### "MCP server failed to start"

```bash
# Check if npx works
npx -y @modelcontextprotocol/server-filesystem --help

# Increase timeout
MCP_TIMEOUT=30000 claude

# Check in session
# /mcp

# Remove and re-add
claude mcp remove <name>
claude mcp add --transport stdio <name> -- <command>
```

### "MCP server authentication failed"

```bash
# Re-add with fresh credentials (masked prompt)
claude-workspace mcp add <name> --api-key ENV_VAR_NAME -- <command>

# For remote servers with expired Bearer token
claude-workspace mcp remote <url> --name <name> --bearer

# For OAuth servers, re-authenticate in Claude Code
# /mcp → select server → Authenticate

# Verify env var is being passed to the server
claude --debug  # Look for MCP env configuration in output
```

### "Context too full" / "Auto-compacting frequently"

```bash
# Check context usage
# In Claude Code: /context

# Manually compact
# /compact

# Adjust threshold (lower = more aggressive)
# In settings.json env:
# "CLAUDE_AUTOCOMPACT_PCT_OVERRIDE": "70"

# Use subagents for context-heavy work
# "Use the explorer agent to find all API endpoints"
```

### "Agent not found"

```bash
# Check agent file exists and has correct format
cat .claude/agents/planner.md | head -10

# Verify YAML frontmatter is valid (--- markers present)
# Agent files MUST start with --- and have a closing ---

# Check for agent in session
# /agents
```

### "Hooks not blocking as expected"

```bash
# Check matcher regex
# "Bash" matches the Bash tool
# "Write|Edit" matches Write OR Edit
# "mcp__memory__.*" matches all memory MCP tools

# Test the matcher
claude --debug
# Then trigger the tool and check hook output
```

### "Worktree won't create"

```bash
# Check if branch already exists
git branch -a | grep <branch-name>

# Check existing worktrees
git worktree list

# Remove stale worktree reference
git worktree prune
```

---

## 11. Rollback Procedures

### Rollback Platform Config in a Project

```bash
# If config is version controlled
cd /path/to/project
git checkout HEAD~1 -- .claude/ .mcp.json

# If using symlinks, checkout previous platform version
cd ~/claude-workspace
git checkout v1.0.0
```

### Rollback Claude Code CLI

```bash
# Reinstall via official installer (version pinning may not be supported)
curl -fsSL https://claude.ai/install.sh | bash
```

### Rollback Settings

Claude Code auto-backs up settings with timestamps:
```bash
# Find backups
ls ~/.claude/backups/.claude.json.backup.*

# Restore (pick the most recent timestamp)
cp ~/.claude/backups/.claude.json.backup.<timestamp> ~/.claude/settings.json
```

### Emergency: Disable All Platform Features

```json
// .claude/settings.local.json
{
  "disableAllHooks": true
}
```

This instantly disables all hooks while preserving the configuration. Remove when the issue is resolved.

---

## 12. Onboarding New Team Members

### Checklist

1. **Prerequisites**
   - [ ] Git configured (name, email, SSH key)
   - [ ] Access to Anthropic Console (for API key) or org API key

2. **Setup**
   - [ ] Install CLI: `curl -fsSL https://raw.githubusercontent.com/lamchakchan/claude-workspace/main/install.sh | bash`
   - [ ] Run setup: `claude-workspace setup`

3. **Project Onboarding**
   - [ ] Clone their project repo
   - [ ] Attach platform: `claude-workspace attach /path/to/project`
   - [ ] Customize `.claude/CLAUDE.local.md` with their personal context
   - [ ] Copy `.claude/settings.local.json.example` → `.claude/settings.local.json`

4. **Verification**
   - [ ] Run doctor: `claude-workspace doctor`
   - [ ] Start Claude Code: `cd /path/to/project && claude`
   - [ ] Verify hooks: `Ctrl+O` and try a test command
   - [ ] Verify MCP: `/mcp`
   - [ ] Verify agents: `/agents`

5. **First Task**
   - [ ] Walk through a simple task together
   - [ ] Show the plan-first workflow
   - [ ] Demonstrate subagent usage
   - [ ] Show how to use `/compact` and context management

### Quick Start Script for New Members

```bash
#!/bin/bash
# new-member-setup.sh
set -euo pipefail

echo "=== New Member Setup ==="

# 1. Install the CLI
curl -fsSL https://raw.githubusercontent.com/lamchakchan/claude-workspace/main/install.sh | bash

# 2. Setup (interactive - API key provisioning)
claude-workspace setup

# 3. Attach to project (pass project path as argument)
PROJECT=${1:?Usage: ./new-member-setup.sh /path/to/project}
claude-workspace attach "$PROJECT"

# 4. Verify
claude-workspace doctor

echo ""
echo "Setup complete. Start Claude Code:"
echo "  cd $PROJECT && claude"
```

---

## 13. Offboarding

### Remove Platform from a Project

```bash
# Remove all platform config
rm -rf /path/to/project/.claude/
rm /path/to/project/.mcp.json

# Or selectively
rm -rf /path/to/project/.claude/agents/
rm -rf /path/to/project/.claude/skills/
rm -rf /path/to/project/.claude/hooks/
# Keep .claude/settings.json and CLAUDE.md if customized
```

### Remove User's Global Config

```bash
# Remove global settings
rm -rf ~/.claude/

# Remove auth
rm ~/.claude.json

# Remove environment variable
# Edit ~/.bashrc or ~/.zshrc: remove ANTHROPIC_API_KEY line
```

### Revoke API Key

1. Go to Anthropic Console
2. Revoke the user's API key
3. If using shared key, rotate it

---

## 14. Security Incident Response

### If a Secret Was Committed

```bash
# 1. Immediately rotate the compromised credential
# 2. Remove from git history
git filter-branch --force --index-filter \
  "git rm --cached --ignore-unmatch path/to/secret" HEAD
# 3. Force push (coordinate with team)
git push --force
# 4. Review validate-secrets.sh hook for gaps
```

### If Claude Executed a Dangerous Command

```bash
# 1. Check what ran
claude --debug  # or review session transcript

# 2. Assess damage
git status  # file changes
git diff    # content changes
git log -5  # recent commits

# 3. Revert if needed
git checkout -- .  # revert all uncommitted changes
git reset HEAD~1   # undo last commit (if Claude committed)

# 4. Review and strengthen hooks
cat .claude/hooks/block-dangerous-commands.sh
# Add the pattern that was missed
```

### If MCP Server Was Compromised

```bash
# 1. Remove the server immediately
claude mcp remove <name>

# 2. Revoke any credentials associated with it
# 3. Review what tools were exposed
# 4. Check session transcripts for unauthorized actions
```

---

## 15. Monitoring and Observability

### OpenTelemetry

The platform enables telemetry by default. Configure your OTEL collector:

```json
// In settings.json
{
  "env": {
    "CLAUDE_CODE_ENABLE_TELEMETRY": "1",
    "OTEL_METRICS_EXPORTER": "otlp",
    "OTEL_EXPORTER_OTLP_ENDPOINT": "http://your-collector:4317"
  }
}
```

### Session Transcripts

All session transcripts are saved to:
```
~/.claude/projects/<path-encoded-working-directory>/<session-uuid>.jsonl
```

The directory name is the working directory with slashes replaced by hyphens (e.g., `/Users/lam/my-project` → `-Users-lam-my-project`).

These are JSONL files containing every message, tool call, and result.

### Cost Monitoring

```bash
# Turn duration shows per-turn timing
# Enabled by default in settings: "showTurnDuration": true

# Disable cost warnings if needed
# "DISABLE_COST_WARNINGS": "1"
```

### Usage Tracking

Track API usage via:
1. Anthropic Console dashboard
2. OpenTelemetry metrics
3. Session transcript analysis

### Alerting

Set up alerts for:
- API key approaching rate limits
- Unusual spending patterns
- Hook failures (exit code 2 spikes)
- MCP server disconnections
