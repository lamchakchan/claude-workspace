// Package memory implements the "memory" command for inspecting and managing
// Claude Code's layered memory system, including overview, show, export, import,
// and provider configuration subcommands.
package memory

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

// knownMemoryProviders is the set of memory MCP server keys this platform manages.
var knownMemoryProviders = []string{providerLibsql, providerEngram, "memory"}

// Run is the entry point for the memory command.
func Run(args []string) error {
	if len(args) == 0 {
		return overview()
	}

	switch args[0] {
	case "show":
		return runShow(args[1:])
	case "export":
		return runExport(args[1:])
	case "import":
		return runImport(args[1:])
	case "configure":
		return runConfigure(args[1:])
	default:
		return fmt.Errorf("unknown memory subcommand: %s\nAvailable: show, export, import, configure", args[0])
	}
}

func runShow(args []string) error {
	scope := "all"
	for i := 0; i < len(args); i++ {
		if args[i] == "--scope" && i+1 < len(args) {
			i++
			scope = args[i]
		}
	}
	return show(ParseScope(scope))
}

func runExport(args []string) error {
	output := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--output" && i+1 < len(args) {
			i++
			output = args[i]
		}
	}
	return export(output)
}

func runImport(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: claude-workspace memory import <file> [--scope=...] [--confirm]")
	}
	file := args[0]
	scope := "auto,mcp"
	confirm := false
	for i := 1; i < len(args); i++ {
		if args[i] == "--confirm" {
			confirm = true
		} else if args[i] == "--scope" && i+1 < len(args) {
			i++
			scope = args[i]
		}
	}
	return importMemory(file, ParseScope(scope), confirm)
}

// overview displays a summary of all memory layers.
func overview() error {
	layers, err := DiscoverLayers()
	if err != nil {
		return err
	}

	w := os.Stdout
	platform.PrintBanner(w, "Memory Layers")

	for i := range layers {
		l := &layers[i]
		fmt.Fprintln(w)
		platform.PrintSectionLabel(w, l.Label+providerSuffix(l))

		switch l.Name {
		case LayerUserClaudeMD, LayerProjectClaudeMD, LayerLocalMD:
			printFileLayer(w, l)

		case LayerAutoMemory:
			printAutoMemoryLayer(w, l)

		case LayerMemoryMCP:
			printMCPLayer(w, l)
		}
	}

	fmt.Fprintln(w)
	return nil
}

// show dumps the contents of specified layers to stdout.
func show(scope map[LayerName]bool) error {
	layers, err := DiscoverLayers()
	if err != nil {
		return err
	}

	w := os.Stdout
	first := true

	for i := range layers {
		l := &layers[i]
		if !scope[l.Name] {
			continue
		}

		if !first {
			fmt.Fprintln(w)
		}
		first = false

		platform.PrintLayerBanner(w, l.Label+providerSuffix(l))

		switch l.Name {
		case LayerUserClaudeMD, LayerProjectClaudeMD, LayerLocalMD:
			showFileContent(w, l)
		case LayerAutoMemory:
			showAutoMemoryContent(w, l)
		case LayerMemoryMCP:
			showMCPContent(w, l)
		}
	}

	return nil
}

func showFileContent(w *os.File, l *Layer) {
	if !l.Exists {
		platform.PrintWarn(w, fmt.Sprintf("%s (not found)", l.Path))
		return
	}
	content := readFileContent(l.Path)
	fmt.Fprintln(w, content)
}

func showAutoMemoryContent(w *os.File, l *Layer) {
	if !l.Exists || len(l.Files) == 0 {
		platform.PrintWarn(w, fmt.Sprintf("%s (empty or not found)", l.Path))
		return
	}
	for name, content := range l.Files {
		platform.PrintSection(w, name)
		fmt.Fprintln(w, content)
	}
}

func showMCPContent(w *os.File, l *Layer) {
	switch l.Provider {
	case providerEngram:
		showEngramContent(w, l)
	case providerLibsql:
		showLibsqlContent(w, l)
	case providerNone:
		platform.PrintWarn(w, "No memory MCP server configured")
	default:
		platform.PrintWarn(w, fmt.Sprintf("Provider %q CLI not available", l.Provider))
	}
}

func showEngramContent(w *os.File, l *Layer) {
	if !platform.Exists(providerEngram) {
		platform.PrintWarn(w, fmt.Sprintf("Provider %q CLI not available", l.Provider))
		return
	}
	fmt.Fprintln(w)
	if l.Stats != "" {
		fmt.Fprintln(w, l.Stats)
	}
	out, err := platform.Output(providerEngram, "search", "*")
	if err == nil && out != "" {
		platform.PrintSection(w, "Recent observations")
		fmt.Fprintln(w, out)
	}
}

func showLibsqlContent(w *os.File, l *Layer) {
	fmt.Fprintf(w, "  DB: %s\n", shortenHome(l.Path))
	if !platform.Exists("claude") {
		fmt.Fprintf(w, "  Search requires Claude: %s\n", platform.Bold("mcp__mcp-memory-libsql__search_nodes"))
		fmt.Fprintf(w, "  Read all: %s\n", platform.Bold("mcp__mcp-memory-libsql__read_graph"))
		return
	}
	fmt.Fprintln(w)
	if err := platform.RunWithSpinner(
		"Querying memory graph via Claude...",
		"claude", "-p",
		"Use mcp__mcp-memory-libsql__read_graph to retrieve all stored memories and display them in a clear, human-readable format grouped by entity type.",
		"--allowedTools", "mcp__mcp-memory-libsql__read_graph",
	); err != nil {
		platform.PrintWarn(w, fmt.Sprintf("claude error: %v", err))
	}
}

func providerSuffix(l *Layer) string {
	if l.Name == LayerMemoryMCP && l.Provider != "" && l.Provider != providerNone {
		return " — " + l.Provider
	}
	return ""
}

func printFileLayer(w *os.File, l *Layer) {
	if l.Exists {
		platform.PrintOK(w, fmt.Sprintf("%s  (%d lines)", shortenHome(l.Path), l.Lines))
	} else {
		platform.PrintWarn(w, fmt.Sprintf("%s  (not found)", shortenHome(l.Path)))
	}
}

func printAutoMemoryLayer(w *os.File, l *Layer) {
	if !l.Exists {
		platform.PrintWarn(w, fmt.Sprintf("%s  (not found)", shortenHome(l.Path)))
		return
	}
	fileCount := len(l.Files)
	if fileCount == 0 {
		platform.PrintWarn(w, fmt.Sprintf("%s  (empty)", shortenHome(l.Path)))
		return
	}
	platform.PrintOK(w, fmt.Sprintf("%s  (%d files)", shortenHome(l.Path), fileCount))
	for name, content := range l.Files {
		lines := countLines(content)
		suffix := ""
		if name == "MEMORY.md" {
			suffix = " (auto-loaded, first 200 lines)"
		}
		fmt.Fprintf(w, "    %s: %d lines%s\n", name, lines, suffix)
	}
}

func printMCPLayer(w *os.File, l *Layer) {
	if l.Provider == providerNone {
		platform.PrintWarn(w, "No memory MCP server configured")
		fmt.Fprintf(w, "  Run %s to configure\n", platform.Bold("claude-workspace memory configure"))
		return
	}
	if l.Exists {
		platform.PrintOK(w, shortenHome(l.Path))
		if l.Stats != "" {
			for _, line := range strings.Split(l.Stats, "\n") {
				if line != "" {
					fmt.Fprintf(w, "    %s\n", line)
				}
			}
		}
		switch l.Provider {
		case providerEngram:
			fmt.Fprintf(w, "  Run %s for interactive browsing\n", platform.Bold("engram tui"))
			fmt.Fprintf(w, "  Run %s to export as JSON\n", platform.Bold("engram export"))
		case providerLibsql:
			fmt.Fprintf(w, "  Search in Claude: %s\n", platform.Bold("mcp__mcp-memory-libsql__search_nodes"))
			fmt.Fprintf(w, "  Read all in Claude: %s\n", platform.Bold("mcp__mcp-memory-libsql__read_graph"))
		}
	} else {
		platform.PrintWarn(w, fmt.Sprintf("%s  (no data yet)", shortenHome(l.Path)))
		if l.Provider == providerLibsql {
			fmt.Fprintf(w, "  DB will be created on first use\n")
		}
	}
}

type configureOpts struct {
	provider string
	dbPath   string
	autoYes  bool
}

func parseConfigureFlags(args []string) configureOpts {
	var opts configureOpts
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--provider" && i+1 < len(args):
			i++
			opts.provider = args[i]
		case strings.HasPrefix(args[i], "--provider="):
			opts.provider = strings.TrimPrefix(args[i], "--provider=")
		case args[i] == "--db-path" && i+1 < len(args):
			i++
			opts.dbPath = args[i]
		case strings.HasPrefix(args[i], "--db-path="):
			opts.dbPath = strings.TrimPrefix(args[i], "--db-path=")
		case args[i] == "--yes", args[i] == "-y":
			opts.autoYes = true
		}
	}
	return opts
}

func promptProvider(w *os.File, reader *bufio.Reader) (string, error) {
	fmt.Fprintln(w, "  Choose a memory provider:")
	fmt.Fprintln(w, "    1) mcp-memory-libsql  (recommended — no extra install, uses npx)")
	fmt.Fprintln(w, "    2) engram              (optional — requires: brew install gentleman-programming/tap/engram)")
	fmt.Fprintln(w, "    3) none                (remove all memory MCP config)")
	platform.PrintPrompt(w, "  Provider [1]: ")
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	switch line {
	case "", "1", providerLibsql:
		return providerLibsql, nil
	case "2", providerEngram:
		return providerEngram, nil
	case "3", providerNone:
		return providerNone, nil
	default:
		return "", fmt.Errorf("unknown provider choice %q; expected 1, 2, or 3", line)
	}
}

func resolveDBPath(w *os.File, reader *bufio.Reader, home, flagDBPath string, autoYes bool) string {
	if flagDBPath != "" {
		return flagDBPath
	}
	defaultDB := filepath.Join(home, ".config", "claude-workspace", "memory.db")
	if autoYes {
		return defaultDB
	}
	platform.PrintPrompt(w, fmt.Sprintf("  DB file path [%s]: ", shortenHome(defaultDB)))
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultDB
	}
	if strings.HasPrefix(line, "~/") {
		return filepath.Join(home, line[2:])
	}
	return line
}

func buildProviderEntry(provider, dbPath string) map[string]interface{} {
	switch provider {
	case providerLibsql:
		return map[string]interface{}{
			"command": "npx",
			"args":    []string{"-y", providerLibsql},
			"env":     map[string]interface{}{"LIBSQL_URL": "file:" + dbPath},
		}
	case providerEngram:
		return map[string]interface{}{
			"command": providerEngram,
			"args":    []string{"mcp"},
		}
	default:
		return nil
	}
}

func printConfigureResult(w *os.File, provider, dbPath, previousProvider string) {
	fmt.Fprintln(w)
	if previousProvider != providerNone && previousProvider != provider {
		platform.PrintOK(w, fmt.Sprintf("Removed: %s", previousProvider))
	}
	switch provider {
	case providerLibsql:
		platform.PrintOK(w, fmt.Sprintf("Configured: mcp-memory-libsql  (DB: %s)", shortenHome(dbPath)))
		fmt.Fprintln(w, "  Tools to use in Claude:")
		fmt.Fprintf(w, "    %s\n", platform.Bold("mcp__mcp-memory-libsql__search_nodes"))
		fmt.Fprintf(w, "    %s\n", platform.Bold("mcp__mcp-memory-libsql__create_entities"))
		fmt.Fprintf(w, "    %s\n", platform.Bold("mcp__mcp-memory-libsql__read_graph"))
	case providerEngram:
		platform.PrintOK(w, "Configured: engram")
		fmt.Fprintln(w, "  Tools to use in Claude:")
		fmt.Fprintf(w, "    %s\n", platform.Bold("mcp__engram__mem_search"))
		fmt.Fprintf(w, "    %s\n", platform.Bold("mcp__engram__mem_save"))
	case providerNone:
		platform.PrintOK(w, "All memory MCP providers removed")
	}
	fmt.Fprintln(w)
}

// runConfigure implements the `memory configure` subcommand.
// It interactively (or via flags) sets the active memory MCP provider in ~/.claude.json.
func runConfigure(args []string) error {
	opts := parseConfigureFlags(args)

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}
	claudeConfig := filepath.Join(home, ".claude.json")

	var config map[string]interface{}
	if platform.FileExists(claudeConfig) {
		if err := platform.ReadJSONFile(claudeConfig, &config); err != nil {
			return fmt.Errorf("reading %s: %w", claudeConfig, err)
		}
	}
	if config == nil {
		config = make(map[string]interface{})
	}

	currentProvider, currentPath := detectProvider(home)
	w := os.Stdout
	platform.PrintBanner(w, "Memory Configure")

	if currentProvider != providerNone {
		fmt.Fprintf(w, "  Current provider: %s\n", currentProvider)
		if currentPath != "" {
			fmt.Fprintf(w, "  Current DB path:  %s\n", shortenHome(currentPath))
		}
	} else {
		fmt.Fprintln(w, "  No memory MCP provider currently configured.")
	}
	fmt.Fprintln(w)

	reader := bufio.NewReader(os.Stdin)

	provider := opts.provider
	if provider == "" {
		provider, err = promptProvider(w, reader)
		if err != nil {
			return err
		}
	}

	dbPath := ""
	if provider == providerLibsql {
		dbPath = resolveDBPath(w, reader, home, opts.dbPath, opts.autoYes)
	}

	config = removeMemoryProviders(config, knownMemoryProviders)

	if newEntry := buildProviderEntry(provider, dbPath); newEntry != nil {
		existing, _ := config["mcpServers"].(map[string]interface{})
		if existing == nil {
			existing = make(map[string]interface{})
		}
		existing[provider] = newEntry
		config["mcpServers"] = existing
	}

	if err := platform.WriteJSONFile(claudeConfig, config); err != nil {
		return fmt.Errorf("writing %s: %w", claudeConfig, err)
	}

	if provider == providerLibsql {
		if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
			platform.PrintWarningLine(w, fmt.Sprintf("could not create DB directory: %v", err))
		}
	}

	printConfigureResult(w, provider, dbPath, currentProvider)
	return nil
}

// removeMemoryProviders removes the given server keys from the mcpServers section.
func removeMemoryProviders(config map[string]interface{}, keys []string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range config {
		result[k] = v
	}
	existing, _ := result["mcpServers"].(map[string]interface{})
	if existing == nil {
		return result
	}
	updated := make(map[string]interface{})
	for k, v := range existing {
		updated[k] = v
	}
	for _, key := range keys {
		delete(updated, key)
	}
	result["mcpServers"] = updated
	return result
}

// shortenHome replaces the home directory prefix with ~.
func shortenHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}
