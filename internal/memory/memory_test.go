package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCountLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty", "", 0},
		{"single line no newline", "hello", 1},
		{"single line with newline", "hello\n", 1},
		{"two lines", "hello\nworld", 2},
		{"two lines trailing newline", "hello\nworld\n", 2},
		{"three lines", "a\nb\nc\n", 3},
		{"blank lines", "\n\n\n", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countLines(tt.input)
			if got != tt.want {
				t.Errorf("countLines(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestEncodeProjectPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/Users/lam/project", "-Users-lam-project"},
		{"/tmp/test", "-tmp-test"},
		{"relative/path", "relative-path"},
	}

	for _, tt := range tests {
		got := encodeProjectPath(tt.input)
		if got != tt.want {
			t.Errorf("encodeProjectPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseScope(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[LayerName]bool
	}{
		{
			name:  "single scope",
			input: "user",
			want:  map[LayerName]bool{LayerUserClaudeMD: true},
		},
		{
			name:  "multiple scopes",
			input: "auto,mcp",
			want:  map[LayerName]bool{LayerAutoMemory: true, LayerMemoryMCP: true},
		},
		{
			name:  "all scope",
			input: "all",
			want: map[LayerName]bool{
				LayerUserClaudeMD:    true,
				LayerProjectClaudeMD: true,
				LayerLocalMD:         true,
				LayerAutoMemory:      true,
				LayerMemoryMCP:       true,
			},
		},
		{
			name:  "empty scope",
			input: "",
			want:  map[LayerName]bool{},
		},
		{
			name:  "with spaces",
			input: "user, project",
			want:  map[LayerName]bool{LayerUserClaudeMD: true, LayerProjectClaudeMD: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseScope(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("ParseScope(%q) has %d entries, want %d", tt.input, len(got), len(tt.want))
				return
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("ParseScope(%q)[%s] = %v, want %v", tt.input, k, got[k], v)
				}
			}
		})
	}
}

func TestShortenHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	tests := []struct {
		input string
		want  string
	}{
		{filepath.Join(home, ".claude", "CLAUDE.md"), "~/.claude/CLAUDE.md"},
		{"/tmp/other/path", "/tmp/other/path"},
		{home, "~"},
	}

	for _, tt := range tests {
		got := shortenHome(tt.input)
		if got != tt.want {
			t.Errorf("shortenHome(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDiscoverFileLayer(t *testing.T) {
	// Test with non-existent file
	l := discoverFileLayer(LayerUserClaudeMD, "User CLAUDE.md", "/nonexistent/path")
	if l.Exists {
		t.Error("expected non-existent file to have Exists=false")
	}
	if l.Lines != 0 {
		t.Errorf("expected 0 lines, got %d", l.Lines)
	}

	// Test with existing file
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte("line1\nline2\nline3\n"), 0644)

	l = discoverFileLayer(LayerLocalMD, "Test MD", path)
	if !l.Exists {
		t.Error("expected existing file to have Exists=true")
	}
	if l.Lines != 3 {
		t.Errorf("expected 3 lines, got %d", l.Lines)
	}
}

func TestDiscoverAutoMemory(t *testing.T) {
	// Test with non-existent directory
	l := discoverAutoMemory("/nonexistent/path")
	if l.Exists {
		t.Error("expected non-existent dir to have Exists=false")
	}

	// Test with existing directory containing files
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "MEMORY.md"), []byte("# Memory\nSome notes\n"), 0644)
	os.WriteFile(filepath.Join(dir, "debugging.md"), []byte("# Debug\n"), 0644)

	l = discoverAutoMemory(dir)
	if !l.Exists {
		t.Error("expected existing dir to have Exists=true")
	}
	if len(l.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(l.Files))
	}
	if _, ok := l.Files["MEMORY.md"]; !ok {
		t.Error("expected MEMORY.md in files")
	}
}

func TestExportImportRoundtrip(t *testing.T) {
	// Create a minimal export data structure and verify it serializes/deserializes
	content := "# Test content"
	data := ExportData{
		Version:    1,
		ExportedAt: "2026-02-24T00:00:00Z",
		Platform:   "claude-workspace",
		Layers: ExportLayers{
			UserClaudeMD: &ExportFile{
				Path:    "~/.claude/CLAUDE.md",
				Content: &content,
			},
			AutoMemory: &ExportAutoMem{
				BasePath: "~/.claude/projects/-test/memory/",
				Files: map[string]string{
					"MEMORY.md": "# Memory\ntest\n",
				},
			},
			MemoryMCP: &ExportMCP{
				Provider: "engram",
				Data:     nil,
			},
		},
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	var parsed ExportData
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatal(err)
	}

	if parsed.Version != 1 {
		t.Errorf("version = %d, want 1", parsed.Version)
	}
	if parsed.Layers.UserClaudeMD == nil || parsed.Layers.UserClaudeMD.Content == nil {
		t.Fatal("expected user CLAUDE.md content")
	}
	if *parsed.Layers.UserClaudeMD.Content != content {
		t.Errorf("content = %q, want %q", *parsed.Layers.UserClaudeMD.Content, content)
	}
	if parsed.Layers.AutoMemory == nil || len(parsed.Layers.AutoMemory.Files) != 1 {
		t.Error("expected auto memory with 1 file")
	}
}

func TestProviderSuffix(t *testing.T) {
	tests := []struct {
		name       string
		layer      Layer
		wantSuffix string
	}{
		{
			name:       "non-MCP layer returns empty",
			layer:      Layer{Name: LayerAutoMemory, Provider: "engram"},
			wantSuffix: "",
		},
		{
			name:       "MCP layer with real provider returns suffix",
			layer:      Layer{Name: LayerMemoryMCP, Provider: "engram"},
			wantSuffix: " — engram",
		},
		{
			name:       "MCP layer with provider=none returns empty",
			layer:      Layer{Name: LayerMemoryMCP, Provider: "none"},
			wantSuffix: "",
		},
		{
			name:       "MCP layer with empty provider returns empty",
			layer:      Layer{Name: LayerMemoryMCP, Provider: ""},
			wantSuffix: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := providerSuffix(tt.layer)
			if got != tt.wantSuffix {
				t.Errorf("providerSuffix(%+v) = %q, want %q", tt.layer, got, tt.wantSuffix)
			}
		})
	}
}

func TestDetectProvider(t *testing.T) {
	// Test with no config and no engram data
	dir := t.TempDir()
	provider, _ := detectProvider(dir)
	if provider != "none" {
		t.Errorf("expected provider 'none', got %q", provider)
	}

	// Test with engram in claude.json
	config := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"engram": map[string]interface{}{
				"command": "engram",
				"args":    []string{"mcp"},
			},
		},
	}
	configData, _ := json.Marshal(config)
	os.WriteFile(filepath.Join(dir, ".claude.json"), configData, 0644)

	provider, dataPath := detectProvider(dir)
	if provider != "engram" {
		t.Errorf("expected provider 'engram', got %q", provider)
	}
	if dataPath == "" {
		t.Error("expected non-empty data path")
	}
}

// --- layers.go ---

func TestAutoMemoryDir(t *testing.T) {
	got := autoMemoryDir("/Users/lam", "/Users/lam/git/myproject")
	want := "/Users/lam/.claude/projects/-Users-lam-git-myproject/memory"
	if got != want {
		t.Errorf("autoMemoryDir = %q, want %q", got, want)
	}
}

// --- export.go ---

func TestReadFileContent(t *testing.T) {
	// Non-existent file returns empty string.
	got := readFileContent("/nonexistent/path/file.md")
	if got != "" {
		t.Errorf("readFileContent(nonexistent) = %q, want empty", got)
	}

	// Existing file returns its full content.
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	content := "# Memory\nSome notes\n"
	os.WriteFile(path, []byte(content), 0644)

	got = readFileContent(path)
	if got != content {
		t.Errorf("readFileContent = %q, want %q", got, content)
	}
}

func TestWriteFileContent(t *testing.T) {
	dir := t.TempDir()
	content := "# Test\nline1\nline2\n"

	// Write to a file in an existing directory.
	path := filepath.Join(dir, "output.md")
	if err := writeFileContent(path, content); err != nil {
		t.Fatalf("writeFileContent: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading written file: %v", err)
	}
	if string(data) != content {
		t.Errorf("written content = %q, want %q", string(data), content)
	}

	// Write to a file whose parent directories don't exist yet.
	deepPath := filepath.Join(dir, "nested", "deep", "file.md")
	if err := writeFileContent(deepPath, content); err != nil {
		t.Fatalf("writeFileContent with nested dirs: %v", err)
	}
	if _, err := os.Stat(deepPath); err != nil {
		t.Errorf("expected file to exist at %s: %v", deepPath, err)
	}
}

func TestImportMemoryUnsupportedVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "export.json")

	data := ExportData{Version: 99, Platform: "test", Layers: ExportLayers{}}
	raw, _ := json.Marshal(data)
	os.WriteFile(path, raw, 0644)

	err := importMemory(path, ParseScope("all"), false)
	if err == nil {
		t.Fatal("expected error for unsupported version, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported export version") {
		t.Errorf("error = %q, want message containing 'unsupported export version'", err.Error())
	}
}

func TestImportMemoryDryRun(t *testing.T) {
	dir := t.TempDir()
	exportPath := filepath.Join(dir, "export.json")
	restoreDir := filepath.Join(dir, "memory")

	data := ExportData{
		Version:  1,
		Platform: "test",
		Layers: ExportLayers{
			AutoMemory: &ExportAutoMem{
				BasePath: restoreDir,
				Files:    map[string]string{"MEMORY.md": "# Memory\nnotes\n"},
			},
		},
	}
	raw, _ := json.MarshalIndent(data, "", "  ")
	os.WriteFile(exportPath, raw, 0644)

	// confirm=false: preview only, no files should be written.
	if err := importMemory(exportPath, ParseScope("auto"), false); err != nil {
		t.Fatalf("importMemory dry run: %v", err)
	}
	if _, err := os.Stat(restoreDir); err == nil {
		t.Error("restore dir should not exist after dry run")
	}
}

func TestImportMemoryAutoMemoryRestore(t *testing.T) {
	dir := t.TempDir()
	exportPath := filepath.Join(dir, "export.json")
	restoreDir := filepath.Join(dir, "memory")

	files := map[string]string{
		"MEMORY.md":    "# Memory\nnotes\n",
		"debugging.md": "# Debug\npatterns\n",
	}
	data := ExportData{
		Version:  1,
		Platform: "test",
		Layers: ExportLayers{
			AutoMemory: &ExportAutoMem{
				BasePath: restoreDir,
				Files:    files,
			},
		},
	}
	raw, _ := json.MarshalIndent(data, "", "  ")
	os.WriteFile(exportPath, raw, 0644)

	if err := importMemory(exportPath, ParseScope("auto"), true); err != nil {
		t.Fatalf("importMemory: %v", err)
	}

	for name, want := range files {
		got, err := os.ReadFile(filepath.Join(restoreDir, name))
		if err != nil {
			t.Errorf("expected file %s to be restored: %v", name, err)
			continue
		}
		if string(got) != want {
			t.Errorf("file %s content = %q, want %q", name, string(got), want)
		}
	}
}

func TestImportMemoryScopeFiltering(t *testing.T) {
	dir := t.TempDir()
	exportPath := filepath.Join(dir, "export.json")
	autoDir := filepath.Join(dir, "memory")

	userContent := "# User CLAUDE.md"
	data := ExportData{
		Version:  1,
		Platform: "test",
		Layers: ExportLayers{
			UserClaudeMD: &ExportFile{
				Path:    filepath.Join(dir, "user-claude.md"),
				Content: &userContent,
			},
			AutoMemory: &ExportAutoMem{
				BasePath: autoDir,
				Files:    map[string]string{"MEMORY.md": "# Memory\n"},
			},
		},
	}
	raw, _ := json.MarshalIndent(data, "", "  ")
	os.WriteFile(exportPath, raw, 0644)

	// Only restore auto scope — user CLAUDE.md should not be written.
	if err := importMemory(exportPath, ParseScope("auto"), true); err != nil {
		t.Fatalf("importMemory: %v", err)
	}

	if _, err := os.Stat(data.Layers.UserClaudeMD.Path); err == nil {
		t.Error("user CLAUDE.md should not have been written when scope=auto")
	}
	if _, err := os.Stat(filepath.Join(autoDir, "MEMORY.md")); err != nil {
		t.Error("auto-memory MEMORY.md should have been written")
	}
}
