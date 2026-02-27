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
   - Name: `plan-YYYY-MM-DD-<description>.md`
   - Header: immediately after the title line, include:
     ```
     Date: YYYY-MM-DD
     Status: Draft
     Last Updated: YYYY-MM-DD
     ```
   - Include: steps, affected files, risks, test strategy
   - Include Mermaid diagrams for complex subjects (architecture, state machines, data flows, sequences) — the planner agent has detailed guidance on when and how to add them
4. **Present the plan** - Show the user what you'll do and ask for approval
5. **Create a todo list** - Use TodoWrite to create trackable items from the plan
6. **Name the session** - Suggest `/rename <plan-description>` so the session is easy to find with `claude --resume`
7. **Log the plan path** - Tell the user: "Plan saved to `./plans/<filename>`. You can resume this in a future session with `/plan-resume`"

## Phase 2: Execution

1. **Work step by step** - Follow the plan in order
2. **Update progress** - As each step completes:
   - Check it off in the plan file (`- [ ]` → `- [x]`)
   - Mark the corresponding todo as completed
3. **Validate each step** - Run tests or verify after each change
4. **Handle deviations** - If the plan needs to change, update it and inform the user

## Phase 3: Verification

1. **Run tests** - Use the test-runner subagent
2. **Review changes** - Use the code-reviewer subagent
3. **Summarize** - Report what was done and any remaining items
4. **Update plan status** - Set the plan file's `Status:` to `Complete` and `Last Updated:` to today's date

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
