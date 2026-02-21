# Architecture & Design Decisions

This document explains why the platform is structured the way it is, what trade-offs were made, and how the pieces fit together. Read this before modifying the platform configuration.

---

## Table of Contents

1. [Design Philosophy](#1-design-philosophy)
2. [Configuration Hierarchy](#2-configuration-hierarchy)
3. [Prompt Layering](#3-prompt-layering)
4. [Subagent Architecture](#4-subagent-architecture)
5. [Hook System Design](#5-hook-system-design)
6. [MCP Strategy](#6-mcp-strategy)
7. [Model Selection Strategy](#7-model-selection-strategy)
8. [Context Management Strategy](#8-context-management-strategy)
9. [Sandboxing & Isolation](#9-sandboxing--isolation)
10. [Extending the Platform](#10-extending-the-platform)

---

## 1. Design Philosophy

### Principles

1. **Safe by default, flexible by choice** - Dangerous operations are blocked out of the box. Teams can loosen restrictions per-project as trust builds.

2. **Plan first, execute second** - AI agents can do damage quickly if they start coding before understanding the problem. The platform enforces planning for non-trivial tasks.

3. **Visible execution** - Every significant action is tracked via TodoWrite. Plans are written to files. Users can always see what Claude is doing and why.

4. **Context is precious** - Large projects can overwhelm the context window. The platform uses subagents to isolate context-heavy operations and aggressive auto-compaction to keep sessions productive.

5. **Team-shared, individually customizable** - Settings, agents, skills, and hooks are committed to git for team consistency. Personal overrides live in `.local` files that are gitignored.

6. **Minimal setup, maximum value** - Three commands to get started: clone, setup, attach. The Go CLI handles installation, API key provisioning, and configuration. A developer should go from "never used Claude Code" to "productive" in under 10 minutes.

### What This Platform Is NOT

- **Not a Claude Code fork** - This is a configuration layer on top of the standard Claude Code CLI. It doesn't modify Claude Code itself.
- **Not a managed service** - This is a toolkit your team self-hosts. There's no SaaS component.
- **Not locked-in** - Any project can remove the platform config and use Claude Code vanilla. The platform adds structure; it doesn't create dependency.

---

## 2. Configuration Hierarchy

Claude Code loads settings from multiple sources with strict precedence. Understanding this is critical for maintaining the platform.

```
Precedence (highest wins):
┌─────────────────────────────────────────────────────┐
│ 1. Managed Settings (managed-settings.json)         │  ← IT-deployed, cannot override
│    Location: /etc/claude-code/ (Linux)              │
│              /Library/Application Support/ClaudeCode/│
├─────────────────────────────────────────────────────┤
│ 2. CLI Arguments (--model, etc.)                    │  ← Session-only
├─────────────────────────────────────────────────────┤
│ 3. Local Settings (.claude/settings.local.json)     │  ← Personal, gitignored
├─────────────────────────────────────────────────────┤
│ 4. Project Settings (.claude/settings.json)         │  ← Team-shared, committed
├─────────────────────────────────────────────────────┤
│ 5. User Settings (~/.claude/settings.json)          │  ← Personal, all projects
└─────────────────────────────────────────────────────┘
```

### Design Decision: Project Settings as the Primary Layer

The platform's settings live in `.claude/settings.json` (layer 4). This means:
- **Team members share the same defaults** - hooks, permissions, model choices
- **Individuals can override** via `.claude/settings.local.json` (layer 3)
- **Organizations can enforce** via managed settings (layer 1)

### What Goes Where

| Setting | Where | Why |
|---------|-------|-----|
| Safety hooks | `.claude/settings.json` | Everyone needs them |
| Permission allow/deny lists | `.claude/settings.json` | Team consistency |
| Default model | `.claude/settings.json` | Cost control |
| Environment variables | `.claude/settings.json` | Feature flags |
| Personal model override | `.claude/settings.local.json` | Individual preference |
| Additional directories | `.claude/settings.local.json` | Personal workspace |
| Org-wide model restrictions | `managed-settings.json` | Governance |
| Org-wide MCP allowlists | `managed-settings.json` | Security policy |

---

## 3. Prompt Layering

### How CLAUDE.md Files Stack

Claude loads instruction files at startup. They concatenate (not override), with higher-priority layers loaded later:

```
Load order (all content is combined):
1. ~/.claude/CLAUDE.md            ← Global: "always plan, never commit secrets"
2. CLAUDE.md (project root)       ← Platform: core principles and workflow
3. .claude/CLAUDE.md              ← Project: tech stack, conventions, key files
4. .claude/CLAUDE.local.md        ← Personal: your name, preferences, env notes
```

### Design Decision: Three-Layer Prompt Architecture

| Layer | Purpose | Managed By |
|-------|---------|------------|
| **Global** (`~/.claude/CLAUDE.md`) | Universal behaviors - plan first, test always, never commit secrets | Platform setup script |
| **Project** (`.claude/CLAUDE.md`) | Project-specific context - tech stack, conventions, directory layout | Team lead / developers |
| **Personal** (`.claude/CLAUDE.local.md`) | Individual preferences - response style, local env details | Each developer |

### Why Not One Big Prompt?

1. **Separation of concerns** - Safety rules don't change per-project. Tech stack context doesn't change per-person.
2. **Version control** - Team prompts are committed. Personal prompts are not.
3. **Composability** - Attaching to a new project only needs to update layer 2, not touch layers 1 or 3.

### Subagent and Skill Prompts

Subagents and skills inject additional context when invoked. These are temporary - they only apply during the subagent's execution, keeping the main conversation clean.

```
During a planner subagent invocation:
  Main context: Global + Platform + Project + Personal
  Subagent context: Planner prompt (detailed planning instructions)
```

---

## 4. Subagent Architecture

### Why Five Agents?

Each agent maps to a distinct phase of the development workflow:

```
    ┌─────────┐     ┌──────────┐     ┌───────────┐     ┌──────────┐
    │ Explorer │────▶│ Planner  │────▶│  (Claude)  │────▶│ Reviewer │
    │ (Haiku)  │     │ (Sonnet) │     │  Executes  │     │ (Sonnet) │
    └─────────┘     └──────────┘     └───────────┘     └──────────┘
         │                                                    │
    Understand              Implement               Validate
    the code                the plan                 the work
                                                         │
                                                  ┌──────────┐
                                                  │  Tester  │
                                                  │ (Sonnet) │
                                                  └──────────┘
                                                         │
                                                  ┌──────────┐
                                                  │ Security │
                                                  │ (Sonnet) │
                                                  └──────────┘
```

### Design Decision: Model Selection Per Agent

| Agent | Model | Rationale |
|-------|-------|-----------|
| Explorer | **Haiku** | Fast, cheap. Exploration is breadth-first - read many files quickly, return a summary. Doesn't need deep reasoning. |
| Planner | **Sonnet** | Plans need good reasoning but not the best. Sonnet balances quality and cost. |
| Code Reviewer | **Sonnet** | Reviews need pattern recognition and security awareness. Sonnet handles this well. |
| Test Runner | **Sonnet** | Needs to analyze test output and diagnose failures. |
| Security Scanner | **Sonnet** | Needs to recognize vulnerability patterns. |

**When to use Opus**: For architectural decisions spanning the entire codebase, switch the main session to Opus (`/model opus`) or use the `opusplan` alias. Don't run all agents on Opus - it's 10x the cost.

### Context Isolation

Each subagent runs in its own context window. This is the key architectural benefit:

```
Main session (200K context):
  ├── Conversation with user
  ├── Project CLAUDE.md loaded
  └── Current working files

Explorer subagent (separate 200K):
  ├── Explorer instructions
  ├── File listings, grep results
  └── Summarized findings → returned to main session
```

The main session only receives the subagent's summary, not all the files it read. This prevents context pollution.

### Memory Persistence

The planner, code-reviewer, and security-scanner agents have `memory: project` enabled. They remember patterns and findings across sessions, building up project knowledge over time.

---

## 5. Hook System Design

### Hook Execution Flow

```
User or Claude triggers a tool call
         │
         ▼
┌─────────────────┐
│  PreToolUse      │ ← Can BLOCK the tool call
│  Hooks           │    - block-dangerous-commands.sh
│                  │    - enforce-branch-policy.sh
│                  │    - validate-secrets.sh (Write/Edit only)
└────────┬────────┘
         │ (if allowed)
         ▼
┌─────────────────┐
│  Tool Executes   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  PostToolUse     │ ← Can provide feedback
│  Hooks           │    - auto-format.sh (Write/Edit only)
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Stop Hook       │ ← Can prevent Claude from stopping
│  (prompt-based)  │    "Are all tasks complete?"
└─────────────────┘
```

### Design Decisions

**Why shell scripts for PreToolUse hooks?**
- Speed: shell scripts execute in milliseconds
- Reliability: no LLM latency or hallucination risk for safety-critical checks
- Determinism: regex pattern matching, not fuzzy AI judgment

**Why a prompt-based Stop hook?**
- The Stop hook checks "are all tasks complete?" - this requires understanding the conversation context
- A shell script can't evaluate task completeness
- Uses Haiku model for speed (< 2 seconds)

**Why not more hooks?**
- Each hook adds latency to every tool call
- The four PreToolUse hooks cover the critical safety surface
- Teams can add project-specific hooks as needed

### Hook Files

| Hook | Event | Matcher | Purpose |
|------|-------|---------|---------|
| `block-dangerous-commands.sh` | PreToolUse | Bash | Blocks rm -rf, force push, curl\|bash, chmod 777 |
| `enforce-branch-policy.sh` | PreToolUse | Bash | Blocks commits to main/master, warns on checkout |
| `validate-secrets.sh` | PreToolUse | Write\|Edit | Scans content for AWS keys, API tokens, passwords |
| `auto-format.sh` | PostToolUse | Write\|Edit | Runs prettier/black/rustfmt on changed files |

---

## 6. MCP Strategy

### Default Servers

```json
{
  "memory": "Persistent knowledge graph across sessions",
  "filesystem": "Secure file operations within project",
  "git": "Git repository operations"
}
```

### Design Decision: Minimal Defaults, Easy Expansion

**Why only three default servers?**
- Each MCP server consumes 500-2,000 tokens of context
- More servers = less room for actual code and conversation
- When tool count exceeds 10% of context, Claude Code enables Tool Search (dynamic loading), which adds latency

**Why templates instead of pre-installing everything?**
- Not every project needs a database MCP
- MCP servers with API keys need per-user configuration
- Templates show the exact command to run - copy, paste, done

### MCP Scopes

```
┌─────────────────────────────────────────────┐
│ Managed MCP (managed-mcp.json)              │ ← Org-wide, IT controls
│ Users CANNOT add/modify/remove these        │
├─────────────────────────────────────────────┤
│ User scope (~/.claude.json global section)  │ ← Personal, all projects
├─────────────────────────────────────────────┤
│ Project scope (.mcp.json in project root)   │ ← Team-shared, committed
├─────────────────────────────────────────────┤
│ Local scope (~/.claude.json per-project)    │ ← Personal, this project
└─────────────────────────────────────────────┘
```

The platform uses **project scope** (`.mcp.json`) for the three default servers so they're shared with the team. Additional servers added via the CLI default to **local scope** (personal).

---

## 7. Model Selection Strategy

### Cost-Performance Matrix

```
                     Cost
                      ▲
                      │
            Opus ●    │    Best reasoning, architecture
                      │    Use for: design decisions, complex refactors
                      │
          Sonnet ●    │    Default for all coding work
                      │    Use for: implementation, review, testing
                      │
           Haiku ●    │    Fast, cheap
                      │    Use for: exploration, quick lookups
                      │
                      └──────────────────────────▶ Speed
```

### Override Points

| Level | How | Example |
|-------|-----|---------|
| Per-session | `/model opus` | Complex architecture session |
| Per-session | `claude --model opus` | Start with specific model |
| Per-agent | `model: haiku` in agent frontmatter | Explorer agent |
| Per-environment | `ANTHROPIC_MODEL=opus` | All sessions on this machine |
| Per-subagent | `CLAUDE_CODE_SUBAGENT_MODEL=opus` | All subagents use Opus |
| Per-org | `availableModels: ["sonnet"]` in managed settings | Restrict to Sonnet only |

---

## 8. Context Management Strategy

### The Problem

Large projects (100K+ lines) can fill Claude's 200K context window quickly. When context is full:
- Claude forgets earlier conversation
- Auto-compaction summarizes and loses detail
- Performance degrades

### The Solution: Layered Context Isolation

```
┌──────────────────────────────────────────────┐
│ Main Session Context (200K)                  │
│                                              │
│  System prompts (~5K)                        │
│  CLAUDE.md layers (~2K)                      │
│  Conversation history                         │
│  Current working files (read on demand)       │
│  Tool outputs                                 │
│                                              │
│  ┌────────────────┐  ┌────────────────┐      │
│  │ Explorer Agent  │  │ Planner Agent  │      │
│  │ (own 200K ctx)  │  │ (own 200K ctx) │      │
│  │ Reads 50 files  │  │ Reads 20 files │      │
│  │ Returns summary │  │ Returns plan   │      │
│  └────────────────┘  └────────────────┘      │
│                                              │
│  Only summaries enter main context ↑          │
└──────────────────────────────────────────────┘
```

### Platform Defaults

| Setting | Value | Purpose |
|---------|-------|---------|
| `CLAUDE_AUTOCOMPACT_PCT_OVERRIDE` | `80%` | Trigger compaction earlier (default is ~90%) |
| `plansDirectory` | `./plans` | Plans written to files, not kept in context |
| `alwaysThinkingEnabled` | `true` | Extended thinking for better reasoning |
| Explorer model | `haiku` | Cheap enough to run frequently |

---

## 9. Sandboxing & Isolation

### Git Worktree Architecture

```
/path/to/project/                    ← Main working copy (main branch)
    ├── .git/                        ← Shared git database
    └── ...

/path/to/project-worktrees/          ← Created by sandbox command
    ├── feature-auth/                ← Worktree (feature-auth branch)
    │   ├── .git (→ pointer to main .git)
    │   ├── .claude/                 ← Copied from main
    │   └── node_modules/            ← Independent install
    ├── feature-api/                 ← Worktree (feature-api branch)
    └── bugfix-payments/             ← Worktree (bugfix-payments branch)
```

**Why git worktrees?**
- True filesystem isolation (different files in each directory)
- Shared git history (can cherry-pick, merge between worktrees)
- Each worktree has its own branch
- Dependencies installed independently (no conflicts)
- Claude Code sees a regular git repo in each worktree

---

## 10. Extending the Platform

### Adding a New Subagent

1. Create `.claude/agents/my-agent.md`:
```yaml
---
name: my-agent
description: What it does (Claude reads this to decide when to use it)
tools: Read, Grep, Glob, Bash    # Restrict tools for safety
model: sonnet                     # sonnet, haiku, or opus
permissionMode: plan              # plan = read-only, default = full access
---

System prompt for the agent. Explain its role, process, and output format.
```

2. Commit to `.claude/agents/` - it's automatically available to all team members.

### Adding a New Skill

1. Create `.claude/skills/my-skill/SKILL.md`:
```yaml
---
name: my-skill
description: When to use this skill
---

Instructions that Claude follows when this skill is invoked.
```

2. Invoke with `/my-skill` in Claude Code.

### Adding a New Hook

1. Create the hook script in `.claude/hooks/my-hook.sh`
2. Make it executable: `chmod +x .claude/hooks/my-hook.sh`
3. Register it in `.claude/settings.json`:
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

### Adding a New MCP Template

1. Create `templates/mcp-configs/my-category.json`
2. Include the JSON config and the CLI command to set it up
3. Document in README.md

### Adding Organization-Wide Policies

Deploy managed settings to system directories:

**Linux:** `/etc/claude-code/managed-settings.json`
**macOS:** `/Library/Application Support/ClaudeCode/managed-settings.json`

```json
{
  "availableModels": ["sonnet", "haiku"],
  "permissions": {
    "deny": ["Bash(curl * | bash)"],
    "disableBypassPermissionsMode": "disable"
  },
  "allowManagedHooksOnly": false,
  "enableAllProjectMcpServers": false
}
```

These settings cannot be overridden by users or project config.
