package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

// LayerName identifies a memory layer.
type LayerName string

const (
	LayerUserClaudeMD    LayerName = "user_claude_md"
	LayerProjectClaudeMD LayerName = "project_claude_md"
	LayerLocalMD         LayerName = "local_md"
	LayerAutoMemory      LayerName = "auto_memory"
	LayerMemoryMCP       LayerName = "memory_mcp"
)

// AllLayers lists every layer in precedence order.
var AllLayers = []LayerName{
	LayerUserClaudeMD,
	LayerProjectClaudeMD,
	LayerLocalMD,
	LayerAutoMemory,
	LayerMemoryMCP,
}

// Layer holds the discovered state of one memory layer.
type Layer struct {
	Name     LayerName
	Label    string // human-readable label
	Path     string // primary file or directory path
	Exists   bool
	Lines    int               // line count (for file-based layers)
	Files    map[string]string // file name → content (for auto-memory directory)
	Provider string            // MCP provider name (for memory_mcp)
	Stats    string            // output from provider stats command
}

// DiscoverLayers inspects the filesystem and returns the state of all layers.
func DiscoverLayers() ([]Layer, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home directory: %w", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}

	var layers []Layer

	// User CLAUDE.md
	userMD := filepath.Join(home, ".claude", "CLAUDE.md")
	layers = append(layers, discoverFileLayer(LayerUserClaudeMD, "User CLAUDE.md", userMD))

	// Project CLAUDE.md — check .claude/CLAUDE.md first, fall back to ./CLAUDE.md
	projectMD := filepath.Join(cwd, ".claude", "CLAUDE.md")
	if !platform.FileExists(projectMD) {
		projectMD = filepath.Join(cwd, "CLAUDE.md")
	}
	layers = append(layers, discoverFileLayer(LayerProjectClaudeMD, "Project CLAUDE.md", projectMD))

	// CLAUDE.local.md
	localMD := filepath.Join(cwd, "CLAUDE.local.md")
	layers = append(layers, discoverFileLayer(LayerLocalMD, "CLAUDE.local.md", localMD))

	// Auto-memory
	autoDir := autoMemoryDir(home, cwd)
	layers = append(layers, discoverAutoMemory(autoDir))

	// Memory MCP
	layers = append(layers, discoverMemoryMCP(home))

	return layers, nil
}

// ParseScope parses a comma-separated scope string into a set of layer names.
// "all" expands to every layer.
func ParseScope(scope string) map[LayerName]bool {
	m := make(map[LayerName]bool)
	for _, s := range strings.Split(scope, ",") {
		s = strings.TrimSpace(s)
		switch s {
		case "all":
			for _, l := range AllLayers {
				m[l] = true
			}
		case "user":
			m[LayerUserClaudeMD] = true
		case "project":
			m[LayerProjectClaudeMD] = true
		case "local":
			m[LayerLocalMD] = true
		case "auto":
			m[LayerAutoMemory] = true
		case "mcp":
			m[LayerMemoryMCP] = true
		}
	}
	return m
}

// encodeProjectPath converts a filesystem path to Claude's directory encoding (/ → -).
func encodeProjectPath(path string) string {
	return strings.ReplaceAll(path, "/", "-")
}

// autoMemoryDir returns the auto-memory directory path for a project.
func autoMemoryDir(home, projectPath string) string {
	encoded := encodeProjectPath(projectPath)
	return filepath.Join(home, ".claude", "projects", encoded, "memory")
}

func discoverFileLayer(name LayerName, label, path string) Layer {
	l := Layer{
		Name:  name,
		Label: label,
		Path:  path,
	}
	if platform.FileExists(path) {
		l.Exists = true
		data, err := os.ReadFile(path)
		if err == nil {
			l.Lines = countLines(string(data))
		}
	}
	return l
}

func discoverAutoMemory(dir string) Layer {
	l := Layer{
		Name:  LayerAutoMemory,
		Label: "Auto-memory",
		Path:  dir,
		Files: make(map[string]string),
	}
	if !platform.FileExists(dir) {
		return l
	}
	l.Exists = true
	entries, err := os.ReadDir(dir)
	if err != nil {
		return l
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err == nil {
			l.Files[e.Name()] = string(data)
		}
	}
	return l
}

func discoverMemoryMCP(home string) Layer {
	l := Layer{
		Name:  LayerMemoryMCP,
		Label: "Memory MCP",
	}

	// Detect provider from ~/.claude.json mcpServers
	provider, dataPath := detectProvider(home)
	l.Provider = provider
	l.Path = dataPath

	if l.Path != "" && platform.FileExists(l.Path) {
		l.Exists = true
	}

	switch provider {
	case "engram":
		if platform.Exists("engram") {
			stats, err := platform.Output("engram", "stats")
			if err == nil {
				l.Stats = stats
			}
		}
	case "mcp-memory-libsql":
		// No CLI stats command — direct users to the Claude tool instead
		l.Stats = "run: mcp__mcp-memory-libsql__read_graph"
	}

	return l
}

// mcpServerConfig is a minimal struct for parsing MCP server entries from ~/.claude.json.
type mcpServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

// libsqlDBPath extracts the database file path from a mcp-memory-libsql server config.
// The LIBSQL_URL env var is expected to be "file:<path>". Falls back to defaultPath.
func libsqlDBPath(raw json.RawMessage, defaultPath string) string {
	var cfg mcpServerConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return defaultPath
	}
	url, ok := cfg.Env["LIBSQL_URL"]
	if !ok || url == "" {
		return defaultPath
	}
	path := strings.TrimPrefix(url, "file:")
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[1:])
		}
	}
	return path
}

// detectProvider reads ~/.claude.json to find the configured memory MCP server.
// Returns (provider name, data path). Priority: mcp-memory-libsql > engram > memory.
func detectProvider(home string) (string, string) {
	claudeJSON := filepath.Join(home, ".claude.json")
	defaultLibsqlPath := filepath.Join(home, ".config", "claude-workspace", "memory.db")

	if !platform.FileExists(claudeJSON) {
		// Fall back to checking if engram data exists
		engramDB := filepath.Join(home, ".engram", "engram.db")
		if platform.FileExists(engramDB) {
			return "engram", engramDB
		}
		return "none", ""
	}

	var config struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if err := platform.ReadJSONFile(claudeJSON, &config); err != nil {
		engramDB := filepath.Join(home, ".engram", "engram.db")
		if platform.FileExists(engramDB) {
			return "engram", engramDB
		}
		return "none", ""
	}

	// Priority 1: mcp-memory-libsql
	if raw, ok := config.MCPServers["mcp-memory-libsql"]; ok {
		return "mcp-memory-libsql", libsqlDBPath(raw, defaultLibsqlPath)
	}
	// Priority 2: engram
	if _, ok := config.MCPServers["engram"]; ok {
		return "engram", filepath.Join(home, ".engram", "engram.db")
	}
	// Priority 3: memory (legacy)
	if _, ok := config.MCPServers["memory"]; ok {
		return "memory", filepath.Join(home, ".memory", "memory.json")
	}

	// No MCP server configured, but check if engram data exists as fallback
	engramDB := filepath.Join(home, ".engram", "engram.db")
	if platform.FileExists(engramDB) {
		return "engram", engramDB
	}
	return "none", ""
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	n := strings.Count(s, "\n")
	// Count trailing content without newline as a line
	if !strings.HasSuffix(s, "\n") {
		n++
	}
	return n
}
