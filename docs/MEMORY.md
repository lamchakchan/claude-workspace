# Memory Management

This document explains the six memory layers available in Claude Code, when to use each, how to clear them, and what to gitignore.

---

## 1. Overview — The Six Layers

| Layer | File Location | Scope | Auto-loaded? | Shared? |
|---|---|---|---|---|
| **Managed policy** | `/Library/Application Support/ClaudeCode/CLAUDE.md` (macOS) | All users in org | Always | Org-wide via MDM |
| **Project CLAUDE.md** | `./CLAUDE.md` or `./.claude/CLAUDE.md` | Per-project | Always | Team via git |
| **User CLAUDE.md** | `~/.claude/CLAUDE.md` | All projects | Always | Just you |
| **CLAUDE.local.md** | `./CLAUDE.local.md` | Per-project | Always | Just you (gitignored) |
| **Auto-memory** | `~/.claude/projects/<project>/memory/MEMORY.md` | Per-project | First 200 lines | Just you |
| **Memory MCP** | `memory.json` (MCP server data file) | Cross-project | Must query explicitly | Just you |

**Rule of thumb:** More specific instructions take precedence over broader ones. Project CLAUDE.md overrides User CLAUDE.md; CLAUDE.local.md overrides both for personal overrides.

---

## 2. Auto-memory (Claude's automatic notes)

Auto-memory is where Claude writes its own notes during sessions — project patterns, debugging insights, architecture discoveries, your preferences. Unlike CLAUDE.md files that you write for Claude, auto-memory contains notes Claude writes for itself.

**Storage:** Each project gets its own directory at `~/.claude/projects/<project>/memory/`, where `<project>` is derived from the git repository root path (e.g., `/Users/lam/git/myproject` → `-Users-lam-git-myproject`).

```
~/.claude/projects/<project>/memory/
├── MEMORY.md          # Index file, first 200 lines loaded every session
├── debugging.md       # Detailed debug patterns (loaded on demand)
├── architecture.md    # Architecture notes (loaded on demand)
└── ...                # Other topic files Claude creates
```

Only `MEMORY.md` is auto-loaded (first 200 lines). Topic files are read on demand when Claude needs them. Keep `MEMORY.md` concise — move details into topic files.

**Enabling auto-memory:**
```bash
export CLAUDE_CODE_DISABLE_AUTO_MEMORY=0  # Force on (rolling out gradually)
export CLAUDE_CODE_DISABLE_AUTO_MEMORY=1  # Force off
```

**How to view and edit:**
- Use `/memory` in any session to open the memory file selector
- Tell Claude directly: *"remember that we use pnpm, not npm"*

**How to clear:**
```bash
# Remove just the index
rm ~/.claude/projects/-Users-lam-git-myproject/memory/MEMORY.md

# Remove all memory files for a project
rm -rf ~/.claude/projects/-Users-lam-git-myproject/memory/

# In-session: tell Claude directly
"forget that the API requires a local Redis instance"
```

---

## 3. CLAUDE.md Files (instruction memory)

CLAUDE.md files contain instructions you write for Claude. They are loaded at session start and persist until you edit them.

**Hierarchy (most specific wins):**

1. **Managed policy** — IT/DevOps-deployed, applies to all org users. Edit via your config management system.
2. **Project CLAUDE.md** — `.claude/CLAUDE.md` in your repo. Shared with the team via git. For architecture, conventions, build commands.
3. **User CLAUDE.md** — `~/.claude/CLAUDE.md`. Personal preferences for all projects. For your tool preferences, communication style.
4. **CLAUDE.local.md** — `./CLAUDE.local.md` in your repo, auto-gitignored. For personal project overrides (sandbox URLs, local test data).

**CLAUDE.md imports:** Use `@path/to/file` syntax to import other files:
```markdown
See @README for project overview.
```

**Path-specific rules:** Files in `.claude/rules/*.md` with YAML frontmatter `paths:` apply only to matching files:
```markdown
---
paths:
  - "src/api/**/*.ts"
---
# API-specific rules here
```

**How to clear/reset:**
- Edit the file directly (`/memory` opens it in your editor)
- Regenerate the project CLAUDE.md: `claude-workspace attach --force` (overwrites with platform template)
- In-session: `/memory` to view and edit

---

## 4. Memory MCP (cross-project knowledge graph)

The `@modelcontextprotocol/server-memory` MCP server maintains a persistent knowledge graph backed by a `memory.json` file. Unlike auto-memory (which is project-scoped and auto-loaded), the memory MCP graph is **cross-project** and must be **queried explicitly**.

**When to use memory MCP (not auto-memory):**
- User preferences that apply everywhere: *"always use bun, not npm"*
- Cross-project patterns: architecture styles, recurring decisions
- Structured relationships: *"Project X uses Framework Y"*
- Facts that need keyword-queryable recall across sessions

**When to use auto-memory instead:**
- Project-specific architecture notes
- Session findings scoped to one codebase
- Anything that should be human-readable and auditable in one place

### Core primitives

- **Entities** — Named nodes with a type and observations. Types: `project`, `preference`, `pattern`, `person`, `decision`
- **Observations** — Discrete, atomic facts on an entity: `"uses TypeScript strict mode"`
- **Relations** — Directed typed connections: `Project X` → `uses` → `React`

### Session workflow

**At session start** — load relevant context:
```
mcp__memory__search_nodes(query: "project-name OR relevant-concept")
```

**During work** — record new facts:
```
# New entity
mcp__memory__create_entities([{
  name: "claude-workspace",
  entityType: "project",
  observations: ["Go CLI", "builds with go build ./...", "tests with go test ./..."]
}])

# Update existing
mcp__memory__add_observations([{
  entityName: "claude-workspace",
  contents: ["uses go:embed for template distribution"]
}])

# Link entities
mcp__memory__create_relations([{
  from: "claude-workspace",
  to: "Go",
  relationType: "uses"
}])
```

### How to clear

**Surgical (in-session):**
```
mcp__memory__delete_entities(entityNames: ["entity-name"])
mcp__memory__delete_observations(observations: [{entityName: "...", observations: ["stale fact"]}])
mcp__memory__delete_relations(relations: [{from: "...", to: "...", relationType: "..."}])
```

**Full wipe (in-session):**
```
# 1. List everything
mcp__memory__read_graph()
# 2. Delete all entities (relations/observations cascade-delete)
mcp__memory__delete_entities(entityNames: ["entity1", "entity2", ...])
```

**Nuclear (delete data file):**
```bash
# Find the file
find ~ -name "memory.json" -path "*/server-memory/*" 2>/dev/null
find ~/.npm -name "memory.json" 2>/dev/null

# Delete it
rm /path/to/memory.json
```

The `memory.json` file location depends on how the server is launched. If no `--memory-path` argument is set, it writes to the working directory where `npx` is invoked.

---

## 5. Native Agent Memory (`memory: project` frontmatter)

Four platform agents (`planner`, `code-reviewer`, `security-scanner`, `incident-responder`) are configured with `memory: project` in their frontmatter. This tells Claude Code to give them persistent memory that survives between sessions.

These agents write their memory into the **auto-memory layer** (`~/.claude/projects/<project>/memory/`). Clearing native agent memory uses the same process as clearing auto-memory — see Section 2 above.

---

## 6. Clearing Reference

| Layer | In-session clear | Manual clear | Full wipe |
|---|---|---|---|
| **Auto-memory** | *"forget that..."* or `/memory` to edit | `rm ~/.claude/projects/<proj>/memory/MEMORY.md` | `rm -rf ~/.claude/projects/<proj>/memory/` |
| **Memory MCP** | `delete_entities` / `delete_observations` / `delete_relations` | `rm /path/to/memory.json` | `rm memory.json` + `delete_entities` for all |
| **User CLAUDE.md** | `/memory` to open editor | Edit `~/.claude/CLAUDE.md` directly | Delete the file (re-run `claude-workspace setup` to regenerate) |
| **Project CLAUDE.md** | `/memory` to open editor | Edit `.claude/CLAUDE.md` | `claude-workspace attach --force` to regenerate from template |
| **CLAUDE.local.md** | `/memory` to open editor | Edit `./CLAUDE.local.md` | Delete the file |
| **Session context** | `/clear` (resets conversation buffer, not files) | N/A | N/A |
| **All of the above** | N/A | `rm -rf ~/.claude/ && rm ~/.claude.json` (full offboarding) | See RUNBOOK.md offboarding section |

---

## 7. .gitignore Rules

**What to ignore:**

| Pattern | Why |
|---|---|
| `memory.json` | MCP memory server data file. Writes to CWD if no `--memory-path` configured — could land in repo root. |
| `.claude/MEMORY.md` | Auto-memory normally lives in `~/.claude/...` (outside repo), but if manually placed in `.claude/` it would be tracked. |
| `.claude/*.jsonl` | JSONL conversation/log files that tools could write into the project's `.claude/` directory. |
| `.claude/CLAUDE.local.md` | Personal project overrides. Already covered by Claude Code convention. |
| `.claude/settings.local.json` | Personal local settings. Already covered. |

**What NOT to ignore:**

- `~/.claude/projects/*/memory/MEMORY.md` — lives **outside the repo** in `~/.claude/`. No gitignore entry needed.
- Session `.jsonl` files in `~/.claude/projects/*/` — outside the repo.
- `~/.claude.json` — outside the repo.

All three new patterns (`memory.json`, `.claude/MEMORY.md`, `.claude/*.jsonl`) are already present in this repo's `.gitignore`, `.claude/.gitignore`, and `_template/.claude/.gitignore`.

---

## 8. Choosing the Right Layer

```
Is this fact relevant to ONE project only?
  YES → Is it an instruction/rule for Claude to follow?
          YES → Project CLAUDE.md (.claude/CLAUDE.md) — shared with team
                Or CLAUDE.local.md — personal override, gitignored
          NO  → Auto-memory (Claude writes it) or tell Claude "remember X"
  NO  → Does it apply to ALL my projects (personal preference)?
          YES + is it an instruction? → User CLAUDE.md (~/.claude/CLAUDE.md)
          YES + is it a fact/pattern?  → Memory MCP (mcp__memory__*)
          YES + is it org policy?      → Managed policy CLAUDE.md
```

**Quick decision:**
- Writing a rule or instruction → CLAUDE.md at the right scope
- Claude learning a fact during work on one project → auto-memory
- Cross-project fact or user preference that spans sessions → memory MCP
