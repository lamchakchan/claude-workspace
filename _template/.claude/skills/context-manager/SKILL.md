---
name: context-manager
description: Strategies for managing context in large codebases. Use when working on projects with many files, when context window is getting full, or when you need to efficiently explore unfamiliar code.
---

# Context Management Strategies

When working on large codebases, follow these strategies to manage your context window effectively:

## Strategy 1: Hierarchical Exploration

1. **Start broad** - List directories and read high-level files (README, package.json)
2. **Narrow down** - Use Grep to find relevant code without reading entire files
3. **Read selectively** - Only read the specific files and line ranges you need
4. **Reference, don't quote** - Use `file:line` references instead of pasting code blocks

## Strategy 2: Subagent Delegation

Offload context-heavy operations to subagents:

- **Explore subagent** (Haiku) - Fast codebase discovery, returns concise summaries
- **Test-runner subagent** - Runs tests in isolated context, returns only results
- **Code-reviewer subagent** - Reviews in isolated context, returns only findings

Each subagent gets its own context window, keeping your main conversation clean.

## Strategy 3: Proactive Compaction

- Monitor your context usage mentally
- When you've accumulated many file reads and tool outputs, suggest compaction
- Before compacting, summarize the key findings you want to preserve
- Set `CLAUDE_AUTOCOMPACT_PCT_OVERRIDE` to 80% for earlier automatic compaction

## Strategy 4: File Reference Patterns

Instead of reading entire files:
```
# Bad - reads entire file into context
Read src/services/auth.ts

# Good - reads only what you need
Read src/services/auth.ts (lines 42-58)

# Best - use Grep to find what you need first
Grep "function authenticate" in src/services/
```

## Strategy 5: Working Memory via Todos

Use TodoWrite as working memory:
- Track which files you've already read
- Note key findings as you discover them
- Plan remaining work to avoid re-reading

## Strategy 6: Persistent Cross-Session Memory

For facts that should survive context compaction and new sessions:
- `mcp__memory__search_nodes` — load relevant prior context at session start
- `mcp__memory__create_entities` + `add_observations` — record stable cross-project facts (user preferences, recurring patterns, architectural decisions)
- `mcp__memory__delete_entities` — prune stale or wrong knowledge

Use memory MCP for cross-project facts only. Project-specific notes belong in auto-memory (`~/.claude/projects/<project>/memory/MEMORY.md`), which is automatically loaded every session. See `docs/MEMORY.md` for the full six-layer memory reference.

## Anti-Patterns to Avoid

- Reading files "just in case"
- Quoting large blocks of unchanged code
- Repeatedly reading the same file in one session
- Running broad searches when you can use targeted ones
