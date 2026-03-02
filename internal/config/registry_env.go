package config

func buildEnvKeys() []ConfigKey {
	keys := make([]ConfigKey, 0, 110)
	ev := []ConfigScope{ScopeEnv}

	// --- Model & Tokens ---
	keys = append(keys, //nolint:gocritic // appendCombine: organized by category for readability
		ConfigKey{
			Key: "ANTHROPIC_MODEL", Category: CatEnvModel, Type: TypeString,
			Description: "Override the default Claude model (alias or full model ID)",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "ANTHROPIC_DEFAULT_SONNET_MODEL", Category: CatEnvModel, Type: TypeString,
			Description: "Model used for the \"sonnet\" alias and opusplan execution phase",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "ANTHROPIC_DEFAULT_OPUS_MODEL", Category: CatEnvModel, Type: TypeString,
			Description: "Model used for the \"opus\" alias and opusplan planning phase",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "ANTHROPIC_DEFAULT_HAIKU_MODEL", Category: CatEnvModel, Type: TypeString,
			Description: "Model used for the \"haiku\" alias and background subagent tasks",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_SUBAGENT_MODEL", Category: CatEnvModel, Type: TypeString,
			Description: "Model to use for spawned subagents",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_EFFORT_LEVEL", Category: CatEnvModel, Type: TypeEnum,
			Default: "high", Description: "Adaptive reasoning effort level for Opus 4.6 / Sonnet 4.6",
			ValidScopes: ev, EnumValues: []string{"low", "medium", "high"},
		},
		ConfigKey{
			Key: "CLAUDE_CODE_MAX_OUTPUT_TOKENS", Category: CatEnvModel, Type: TypeInt,
			Default: "32000", Description: "Maximum output tokens per response (max: 64000)",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_FILE_READ_MAX_OUTPUT_TOKENS", Category: CatEnvModel, Type: TypeInt,
			Description: "Override token limit specifically for file Read tool operations",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "MAX_THINKING_TOKENS", Category: CatEnvModel, Type: TypeInt,
			Default: "31999", Description: "Maximum tokens for extended thinking budget (reduce to free output tokens)",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_DISABLE_ADAPTIVE_THINKING", Category: CatEnvModel, Type: TypeBool,
			Description: "Disable adaptive reasoning; revert to fixed MAX_THINKING_TOKENS budget",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_DISABLE_THINKING", Category: CatEnvModel, Type: TypeBool,
			Description: "Completely disable extended thinking",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_DISABLE_1M_CONTEXT", Category: CatEnvModel, Type: TypeBool,
			Description: "Disable 1M context window support",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "ANTHROPIC_SMALL_FAST_MODEL", Category: CatEnvModel, Type: TypeString,
			Description: "DEPRECATED: use ANTHROPIC_DEFAULT_HAIKU_MODEL instead",
			ValidScopes: ev,
		},
	)

	// --- Authentication ---
	keys = append(keys,
		ConfigKey{
			Key: "ANTHROPIC_API_KEY", Category: CatEnvAuth, Type: TypeString,
			Description: "Anthropic API key (unset to use claude.ai subscription instead)",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "ANTHROPIC_AUTH_TOKEN", Category: CatEnvAuth, Type: TypeString,
			Description: "Custom Authorization: Bearer header value",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "ANTHROPIC_BASE_URL", Category: CatEnvAuth, Type: TypeString,
			Description: "Custom API base URL (for proxies or enterprise endpoints)",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "ANTHROPIC_CUSTOM_HEADERS", Category: CatEnvAuth, Type: TypeString,
			Description: "Custom request headers (newline-separated \"Name: Value\" pairs)",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "ANTHROPIC_BETAS", Category: CatEnvAuth, Type: TypeString,
			Description: "Comma-separated Anthropic API beta feature headers to enable",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "API_TIMEOUT_MS", Category: CatEnvAuth, Type: TypeInt,
			Default: "600000", Description: "API request timeout in milliseconds",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_API_KEY_HELPER_TTL_MS", Category: CatEnvAuth, Type: TypeInt,
			Description: "Credential refresh interval for apiKeyHelper script (ms)",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_MAX_RETRIES", Category: CatEnvAuth, Type: TypeInt,
			Description: "Maximum number of API request retries on failure",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_OAUTH_TOKEN", Category: CatEnvAuth, Type: TypeString,
			Description: "Pre-configured OAuth access token",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_OAUTH_REFRESH_TOKEN", Category: CatEnvAuth, Type: TypeString,
			Description: "OAuth refresh token for token renewal",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_CLIENT_CERT", Category: CatEnvAuth, Type: TypeString,
			Description: "Path to client certificate for mutual TLS (mTLS)",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_CLIENT_KEY", Category: CatEnvAuth, Type: TypeString,
			Description: "Path to client private key for mTLS",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_CLIENT_KEY_PASSPHRASE", Category: CatEnvAuth, Type: TypeString,
			Description: "Passphrase for encrypted client private key",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "NODE_EXTRA_CA_CERTS", Category: CatEnvAuth, Type: TypeString,
			Description: "Path to additional CA certificates for TLS verification",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "HTTP_PROXY", Category: CatEnvAuth, Type: TypeString,
			Description: "HTTP proxy server URL",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "HTTPS_PROXY", Category: CatEnvAuth, Type: TypeString,
			Description: "HTTPS proxy server URL",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "NO_PROXY", Category: CatEnvAuth, Type: TypeString,
			Description: "Comma-separated domains that bypass the proxy",
			ValidScopes: ev,
		},
	)

	// --- Cloud Providers (Bedrock, Vertex, Foundry) ---
	keys = append(keys,
		// Bedrock
		ConfigKey{
			Key: "CLAUDE_CODE_USE_BEDROCK", Category: CatEnvCloud, Type: TypeBool,
			Description: "Route all API calls through AWS Bedrock",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_SKIP_BEDROCK_AUTH", Category: CatEnvCloud, Type: TypeBool,
			Description: "Skip AWS authentication for Bedrock (useful for testing)",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "BEDROCK_BASE_URL", Category: CatEnvCloud, Type: TypeString,
			Description: "Custom AWS Bedrock endpoint URL",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "AWS_ACCESS_KEY_ID", Category: CatEnvCloud, Type: TypeString,
			Description: "AWS access key ID for Bedrock authentication",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "AWS_SECRET_ACCESS_KEY", Category: CatEnvCloud, Type: TypeString,
			Description: "AWS secret access key for Bedrock authentication",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "AWS_SESSION_TOKEN", Category: CatEnvCloud, Type: TypeString,
			Description: "AWS session token for temporary credentials",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "AWS_REGION", Category: CatEnvCloud, Type: TypeString,
			Description: "AWS region for Bedrock API calls",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "AWS_PROFILE", Category: CatEnvCloud, Type: TypeString,
			Description: "AWS credentials profile name",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "AWS_BEARER_TOKEN_BEDROCK", Category: CatEnvCloud, Type: TypeString,
			Description: "Bearer token for Bedrock authentication",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "ENABLE_PROMPT_CACHING_1H_BEDROCK", Category: CatEnvCloud, Type: TypeBool,
			Description: "Enable 1-hour prompt caching on AWS Bedrock",
			ValidScopes: ev,
		},
		// Vertex
		ConfigKey{
			Key: "CLAUDE_CODE_USE_VERTEX", Category: CatEnvCloud, Type: TypeBool,
			Description: "Route all API calls through Google Cloud Vertex AI",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_SKIP_VERTEX_AUTH", Category: CatEnvCloud, Type: TypeBool,
			Description: "Skip GCP authentication for Vertex AI",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "ANTHROPIC_VERTEX_PROJECT_ID", Category: CatEnvCloud, Type: TypeString,
			Description: "Google Cloud project ID for Vertex AI",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLOUD_ML_REGION", Category: CatEnvCloud, Type: TypeString,
			Description: "Google Cloud ML region for Vertex AI",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "GOOGLE_APPLICATION_CREDENTIALS", Category: CatEnvCloud, Type: TypeString,
			Description: "Path to GCP service account JSON file",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "VERTEX_BASE_URL", Category: CatEnvCloud, Type: TypeString,
			Description: "Custom Vertex AI endpoint URL",
			ValidScopes: ev,
		},
		// Foundry (Microsoft Azure)
		ConfigKey{
			Key: "CLAUDE_CODE_USE_FOUNDRY", Category: CatEnvCloud, Type: TypeBool,
			Description: "Route all API calls through Microsoft Azure Foundry",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_SKIP_FOUNDRY_AUTH", Category: CatEnvCloud, Type: TypeBool,
			Description: "Skip Azure authentication for Foundry",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "ANTHROPIC_FOUNDRY_API_KEY", Category: CatEnvCloud, Type: TypeString,
			Description: "API key for Microsoft Foundry",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "ANTHROPIC_FOUNDRY_BASE_URL", Category: CatEnvCloud, Type: TypeString,
			Description: "Full base URL for Microsoft Foundry endpoint",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "ANTHROPIC_FOUNDRY_RESOURCE", Category: CatEnvCloud, Type: TypeString,
			Description: "Azure Foundry resource name",
			ValidScopes: ev,
		},
	)

	// --- Shell & Bash ---
	keys = append(keys,
		ConfigKey{
			Key: "BASH_DEFAULT_TIMEOUT_MS", Category: CatEnvBash, Type: TypeInt,
			Description: "Default timeout for bash command execution (ms)",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "BASH_MAX_TIMEOUT_MS", Category: CatEnvBash, Type: TypeInt,
			Description: "Maximum timeout the model can set for bash commands (ms)",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "BASH_MAX_OUTPUT_LENGTH", Category: CatEnvBash, Type: TypeInt,
			Default: "30000", Description: "Maximum characters in bash output before middle-truncation",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_SHELL", Category: CatEnvBash, Type: TypeString,
			Description: "Override shell detection (e.g., \"/bin/bash\")",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_SHELL_PREFIX", Category: CatEnvBash, Type: TypeString,
			Description: "Command prefix prepended to all bash executions (for logging/auditing)",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_BASH_MAINTAIN_PROJECT_WORKING_DIR", Category: CatEnvBash, Type: TypeBool,
			Description: "Return to original working directory after each Bash command",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_DONT_INHERIT_ENV", Category: CatEnvBash, Type: TypeBool,
			Description: "Start shell with a clean environment (no inherited env vars)",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_GLOB_TIMEOUT_SECONDS", Category: CatEnvBash, Type: TypeInt,
			Description: "Timeout in seconds for Glob tool operations",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_GLOB_HIDDEN", Category: CatEnvBash, Type: TypeBool,
			Description: "Include hidden files (dotfiles) in Glob tool results",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_GLOB_NO_IGNORE", Category: CatEnvBash, Type: TypeBool,
			Description: "Ignore .gitignore rules in Glob tool results",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_MAX_TOOL_USE_CONCURRENCY", Category: CatEnvBash, Type: TypeInt,
			Description: "Maximum number of tool calls that can execute concurrently",
			ValidScopes: ev,
		},
	)

	// --- Feature Flags ---
	keys = append(keys,
		ConfigKey{
			Key: "CLAUDE_CODE_SIMPLE", Category: CatEnvFeatures, Type: TypeBool,
			Description: "Minimal mode: strips system prompt, disables MCP, hooks, and CLAUDE.md",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_DISABLE_FAST_MODE", Category: CatEnvFeatures, Type: TypeBool,
			Description: "Disable fast mode globally",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_DISABLE_AUTO_MEMORY", Category: CatEnvFeatures, Type: TypeBool,
			Description: "Disable automatic context saving to memory (1=disable, 0=force on)",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_DISABLE_BACKGROUND_TASKS", Category: CatEnvFeatures, Type: TypeBool,
			Description: "Disable all background task functionality",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_ENABLE_TASKS", Category: CatEnvFeatures, Type: TypeBool,
			Default: "true", Description: "Enable the task tracking system",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_ENABLE_PROMPT_SUGGESTION", Category: CatEnvFeatures, Type: TypeBool,
			Description: "Enable or disable prompt suggestions",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS", Category: CatEnvFeatures, Type: TypeBool,
			Description: "Enable the agent teams feature",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "ENABLE_TOOL_SEARCH", Category: CatEnvFeatures, Type: TypeString,
			Description: "Control MCP tool search: \"auto\", \"auto:N\", \"true\", or \"false\"",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "ENABLE_CLAUDEAI_MCP_SERVERS", Category: CatEnvFeatures, Type: TypeBool,
			Default: "true", Description: "Enable claude.ai MCP servers for logged-in users",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "ENABLE_LSP_TOOL", Category: CatEnvFeatures, Type: TypeBool,
			Description: "Enable the Language Server Protocol tool",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "ENABLE_SESSION_BACKGROUNDING", Category: CatEnvFeatures, Type: TypeBool,
			Description: "Enable session backgrounding (Ctrl+B)",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_DISABLE_CLAUDE_MDS", Category: CatEnvFeatures, Type: TypeBool,
			Description: "Completely disable loading of all CLAUDE.md files",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_DISABLE_TERMINAL_TITLE", Category: CatEnvFeatures, Type: TypeBool,
			Description: "Disable automatic terminal title updates",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_DISABLE_ATTACHMENTS", Category: CatEnvFeatures, Type: TypeBool,
			Description: "Disable file and image attachment support",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_DISABLE_FILE_CHECKPOINTING", Category: CatEnvFeatures, Type: TypeBool,
			Description: "Disable git-based file checkpointing",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_DISABLE_COMMAND_INJECTION_CHECK", Category: CatEnvFeatures, Type: TypeBool,
			Description: "Disable command injection safety check",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC", Category: CatEnvFeatures, Type: TypeBool,
			Description: "Disable autoupdater, /bug command, error reporting, and telemetry",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", Category: CatEnvFeatures, Type: TypeBool,
			Description: "Disable Anthropic API beta feature headers",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_DISABLE_FEEDBACK_SURVEY", Category: CatEnvFeatures, Type: TypeBool,
			Description: "Disable session quality surveys",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_SKIP_PROMPT_HISTORY", Category: CatEnvFeatures, Type: TypeBool,
			Description: "Disable saving prompt history to disk",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_SYNTAX_HIGHLIGHT", Category: CatEnvFeatures, Type: TypeBool,
			Description: "Enable or disable syntax highlighting in output",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_ACCESSIBILITY", Category: CatEnvFeatures, Type: TypeBool,
			Description: "Enable accessibility mode for screen readers",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "DISABLE_INTERLEAVED_THINKING", Category: CatEnvFeatures, Type: TypeBool,
			Description: "Disable interleaved thinking beta feature",
			ValidScopes: ev,
		},
	)

	// --- Context & Compaction ---
	keys = append(keys,
		ConfigKey{
			Key: "CLAUDE_AUTOCOMPACT_PCT_OVERRIDE", Category: CatEnvContext, Type: TypeInt,
			Default: "95", Description: "Percentage of context (1-100) that triggers auto-compaction",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "DISABLE_COMPACT", Category: CatEnvContext, Type: TypeBool,
			Description: "Disable all context compaction",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "DISABLE_AUTO_COMPACT", Category: CatEnvContext, Type: TypeBool,
			Description: "Disable only automatic context compaction (manual still works)",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "DISABLE_MICROCOMPACT", Category: CatEnvContext, Type: TypeBool,
			Description: "Disable microcompact optimization that runs between tool calls",
			ValidScopes: ev,
		},
	)

	// --- MCP ---
	keys = append(keys,
		ConfigKey{
			Key: "MCP_TIMEOUT", Category: CatEnvMCP, Type: TypeInt,
			Default: "30000", Description: "MCP server connection timeout (ms)",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "MCP_TOOL_TIMEOUT", Category: CatEnvMCP, Type: TypeInt,
			Description: "Individual MCP tool execution timeout (ms)",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "MAX_MCP_OUTPUT_TOKENS", Category: CatEnvMCP, Type: TypeInt,
			Default: "25000", Description: "Maximum tokens allowed in MCP tool output",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "MCP_SERVER_CONNECTION_BATCH_SIZE", Category: CatEnvMCP, Type: TypeInt,
			Default: "3", Description: "Number of MCP servers to connect concurrently",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "MCP_CONNECTION_NONBLOCKING", Category: CatEnvMCP, Type: TypeBool,
			Description: "Make MCP server connections non-blocking at startup",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "SLASH_COMMAND_TOOL_CHAR_BUDGET", Category: CatEnvMCP, Type: TypeInt,
			Description: "Character budget for slash command tool output",
			ValidScopes: ev,
		},
	)

	// --- Updates & Commands ---
	keys = append(keys,
		ConfigKey{
			Key: "DISABLE_AUTOUPDATER", Category: CatEnvUpdates, Type: TypeBool,
			Description: "Disable the automatic update checker",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "DISABLE_BUG_COMMAND", Category: CatEnvUpdates, Type: TypeBool,
			Description: "Disable the /bug command",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "DISABLE_COST_WARNINGS", Category: CatEnvUpdates, Type: TypeBool,
			Description: "Disable cost warning messages",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "DISABLE_ERROR_REPORTING", Category: CatEnvUpdates, Type: TypeBool,
			Description: "Opt out of Sentry error reporting",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "DISABLE_TELEMETRY", Category: CatEnvUpdates, Type: TypeBool,
			Description: "Opt out of Statsig telemetry",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_ENABLE_TELEMETRY", Category: CatEnvUpdates, Type: TypeBool,
			Description: "Enable OpenTelemetry data collection",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "DISABLE_INSTALLATION_CHECKS", Category: CatEnvUpdates, Type: TypeBool,
			Description: "Disable installation dependency warnings",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "DISABLE_NON_ESSENTIAL_MODEL_CALLS", Category: CatEnvUpdates, Type: TypeBool,
			Description: "Disable model calls used for flavor text and non-essential content",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "DISABLE_DOCTOR_COMMAND", Category: CatEnvUpdates, Type: TypeBool,
			Description: "Disable the /doctor command",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "DISABLE_LOGIN_COMMAND", Category: CatEnvUpdates, Type: TypeBool,
			Description: "Disable the /login command",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "DISABLE_LOGOUT_COMMAND", Category: CatEnvUpdates, Type: TypeBool,
			Description: "Disable the /logout command",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "DISABLE_UPGRADE_COMMAND", Category: CatEnvUpdates, Type: TypeBool,
			Description: "Disable the /upgrade command",
			ValidScopes: ev,
		},
	)

	// --- Prompt Caching ---
	keys = append(keys,
		ConfigKey{
			Key: "DISABLE_PROMPT_CACHING", Category: CatEnvCaching, Type: TypeBool,
			Description: "Disable prompt caching globally (takes precedence over model-specific flags)",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "DISABLE_PROMPT_CACHING_HAIKU", Category: CatEnvCaching, Type: TypeBool,
			Description: "Disable prompt caching for Haiku models only",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "DISABLE_PROMPT_CACHING_SONNET", Category: CatEnvCaching, Type: TypeBool,
			Description: "Disable prompt caching for Sonnet models only",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "DISABLE_PROMPT_CACHING_OPUS", Category: CatEnvCaching, Type: TypeBool,
			Description: "Disable prompt caching for Opus models only",
			ValidScopes: ev,
		},
	)

	// --- Paths ---
	keys = append(keys,
		ConfigKey{
			Key: "CLAUDE_CONFIG_DIR", Category: CatEnvPaths, Type: TypeString,
			Default: "~/.claude", Description: "Override the root config and data directory",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_TMPDIR", Category: CatEnvPaths, Type: TypeString,
			Default: "/tmp", Description: "Override the temporary file directory",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_DEBUG_LOGS_DIR", Category: CatEnvPaths, Type: TypeString,
			Description: "Custom directory for debug log output",
			ValidScopes: ev,
		},
	)

	// --- Account Info ---
	keys = append(keys,
		ConfigKey{
			Key: "CLAUDE_CODE_ACCOUNT_UUID", Category: CatEnvAccount, Type: TypeString,
			Description: "Account UUID for the authenticated user (SDK callers)",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_USER_EMAIL", Category: CatEnvAccount, Type: TypeString,
			Description: "Email address for the authenticated user (SDK callers)",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_ORGANIZATION_UUID", Category: CatEnvAccount, Type: TypeString,
			Description: "Organization UUID (SDK callers)",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_HIDE_ACCOUNT_INFO", Category: CatEnvAccount, Type: TypeBool,
			Description: "Hide email and org info from Claude Code UI",
			ValidScopes: ev,
		},
	)

	// --- OpenTelemetry ---
	keys = append(keys,
		ConfigKey{
			Key: "OTEL_EXPORTER_OTLP_ENDPOINT", Category: CatTelemetry, Type: TypeString,
			Description: "OTLP exporter endpoint URL",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "OTEL_EXPORTER_OTLP_HEADERS", Category: CatTelemetry, Type: TypeString,
			Description: "OTLP exporter headers (comma-separated key=value pairs)",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "OTEL_EXPORTER_OTLP_PROTOCOL", Category: CatTelemetry, Type: TypeEnum,
			Description: "OTLP protocol",
			ValidScopes: ev, EnumValues: []string{"grpc", "http/protobuf"},
		},
		ConfigKey{
			Key: "OTEL_LOG_USER_PROMPTS", Category: CatTelemetry, Type: TypeBool,
			Description: "Include user prompts in OpenTelemetry telemetry data",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "OTEL_LOG_TOOL_CONTENT", Category: CatTelemetry, Type: TypeBool,
			Description: "Include tool content in OpenTelemetry telemetry data",
			ValidScopes: ev,
		},
	)

	// --- Miscellaneous ---
	keys = append(keys,
		ConfigKey{
			Key: "CLAUDE_CODE_EXTRA_BODY", Category: CatEnvMisc, Type: TypeString,
			Description: "Additional JSON merged into every API request body",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_BLOCKING_LIMIT_OVERRIDE", Category: CatEnvMisc, Type: TypeInt,
			Description: "Override the token blocking limit threshold",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_RESUME_INTERRUPTED_TURN", Category: CatEnvMisc, Type: TypeBool,
			Description: "Automatically resume an interrupted turn when restarting",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_SLOW_OPERATION_THRESHOLD_MS", Category: CatEnvMisc, Type: TypeInt,
			Description: "Threshold (ms) above which operations are flagged as slow",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "FORCE_AUTOUPDATE_PLUGINS", Category: CatEnvMisc, Type: TypeBool,
			Description: "Force plugin auto-updates regardless of user preference",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "TASK_MAX_OUTPUT_LENGTH", Category: CatEnvMisc, Type: TypeInt,
			Description: "Maximum output length for background task results",
			ValidScopes: ev,
		},
		ConfigKey{
			Key: "CLAUDE_CODE_API_KEY_FILE_DESCRIPTOR", Category: CatEnvMisc, Type: TypeInt,
			Description: "File descriptor number to read the API key from",
			ValidScopes: ev,
		},
	)

	return keys
}
