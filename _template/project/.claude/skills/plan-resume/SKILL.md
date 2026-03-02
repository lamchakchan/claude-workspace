---
name: plan-resume
description: Resume a previously parked plan. Use when the user wants to pick up work from a prior session, review existing plans, or continue implementing a saved plan.
---

# Plan Resume Workflow

Use this skill to resume a previously created plan from `./plans/`.

## Step 1: Discover Plans

1. List all `./plans/*.md` files
2. Read the first 5 lines of each to extract:
   - Title (first `# ` line)
   - `Status:` field
3. Present as a numbered list sorted by modification date (newest first):
   ```
   1. plan-2025-01-15-auth-refactor.md — "Auth Refactor" — Status: In Progress — Modified: 2025-01-15
   2. plan-2025-01-10-api-endpoints.md — "API Endpoints" — Status: Draft — Modified: 2025-01-10
   ```

## Step 2: Select a Plan

- Ask the user which plan to resume
- If only one non-Complete plan exists, suggest it automatically:
  "Only one active plan found: [title]. Resume it?"

## Step 3: Assess Progress

Read the selected plan fully, then:

1. Count `- [x]` (done) vs `- [ ]` (remaining) checkboxes in the Implementation Steps
2. Fall back to checking referenced files exist if the plan predates checkbox tracking
3. Summarize: "X of Y steps complete"

## Step 4: Detect Team Context

Check if this plan was previously executed with a team or is annotated for team execution:

1. **Check for existing tasks**: Run `TaskList` to see if tasks from a prior session exist (team or standalone).
2. **Check plan annotations**: Look for a `## Team Execution Feasibility` section in the plan. Check the `Recommended mode` field.

### If existing tasks are found with incomplete items:
- Present the status: "Found X completed and Y remaining tasks from a prior session"
- Ask the user: "Resume with existing tasks, start fresh, or proceed without task tracking?"
- If resuming: pick up from the next incomplete task

### If no existing tasks but the plan has team annotations:
- Check the plan's `Recommended mode` field
- If it says `solo-team` or `multi-agent-team`, suggest: "This plan was designed for team execution. Create a team for the remaining phases?"
- If the user agrees, follow the same team creation flow as plan-and-execute Phase 1.5

### If no team context at all:
- Proceed with sequential execution (existing behavior)

## Step 5: Update the Plan File

1. Set `Status: In Progress`
2. Set or add `Last Updated: YYYY-MM-DD` (use today's date)
3. Add a `## Progress` section (or update existing) with assessment results:
   ```markdown
   ## Progress
   - Phase 1: Complete (3/3 steps done)
   - Phase 2: Partial (1/4 steps done)
   - Phase 3: Not started
   ```

## Step 6: Create Todo List

Generate todos from remaining incomplete items using TaskCreate/TaskUpdate:
- One task per remaining phase or logical group of steps
- Set dependencies between tasks where phases are sequential

## Step 7: Session Hygiene

Suggest the user run `/rename <plan-description>` so this session is easy to find later with `claude --resume`.

## Status Values

| Status | Meaning |
|--------|---------|
| `Draft` | Plan written, not yet reviewed by the user |
| `Approved` | User has reviewed and approved; ready to execute |
| `In Progress` | Actively being worked on (set this on resume) |
| `Complete` | All steps done and verified |

## Rules

- NEVER skip the progress assessment — always verify what's already done
- NEVER modify completed steps or phases
- If the plan is outdated or no longer relevant, inform the user and suggest starting fresh
- If all steps are complete, set Status to Complete and congratulate the user
