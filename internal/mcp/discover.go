package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

const (
	scopeUser    = "user"
	scopeProject = "project"
	scopeManaged = "managed"
)

// Server represents a configured MCP server with its scope.
type Server struct {
	Name  string
	Scope string // "user", "project", or "managed"
}

// DiscoverServers returns all configured MCP servers across all scopes.
// Managed servers are included but marked as non-removable via their scope.
func DiscoverServers() ([]Server, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home directory: %w", err)
	}

	userServers := discoverUserServers(home)
	projectServers := discoverProjectServers()
	managedServers := discoverManagedServers()

	servers := make([]Server, 0, len(userServers)+len(projectServers)+len(managedServers))
	servers = append(servers, userServers...)
	servers = append(servers, projectServers...)
	servers = append(servers, managedServers...)

	sort.Slice(servers, func(i, j int) bool {
		if servers[i].Scope != servers[j].Scope {
			return scopeOrder(servers[i].Scope) < scopeOrder(servers[j].Scope)
		}
		return servers[i].Name < servers[j].Name
	})

	return servers, nil
}

// scopeOrder returns a sort key for scope ordering: user, project, managed.
func scopeOrder(scope string) int {
	switch scope {
	case scopeUser:
		return 0
	case scopeProject:
		return 1
	case scopeManaged:
		return 2
	default:
		return 3
	}
}

// discoverUserServers reads ~/.claude.json and extracts mcpServers names.
func discoverUserServers(home string) []Server {
	path := filepath.Join(home, ".claude.json")
	if !platform.FileExists(path) {
		return nil
	}
	var root map[string]json.RawMessage
	if err := platform.ReadJSONFile(path, &root); err != nil {
		return nil
	}
	rawServers, ok := root["mcpServers"]
	if !ok {
		return nil
	}
	var serverMap map[string]json.RawMessage
	if json.Unmarshal(rawServers, &serverMap) != nil {
		return nil
	}
	servers := make([]Server, 0, len(serverMap))
	for name := range serverMap {
		servers = append(servers, Server{Name: name, Scope: scopeUser})
	}
	return servers
}

// discoverProjectServers reads .mcp.json and extracts server names.
func discoverProjectServers() []Server {
	path := filepath.Join(".", ".mcp.json")
	if !platform.FileExists(path) {
		return nil
	}
	return readServerNames(path, scopeProject)
}

// discoverManagedServers reads the managed MCP config file.
func discoverManagedServers() []Server {
	var path string
	if runtime.GOOS == "darwin" {
		path = "/Library/Application Support/ClaudeCode/managed-mcp.json"
	} else {
		path = "/etc/claude-code/managed-mcp.json"
	}
	if !platform.FileExists(path) {
		return nil
	}
	return readServerNames(path, scopeManaged)
}

// readServerNames reads a .mcp.json-format file and returns servers tagged with the given scope.
func readServerNames(path, scope string) []Server {
	var root struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if err := platform.ReadJSONFile(path, &root); err != nil {
		return nil
	}
	servers := make([]Server, 0, len(root.MCPServers))
	for name := range root.MCPServers {
		servers = append(servers, Server{Name: name, Scope: scope})
	}
	return servers
}
