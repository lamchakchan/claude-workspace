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
var knownMemoryProviders = []string{"mcp-memory-libsql", "engram", "memory"}

// Run is the entry point for the memory command.
func Run(args []string) error {
	if len(args) == 0 {
		return overview()
	}

	switch args[0] {
	case "show":
		scope := "all"
		for i := 1; i < len(args); i++ {
			if args[i] == "--scope" && i+1 < len(args) {
				i++
				scope = args[i]
			}
		}
		return show(ParseScope(scope))
	case "export":
		output := ""
		for i := 1; i < len(args); i++ {
			if args[i] == "--output" && i+1 < len(args) {
				i++
				output = args[i]
			}
		}
		return export(output)
	case "import":
		if len(args) < 2 {
			return fmt.Errorf("usage: claude-workspace memory import <file> [--scope=...] [--confirm]")
		}
		file := args[1]
		scope := "auto,mcp"
		confirm := false
		for i := 2; i < len(args); i++ {
			if args[i] == "--confirm" {
				confirm = true
			} else if args[i] == "--scope" && i+1 < len(args) {
				i++
				scope = args[i]
			}
		}
		return importMemory(file, ParseScope(scope), confirm)
	case "configure":
		return runConfigure(args[1:])
	default:
		return fmt.Errorf("unknown memory subcommand: %s\nAvailable: show, export, import, configure", args[0])
	}
}

// overview displays a summary of all memory layers.
func overview() error {
	layers, err := DiscoverLayers()
	if err != nil {
		return err
	}

	w := os.Stdout
	platform.PrintBanner(w, "Memory Layers")

	for _, l := range layers {
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

	for _, l := range layers {
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
			if !l.Exists {
				platform.PrintWarn(w, fmt.Sprintf("%s (not found)", l.Path))
				continue
			}
			content := readFileContent(l.Path)
			fmt.Fprintln(w, content)

		case LayerAutoMemory:
			if !l.Exists || len(l.Files) == 0 {
				platform.PrintWarn(w, fmt.Sprintf("%s (empty or not found)", l.Path))
				continue
			}
			for name, content := range l.Files {
				platform.PrintSection(w, name)
				fmt.Fprintln(w, content)
			}

		case LayerMemoryMCP:
			switch l.Provider {
			case "engram":
				if platform.Exists("engram") {
					fmt.Fprintln(w)
					if l.Stats != "" {
						fmt.Fprintln(w, l.Stats)
					}
					out, err := platform.Output("engram", "search", "*")
					if err == nil && out != "" {
						platform.PrintSection(w, "Recent observations")
						fmt.Fprintln(w, out)
					}
				} else {
					platform.PrintWarn(w, fmt.Sprintf("Provider %q CLI not available", l.Provider))
				}
			case "mcp-memory-libsql":
				fmt.Fprintf(w, "  DB: %s\n", shortenHome(l.Path))
				if !platform.Exists("claude") {
					fmt.Fprintf(w, "  Search requires Claude: %s\n", platform.Bold("mcp__mcp-memory-libsql__search_nodes"))
					fmt.Fprintf(w, "  Read all: %s\n", platform.Bold("mcp__mcp-memory-libsql__read_graph"))
				} else {
					fmt.Fprintln(w)
					if err := platform.Run("claude", "-p",
						"Use mcp__mcp-memory-libsql__read_graph to retrieve all stored memories and display them in a clear, human-readable format grouped by entity type.",
						"--allowedTools", "mcp__mcp-memory-libsql__read_graph",
					); err != nil {
						platform.PrintWarn(w, fmt.Sprintf("claude error: %v", err))
					}
				}
			case "none":
				platform.PrintWarn(w, "No memory MCP server configured")
			default:
				platform.PrintWarn(w, fmt.Sprintf("Provider %q CLI not available", l.Provider))
			}
		}
	}

	return nil
}

func providerSuffix(l Layer) string {
	if l.Name == LayerMemoryMCP && l.Provider != "" && l.Provider != "none" {
		return " — " + l.Provider
	}
	return ""
}

func printFileLayer(w *os.File, l Layer) {
	if l.Exists {
		platform.PrintOK(w, fmt.Sprintf("%s  (%d lines)", shortenHome(l.Path), l.Lines))
	} else {
		platform.PrintWarn(w, fmt.Sprintf("%s  (not found)", shortenHome(l.Path)))
	}
}

func printAutoMemoryLayer(w *os.File, l Layer) {
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

func printMCPLayer(w *os.File, l Layer) {
	if l.Provider == "none" {
		platform.PrintWarn(w, "No memory MCP server configured")
		fmt.Fprintf(w, "  Run %s to configure\n", platform.Bold("claude-workspace memory configure"))
		return
	}
	if l.Exists {
		platform.PrintOK(w, fmt.Sprintf("%s", shortenHome(l.Path)))
		if l.Stats != "" {
			for _, line := range strings.Split(l.Stats, "\n") {
				if line != "" {
					fmt.Fprintf(w, "    %s\n", line)
				}
			}
		}
		switch l.Provider {
		case "engram":
			fmt.Fprintf(w, "  Run %s for interactive browsing\n", platform.Bold("engram tui"))
			fmt.Fprintf(w, "  Run %s to export as JSON\n", platform.Bold("engram export"))
		case "mcp-memory-libsql":
			fmt.Fprintf(w, "  Search in Claude: %s\n", platform.Bold("mcp__mcp-memory-libsql__search_nodes"))
			fmt.Fprintf(w, "  Read all in Claude: %s\n", platform.Bold("mcp__mcp-memory-libsql__read_graph"))
		}
	} else {
		platform.PrintWarn(w, fmt.Sprintf("%s  (no data yet)", shortenHome(l.Path)))
		if l.Provider == "mcp-memory-libsql" {
			fmt.Fprintf(w, "  DB will be created on first use\n")
		}
	}
}

// runConfigure implements the `memory configure` subcommand.
// It interactively (or via flags) sets the active memory MCP provider in ~/.claude.json.
func runConfigure(args []string) error {
	var flagProvider, flagDBPath string
	autoYes := false
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--provider" && i+1 < len(args):
			i++
			flagProvider = args[i]
		case strings.HasPrefix(args[i], "--provider="):
			flagProvider = strings.TrimPrefix(args[i], "--provider=")
		case args[i] == "--db-path" && i+1 < len(args):
			i++
			flagDBPath = args[i]
		case strings.HasPrefix(args[i], "--db-path="):
			flagDBPath = strings.TrimPrefix(args[i], "--db-path=")
		case args[i] == "--yes", args[i] == "-y":
			autoYes = true
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}
	claudeConfig := filepath.Join(home, ".claude.json")

	// Read current config
	var config map[string]interface{}
	if platform.FileExists(claudeConfig) {
		if err := platform.ReadJSONFile(claudeConfig, &config); err != nil {
			return fmt.Errorf("reading %s: %w", claudeConfig, err)
		}
	}
	if config == nil {
		config = make(map[string]interface{})
	}

	// Detect and display current memory provider
	currentProvider, currentPath := detectProvider(home)
	w := os.Stdout
	platform.PrintBanner(w, "Memory Configure")

	if currentProvider != "none" {
		fmt.Fprintf(w, "  Current provider: %s\n", currentProvider)
		if currentPath != "" {
			fmt.Fprintf(w, "  Current DB path:  %s\n", shortenHome(currentPath))
		}
	} else {
		fmt.Fprintln(w, "  No memory MCP provider currently configured.")
	}
	fmt.Fprintln(w)

	reader := bufio.NewReader(os.Stdin)

	// Prompt for provider
	provider := flagProvider
	if provider == "" {
		fmt.Fprintln(w, "  Choose a memory provider:")
		fmt.Fprintln(w, "    1) mcp-memory-libsql  (recommended — no extra install, uses npx)")
		fmt.Fprintln(w, "    2) engram              (optional — requires: brew install gentleman-programming/tap/engram)")
		fmt.Fprintln(w, "    3) none                (remove all memory MCP config)")
		platform.PrintPrompt(w, "  Provider [1]: ")
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		switch line {
		case "", "1", "mcp-memory-libsql":
			provider = "mcp-memory-libsql"
		case "2", "engram":
			provider = "engram"
		case "3", "none":
			provider = "none"
		default:
			return fmt.Errorf("unknown provider choice %q; expected 1, 2, or 3", line)
		}
	}

	// For mcp-memory-libsql, prompt for DB path
	dbPath := flagDBPath
	if provider == "mcp-memory-libsql" && dbPath == "" && !autoYes {
		defaultDB := filepath.Join(home, ".config", "claude-workspace", "memory.db")
		platform.PrintPrompt(w, fmt.Sprintf("  DB file path [%s]: ", shortenHome(defaultDB)))
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			dbPath = defaultDB
		} else {
			// Expand ~ if present
			if strings.HasPrefix(line, "~/") {
				dbPath = filepath.Join(home, line[2:])
			} else {
				dbPath = line
			}
		}
	} else if provider == "mcp-memory-libsql" && dbPath == "" {
		dbPath = filepath.Join(home, ".config", "claude-workspace", "memory.db")
	}

	// Remove all known memory provider keys
	config = removeMemoryProviders(config, knownMemoryProviders)

	// Add the new provider (unless "none")
	var newEntry map[string]interface{}
	switch provider {
	case "mcp-memory-libsql":
		newEntry = map[string]interface{}{
			"command": "npx",
			"args":    []string{"-y", "mcp-memory-libsql"},
			"env":     map[string]interface{}{"LIBSQL_URL": "file:" + dbPath},
		}
	case "engram":
		newEntry = map[string]interface{}{
			"command": "engram",
			"args":    []string{"mcp"},
		}
	case "none":
		// nothing to add
	}

	if newEntry != nil {
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

	// Create DB parent directory for mcp-memory-libsql
	if provider == "mcp-memory-libsql" {
		if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
			platform.PrintWarningLine(w, fmt.Sprintf("could not create DB directory: %v", err))
		}
	}

	// Print confirmation
	fmt.Fprintln(w)
	if currentProvider != "none" && currentProvider != provider {
		platform.PrintOK(w, fmt.Sprintf("Removed: %s", currentProvider))
	}
	switch provider {
	case "mcp-memory-libsql":
		platform.PrintOK(w, fmt.Sprintf("Configured: mcp-memory-libsql  (DB: %s)", shortenHome(dbPath)))
		fmt.Fprintln(w, "  Tools to use in Claude:")
		fmt.Fprintf(w, "    %s\n", platform.Bold("mcp__mcp-memory-libsql__search_nodes"))
		fmt.Fprintf(w, "    %s\n", platform.Bold("mcp__mcp-memory-libsql__create_entities"))
		fmt.Fprintf(w, "    %s\n", platform.Bold("mcp__mcp-memory-libsql__read_graph"))
	case "engram":
		platform.PrintOK(w, "Configured: engram")
		fmt.Fprintln(w, "  Tools to use in Claude:")
		fmt.Fprintf(w, "    %s\n", platform.Bold("mcp__engram__mem_search"))
		fmt.Fprintf(w, "    %s\n", platform.Bold("mcp__engram__mem_save"))
	case "none":
		platform.PrintOK(w, "All memory MCP providers removed")
	}
	fmt.Fprintln(w)
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
