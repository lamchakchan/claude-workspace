---
name: onboarding
description: Helps onboard Claude Code to a new project. Use when attaching the platform to a new repository or when first exploring an unfamiliar codebase.
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
- Find and read the package manifest (package.json, Cargo.toml, go.mod, pyproject.toml, requirements.txt, pom.xml, build.gradle, build.gradle.kts, Gemfile, *.csproj, *.sln, mix.exs, composer.json, Package.swift, build.sbt, MODULE.bazel, CMakeLists.txt)
- Identify the build command
- Identify the test command
- Identify linting/formatting tools

### 3. Architecture
- Map the top-level directory structure
- Identify the architectural pattern (MVC, hexagonal, microservices, etc.)
- Find the entry point(s)
- Understand the module/package organization

### 4. Key Files
- Configuration files (webpack, vite, tsconfig, pom.xml, build.gradle, Rakefile, Guardfile, phpunit.xml, .swiftlint.yml, .scalafmt.conf, .clang-format, BUILD, WORKSPACE, etc.)
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
- `mcp-memory-libsql` or similar memory server → cross-project persistent memory

### 7. Initialize Persistent Memory (if a memory MCP server is available)

Check for memory tools in your available tool list (e.g., tools matching
`mcp__mcp-memory-libsql__*`). If found:
1. Call `read_graph` or `search_nodes` for the project name — skip if entities already exist
2. If not found, call `create_entities` with an entity for the project summarizing:
   tech stack, purpose, build command, test command, key directories

This creates a persistent cross-project record that survives context compaction and new sessions.

### 8. Generate CLAUDE.md
Based on your findings, create or update the project's `.claude/CLAUDE.md` with:
- Project description and tech stack
- Build, test, and lint commands
- Key directories and their purposes
- Coding conventions and patterns
- Important files to know about
- **MCP Tool Preferences** section: list capability categories covered by detected MCP servers, using the same capability-based format (not specific tool names) so the entry survives MCP provider changes
- **Team Execution** section: if team agents (`.claude/agents/team-lead.md`) or hooks (`.claude/hooks/verify-task-completed.sh`, `.claude/hooks/check-teammate-idle.sh`) are detected, document available execution modes (sequential, solo team, multi-agent team), key tools (`TeamCreate`, `TaskCreate`/`TaskUpdate`/`TaskList`, `Agent` with `team_name`, `SendMessage`, `TeamDelete`), and configured hooks

## Output

Provide a structured onboarding report and suggest a CLAUDE.md content for the project.
