---
name: onboarding
description: Helps onboard Claude Code to a new project. Use when attaching the platform to a new repository or when first exploring an unfamiliar codebase.
context: fork
---

# Project Onboarding

You are onboarding to a new project. Your goal is to understand the project structure, conventions, and key files so you can work effectively.

## Onboarding Steps

### 1. Project Identity
- Read README.md or equivalent documentation
- Identify the project type (web app, API, library, CLI, etc.)
- Identify the primary programming language(s)
- Identify the framework(s) in use

### 2. Build System
- Find and read the package manifest (package.json, Cargo.toml, go.mod, pyproject.toml, etc.)
- Identify the build command
- Identify the test command
- Identify linting/formatting tools

### 3. Architecture
- Map the top-level directory structure
- Identify the architectural pattern (MVC, hexagonal, microservices, etc.)
- Find the entry point(s)
- Understand the module/package organization

### 4. Key Files
- Configuration files (webpack, vite, tsconfig, etc.)
- Environment files (.env.example)
- CI/CD configuration (.github/workflows, Jenkinsfile, etc.)
- Database migrations or schemas

### 5. Conventions
- Code style (linting config, prettier, etc.)
- Git conventions (branch naming, commit format)
- Testing patterns and frameworks
- Documentation standards

### 6. Detect Installed MCP Servers

Run `claude mcp list` to identify which capability categories are covered by installed MCP servers. Map each detected server to a capability:

- Search providers (brave, perplexity, tavily, exa) → web search
- `filesystem` server → filesystem operations
- GitHub/GitLab/Bitbucket servers → version control
- Observability servers (honeycomb, datadog, grafana, newrelic, sentry) → traces/logs/metrics

### 7. Generate CLAUDE.md
Based on your findings, create or update the project's `.claude/CLAUDE.md` with:
- Project description and tech stack
- Build, test, and lint commands
- Key directories and their purposes
- Coding conventions and patterns
- Important files to know about
- **MCP Tool Preferences** section: list capability categories covered by detected MCP servers, using the same capability-based format (not specific tool names) so the entry survives MCP provider changes

## Output

Provide a structured onboarding report and suggest a CLAUDE.md content for the project.
