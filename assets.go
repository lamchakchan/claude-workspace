package main

import "embed"

// PlatformFS embeds the _template directory (containing .claude and .mcp.json)
// into the binary. The "all:" prefix includes dotfiles (files starting with ".").
//
//go:embed all:_template
var PlatformFS embed.FS

// McpConfigFS embeds the docs/mcp-configs directory containing pre-defined
// MCP server recipe JSON files used by the TUI recipe picker.
//
//go:embed all:docs/mcp-configs
var McpConfigFS embed.FS

// MarketplaceRegistryFS embeds the docs/plugin-marketplaces directory containing
// the curated marketplace registry JSON file.
//
//go:embed all:docs/plugin-marketplaces
var MarketplaceRegistryFS embed.FS
