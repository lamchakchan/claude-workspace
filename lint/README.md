# Template Linting

CUE-based schema validation for `_template/` configuration files. Catches invalid JSON, wrong enum values, missing required fields, and structural drift at lint time rather than runtime.

## Quick Start

```bash
# Install CUE
brew install cue-lang/tap/cue

# Run lint
make lint
```

## What Gets Validated

| Category | Files | Schema Definition |
|---|---|---|
| Settings (3) | project settings.json, global settings.json, settings.local.json.example | `#ProjectSettings`, `#GlobalSettings`, `#SettingsLocalExample` |
| MCP config (1) | project .mcp.json | `#McpConfig` |
| Agent frontmatter (9) | All agents in `_template/project/.claude/agents/` | `#AgentFrontmatter` |
| Skill frontmatter (5) | All skills in `_template/project/.claude/skills/` | `#SkillFrontmatter` |

## Schema Files

- `settings.cue` — Settings JSON validation (base + project/global/local variants)
- `mcp.cue` — MCP server configuration validation
- `agents.cue` — Agent YAML frontmatter validation
- `skills.cue` — Skill YAML frontmatter validation

## Vendored Schema

`schemas/claude-code-settings.schema.json` is the official JSON Schema from
[schemastore.org](https://json.schemastore.org/claude-code-settings.json),
vendored for provenance. The hand-written CUE schemas are the actual validation source.

To refresh:

```bash
curl -fsSL -o lint/schemas/claude-code-settings.schema.json \
  https://json.schemastore.org/claude-code-settings.json
```

## Adding New Validations

1. Add a CUE definition to the appropriate `.cue` file (or create a new one)
2. Add a `vet` or `vet_frontmatter` call in `scripts/lint-templates.sh`
3. Run `make lint` to verify

## CI

The lint step runs automatically in CI via `make lint`. CUE is installed using the `cue-lang/setup-cue` GitHub Action. Locally, if CUE is not installed, the lint step prints a warning and skips; in CI (`$CI=true`) it fails hard.
