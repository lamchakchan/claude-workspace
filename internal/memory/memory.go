package memory

import (
	"fmt"
	"os"
	"strings"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

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
	default:
		return fmt.Errorf("unknown memory subcommand: %s\nAvailable: show, export, import", args[0])
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
			if l.Provider == "engram" && platform.Exists("engram") {
				fmt.Fprintln(w)
				// Shell out to engram for stats and recent observations
				if l.Stats != "" {
					fmt.Fprintln(w, l.Stats)
				}
				// Also show recent search results
				out, err := platform.Output("engram", "search", "*")
				if err == nil && out != "" {
					platform.PrintSection(w, "Recent observations")
					fmt.Fprintln(w, out)
				}
			} else if l.Provider == "none" {
				platform.PrintWarn(w, "No memory MCP server configured")
			} else {
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
		return
	}
	if l.Exists {
		platform.PrintOK(w, fmt.Sprintf("%s", shortenHome(l.Path)))
		if l.Stats != "" {
			// Print each line of stats indented
			for _, line := range strings.Split(l.Stats, "\n") {
				if line != "" {
					fmt.Fprintf(w, "    %s\n", line)
				}
			}
		}
		if l.Provider == "engram" {
			fmt.Fprintf(w, "  Run %s for interactive browsing\n", platform.Bold("engram tui"))
			fmt.Fprintf(w, "  Run %s to export as JSON\n", platform.Bold("engram export"))
		}
	} else {
		platform.PrintWarn(w, fmt.Sprintf("%s  (no data)", shortenHome(l.Path)))
	}
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
