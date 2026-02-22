package main

import "embed"

// PlatformFS embeds the _template directory (containing .claude and .mcp.json)
// into the binary. The "all:" prefix includes dotfiles (files starting with ".").
//
//go:embed all:_template
var PlatformFS embed.FS
