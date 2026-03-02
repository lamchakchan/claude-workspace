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

## Phase 1.5: Execution Mode Assessment

After the plan is approved, determine the best execution mode. If the plan includes a "Team Execution Feasibility" section (from the planner agent), use its annotations as input.

### Three execution modes

| Mode | When to use | Who creates the team? |
|------|-------------|-----------------------|
| **Sequential** (default) | Simple work, overlapping files, or user prefers it | No team needed |
| **Solo team** | Complex sequential work benefiting from structured task tracking and automated verification hooks | You (the main agent) create the team directly |
| **Multi-agent team** | 2+ phases can run in parallel on isolated file sets | You (simple) or the team-lead agent (complex) |

### Scoring rubric

Score the plan against these criteria:

| Criterion | Points | Description |
|-----------|--------|-------------|
| Multiple phases | +1 | Plan has 3+ implementation phases |
| Independent work | +2 | At least 2 phases have no dependencies on each other |
| File isolation | +1 | Parallel phases modify DIFFERENT files (no merge conflict risk) |
| Substantial phases | +1 | Each parallel phase involves substantial work (not a 5-line change) |
| Automated gates desired | +1 | Plan would benefit from TaskCompleted hooks running tests between phases |

**Hard disqualifiers** (override the score — never use multi-agent teams):
- Phases modify overlapping files
- The user explicitly prefers sequential execution

### Mode selection

| Score | Recommended mode |
|-------|-----------------|
| 0-1 | **Sequential** — proceed directly to Phase 2 |
| 2 | **Solo team** — you create a team, create tasks for each phase, and execute them yourself sequentially. TaskCompleted hooks run tests between phases automatically. |
| 3+ with independent phases | **Multi-agent team** — parallelize independent phases |
| 3+ without independent phases | **Solo team** — structured tracking with automated gates |

### How to propose the execution mode

Present the assessment to the user:
- Which phases can run concurrently (if any)
- Which must be sequential (and why)
- The recommended mode and its benefits
- **For solo teams**: "I'll create a team with task tracking. Each phase gets a task, and automated hooks will run tests between phases. No additional agents needed."
- **For multi-agent teams**: Estimated time savings from parallelism. Example: "Phases A, B, C take ~5 min each sequentially (15 min). Running A+B in parallel then C takes ~10 min — saving ~5 min (~33%)."

### Using the Team Coordination Plan

When the plan includes a **Team Coordination Plan** section (produced by the planner agent for solo-team or multi-agent-team modes), use it as the execution blueprint:

- **Roles table** tells you who the coordinator is and what agent types to spawn
- **Task Assignment table** maps directly to `TaskCreate` calls — use the dependencies column for `addBlockedBy`
- **Execution Phases** shows the parallel/sequential structure to follow
- **Communication Protocol** and **Error Handling** set the rules for the team

Do not re-derive the coordination strategy. The plan already specifies it.

### Executing the chosen mode

**Sequential** (score 0-1): Proceed directly to Phase 2.

**Solo team** (score 2, or 3+ without parallelism):
1. Use `TeamCreate` to create a team for this plan
2. Use the plan's **Task Assignment** table to create tasks via `TaskCreate` — one per row, with dependencies from the "Depends On" column
3. Work through tasks yourself in the order specified by the plan's **Execution Phases** list
4. Mark each task completed as you go — TaskCompleted hooks verify each phase automatically
5. When all tasks are complete, use `TeamDelete` to clean up
6. Resume at Phase 3 (Verification)

**Multi-agent team** (score 3+ with parallelism):
- **Simple teams (2 teammates, clear phases)**: Create the team yourself using `TeamCreate`. Use the plan's **Roles** and **Task Assignment** tables to spawn teammates with the correct `subagent_type` and assign tasks via `TaskUpdate`. Follow the plan's **Communication Protocol** for monitoring. Send `shutdown_request` to teammates when done.
- **Complex teams (3+ teammates, multi-phase dependencies)**: Spawn the `team-lead` agent and include the plan file path in its prompt. The team-lead reads the **Team Coordination Plan** section and uses it as the coordination blueprint — roles, task assignments, phase structure, communication rules, and error handling are all specified in the plan.

In both cases, TaskCompleted hooks verify each phase's work automatically. Resume at Phase 3 (Verification) when all team tasks are complete.

## Phase 2: Execution

> **Note:** If a solo team or multi-agent team was chosen in Phase 1.5, skip Phase 2.
> Team execution (solo or multi-agent) handles the work with automated verification.
> Resume at Phase 3 when all team tasks are complete.

1. **Work step by step** - Follow the plan in order
   - **Security checkpoint**: If a step involves auth, user input handling, file I/O with external paths, or shell command construction, run the security-scanner subagent on that step's changes before proceeding to the next step. Do not wait until Phase 3.
2. **Update progress** - As each step completes:
   - Check it off in the plan file (`- [ ]` → `- [x]`)
   - Mark the corresponding todo as completed
3. **Validate each step** - Run tests or verify after each change
4. **Update documentation** - Use the documentation-writer subagent to update affected docs after code or architecture changes. Skip only when changes are purely internal with zero impact on documented behavior
5. **Handle deviations** - If the plan needs to change, update it and inform the user

## Phase 3: Verification

Run verification agents **in parallel** where possible. Steps 1-5 are independent and should be spawned concurrently as subagents.

1. **Run tests** - Use the test-runner subagent
2. **Review changes** - Use the code-reviewer subagent
3. **Security scan** (default: on) - Run the security-scanner subagent unless the plan is documentation-only or covers only test/config files with no code logic changes. It writes a full report to `.claude/audits/` and returns a brief summary. When in doubt, run it.
4. **Infrastructure review** (when applicable) - If the plan touched CI/CD pipelines, Dockerfiles, docker-compose files, or Kubernetes manifests, use the infra-reviewer subagent.
5. **Verify documentation** - Use the documentation-writer subagent to verify all affected docs are consistent with the changes

These agents read different inputs (test output, git diff, security patterns, doc files, infra files) and produce independent reports. Spawn all applicable agents at once and wait for all to complete.

6. **Performance validation** (when applicable) - After test-runner completes, if the plan identified performance-sensitive changes:
   - Use the test-runner subagent in benchmark mode
   - Compare with baseline if available
   - Include results in the verification summary
7. **Summarize** - Collect results from all verification agents and report what was done and any remaining items
8. **Update plan status** - Set the plan file's `Status:` to `Complete` and `Last Updated:` to today's date

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
