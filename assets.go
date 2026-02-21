package main

import "embed"

// PlatformFS embeds the .claude directory and .mcp.json into the binary.
// The "all:" prefix includes dotfiles (files starting with ".").
//
//go:embed all:.claude .mcp.json
var PlatformFS embed.FS
