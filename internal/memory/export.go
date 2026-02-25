package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

	for _, l := range layers {
		switch l.Name {
		case LayerUserClaudeMD:
			ef := &ExportFile{Path: l.Path}
			if l.Exists {
				content := readFileContent(l.Path)
				ef.Content = &content
			}
			data.Layers.UserClaudeMD = ef

		case LayerProjectClaudeMD:
			ef := &ExportFile{Path: l.Path, Project: cwd}
			if l.Exists {
				content := readFileContent(l.Path)
				ef.Content = &content
			}
			data.Layers.ProjectClaudeMD = ef

		case LayerLocalMD:
			ef := &ExportFile{Path: l.Path}
			if l.Exists {
				content := readFileContent(l.Path)
				ef.Content = &content
			}
			data.Layers.LocalMD = ef

		case LayerAutoMemory:
			am := &ExportAutoMem{
				BasePath: l.Path,
				Files:    l.Files,
			}
			data.Layers.AutoMemory = am

		case LayerMemoryMCP:
			em := &ExportMCP{Provider: l.Provider}
			if l.Provider == "engram" && platform.Exists("engram") {
				raw := exportEngram()
				if raw != nil {
					em.Data = raw
				}
			}
			data.Layers.MemoryMCP = em
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

	// Show what will be restored
	platform.PrintBanner(w, "Memory Import Preview")
	count := 0

	if scope[LayerUserClaudeMD] && data.Layers.UserClaudeMD != nil && data.Layers.UserClaudeMD.Content != nil {
		fmt.Fprintf(w, "  Will write: %s (%d lines)\n", data.Layers.UserClaudeMD.Path, countLines(*data.Layers.UserClaudeMD.Content))
		count++
	}
	if scope[LayerProjectClaudeMD] && data.Layers.ProjectClaudeMD != nil && data.Layers.ProjectClaudeMD.Content != nil {
		fmt.Fprintf(w, "  Will write: %s (%d lines)\n", data.Layers.ProjectClaudeMD.Path, countLines(*data.Layers.ProjectClaudeMD.Content))
		count++
	}
	if scope[LayerLocalMD] && data.Layers.LocalMD != nil && data.Layers.LocalMD.Content != nil {
		fmt.Fprintf(w, "  Will write: %s (%d lines)\n", data.Layers.LocalMD.Path, countLines(*data.Layers.LocalMD.Content))
		count++
	}
	if scope[LayerAutoMemory] && data.Layers.AutoMemory != nil && len(data.Layers.AutoMemory.Files) > 0 {
		fmt.Fprintf(w, "  Will write: %d file(s) to %s\n", len(data.Layers.AutoMemory.Files), data.Layers.AutoMemory.BasePath)
		count++
	}
	if scope[LayerMemoryMCP] && data.Layers.MemoryMCP != nil && data.Layers.MemoryMCP.Data != nil {
		fmt.Fprintf(w, "  Will import: Memory MCP data via %s\n", data.Layers.MemoryMCP.Provider)
		count++
	}

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
	if scope[LayerUserClaudeMD] && data.Layers.UserClaudeMD != nil && data.Layers.UserClaudeMD.Content != nil {
		if err := writeFileContent(data.Layers.UserClaudeMD.Path, *data.Layers.UserClaudeMD.Content); err != nil {
			platform.PrintFail(w, fmt.Sprintf("User CLAUDE.md: %v", err))
		} else {
			platform.PrintOK(w, "User CLAUDE.md restored")
		}
	}

	if scope[LayerProjectClaudeMD] && data.Layers.ProjectClaudeMD != nil && data.Layers.ProjectClaudeMD.Content != nil {
		if err := writeFileContent(data.Layers.ProjectClaudeMD.Path, *data.Layers.ProjectClaudeMD.Content); err != nil {
			platform.PrintFail(w, fmt.Sprintf("Project CLAUDE.md: %v", err))
		} else {
			platform.PrintOK(w, "Project CLAUDE.md restored")
		}
	}

	if scope[LayerLocalMD] && data.Layers.LocalMD != nil && data.Layers.LocalMD.Content != nil {
		if err := writeFileContent(data.Layers.LocalMD.Path, *data.Layers.LocalMD.Content); err != nil {
			platform.PrintFail(w, fmt.Sprintf("CLAUDE.local.md: %v", err))
		} else {
			platform.PrintOK(w, "CLAUDE.local.md restored")
		}
	}

	if scope[LayerAutoMemory] && data.Layers.AutoMemory != nil && len(data.Layers.AutoMemory.Files) > 0 {
		basePath := data.Layers.AutoMemory.BasePath
		if err := os.MkdirAll(basePath, 0755); err != nil {
			platform.PrintFail(w, fmt.Sprintf("Auto-memory dir: %v", err))
		} else {
			for name, content := range data.Layers.AutoMemory.Files {
				path := filepath.Join(basePath, name)
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					platform.PrintFail(w, fmt.Sprintf("Auto-memory %s: %v", name, err))
				}
			}
			platform.PrintOK(w, fmt.Sprintf("Auto-memory: %d file(s) restored", len(data.Layers.AutoMemory.Files)))
		}
	}

	if scope[LayerMemoryMCP] && data.Layers.MemoryMCP != nil && data.Layers.MemoryMCP.Data != nil {
		if data.Layers.MemoryMCP.Provider == "engram" && platform.Exists("engram") {
			tmpFile, err := os.CreateTemp("", "memory-import-*.json")
			if err != nil {
				platform.PrintFail(w, fmt.Sprintf("Memory MCP temp file: %v", err))
			} else {
				tmpFile.Write([]byte(*data.Layers.MemoryMCP.Data))
				tmpFile.Close()
				defer os.Remove(tmpFile.Name())

				if err := platform.Run("engram", "import", tmpFile.Name()); err != nil {
					platform.PrintFail(w, fmt.Sprintf("Memory MCP import: %v", err))
				} else {
					platform.PrintOK(w, "Memory MCP data imported via engram")
				}
			}
		} else {
			platform.PrintWarn(w, fmt.Sprintf("Cannot import Memory MCP: provider %q not available", data.Layers.MemoryMCP.Provider))
		}
	}

	fmt.Fprintln(w)
	return nil
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
