# Skills Reference

Skills are reusable workflows that Claude follows when you invoke them as slash commands. Unlike agents (which Claude spawns autonomously to handle delegated work), skills are user-invoked — you type `/skill-name` and Claude reads the skill's instructions, then follows that workflow for your current task.

---

## Table of Contents

1. [Slash Commands](#slash-commands)
2. [Using Skills](#using-skills)
3. [Built-in Skills](#built-in-skills)
4. [Creating Custom Skills](#creating-custom-skills)
5. [Skills vs Agents vs Commands](#skills-vs-agents-vs-commands)

---

## Slash Commands

Type `/` in Claude Code to see all available commands. Skills are auto-discovered from two locations:

| Location | Scope | Shared with team? |
|----------|-------|--------------------|
| `.claude/skills/*/SKILL.md` | Project | Yes — checked into git |
| `~/.claude/commands/*.md` | Personal | No — local to your machine |

When you type `/onboarding`, Claude reads the corresponding `SKILL.md` file and follows the workflow defined inside. The skill's instructions are injected as temporary context for that session only — they don't persist after the task is complete.

**Project skills** live in your repository and are shared with everyone on the team. They use YAML frontmatter for metadata (name, description) and contain detailed multi-step workflows.

**Personal commands** live in your home directory and are only available to you. They use a simpler format (plain Markdown, no frontmatter) and are useful for personal shortcuts that don't need to be shared.

---

## Using Skills

### When to use skills

- `/plan-and-execute` — Before any non-trivial implementation task
- `/plan-resume` — When returning to a project with an in-progress plan
- `/pr-workflow` — When your changes are ready for review
- `/onboarding` — After running `claude-workspace attach` on a new project
- `/context-manager` — When working in a large codebase and context is filling up
- `/statusline-setup` — When you want live cost and context metrics in your terminal

### How skills work

1. You type `/skill-name` (or Claude suggests it based on context)
2. Claude reads the skill's `SKILL.md` file
3. The instructions are injected into the current session as temporary context
4. Claude follows the workflow step by step
5. When the task completes, the skill's context is no longer active

### Chaining skills

Skills can be used in sequence within a session. A common pattern:

1. `/plan-and-execute` — Plan and implement a feature
2. `/pr-workflow` — Create a PR for the completed work

Or across sessions:

1. Session 1: `/plan-and-execute` → plan is saved to `./plans/`
2. Session 2: `/plan-resume` → pick up where you left off

---

## Built-in Skills

The platform ships with 6 skills:

| Skill | Description | When to use |
|-------|-------------|-------------|
| `plan-and-execute` | Plan-first development workflow | Complex tasks, multi-file changes, new features |
| `plan-resume` | Resume parked plans from prior sessions | Returning to in-progress work |
| `pr-workflow` | Create or update PRs with structured output | Changes ready for review |
| `onboarding` | Post-attach project analysis and memory init | First time in a new project |
| `context-manager` | Context management strategies | Large codebases, full context windows |
| `statusline-setup` | Configure the Claude Code statusline | Setting up live cost/context metrics |

### plan-and-execute

Enforces a plan-first workflow for non-trivial tasks. Claude analyzes the request, researches the codebase, writes a plan to `./plans/`, gets your approval, then executes step by step with validation after each change.

**Workflow:** Resume check → Research → Write plan → Get approval → Execute → Validate → Update docs

Plans use status tracking (`Draft` → `Approved` → `In Progress` → `Complete`) and checkboxes for progress. The plan file is saved to disk so it can be resumed in future sessions.

### plan-resume

Picks up a previously parked plan. Lists all plans in `./plans/`, shows their status, assesses what's already done, and creates a todo list from the remaining steps.

**Workflow:** Discover plans → Select one → Assess progress → Update status → Create todos → Resume execution

### pr-workflow

Creates or updates a pull request with a structured title, body, and test plan. Detects PR stacks (chained branches), generates Mermaid diagrams for stacked PRs, and actively executes the test plan (running tests locally, checking CI).

**Workflow:** Detect create/update mode → Detect PR stack → Gather full branch context → Derive title with tags → Build body → Execute test plan → Create/update PR

### onboarding

Explores a new project to understand its structure, conventions, and key files. Detects installed MCP servers, initializes persistent memory, and generates or updates the project's `.claude/CLAUDE.md`.

**Workflow:** Read project identity → Map build system → Understand architecture → Find key files → Detect conventions → Check MCP servers → Initialize memory → Generate CLAUDE.md

### context-manager

Provides strategies for managing the context window in large codebases: hierarchical exploration (broad to narrow), subagent delegation, proactive compaction, selective file reading, working memory via todos, and persistent cross-session memory.

Not a step-by-step workflow — more of a reference guide that Claude applies throughout the session.

### statusline-setup

Configures the Claude Code statusline to display live session cost, context usage, model name, weekly reset countdown, and service status alerts (GitHub, Claude, Cloudflare, AWS, Google Cloud, Azure DevOps). Detects the best available runtime (bun/npx/jq) and writes a wrapper script to `~/.claude/statusline.sh`.

**Workflow:** Check existing config → Write wrapper script → Register in settings → Confirm

Can also be run from the terminal: `claude-workspace statusline`

---

## Creating Custom Skills

### Project skills (shared with team)

Create a file at `.claude/skills/my-skill/SKILL.md`:

```yaml
---
name: my-skill
description: When to use this skill
---

Instructions that Claude follows when this skill is invoked.
Include step-by-step workflows, rules, and output format.
```

The skill is immediately available as `/my-skill` to anyone working in the project.

**Frontmatter fields:**

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Slash command name (used as `/name`) |
| `description` | Yes | When Claude should suggest this skill |

### Personal commands (local only)

Create a file at `~/.claude/commands/my-command.md`:

```markdown
Your instructions here. No frontmatter needed.
Claude follows these instructions when you type /my-command.
```

Personal commands use a simpler format — just plain Markdown. They're useful for shortcuts you use frequently but don't need to share.

### Tips for writing skills

- Be specific about the workflow steps — Claude follows them literally
- Include rules and constraints (e.g., "NEVER skip the planning phase")
- Define output format if the skill should produce structured results
- Reference subagents when the skill needs delegated work (e.g., "Use the explorer subagent")
- Keep skills focused on one workflow — chain multiple skills for complex flows

---

## Skills vs Agents vs Commands

| | Skills | Agents | Commands |
|---|--------|--------|----------|
| **Location** | `.claude/skills/*/SKILL.md` | `.claude/agents/*.md` | `~/.claude/commands/*.md` |
| **Invoked by** | User (`/name`) | Claude (automatically) | User (`/name`) |
| **Scope** | Project (shared via git) | Project (shared via git) | Personal (local only) |
| **Format** | YAML frontmatter + Markdown | YAML frontmatter + Markdown | Plain Markdown |
| **Purpose** | Reusable workflows for the current task | Delegated work in isolated context | Personal shortcuts |
| **Context** | Injected into current session | Runs in separate context window | Injected into current session |
| **Example** | `/plan-and-execute` | Planner, Explorer, Code Reviewer | `/my-shortcut` |

**When to use which:**

- **Skill** — You want a repeatable workflow that the whole team follows (e.g., PR creation, onboarding)
- **Agent** — You want Claude to delegate a subtask to a specialized worker with its own context (e.g., code review, security scanning)
- **Command** — You want a personal shortcut that only you need (e.g., your preferred commit message format)
