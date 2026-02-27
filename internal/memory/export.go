package memory

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

// ExportData is the top-level export structure.
type ExportData struct {
	Version    int          `json:"version"`
	ExportedAt string       `json:"exported_at"`
	Platform   string       `json:"platform"`
	Layers     ExportLayers `json:"layers"`
}

// ExportLayers contains all exported memory layers.
type ExportLayers struct {
	UserClaudeMD    *ExportFile    `json:"user_claude_md"`
	ProjectClaudeMD *ExportFile    `json:"project_claude_md"`
	LocalMD         *ExportFile    `json:"local_md"`
	AutoMemory      *ExportAutoMem `json:"auto_memory"`
	MemoryMCP       *ExportMCP     `json:"memory_mcp"`
}

// ExportFile represents a single-file layer in the export.
type ExportFile struct {
	Path    string  `json:"path"`
	Project string  `json:"project,omitempty"`
	Content *string `json:"content"` // nil means file doesn't exist
}

// ExportAutoMem represents the auto-memory directory in the export.
type ExportAutoMem struct {
	BasePath string            `json:"base_path"`
	Files    map[string]string `json:"files"` // filename → content
}

// ExportMCP represents the memory MCP layer in the export.
type ExportMCP struct {
	Provider string           `json:"provider"`
	Data     *json.RawMessage `json:"data"` // raw JSON from provider export
}

// export writes all memory layers to JSON.
func export(outputPath string) error {
	layers, err := DiscoverLayers()
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	data := ExportData{
		Version:    1,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Platform:   "claude-workspace",
		Layers:     ExportLayers{},
	}

	for i := range layers {
		l := &layers[i]
		switch l.Name {
		case LayerUserClaudeMD:
			data.Layers.UserClaudeMD = exportFileLayer(l, "")
		case LayerProjectClaudeMD:
			data.Layers.ProjectClaudeMD = exportFileLayer(l, cwd)
		case LayerLocalMD:
			data.Layers.LocalMD = exportFileLayer(l, "")
		case LayerAutoMemory:
			data.Layers.AutoMemory = &ExportAutoMem{BasePath: l.Path, Files: l.Files}
		case LayerMemoryMCP:
			data.Layers.MemoryMCP = exportMCPLayer(l)
		}
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling export: %w", err)
	}
	jsonData = append(jsonData, '\n')

	if outputPath == "" || outputPath == "-" {
		_, err = os.Stdout.Write(jsonData)
		return err
	}

	return os.WriteFile(outputPath, jsonData, 0644)
}

// exportFileLayer builds an ExportFile from a discovered file layer.
// If project is non-empty, it is set on the returned ExportFile.
func exportFileLayer(l *Layer, project string) *ExportFile {
	ef := &ExportFile{Path: l.Path, Project: project}
	if l.Exists {
		content := readFileContent(l.Path)
		ef.Content = &content
	}
	return ef
}

// exportMCPLayer builds an ExportMCP from a discovered MCP layer,
// fetching data from the appropriate provider.
func exportMCPLayer(l *Layer) *ExportMCP {
	em := &ExportMCP{Provider: l.Provider}
	switch l.Provider {
	case "engram":
		if platform.Exists("engram") {
			if raw := exportEngram(); raw != nil {
				em.Data = raw
			}
		}
	case "mcp-memory-libsql":
		if !platform.Exists("claude") {
			fmt.Fprintln(os.Stderr, "  Warning: claude CLI not available — cannot export mcp-memory-libsql data.")
			fmt.Fprintln(os.Stderr, "  To back up memories, use Claude with: mcp__mcp-memory-libsql__read_graph")
		} else {
			raw, err := exportLibsqlViaClaude()
			if err != nil {
				fmt.Fprintf(os.Stderr, "  Warning: mcp-memory-libsql export via Claude failed: %v\n", err)
			} else if raw != nil {
				em.Data = raw
			}
		}
	}
	return em
}

// importMemory restores layers from a previously exported JSON file.
func importMemory(filePath string, scope map[LayerName]bool, confirm bool) error {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading import file: %w", err)
	}

	var data ExportData
	if err := json.Unmarshal(raw, &data); err != nil {
		return fmt.Errorf("parsing import file: %w", err)
	}

	if data.Version != 1 {
		return fmt.Errorf("unsupported export version: %d", data.Version)
	}

	w := os.Stdout

	platform.PrintBanner(w, "Memory Import Preview")
	count := previewImport(w, &data, scope)

	if count == 0 {
		fmt.Fprintln(w, "  Nothing to import for the selected scope.")
		return nil
	}

	if !confirm {
		fmt.Fprintf(w, "\n  Re-run with --confirm to apply, or add --scope to limit.\n")
		return nil
	}

	fmt.Fprintln(w)

	// Apply changes
	importFileLayer(w, scope, LayerUserClaudeMD, data.Layers.UserClaudeMD, "User CLAUDE.md")
	importFileLayer(w, scope, LayerProjectClaudeMD, data.Layers.ProjectClaudeMD, "Project CLAUDE.md")
	importFileLayer(w, scope, LayerLocalMD, data.Layers.LocalMD, "CLAUDE.local.md")
	importAutoMemory(w, scope, data.Layers.AutoMemory)
	importMemoryMCP(w, scope, data.Layers.MemoryMCP)

	fmt.Fprintln(w)
	return nil
}

// previewImport prints a preview of what will be imported and returns the count of items.
func previewImport(w io.Writer, data *ExportData, scope map[LayerName]bool) int {
	count := 0
	count += previewFileImport(w, scope, LayerUserClaudeMD, data.Layers.UserClaudeMD)
	count += previewFileImport(w, scope, LayerProjectClaudeMD, data.Layers.ProjectClaudeMD)
	count += previewFileImport(w, scope, LayerLocalMD, data.Layers.LocalMD)
	if scope[LayerAutoMemory] && data.Layers.AutoMemory != nil && len(data.Layers.AutoMemory.Files) > 0 {
		fmt.Fprintf(w, "  Will write: %d file(s) to %s\n", len(data.Layers.AutoMemory.Files), data.Layers.AutoMemory.BasePath)
		count++
	}
	if scope[LayerMemoryMCP] && data.Layers.MemoryMCP != nil && data.Layers.MemoryMCP.Data != nil {
		fmt.Fprintf(w, "  Will import: Memory MCP data via %s\n", data.Layers.MemoryMCP.Provider)
		count++
	}
	return count
}

// previewFileImport previews a single file layer import. Returns 1 if the layer will be imported, 0 otherwise.
func previewFileImport(w io.Writer, scope map[LayerName]bool, name LayerName, ef *ExportFile) int {
	if !scope[name] || ef == nil || ef.Content == nil {
		return 0
	}
	fmt.Fprintf(w, "  Will write: %s (%d lines)\n", ef.Path, countLines(*ef.Content))
	return 1
}

// importFileLayer restores a single file-based layer if it is in scope and has content.
func importFileLayer(w io.Writer, scope map[LayerName]bool, name LayerName, ef *ExportFile, label string) {
	if !scope[name] || ef == nil || ef.Content == nil {
		return
	}
	if err := writeFileContent(ef.Path, *ef.Content); err != nil {
		platform.PrintFail(w, fmt.Sprintf("%s: %v", label, err))
	} else {
		platform.PrintOK(w, label+" restored")
	}
}

// importAutoMemory restores the auto-memory directory files if in scope.
func importAutoMemory(w io.Writer, scope map[LayerName]bool, am *ExportAutoMem) {
	if !scope[LayerAutoMemory] || am == nil || len(am.Files) == 0 {
		return
	}
	if err := os.MkdirAll(am.BasePath, 0755); err != nil {
		platform.PrintFail(w, fmt.Sprintf("Auto-memory dir: %v", err))
		return
	}
	for name, content := range am.Files {
		path := filepath.Join(am.BasePath, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			platform.PrintFail(w, fmt.Sprintf("Auto-memory %s: %v", name, err))
		}
	}
	platform.PrintOK(w, fmt.Sprintf("Auto-memory: %d file(s) restored", len(am.Files)))
}

// importMemoryMCP restores the memory MCP data if in scope, dispatching to the appropriate provider.
func importMemoryMCP(w io.Writer, scope map[LayerName]bool, mcp *ExportMCP) {
	if !scope[LayerMemoryMCP] || mcp == nil || mcp.Data == nil {
		return
	}
	switch mcp.Provider {
	case "engram":
		importViaEngram(w, mcp.Data)
	case "mcp-memory-libsql":
		importViaLibsql(w, mcp.Data)
	default:
		platform.PrintWarn(w, fmt.Sprintf("Cannot import Memory MCP: provider %q not available", mcp.Provider))
	}
}

// importViaEngram imports MCP memory data using the engram CLI.
func importViaEngram(w io.Writer, data *json.RawMessage) {
	if !platform.Exists("engram") {
		platform.PrintWarn(w, "Cannot import Memory MCP: engram not available")
		return
	}
	tmpFile, err := os.CreateTemp("", "memory-import-*.json")
	if err != nil {
		platform.PrintFail(w, fmt.Sprintf("Memory MCP temp file: %v", err))
		return
	}
	if _, err := tmpFile.Write(*data); err != nil {
		platform.PrintFail(w, fmt.Sprintf("Memory MCP write: %v", err))
		return
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if err := platform.Run("engram", "import", tmpFile.Name()); err != nil {
		platform.PrintFail(w, fmt.Sprintf("Memory MCP import: %v", err))
	} else {
		platform.PrintOK(w, "Memory MCP data imported via engram")
	}
}

// importViaLibsql imports MCP memory data using the claude CLI and mcp-memory-libsql tools.
func importViaLibsql(w io.Writer, data *json.RawMessage) {
	if !platform.Exists("claude") {
		platform.PrintWarn(w, "Cannot import Memory MCP: claude CLI not available.")
		fmt.Fprintln(w, "  Use mcp__mcp-memory-libsql__create_entities in Claude directly to restore memories.")
		return
	}
	dataJSON := string(*data)
	prompt := fmt.Sprintf(
		"Use mcp__mcp-memory-libsql__create_entities and mcp__mcp-memory-libsql__create_relations to import this memory data, preserving all entities and their observations exactly:\n\n%s",
		dataJSON,
	)
	if err := platform.RunWithSpinner(
		"Importing memories via Claude...",
		"claude", "-p", prompt,
		"--allowedTools", "mcp__mcp-memory-libsql__create_entities,mcp__mcp-memory-libsql__create_relations",
	); err != nil {
		platform.PrintFail(w, fmt.Sprintf("Memory MCP import via Claude: %v", err))
	} else {
		platform.PrintOK(w, "Memory MCP data imported via Claude + mcp-memory-libsql")
	}
}

func readFileContent(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func writeFileContent(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// exportLibsqlViaClaude invokes the claude CLI to call read_graph and returns the JSON data.
func exportLibsqlViaClaude() (*json.RawMessage, error) {
	spinner := platform.StartSpinner(os.Stderr, "Exporting memories via Claude...")
	out, err := platform.Output("claude", "-p",
		"Call mcp__mcp-memory-libsql__read_graph and output ONLY the raw JSON result — no commentary, no markdown fences, just valid JSON.",
		"--allowedTools", "mcp__mcp-memory-libsql__read_graph",
	)
	spinner.Stop()
	if err != nil {
		return nil, err
	}
	extracted := extractJSON(out)
	if !json.Valid([]byte(extracted)) {
		return nil, fmt.Errorf("claude output was not valid JSON")
	}
	raw := json.RawMessage(extracted)
	return &raw, nil
}

// extractJSON strips optional markdown code fences from a string and returns the inner content.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		// strip opening fence (```json or ```)
		end := strings.Index(s, "\n")
		if end < 0 {
			return s
		}
		s = s[end+1:]
		// strip closing fence
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
		s = strings.TrimSpace(s)
	}
	return s
}

func exportEngram() *json.RawMessage {
	out, err := platform.Output("engram", "export")
	if err != nil || out == "" {
		return nil
	}
	raw := json.RawMessage(out)
	// Validate it's valid JSON
	if !json.Valid(raw) {
		return nil
	}
	return &raw
}
