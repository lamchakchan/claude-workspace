# Agents Reference

Agents are specialized AI workers that Claude spawns autonomously to handle delegated subtasks. Unlike skills (which you invoke with `/name`), agents run automatically when Claude determines that a task benefits from isolated, focused work — such as code review, test execution, or security scanning. Each agent runs in its own context window and returns results to the main session.

---

## Table of Contents

1. [How Agents Work](#how-agents-work)
2. [Built-in Agents](#built-in-agents)
3. [Agent Configuration](#agent-configuration)
4. [Creating Custom Agents](#creating-custom-agents)
5. [Agents vs Skills vs Commands](#agents-vs-skills-vs-commands)

---

## How Agents Work

Agents are auto-discovered from two locations:

| Location | Scope | Shared with team? |
|----------|-------|--------------------|
| `.claude/agents/*.md` | Project | Yes — checked into git |
| `~/.claude/agents/*.md` | Personal | No — local to your machine |

When Claude decides to delegate work, it:

1. Selects the appropriate agent based on the task (e.g., code-reviewer for reviewing changes)
2. Spawns the agent in an **isolated context window** — the agent cannot see or pollute the main conversation
3. The agent works autonomously using its allowed tools (Read, Grep, Bash, etc.)
4. The agent returns a structured summary to the main session
5. The main session incorporates the findings and continues

You can also request agents explicitly:

```
> Use the explorer to map the authentication module
> Run a code review on my changes
> Scan for security vulnerabilities before we ship
```

### Key properties

- **Isolated context**: Each agent runs in its own context window, protecting the main session from context bloat
- **Tool-restricted**: Agents only have access to the tools listed in their configuration (e.g., read-only agents cannot modify files)
- **Model-specific**: Agents can run on different models — Haiku for fast exploration, Sonnet for detailed analysis, Opus for complex planning
- **Permission modes**: Agents can be restricted to read-only (`plan`), edit-allowed (`acceptEdits`), or unrestricted (default)

---

## Built-in Agents

The platform ships with 10 agents:

| Agent | Model | Purpose | When to use |
|-------|-------|---------|-------------|
| `explorer` | Haiku | Fast codebase exploration | Before planning or implementing, to understand code structure |
| `planner` | Opus | Deep implementation planning | Complex multi-step tasks, refactoring, multi-file changes |
| `code-reviewer` | Sonnet | Code quality and correctness review | After any code changes, before PRs |
| `test-runner` | Sonnet | Test execution and failure diagnosis | After implementation, before PRs or merging |
| `security-scanner` | Sonnet | Security vulnerability analysis | Before PRs involving auth, input handling, or dependency changes |
| `dependency-updater` | Sonnet | Dependency update and maintenance | Updating packages, analyzing breaking changes, license review |
| `infra-reviewer` | Sonnet | Infrastructure config review | Reviewing Dockerfiles, CI/CD, docker-compose, K8s manifests |
| `documentation-writer` | Sonnet | Technical documentation updates | After implementing features, changing APIs, or refactoring |
| `incident-responder` | Sonnet | Production incident diagnosis | Stack traces, error spikes, production incidents |
| `team-lead` | Opus | Multi-agent team coordination | Parallel task execution with 3+ teammates |

### explorer

Fast codebase exploration and context gathering. Optimized for speed — uses Haiku to quickly scan directories, search for patterns, and return concise summaries with `file:line` references.

**Purpose:** Understand code structure before planning or implementing. Protects the main context window from being filled with exploration output.

**Tools:** Read, Grep, Glob, Bash (read-only mode)

### planner

Senior software architect that creates thorough, actionable implementation plans. Researches the codebase, identifies risks, and writes detailed plans with checkboxes, file references, and success criteria.

**Purpose:** Design implementation approaches for complex tasks. Plans are saved to `./plans/` and can be resumed across sessions.

**Tools:** Read, Grep, Glob, Bash, WebSearch, WebFetch (read-only mode)

### code-reviewer

Reviews code changes for correctness, performance, and maintainability. Checks for bugs, logic errors, missing error handling, DRY violations, and language-specific best practices. Includes framework-specific checks (Spring Boot, React, Django, etc.).

**Purpose:** Catch quality issues after code changes, before they reach PR review.

**Tools:** Read, Grep, Glob, Bash (read-only mode)

### test-runner

Runs the project's test suite, parses results, and provides clear pass/fail reports with root cause analysis for failures. Supports benchmark mode for performance validation.

**Purpose:** Validate correctness after implementation. Always run before creating a PR or merging.

**Tools:** Read, Grep, Glob, Bash

### security-scanner

Scans for vulnerabilities across 9 categories: input validation, auth, data protection, dependencies, configuration, language-specific issues, supply chain, cryptographic misuse, and API security. Writes detailed findings to `.claude/audits/` and returns a brief summary.

**Purpose:** Catch security vulnerabilities before they ship. Use proactively for auth, input handling, or dependency changes.

**Tools:** Read, Grep, Glob, Bash, WebSearch, WebFetch (read-only mode)

### dependency-updater

Analyzes, updates, and maintains project dependencies. Inventories current versions, checks for updates, reads changelogs, identifies breaking changes, and plans staged updates.

**Purpose:** Keep dependencies current and resolve version conflicts. Distinct from security-scanner (which checks for known vulnerabilities).

**Tools:** Read, Grep, Glob, Bash, WebSearch, WebFetch (edit mode)

### infra-reviewer

Reviews infrastructure configuration files for correctness, security, and best practices. Covers Dockerfiles, CI/CD pipelines, docker-compose files, and Kubernetes manifests. Read-only advisory — does not provision or modify infrastructure.

**Purpose:** Catch infrastructure misconfigurations before deployment.

**Tools:** Read, Grep, Glob, Bash (read-only mode)

### documentation-writer

Updates project documentation to keep it in sync with the codebase. Identifies changed APIs and behaviors, audits existing docs for staleness, and makes targeted updates.

**Purpose:** Prevent documentation drift after implementing features, changing APIs, or refactoring.

**Tools:** Read, Write, Grep, Glob, Bash (edit mode)

### incident-responder

Diagnoses production incidents by correlating evidence from stack traces, error logs, Sentry, and Grafana metrics. Maps errors to specific code locations, checks git history for recent changes, and recommends mitigations.

**Purpose:** Triage production incidents — identify root cause, assess blast radius, and recommend fixes.

**Tools:** Read, Grep, Glob, Bash (read-only mode)

### team-lead

Coordinates multi-agent teams for parallel task execution. Reads implementation plans, decomposes them into parallelizable phases, spawns teammates, assigns tasks, monitors progress, and verifies completed work.

**Purpose:** Orchestrate complex plans that benefit from parallel execution by multiple agents.

**Tools:** Read, Grep, Glob, Bash, WebSearch, WebFetch, Write, Edit

---

## Agent Configuration

Agents are defined as Markdown files with YAML frontmatter. The frontmatter controls how Claude spawns and constrains the agent.

**Frontmatter fields:**

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Agent identifier (used when Claude selects the agent) |
| `description` | Yes | When and why to use this agent |
| `tools` | Yes | Comma-separated list of allowed tools (e.g., `Read, Grep, Glob, Bash`) |
| `model` | No | Model to use: `haiku` (fast), `sonnet` (balanced), `opus` (strongest). Defaults to sonnet |
| `permissionMode` | No | `plan` (read-only), `acceptEdits` (can modify files), or omit for default |
| `maxTurns` | No | Maximum API round-trips before the agent stops. Default varies by agent |
| `memory` | No | Set to `project` to give the agent access to project-scoped persistent memory |

### Model selection guidelines

| Model | Cost | Speed | Use for |
|-------|------|-------|---------|
| `haiku` | Low | Fast | Exploration, quick lookups, simple searches |
| `sonnet` | Medium | Balanced | Code review, test execution, security scanning, documentation |
| `opus` | High | Thorough | Complex planning, architecture decisions, team coordination |

### Permission modes

| Mode | Can read? | Can write/edit? | Can run commands? | Use for |
|------|-----------|-----------------|-------------------|---------|
| `plan` | Yes | No | Read-only commands | Exploration, review, scanning |
| `acceptEdits` | Yes | Yes | Yes | Documentation, dependency updates |
| _(default)_ | Yes | Yes | Yes | Implementation, team coordination |

---

## Creating Custom Agents

### Project agents (shared with team)

Create a file at `.claude/agents/my-agent.md`:

```yaml
---
name: my-agent
description: When and why Claude should use this agent
tools: Read, Grep, Glob, Bash
model: sonnet
permissionMode: plan
maxTurns: 20
---

You are a [role] specialist. Your job is to [purpose].

## Process

1. **Step one** — what to do first
2. **Step two** — what to do next

## Output Format

[How to structure the response]

## Guidelines

- Be specific: include file paths and line numbers
- Be concise: summarize rather than quote large blocks
```

The agent is immediately available to Claude for anyone working in the project.

### Tips for writing agents

- **Be specific about the role** — agents work best with a clear, focused purpose
- **Restrict tools** — only grant the tools the agent actually needs (principle of least privilege)
- **Set permission mode** — use `plan` for read-only agents to prevent accidental modifications
- **Choose the right model** — Haiku for speed, Sonnet for most tasks, Opus for complex reasoning
- **Define output format** — structured output helps the main session parse and use results
- **Set maxTurns** — prevents runaway agents from consuming excessive resources

---

## Agents vs Skills vs Commands

| | Agents | Skills | Commands |
|---|--------|--------|----------|
| **Location** | `.claude/agents/*.md` | `.claude/skills/*/SKILL.md` | `~/.claude/commands/*.md` |
| **Invoked by** | Claude (automatically) | User (`/name`) | User (`/name`) |
| **Scope** | Project (shared via git) | Project (shared via git) | Personal (local only) |
| **Format** | YAML frontmatter + Markdown | YAML frontmatter + Markdown | Plain Markdown |
| **Purpose** | Delegated work in isolated context | Reusable workflows for the current task | Personal shortcuts |
| **Context** | Runs in separate context window | Injected into current session | Injected into current session |
| **Example** | Explorer, Code Reviewer, Planner | `/plan-and-execute`, `/pr-workflow` | `/my-shortcut` |

**When to use which:**

- **Agent** — You want Claude to delegate a subtask to a specialized worker with its own context (e.g., code review, security scanning, test execution)
- **Skill** — You want a repeatable workflow that the whole team follows (e.g., PR creation, onboarding)
- **Command** — You want a personal shortcut that only you need (e.g., your preferred commit message format)

For the full comparison, see [Skills Reference](SKILLS.md#skills-vs-agents-vs-commands).
