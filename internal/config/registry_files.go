package config

// buildFileKeys returns ConfigKey entries describing file-based Claude Code
// configuration artifacts (CLAUDE.md layers, MCP servers, agents, skills, etc.).
// These are not key=value pairs but rather presence/count summaries of files
// that influence Claude's behavior.
func buildFileKeys() []ConfigKey {
	keys := make([]ConfigKey, 0, 20)
	fs := []ConfigScope{ScopeEnv} // reused for file-based entries

	// CLAUDE.md instruction layers
	keys = append(keys,
		ConfigKey{
			Key: "file:claudemd.user", Category: CatFiles, Type: TypeString,
			Description: "User-global instructions loaded for all projects (~/.claude/CLAUDE.md)",
			ValidScopes: fs,
		},
		ConfigKey{
			Key: "file:claudemd.project", Category: CatFiles, Type: TypeString,
			Description: "Project-level instructions loaded when Claude is in this project (CLAUDE.md or .claude/CLAUDE.md)",
			ValidScopes: fs,
		},
		ConfigKey{
			Key: "file:claudemd.local", Category: CatFiles, Type: TypeString,
			Description: "Local project instructions not committed to git (CLAUDE.local.md)",
			ValidScopes: fs,
		},

		// MCP servers
		ConfigKey{
			Key: "file:mcp.project", Category: CatFiles, Type: TypeString,
			Description: "Project MCP server definitions (.mcp.json) — committed to git",
			ValidScopes: fs,
		},
		ConfigKey{
			Key: "file:mcp.user", Category: CatFiles, Type: TypeString,
			Description: "User-global MCP server definitions (~/.claude.json mcpServers)",
			ValidScopes: fs,
		},
		ConfigKey{
			Key: "file:mcp.managed", Category: CatFiles, Type: TypeString,
			Description: "Enterprise-managed MCP servers (managed-mcp.json)",
			ValidScopes: fs,
		},

		// Keybindings
		ConfigKey{
			Key: "file:keybindings", Category: CatFiles, Type: TypeString,
			Description: "Custom keyboard shortcut bindings (~/.claude/keybindings.json)",
			ValidScopes: fs,
		},

		// Agents
		ConfigKey{
			Key: "file:agents.project", Category: CatFiles, Type: TypeString,
			Description: "Project subagent definitions (.claude/agents/*.md)",
			ValidScopes: fs,
		},
		ConfigKey{
			Key: "file:agents.user", Category: CatFiles, Type: TypeString,
			Description: "User-global subagent definitions (~/.claude/agents/*.md)",
			ValidScopes: fs,
		},

		// Skills
		ConfigKey{
			Key: "file:skills.project", Category: CatFiles, Type: TypeString,
			Description: "Project skill definitions (.claude/skills/*/SKILL.md or *.md)",
			ValidScopes: fs,
		},
		ConfigKey{
			Key: "file:skills.user", Category: CatFiles, Type: TypeString,
			Description: "User-global skill definitions (~/.claude/skills/*/SKILL.md or *.md)",
			ValidScopes: fs,
		},

		// Hooks
		ConfigKey{
			Key: "file:hooks.project", Category: CatFiles, Type: TypeString,
			Description: "Project hook scripts (.claude/hooks/*.sh)",
			ValidScopes: fs,
		},
		ConfigKey{
			Key: "file:hooks.user", Category: CatFiles, Type: TypeString,
			Description: "User-global hook scripts (~/.claude/hooks/*.sh)",
			ValidScopes: fs,
		},

		// Rules
		ConfigKey{
			Key: "file:rules.project", Category: CatFiles, Type: TypeString,
			Description: "Modular instruction files (.claude/rules/*.md)",
			ValidScopes: fs,
		},
		ConfigKey{
			Key: "file:rules.user", Category: CatFiles, Type: TypeString,
			Description: "User-global modular instruction files (~/.claude/rules/*.md)",
			ValidScopes: fs,
		},

		// Settings files (presence check)
		ConfigKey{
			Key: "file:settings.managed", Category: CatFiles, Type: TypeString,
			Description: "Enterprise-managed settings file (managed-settings.json)",
			ValidScopes: fs,
		},
		ConfigKey{
			Key: "file:settings.user", Category: CatFiles, Type: TypeString,
			Description: "User settings file (~/.claude/settings.json)",
			ValidScopes: fs,
		},
		ConfigKey{
			Key: "file:settings.project", Category: CatFiles, Type: TypeString,
			Description: "Project settings file (.claude/settings.json) — committed to git",
			ValidScopes: fs,
		},
		ConfigKey{
			Key: "file:settings.local", Category: CatFiles, Type: TypeString,
			Description: "Local settings overrides (.claude/settings.local.json) — not committed",
			ValidScopes: fs,
		},

		// Auth / main config
		ConfigKey{
			Key: "file:claude.json", Category: CatFiles, Type: TypeString,
			Description: "Main Claude config file (~/.claude.json) — OAuth tokens, MCP servers, preferences",
			ValidScopes: fs,
		},
	)

	return keys
}
