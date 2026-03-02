package config

func buildSettingsKeys() []ConfigKey {
	keys := make([]ConfigKey, 0, 90)

	// --- Core ---
	keys = append(keys, //nolint:gocritic // appendCombine: organized by category for readability
		ConfigKey{
			Key: "model", Category: CatCore, Type: TypeString,
			Description: "Override the default model (alias like \"opus\" or full ID like \"claude-opus-4-6\")",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "availableModels", Category: CatCore, Type: TypeStringArray,
			Description: "Restrict which models users can select via /model or CLI",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "effortLevel", Category: CatCore, Type: TypeEnum,
			Default: "high", Description: "Opus 4.6 adaptive reasoning effort level",
			ValidScopes: allScopes, EnumValues: []string{"low", "medium", "high"},
		},
		ConfigKey{
			Key: "alwaysThinkingEnabled", Category: CatCore, Type: TypeBool,
			Default: "false", Description: "Enable extended thinking by default for all sessions",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "fastModePerSessionOptIn", Category: CatCore, Type: TypeBool,
			Description: "When true, fast mode does not persist across sessions",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "language", Category: CatCore, Type: TypeString,
			Description: "Claude's preferred response language (e.g., \"japanese\")",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "outputStyle", Category: CatCore, Type: TypeString,
			Description: "Output style hint passed to Claude (e.g., \"Explanatory\")",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "apiKeyHelper", Category: CatCore, Type: TypeString,
			Description: "Script path to generate auth values (X-Api-Key and Authorization: Bearer)",
			ValidScopes: userScopes,
		},
		ConfigKey{
			Key: "autoUpdatesChannel", Category: CatCore, Type: TypeEnum,
			Default: "latest", Description: "Auto-update release channel",
			ValidScopes: allScopes, EnumValues: []string{"stable", "latest"},
		},
		ConfigKey{
			Key: "cleanupPeriodDays", Category: CatCore, Type: TypeInt,
			Default: "30", Description: "Days to retain session transcripts (0 = delete immediately)",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "plansDirectory", Category: CatCore, Type: TypeString,
			Default: "~/.claude/plans", Description: "Directory where plan files are stored (relative to project root or absolute)",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "autoMemoryEnabled", Category: CatCore, Type: TypeBool,
			Default: "true", Description: "Enable automatic context saving to memory",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "claudeMdExcludes", Category: CatCore, Type: TypeStringArray,
			Description: "Glob patterns for CLAUDE.md files to exclude from loading",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "respectGitignore", Category: CatCore, Type: TypeBool,
			Default: "true", Description: "Whether the @ file picker respects .gitignore",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "env", Category: CatCore, Type: TypeObject,
			Description: "Environment variables applied to every session (e.g., {\"NODE_ENV\": \"development\"})",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "$schema", Category: CatCore, Type: TypeString,
			Default:     "https://json.schemastore.org/claude-code-settings.json",
			Description: "JSON Schema URL for IDE autocomplete support",
			ValidScopes: userScopes,
		},
		ConfigKey{
			Key: "skipWebFetchPreflight", Category: CatCore, Type: TypeBool,
			Default: "false", Description: "Skip WebFetch domain blocklist preflight check",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "companyAnnouncements", Category: CatCore, Type: TypeStringArray,
			Description: "Messages displayed at startup (one selected randomly each session)",
			ValidScopes: allScopes,
		},
	)

	// --- UI & Display ---
	keys = append(keys,
		ConfigKey{
			Key: "showTurnDuration", Category: CatUI, Type: TypeBool,
			Default: "true", Description: "Show turn duration after each Claude response",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "spinnerTipsEnabled", Category: CatUI, Type: TypeBool,
			Default: "true", Description: "Show tips in the spinner while Claude is working",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "spinnerVerbs", Category: CatUI, Type: TypeObject,
			Description: "Customize spinner verbs: {\"mode\": \"append\"|\"replace\", \"verbs\": [\"Pondering\"]}",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "spinnerTipsOverride", Category: CatUI, Type: TypeObject,
			Description: "Override spinner tips: {\"excludeDefault\": true, \"tips\": [\"Custom tip\"]}",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "terminalProgressBarEnabled", Category: CatUI, Type: TypeBool,
			Default: "true", Description: "Enable terminal progress bar during operations",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "prefersReducedMotion", Category: CatUI, Type: TypeBool,
			Default: "false", Description: "Reduce or disable UI animations for accessibility",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "statusLine", Category: CatUI, Type: TypeObject,
			Description: "Custom status line config: {\"type\": \"command\", \"command\": \"~/.claude/statusline.sh\", \"padding\": 0}",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "fileSuggestion", Category: CatUI, Type: TypeObject,
			Description: "Custom @ file autocomplete: {\"type\": \"command\", \"command\": \"~/.claude/file-suggestion.sh\"}",
			ValidScopes: allScopes,
		},
	)

	// --- Attribution ---
	keys = append(keys,
		ConfigKey{
			Key: "attribution.commit", Category: CatAttribution, Type: TypeString,
			Description: "Git commit attribution trailer text (empty string hides it)",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "attribution.pr", Category: CatAttribution, Type: TypeString,
			Description: "PR description attribution text (empty string hides it)",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "includeCoAuthoredBy", Category: CatAttribution, Type: TypeBool,
			Default: "true", Description: "DEPRECATED: Use attribution instead. Add Co-Authored-By trailer to commits",
			ValidScopes: allScopes,
		},
	)

	// --- Permissions ---
	keys = append(keys,
		ConfigKey{
			Key: "permissions.allow", Category: CatPermissions, Type: TypeStringArray,
			Description: "Tools/patterns allowed without prompting (e.g., \"Bash(npm run *)\", \"Read(~/.zshrc)\")",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "permissions.ask", Category: CatPermissions, Type: TypeStringArray,
			Description: "Tools/patterns that always require confirmation (e.g., \"Bash(git push *)\")",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "permissions.deny", Category: CatPermissions, Type: TypeStringArray,
			Description: "Tools/patterns blocked entirely (e.g., \"WebFetch\", \"Read(./.env)\")",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "permissions.additionalDirectories", Category: CatPermissions, Type: TypeStringArray,
			Description: "Additional working directories Claude is allowed to access",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "permissions.defaultMode", Category: CatPermissions, Type: TypeEnum,
			Default:     "default",
			Description: "Default permission mode for the session",
			ValidScopes: allScopes,
			EnumValues:  []string{"default", "acceptEdits", "plan", "dontAsk", "delegate", "bypassPermissions"},
		},
		ConfigKey{
			Key: "permissions.disableBypassPermissionsMode", Category: CatPermissions, Type: TypeEnum,
			Description: "Set to \"disable\" to prevent bypassPermissions mode from being activated",
			ValidScopes: allScopes, EnumValues: []string{"disable"},
		},
		ConfigKey{
			Key: "allowManagedPermissionRulesOnly", Category: CatPermissions, Type: TypeBool,
			Description: "Only managed permission rules apply; project/user rules are ignored",
			ValidScopes: managedOnly, ReadOnly: true,
		},
	)

	// --- Sandbox ---
	keys = append(keys,
		ConfigKey{
			Key: "sandbox.enabled", Category: CatSandbox, Type: TypeBool,
			Default: "false", Description: "Enable bash sandboxing (macOS Seatbelt / Linux bubblewrap)",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "sandbox.autoAllowBashIfSandboxed", Category: CatSandbox, Type: TypeBool,
			Default: "true", Description: "Auto-approve bash commands when sandbox is active",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "sandbox.excludedCommands", Category: CatSandbox, Type: TypeStringArray,
			Description: "Commands that run outside the sandbox even when sandbox is enabled",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "sandbox.allowUnsandboxedCommands", Category: CatSandbox, Type: TypeBool,
			Default: "true", Description: "Allow dangerouslyDisableSandbox parameter in tool calls",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "sandbox.enableWeakerNestedSandbox", Category: CatSandbox, Type: TypeBool,
			Default: "false", Description: "Weaker sandbox for unprivileged Docker (Linux/WSL2 only)",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "sandbox.filesystem.allowWrite", Category: CatSandbox, Type: TypeStringArray,
			Description: "Additional writable paths inside the sandbox (merged across scopes)",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "sandbox.filesystem.denyWrite", Category: CatSandbox, Type: TypeStringArray,
			Description: "Paths denied write access inside the sandbox (merged across scopes)",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "sandbox.filesystem.denyRead", Category: CatSandbox, Type: TypeStringArray,
			Description: "Paths denied read access inside the sandbox (merged across scopes)",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "sandbox.network.allowedDomains", Category: CatSandbox, Type: TypeStringArray,
			Description: "Allowed outbound domains (supports wildcards like \"*.npmjs.org\")",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "sandbox.network.allowUnixSockets", Category: CatSandbox, Type: TypeStringArray,
			Description: "Unix socket paths the sandbox may connect to",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "sandbox.network.allowAllUnixSockets", Category: CatSandbox, Type: TypeBool,
			Default: "false", Description: "Allow all Unix socket connections from the sandbox",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "sandbox.network.allowLocalBinding", Category: CatSandbox, Type: TypeBool,
			Default: "false", Description: "Allow binding to localhost ports (macOS only)",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "sandbox.network.httpProxyPort", Category: CatSandbox, Type: TypeInt,
			Description: "HTTP proxy port for sandboxed network access",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "sandbox.network.socksProxyPort", Category: CatSandbox, Type: TypeInt,
			Description: "SOCKS5 proxy port for sandboxed network access",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "sandbox.network.allowManagedDomainsOnly", Category: CatSandbox, Type: TypeBool,
			Description: "Only managed-policy allowed domains are respected",
			ValidScopes: managedOnly, ReadOnly: true,
		},
	)

	// --- Hooks ---
	keys = append(keys,
		ConfigKey{
			Key: "hooks", Category: CatHooks, Type: TypeObject,
			Description: "Custom lifecycle hook handlers (SessionStart, PreToolUse, PostToolUse, Stop, etc.)",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "disableAllHooks", Category: CatHooks, Type: TypeBool,
			Default: "false", Description: "Disable all hooks and custom status line",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "allowManagedHooksOnly", Category: CatHooks, Type: TypeBool,
			Description: "Only managed and SDK hooks are allowed; project/user hooks ignored",
			ValidScopes: managedOnly, ReadOnly: true,
		},
		ConfigKey{
			Key: "allowedHttpHookUrls", Category: CatHooks, Type: TypeStringArray,
			Description: "URL patterns that HTTP hooks may target (supports * wildcard)",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "httpHookAllowedEnvVars", Category: CatHooks, Type: TypeStringArray,
			Description: "Environment variable names that HTTP hooks are allowed to interpolate",
			ValidScopes: allScopes,
		},
	)

	// --- MCP Servers ---
	keys = append(keys,
		ConfigKey{
			Key: "enableAllProjectMcpServers", Category: CatMCP, Type: TypeBool,
			Default: "false", Description: "Auto-approve all MCP servers from the project .mcp.json",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "enabledMcpjsonServers", Category: CatMCP, Type: TypeStringArray,
			Description: "Specific MCP servers from .mcp.json to approve automatically",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "disabledMcpjsonServers", Category: CatMCP, Type: TypeStringArray,
			Description: "Specific MCP servers from .mcp.json to reject automatically",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "allowedMcpServers", Category: CatMCP, Type: TypeStringArray,
			Description: "Allowlist of MCP servers (undefined = no restriction, [] = lockdown)",
			ValidScopes: managedOnly, ReadOnly: true,
		},
		ConfigKey{
			Key: "deniedMcpServers", Category: CatMCP, Type: TypeStringArray,
			Description: "Denylist of MCP servers (takes precedence over allowlist)",
			ValidScopes: managedOnly, ReadOnly: true,
		},
		ConfigKey{
			Key: "allowManagedMcpServersOnly", Category: CatMCP, Type: TypeBool,
			Description: "Only managed MCP server policies are respected",
			ValidScopes: managedOnly, ReadOnly: true,
		},
	)

	// --- Plugins ---
	keys = append(keys,
		ConfigKey{
			Key: "enabledPlugins", Category: CatPlugins, Type: TypeObject,
			Description: "Control which plugins are enabled (\"plugin-name@marketplace\": true/false)",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "extraKnownMarketplaces", Category: CatPlugins, Type: TypeObject,
			Description: "Additional plugin marketplaces for the repository",
			ValidScopes: allScopes,
		},
		ConfigKey{
			Key: "strictKnownMarketplaces", Category: CatPlugins, Type: TypeStringArray,
			Description: "Marketplace allowlist; only listed marketplaces are permitted",
			ValidScopes: managedOnly, ReadOnly: true,
		},
		ConfigKey{
			Key: "blockedMarketplaces", Category: CatPlugins, Type: TypeStringArray,
			Description: "Marketplace denylist; plugins from these sources are blocked",
			ValidScopes: managedOnly, ReadOnly: true,
		},
		ConfigKey{
			Key: "skippedPlugins", Category: CatPlugins, Type: TypeStringArray,
			Description: "User-skipped plugins (not auto-enabled at session start)",
			ValidScopes: userScopes,
		},
		ConfigKey{
			Key: "pluginConfigs", Category: CatPlugins, Type: TypeObject,
			Description: "Per-plugin configuration keyed by plugin ID",
			ValidScopes: allScopes,
		},
	)

	// --- Telemetry ---
	keys = append(keys,
		ConfigKey{
			Key: "otelHeadersHelper", Category: CatTelemetry, Type: TypeString,
			Description: "Script path to generate dynamic OpenTelemetry export headers",
			ValidScopes: allScopes,
		},
	)

	// --- Cloud Providers (settings.json) ---
	keys = append(keys,
		ConfigKey{
			Key: "awsAuthRefresh", Category: CatCloud, Type: TypeString,
			Description: "Script to refresh AWS auth (modifies .aws directory)",
			ValidScopes: userScopes,
		},
		ConfigKey{
			Key: "awsCredentialExport", Category: CatCloud, Type: TypeString,
			Description: "Script outputting JSON with AWS credentials for Bedrock auth",
			ValidScopes: userScopes,
		},
	)

	// --- Login & Auth ---
	keys = append(keys,
		ConfigKey{
			Key: "forceLoginMethod", Category: CatCore, Type: TypeEnum,
			Description: "Restrict login to a specific method",
			ValidScopes: managedOnly, ReadOnly: true,
			EnumValues: []string{"claudeai", "console"},
		},
		ConfigKey{
			Key: "forceLoginOrgUUID", Category: CatCore, Type: TypeString,
			Description: "Auto-select organization UUID during login",
			ValidScopes: managedOnly, ReadOnly: true,
		},
	)

	// --- Agent Teams ---
	keys = append(keys,
		ConfigKey{
			Key: "teammateMode", Category: CatAgentTeams, Type: TypeEnum,
			Default: "auto", Description: "How agent teammates are displayed during team execution",
			ValidScopes: allScopes, EnumValues: []string{"auto", "in-process", "tmux"},
		},
	)

	return keys
}
