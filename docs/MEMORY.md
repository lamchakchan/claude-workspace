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
| **Memory MCP** | `~/.config/claude-workspace/memory.db` (libsql database) | Cross-project | Must query explicitly | Just you |

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

**Example prompts that trigger auto-memory:**

- *"Remember that this project uses `make build-all`, not `go build ./...` — it cross-compiles for 4 targets."*
- *"We just figured out that `_template/` must be rebuilt into the binary after any asset change — save that for next session."*
- *"Note that the attach command spawns a `claude -p` subprocess with a 180s timeout when enriching CLAUDE.md."*

Claude also writes to auto-memory unprompted when it discovers something significant about the codebase mid-session (build quirks, debugging insights, architecture patterns). The four platform agents (`planner`, `code-reviewer`, `security-scanner`, `incident-responder`) write here automatically due to `memory: project` in their frontmatter.

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

**Example Project CLAUDE.md instructions** (shared with the whole team):

- *"Build with `make build-all`. Never run `go build .` directly — it won't cross-compile."*
- *"All PRs require a test plan in the description. Never merge directly to main."*
- *"Use `internal/platform/color.go` helpers for all terminal output — do not use `fmt.Println` for user-facing messages."*

**Example User CLAUDE.md instructions** (personal, all projects):

- *"I prefer short, direct responses. Skip preamble and summaries unless I ask."*
- *"Always use `bun` instead of `npm` or `yarn`."*
- *"When writing commit messages, use imperative mood and explain the 'why', not the 'what'."*

**Example CLAUDE.local.md instructions** (personal, this project only):

- *"My local API runs on port 9000, not the default 8080."*
- *"Skip the smoke tests when iterating locally — they take 5 minutes."*

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

## 4. Memory MCP (cross-project persistent memory)

The default memory MCP provider is **`mcp-memory-libsql`** — a knowledge graph backed by a libsql database at `~/.config/claude-workspace/memory.db`. It runs via `npx` (no extra install beyond Node.js) and is registered automatically by `claude-workspace setup`.

Unlike auto-memory (which is project-scoped and auto-loaded), Memory MCP is **cross-project** and must be **queried explicitly**.

Run `claude-workspace memory` to see your configured provider.
Run `claude-workspace memory configure` to switch providers.

**Example prompts that trigger Memory MCP saves:**

- *"Always use `bun` instead of `npm` for any JavaScript project — save this to memory."*
- *"I prefer imperative commit messages that explain the 'why', not the 'what'. Remember this across all projects."*
- *"Save that I work with Go microservices and prefer one package per subcommand with a `Run()` entry point."*

Unlike auto-memory, Claude will not write to Memory MCP unprompted — you must ask explicitly, or configure your `~/.claude/CLAUDE.md` to instruct Claude to save cross-project facts at session end.

**When to use memory MCP (not auto-memory):**
- User preferences that apply everywhere: *"always use bun, not npm"*
- Cross-project patterns: architecture styles, recurring decisions
- Facts that need full-text-searchable recall across sessions

**When to use auto-memory instead:**
- Project-specific architecture notes
- Session findings scoped to one codebase
- Anything that should be human-readable and auditable in one place

### Core tools (mcp-memory-libsql)

- `mcp__mcp-memory-libsql__search_nodes` — Search the knowledge graph by keyword
- `mcp__mcp-memory-libsql__create_entities` — Add new entities/facts to the graph
- `mcp__mcp-memory-libsql__create_relations` — Link entities with named relationships
- `mcp__mcp-memory-libsql__read_graph` — Read the full knowledge graph
- `mcp__mcp-memory-libsql__delete_entity` — Remove an entity from the graph

### Session workflow

**At session start** — load relevant context:
```
mcp__mcp-memory-libsql__search_nodes({"query": "preferences"})
```

**During work** — record new facts:
```
mcp__mcp-memory-libsql__create_entities({
  "entities": [{"name": "claude-workspace", "entityType": "project",
    "observations": ["Go CLI, builds with go build ./..., tests with go test ./..."]}]
})
```

### How to clear

**Surgical (in-session):**
```
mcp__mcp-memory-libsql__delete_entity({"entityName": "entity-name"})
```

**Nuclear (delete database):**
```bash
rm ~/.config/claude-workspace/memory.db
```

### Alternative: engram (optional)

`engram` is still supported as an optional legacy provider. To switch:
```bash
claude-workspace memory configure --provider engram
```
Requires: `brew install gentleman-programming/tap/engram`. Data stored at `~/.engram/engram.db`.

> **Important:** `~/.claude/CLAUDE.md` must stay in sync with the active memory MCP provider.
> The platform writes this file once during `claude-workspace setup` and will not overwrite it
> on subsequent runs. If you switch memory providers, run:
> ```bash
> claude-workspace memory configure
> ```
> then update the MCP Tool Preferences and Memory Strategy sections in `~/.claude/CLAUDE.md`
> to reference the new provider's tool names.

---

## 5. Native Agent Memory (`memory: project` frontmatter)

Four platform agents (`planner`, `code-reviewer`, `security-scanner`, `incident-responder`) are configured with `memory: project` in their frontmatter. This tells Claude Code to give them persistent memory that survives between sessions.

These agents write their memory into the **auto-memory layer** (`~/.claude/projects/<project>/memory/`). Clearing native agent memory uses the same process as clearing auto-memory — see Section 2 and Section 7 above.

---

## 6. Slash Commands & In-Session Writes

Two built-in Claude Code slash commands interact with memory:

### `/memory` — edit any loaded memory file

Opens an interactive file picker showing every memory file currently loaded in the session. Selecting a file opens it in your `$EDITOR`.

```
/memory
```

Use this to directly edit any scope:

| Want to write to... | Select in the picker |
|---|---|
| User CLAUDE.md | `~/.claude/CLAUDE.md` |
| Project CLAUDE.md | `./.claude/CLAUDE.md` |
| Personal project override | `./CLAUDE.local.md` |
| Auto-memory index | `~/.claude/projects/<project>/memory/MEMORY.md` |

### `/init` — bootstrap a project CLAUDE.md

Analyzes your codebase and generates a `CLAUDE.md` with build commands, conventions, and architecture notes. Run once when setting up a new project (or use `claude-workspace attach` which does this as part of full platform setup).

```
/init
```

### Conversational writes (no slash command needed)

For scopes without a dedicated command, just tell Claude what to remember:

**Auto-memory (project-scoped facts):**
```
"Remember that this repo's integration tests require a running Docker daemon."
"Note that `make build-all` must be used instead of `go build` — save this for next session."
```

**User CLAUDE.md (personal standing instructions):**
```
"Add to my user CLAUDE.md that I always want test coverage checked before declaring a task done."
"Update my global instructions to prefer bun over npm."
```

**Memory MCP (cross-project facts):**
```
"Save to memory that I prefer flat package structures with one responsibility per file."
"Remember across all projects that I use 1Password CLI for secrets — never ask me to paste credentials."
```

**CLAUDE.local.md (personal project overrides):**
```
"Add a CLAUDE.local.md note that my local Postgres runs on port 5433."
```

---

## 7. Clearing Reference

| Layer | In-session clear | Manual clear | Full wipe |
|---|---|---|---|
| **Auto-memory** | *"forget that..."* or `/memory` to edit | `rm ~/.claude/projects/<proj>/memory/MEMORY.md` | `rm -rf ~/.claude/projects/<proj>/memory/` |
| **Memory MCP** | `mcp__mcp-memory-libsql__delete_entity` | `rm ~/.config/claude-workspace/memory.db` | `rm ~/.config/claude-workspace/memory.db` |
| **User CLAUDE.md** | `/memory` to open editor | Edit `~/.claude/CLAUDE.md` directly | Delete the file (re-run `claude-workspace setup` to regenerate) |
| **Project CLAUDE.md** | `/memory` to open editor | Edit `.claude/CLAUDE.md` | `claude-workspace attach --force` to regenerate from template |
| **CLAUDE.local.md** | `/memory` to open editor | Edit `./CLAUDE.local.md` | Delete the file |
| **Session context** | `/clear` (resets conversation buffer, not files) | N/A | N/A |
| **All of the above** | N/A | `rm -rf ~/.claude/ && rm ~/.claude.json` (full offboarding) | See RUNBOOK.md offboarding section |

---

## 8. .gitignore Rules

**What to ignore:**

| Pattern | Why |
|---|---|
| `.claude/MEMORY.md` | Auto-memory normally lives in `~/.claude/...` (outside repo), but if manually placed in `.claude/` it would be tracked. |
| `.claude/*.jsonl` | JSONL conversation/log files that tools could write into the project's `.claude/` directory. |
| `.claude/CLAUDE.local.md` | Personal project overrides. Already covered by Claude Code convention. |
| `.claude/settings.local.json` | Personal local settings. Already covered. |

**What NOT to ignore:**

- `~/.claude/projects/*/memory/MEMORY.md` — lives **outside the repo** in `~/.claude/`. No gitignore entry needed.
- Session `.jsonl` files in `~/.claude/projects/*/` — outside the repo.
- `~/.claude.json` — outside the repo.

Both patterns (`.claude/MEMORY.md`, `.claude/*.jsonl`) are already present in this repo's `.gitignore`, `.claude/.gitignore`, and `_template/.claude/.gitignore`.

---

## 9. Choosing the Right Layer

```
Is this fact relevant to ONE project only?
  YES → Is it an instruction/rule for Claude to follow?
          YES → Project CLAUDE.md (.claude/CLAUDE.md) — shared with team
                Or CLAUDE.local.md — personal override, gitignored
          NO  → Auto-memory (Claude writes it) or tell Claude "remember X"
  NO  → Does it apply to ALL my projects (personal preference)?
          YES + is it an instruction? → User CLAUDE.md (~/.claude/CLAUDE.md)
          YES + is it a fact/pattern?  → Memory MCP (search/save via your configured provider)
          YES + is it org policy?      → Managed policy CLAUDE.md
```

**Quick decision:**
- Writing a rule or instruction → CLAUDE.md at the right scope
- Claude learning a fact during work on one project → auto-memory
- Cross-project fact or user preference that spans sessions → memory MCP
