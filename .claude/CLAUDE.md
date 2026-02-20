# Project-Level Instructions

This file contains team-shared instructions loaded for every Claude Code session in this project. Customize this for your specific project.

## Project Context

<!-- Describe your project here -->
Project: Claude Code Platform Engineering Kit
Purpose: Preconfigured AI agent platform for teams adopting Claude Code
Tech Stack: Shell scripts, YAML, JSON, Markdown

## Team Conventions

### Code Style
- Shell scripts: Use `set -euo pipefail`, quote variables, use shellcheck
- JSON: 2-space indentation, no trailing commas
- Markdown: ATX headings, fenced code blocks with language tags

### Testing
- All hooks must be tested before deployment
- Scripts should handle edge cases gracefully
- Validate with `shellcheck` for shell scripts

### Documentation
- Every new feature needs a corresponding docs update
- Configuration changes must be reflected in README.md
- Use inline comments for non-obvious logic

## Directory Layout

```
.claude/agents/   - Custom subagent definitions (Markdown + YAML frontmatter)
.claude/skills/   - Reusable skill definitions
.claude/hooks/    - Safety and quality gate scripts
scripts/          - Setup and management scripts
templates/        - Templates for project adaptation
docs/             - Detailed documentation
```

## Important Files

- `README.md` - Main documentation and quick start guide
- `.claude/settings.json` - Team settings with safe defaults
- `.mcp.json` - MCP server configurations
- `scripts/setup.sh` - First-time setup script
