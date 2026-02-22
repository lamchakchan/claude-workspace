---
name: pr-workflow
description: Guides the full pull request workflow from branch creation through PR submission. Use when creating PRs, preparing changes for review, or managing the git workflow.
---

# Pull Request Workflow

## Creating a PR

### Step 1: Prepare the Branch
```bash
# Ensure you're on a feature branch, never main
git checkout -b feature/<descriptive-name>

# Or if branch exists
git checkout feature/<descriptive-name>
```

### Step 2: Review Changes
Before creating the PR, always review what will be included:
- Run `git status` to see all changes
- Run `git diff` to review the actual changes
- Use the code-reviewer subagent for automated review

### Step 3: Stage and Commit
- Stage specific files (avoid `git add -A`)
- Write clear commit messages in imperative mood
- Reference issue numbers if applicable

### Step 4: Create the PR
Use `gh pr create` with:
- **Title**: Short (under 70 characters), descriptive
- **Body**: Include Summary, Changes, Test Plan sections
- **Labels**: Add appropriate labels if available

### PR Template

```markdown
## Summary
[1-3 bullet points explaining what and why]

## Changes
- [List of significant changes]

## Test Plan
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual verification of [behavior]

## Screenshots
[If UI changes, include before/after]
```

## Reviewing a PR

1. Read the PR description and linked issues
2. Check out the branch locally
3. Use the code-reviewer subagent for automated review
4. Run the test suite
5. Provide structured feedback (Critical / Warning / Suggestion)

## Rules

- Never push directly to main/master
- Always include a test plan
- Keep PRs focused - one logical change per PR
- Reference related issues
