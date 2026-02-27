package sessions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExtractContent(t *testing.T) {
	tests := []struct {
		name string
		raw  json.RawMessage
		want string
	}{
		{
			name: "plain string",
			raw:  json.RawMessage(`"hello world"`),
			want: "hello world",
		},
		{
			name: "string with whitespace",
			raw:  json.RawMessage(`"  hello  "`),
			want: "hello",
		},
		{
			name: "array with text block",
			raw:  json.RawMessage(`[{"type":"text","text":"some text"}]`),
			want: "some text",
		},
		{
			name: "array with multiple text blocks",
			raw:  json.RawMessage(`[{"type":"text","text":"first"},{"type":"text","text":"second"}]`),
			want: "first\nsecond",
		},
		{
			name: "array with non-text blocks filtered",
			raw:  json.RawMessage(`[{"type":"tool_use","text":""},{"type":"text","text":"real content"}]`),
			want: "real content",
		},
		{
			name: "empty raw message",
			raw:  json.RawMessage(``),
			want: "",
		},
		{
			name: "null raw message",
			raw:  nil,
			want: "",
		},
		{
			name: "command-name tag filtered",
			raw:  json.RawMessage(`"<command-name>/exit</command-name>"`),
			want: "",
		},
		{
			name: "local-command-stdout filtered",
			raw:  json.RawMessage(`"<local-command-stdout>Goodbye!</local-command-stdout>"`),
			want: "",
		},
		{
			name: "local-command-caveat filtered",
			raw:  json.RawMessage(`"<local-command-caveat>some caveat</local-command-caveat>"`),
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractContent(tt.raw)
			if got != tt.want {
				t.Errorf("extractContent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFirstLine(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "short single line",
			input:  "hello",
			maxLen: 80,
			want:   "hello",
		},
		{
			name:   "multiline returns first",
			input:  "first line\nsecond line\nthird line",
			maxLen: 80,
			want:   "first line",
		},
		{
			name:   "truncates long line",
			input:  "this is a very long line that should be truncated",
			maxLen: 20,
			want:   "this is a very lo...",
		},
		{
			name:   "trims whitespace",
			input:  "  hello  \nworld",
			maxLen: 80,
			want:   "hello",
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 80,
			want:   "",
		},
		{
			name:   "exactly at max length",
			input:  "12345",
			maxLen: 5,
			want:   "12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := firstLine(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("firstLine(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
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

func TestDecodeProjectPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"-Users-lam-project", "/Users/lam/project"},
		{"-tmp-test", "/tmp/test"},
		{"relative-path", "relative/path"},
	}

	for _, tt := range tests {
		got := decodeProjectPath(tt.input)
		if got != tt.want {
			t.Errorf("decodeProjectPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// writeTestSession creates a minimal JSONL session file for testing.
func writeTestSession(t *testing.T, dir, id string, messages []record) string {
	t.Helper()
	path := filepath.Join(dir, id+".jsonl")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	// Write a file-history-snapshot first (as real sessions do)
	_, _ = f.WriteString(`{"type":"file-history-snapshot","messageId":"snap-1"}` + "\n")

	enc := json.NewEncoder(f)
	for i := range messages {
		if err := enc.Encode(&messages[i]); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func makeUserRecord(content, ts, cwd string, isMeta bool) record {
	raw, _ := json.Marshal(content)
	return record{
		Type:      "user",
		Timestamp: ts,
		CWD:       cwd,
		IsMeta:    isMeta,
		Message: struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		}{
			Role:    "user",
			Content: raw,
		},
	}
}

func TestParseSessionMeta(t *testing.T) {
	dir := t.TempDir()
	id := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	messages := []record{
		makeUserRecord("first real prompt", "2026-02-24T10:00:00.000Z", "/home/user/project", false),
		makeUserRecord("second prompt", "2026-02-24T10:05:00.000Z", "/home/user/project", false),
	}
	path := writeTestSession(t, dir, id, messages)

	s, err := parseSessionMeta(path, id, "fallback-project")
	if err != nil {
		t.Fatal(err)
	}

	if s.ID != id {
		t.Errorf("ID = %q, want %q", s.ID, id)
	}
	if s.Title != "first real prompt" {
		t.Errorf("Title = %q, want %q", s.Title, "first real prompt")
	}
	if s.Project != "/home/user/project" {
		t.Errorf("Project = %q, want %q", s.Project, "/home/user/project")
	}
	wantTime, _ := time.Parse(time.RFC3339Nano, "2026-02-24T10:00:00.000Z")
	if !s.StartTime.Equal(wantTime) {
		t.Errorf("StartTime = %v, want %v", s.StartTime, wantTime)
	}
}

func TestParseSessionMeta_SkipsMetaMessages(t *testing.T) {
	dir := t.TempDir()
	id := "11111111-2222-3333-4444-555555555555"

	messages := []record{
		makeUserRecord("<command-name>/clear</command-name>", "2026-02-24T09:00:00.000Z", "/home/user/project", false),
		makeUserRecord("the real first prompt", "2026-02-24T09:01:00.000Z", "/home/user/project", false),
	}
	path := writeTestSession(t, dir, id, messages)

	s, err := parseSessionMeta(path, id, "fallback")
	if err != nil {
		t.Fatal(err)
	}

	if s.Title != "the real first prompt" {
		t.Errorf("Title = %q, want %q", s.Title, "the real first prompt")
	}
}

func TestParseSessionMeta_EmptySession(t *testing.T) {
	dir := t.TempDir()
	id := "00000000-0000-0000-0000-000000000000"

	// Session with only system messages, no real user prompts
	messages := []record{
		makeUserRecord("<command-name>/exit</command-name>", "2026-02-24T09:00:00.000Z", "/tmp", false),
	}
	path := writeTestSession(t, dir, id, messages)

	s, err := parseSessionMeta(path, id, "fallback")
	if err != nil {
		t.Fatal(err)
	}

	if s.Title != "" {
		t.Errorf("Title = %q, want empty for session with no real prompts", s.Title)
	}
}

func TestParseSessionPrompts(t *testing.T) {
	dir := t.TempDir()
	id := "abcdef01-2345-6789-abcd-ef0123456789"

	messages := []record{
		makeUserRecord("<command-name>/clear</command-name>", "2026-02-24T10:00:00.000Z", "/tmp", false),
		makeUserRecord("first prompt", "2026-02-24T10:01:00.000Z", "/tmp", false),
		makeUserRecord("second prompt", "2026-02-24T10:05:00.000Z", "/tmp", false),
		makeUserRecord("meta message", "2026-02-24T10:06:00.000Z", "/tmp", true),
		makeUserRecord("third prompt", "2026-02-24T10:10:00.000Z", "/tmp", false),
	}
	path := writeTestSession(t, dir, id, messages)

	prompts, _, err := parseSessionPrompts(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(prompts) != 3 {
		t.Fatalf("got %d prompts, want 3", len(prompts))
	}

	expected := []string{"first prompt", "second prompt", "third prompt"}
	for i, want := range expected {
		if prompts[i].Content != want {
			t.Errorf("prompts[%d].Content = %q, want %q", i, prompts[i].Content, want)
		}
	}
}

func TestParseSessionPrompts_ExtractsSlug(t *testing.T) {
	dir := t.TempDir()
	id := "slug-test-0000-0000-000000000000"

	messages := []record{
		makeUserRecord("hello", "2026-02-24T10:00:00.000Z", "/tmp", false),
	}
	// Manually set slug on the first user record
	messages[0].Slug = "happy-bouncing-walrus"

	path := writeTestSession(t, dir, id, messages)

	_, slug, err := parseSessionPrompts(path)
	if err != nil {
		t.Fatal(err)
	}

	if slug != "happy-bouncing-walrus" {
		t.Errorf("slug = %q, want %q", slug, "happy-bouncing-walrus")
	}
}

func TestScanProjectSessions(t *testing.T) {
	dir := t.TempDir()

	// Create two sessions — one real, one with only slash commands
	writeTestSession(t, dir, "real-session-uuid", []record{
		makeUserRecord("implement the auth feature", "2026-02-24T10:00:00.000Z", "/home/user/project", false),
		makeUserRecord("add tests too", "2026-02-24T10:05:00.000Z", "/home/user/project", false),
	})
	writeTestSession(t, dir, "empty-session-uuid", []record{
		makeUserRecord("<command-name>/exit</command-name>", "2026-02-24T09:00:00.000Z", "/tmp", false),
	})

	sessions, err := scanProjectSessions(dir, "test-project")
	if err != nil {
		t.Fatal(err)
	}

	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1 (empty session should be filtered)", len(sessions))
	}

	if sessions[0].Title != "implement the auth feature" {
		t.Errorf("Title = %q, want %q", sessions[0].Title, "implement the auth feature")
	}
}

func TestScanProjectSessions_IgnoresSubdirectories(t *testing.T) {
	dir := t.TempDir()

	// Create a real session
	writeTestSession(t, dir, "main-session", []record{
		makeUserRecord("main prompt", "2026-02-24T10:00:00.000Z", "/tmp", false),
	})

	// Create a subdirectory (subagent dir) — should be ignored
	subDir := filepath.Join(dir, "main-session")
	_ = os.MkdirAll(filepath.Join(subDir, "subagents"), 0755)

	sessions, err := scanProjectSessions(dir, "test-project")
	if err != nil {
		t.Fatal(err)
	}

	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}
}
