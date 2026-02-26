package lint

import "strings"

// MCP server definition
#McpServer: {
	command: string & strings.MinRunes(1)
	args: [...string]
	env?: {[string]: string}
	...
}

// Top-level .mcp.json config
#McpConfig: {
	mcpServers: {[string]: #McpServer}
}
