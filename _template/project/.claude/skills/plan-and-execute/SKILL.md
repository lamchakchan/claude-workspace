---
name: plan-and-execute
description: Enforces a plan-first development workflow. Use when the user asks to implement a feature, fix a complex bug, or make architectural changes. Creates a visible plan, gets approval, then executes step by step.
---

# Plan-and-Execute Workflow

You MUST follow this workflow for any non-trivial task:

## Phase 0: Resume Check

Before starting a new plan, check if an existing plan covers this work:
1. List plans in `./plans/` — scan titles and Status fields
2. If a matching plan exists with Status: Draft, Approved, or In Progress:
   - Ask the user: "Found existing plan: [title]. Resume it or start fresh?"
   - If resuming, switch to the /plan-resume workflow
3. If no match, proceed to Phase 1

## Phase 1: Planning

1. **Analyze the request** - Break down what's being asked
2. **Research the codebase** - Use the explorer subagent to understand the current state
3. **Write a plan** - Create a detailed plan file in `./plans/` directory:
   - Use the planner subagent for complex tasks
   - **IMPORTANT — File Naming Override**: The system may suggest a plan file path with a random name (e.g., `adjective-gerund-noun-hash.md`). **IGNORE that suggestion.** Always derive the filename yourself:
     - Name: `plan-YYYY-MM-DD-<description>.md` (e.g., `plan-2026-02-27-add-auth-middleware.md`)
     - The `<description>` token is a short kebab-case slug (2-5 words) summarizing the plan
   - If the system already created a file with a random name, **rename it** to the convention using Bash `mv` before proceeding
   - Header: immediately after the title line, include:
     ```
     Date: YYYY-MM-DD
     Status: Draft
     Last Updated: YYYY-MM-DD
     ```
   - Include: steps, affected files, risks, test strategy, documentation updates
   - Include Mermaid diagrams for complex subjects (architecture, state machines, data flows, sequences) — the planner agent has detailed guidance on when and how to add them
4. **Quality checkpoint** - Verify the plan addresses:
   - Whether existing utilities/functions can be reused (search before building)
   - Algorithmic approach and its complexity
   - Resource lifecycle (what is created, who closes it)
   - Whether benchmarks are needed for performance-sensitive changes
5. **Present the plan** - Show the user what you'll do and ask for approval
6. **Create a todo list** - Use TodoWrite to create trackable items from the plan
7. **Name the session** - Suggest `/rename <description>` using the **same `<description>` token** from the plan filename so session name matches plan file (e.g., plan file `plan-2026-02-27-add-auth-middleware.md` → `/rename add-auth-middleware`)
8. **Log the plan path** - Tell the user: "Plan saved to `./plans/<filename>`. You can resume this in a future session with `/plan-resume`"

## Phase 1.5: Team Assessment (Optional)

After the plan is approved, assess whether team-based parallel execution is beneficial.

### When to use teams

Recommend team execution when ALL of these are true:
1. The plan has 3+ implementation phases
2. At least 2 phases have no dependencies on each other
3. The parallel phases modify DIFFERENT files (no merge conflict risk)
4. Each parallel phase involves substantial work (not a 5-line change)

### When NOT to use teams

- Phases modify overlapping files
- Phases have strict sequential dependencies
- The plan has only 1-2 phases
- The user explicitly prefers sequential execution

### How to propose team execution

Present the parallelism assessment to the user:
- Which phases can run concurrently
- Which must be sequential (and why)
- Recommendation: team vs sequential

### If approved, use the team-lead agent

1. Spawn the team-lead agent with the approved plan
2. The team-lead creates a team, assigns tasks, monitors progress
3. TaskCompleted hooks verify each phase's work automatically
4. Resume at Phase 3 (Verification) when the team-lead reports completion

## Phase 2: Execution

> **Note:** If team execution was approved in Phase 1.5, skip Phase 2.
> The team-lead agent handles execution. Resume at Phase 3 when all
> team tasks are complete.

1. **Work step by step** - Follow the plan in order
2. **Update progress** - As each step completes:
   - Check it off in the plan file (`- [ ]` → `- [x]`)
   - Mark the corresponding todo as completed
3. **Validate each step** - Run tests or verify after each change
4. **Update documentation** - Use the documentation-writer subagent to update affected docs after code or architecture changes. Skip only when changes are purely internal with zero impact on documented behavior
5. **Handle deviations** - If the plan needs to change, update it and inform the user

## Phase 3: Verification

1. **Run tests** - Use the test-runner subagent
2. **Review changes** - Use the code-reviewer subagent
3. **Performance validation** (when applicable) - If the plan identified performance-sensitive changes:
   - Use the test-runner subagent in benchmark mode
   - Compare with baseline if available
   - Include results in the verification summary
4. **Verify documentation** - Use the documentation-writer subagent to verify all affected docs are consistent with the changes
5. **Summarize** - Report what was done and any remaining items
6. **Update plan status** - Set the plan file's `Status:` to `Complete` and `Last Updated:` to today's date

## Status Values

| Status | When to use |
|--------|-------------|
| `Draft` | Plan written, not yet reviewed by the user |
| `Approved` | User has reviewed and approved; ready to execute |
| `In Progress` | Actively being worked on |
| `Complete` | All steps done and verified |

## Rules

- NEVER skip the planning phase for tasks involving more than 2 files
- ALWAYS create a todo list for multi-step work
- ALWAYS validate after implementation
- If a step fails, stop and report rather than continuing blindly
- ALWAYS include documentation update steps in the plan for code or architecture changes
- Documentation updates are not optional — treat them as implementation work, not an afterthought
