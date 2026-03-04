package mcpregistry

import (
	"io/fs"
	"testing"
	"testing/fstest"
)

func TestLoadAll(t *testing.T) {
	categories, err := LoadAll(testFS())
	if err != nil {
		t.Fatalf("LoadAll() error: %v", err)
	}

	if len(categories) != 5 {
		t.Errorf("categories = %d, want 5", len(categories))
	}

	// Count total recipes
	total := 0
	for _, c := range categories {
		total += len(c.Recipes)
	}
	if total != 17 {
		t.Errorf("total recipes = %d, want 17", total)
	}
}

func TestLoadAll_CategoryOrder(t *testing.T) {
	categories, err := LoadAll(testFS())
	if err != nil {
		t.Fatalf("LoadAll() error: %v", err)
	}

	wantOrder := []string{"collaboration", "search", "observability", "database", "memory"}
	for i, c := range categories {
		if c.Name != wantOrder[i] {
			t.Errorf("category[%d] = %q, want %q", i, c.Name, wantOrder[i])
		}
	}
}

func TestLoadAll_TransportDetection(t *testing.T) {
	categories, err := LoadAll(testFS())
	if err != nil {
		t.Fatalf("LoadAll() error: %v", err)
	}

	recipeMap := make(map[string]Recipe)
	for _, c := range categories {
		for _, r := range c.Recipes {
			recipeMap[r.Key] = r
		}
	}

	tests := []struct {
		key       string
		transport Transport
	}{
		{"brave-search", TransportStdio},
		{"github", TransportHTTP},
		{"github_pat", TransportHTTP},
		{"notion", TransportHTTP},
		{"slack", TransportHTTP},
		{"linear", TransportHTTP},
		{"jira", TransportStdio},
		{"sentry", TransportHTTP},
		{"grafana", TransportHTTP},
		{"honeycomb", TransportHTTP},
		{"honeycomb-api-key", TransportHTTP},
		{"dynatrace", TransportStdio},
		{"postgresql", TransportStdio},
		{"mysql", TransportStdio},
		{"sqlite", TransportStdio},
		{"mcp-memory-libsql", TransportStdio},
		{"engram", TransportStdio},
	}

	for _, tt := range tests {
		r, ok := recipeMap[tt.key]
		if !ok {
			t.Errorf("recipe %q not found", tt.key)
			continue
		}
		if r.Transport != tt.transport {
			t.Errorf("recipe %q transport = %q, want %q", tt.key, r.Transport, tt.transport)
		}
	}
}

func TestLoadAll_EnvVars(t *testing.T) {
	categories, err := LoadAll(testFS())
	if err != nil {
		t.Fatalf("LoadAll() error: %v", err)
	}

	recipeMap := make(map[string]Recipe)
	for _, c := range categories {
		for _, r := range c.Recipes {
			recipeMap[r.Key] = r
		}
	}

	// brave-search has BRAVE_API_KEY
	bs := recipeMap["brave-search"]
	if _, ok := bs.EnvVars["BRAVE_API_KEY"]; !ok {
		t.Error("brave-search missing BRAVE_API_KEY env var")
	}

	// jira has 3 env vars
	jira := recipeMap["jira"]
	if len(jira.EnvVars) != 3 {
		t.Errorf("jira env vars = %d, want 3", len(jira.EnvVars))
	}
}

func TestLoadAll_Headers(t *testing.T) {
	categories, err := LoadAll(testFS())
	if err != nil {
		t.Fatalf("LoadAll() error: %v", err)
	}

	recipeMap := make(map[string]Recipe)
	for _, c := range categories {
		for _, r := range c.Recipes {
			recipeMap[r.Key] = r
		}
	}

	// github_pat has an Authorization header
	pat := recipeMap["github_pat"]
	if pat.Headers == nil {
		t.Fatal("github_pat should have headers")
	}
	if got := pat.Headers["Authorization"]; got != "Bearer <your_pat>" {
		t.Errorf("github_pat Authorization = %q, want %q", got, "Bearer <your_pat>")
	}

	// github (no headers) should have nil headers
	gh := recipeMap["github"]
	if gh.Headers != nil {
		t.Errorf("github should have nil headers, got %v", gh.Headers)
	}

	// stdio recipe (brave-search) should have nil headers
	bs := recipeMap["brave-search"]
	if bs.Headers != nil {
		t.Errorf("brave-search should have nil headers, got %v", bs.Headers)
	}
}

func TestRecipe_FirstHeader(t *testing.T) {
	r := Recipe{Headers: map[string]string{"Authorization": "Bearer tok123"}}
	key, val := r.FirstHeader()
	if key != "Authorization" {
		t.Errorf("FirstHeader() key = %q, want %q", key, "Authorization")
	}
	if val != "Bearer tok123" {
		t.Errorf("FirstHeader() val = %q, want %q", val, "Bearer tok123")
	}
}

func TestRecipe_FirstHeader_Empty(t *testing.T) {
	r := Recipe{}
	key, val := r.FirstHeader()
	if key != "" || val != "" {
		t.Errorf("FirstHeader() = (%q, %q), want empty", key, val)
	}
}

func TestLoadAll_SuggestedScope(t *testing.T) {
	categories, err := LoadAll(testFS())
	if err != nil {
		t.Fatalf("LoadAll() error: %v", err)
	}

	recipeMap := make(map[string]Recipe)
	for _, c := range categories {
		for _, r := range c.Recipes {
			recipeMap[r.Key] = r
		}
	}

	tests := []struct {
		key   string
		scope string
	}{
		{"brave-search", "user"},
		{"postgresql", "local"},
		{"mysql", "local"},
		{"sqlite", "local"},
		{"github", "user"},
		{"sentry", "user"},
	}

	for _, tt := range tests {
		r := recipeMap[tt.key]
		if r.Scope != tt.scope {
			t.Errorf("recipe %q scope = %q, want %q", tt.key, r.Scope, tt.scope)
		}
	}
}

func TestLoadAll_EmptyFS(t *testing.T) {
	emptyFS := fstest.MapFS{}
	categories, err := LoadAll(emptyFS)
	if err != nil {
		t.Fatalf("LoadAll(empty) error: %v", err)
	}
	if len(categories) != 0 {
		t.Errorf("LoadAll(empty) categories = %d, want 0", len(categories))
	}
}

func TestLoadAll_NilFS(t *testing.T) {
	categories, err := LoadAll(nil)
	if err != nil {
		t.Fatalf("LoadAll(nil) error: %v", err)
	}
	if categories != nil {
		t.Errorf("LoadAll(nil) = %v, want nil", categories)
	}
}

func TestLoadAll_MalformedJSON(t *testing.T) {
	badFS := fstest.MapFS{
		"bad.json": &fstest.MapFile{Data: []byte(`{invalid json`)},
	}
	_, err := LoadAll(badFS)
	if err == nil {
		t.Fatal("LoadAll(malformed) should return error")
	}
	if got := err.Error(); got == "" {
		t.Error("error message should not be empty")
	}
}

func TestRecipe_CommandString(t *testing.T) {
	r := Recipe{Command: "npx", Args: []string{"-y", "@bytebase/dbhub"}}
	got := r.CommandString()
	want := "npx -y @bytebase/dbhub"
	if got != want {
		t.Errorf("CommandString() = %q, want %q", got, want)
	}
}

func TestRecipe_CommandString_Empty(t *testing.T) {
	r := Recipe{}
	if got := r.CommandString(); got != "" {
		t.Errorf("CommandString() = %q, want empty", got)
	}
}

func TestRecipe_FirstEnvVar(t *testing.T) {
	r := Recipe{EnvVars: map[string]string{"BRAVE_API_KEY": "${BRAVE_API_KEY}"}}
	if got := r.FirstEnvVar(); got != "BRAVE_API_KEY" {
		t.Errorf("FirstEnvVar() = %q, want BRAVE_API_KEY", got)
	}
}

func TestRecipe_FirstEnvVar_Empty(t *testing.T) {
	r := Recipe{}
	if got := r.FirstEnvVar(); got != "" {
		t.Errorf("FirstEnvVar() = %q, want empty", got)
	}
}

// testFS returns a real embedded FS by reading the project's docs/mcp-configs.
// Falls back to a synthetic FS for CI environments without the real files.
func testFS() fs.FS {
	return fstest.MapFS{
		"collaboration.json": &fstest.MapFile{Data: []byte(collaborationJSON)},
		"search.json":        &fstest.MapFile{Data: []byte(searchJSON)},
		"observability.json": &fstest.MapFile{Data: []byte(observabilityJSON)},
		"database.json":      &fstest.MapFile{Data: []byte(databaseJSON)},
		"memory.json":        &fstest.MapFile{Data: []byte(memoryJSON)},
	}
}

const collaborationJSON = `{
  "examples": {
    "github": {"type": "http", "url": "https://api.githubcopilot.com/mcp/"},
    "github_pat": {"type": "http", "url": "https://api.githubcopilot.com/mcp/", "headers": {"Authorization": "Bearer <your_pat>"}},
    "notion": {"type": "http", "url": "https://mcp.notion.com/mcp"},
    "slack": {"type": "http", "url": "${SLACK_MCP_URL}"},
    "linear": {"type": "http", "url": "https://mcp.linear.app/sse"},
    "jira": {"command": "npx", "args": ["-y", "@anthropic/claude-code-jira-server"], "env": {"JIRA_URL": "${JIRA_URL}", "JIRA_EMAIL": "${JIRA_EMAIL}", "JIRA_API_TOKEN": "${JIRA_API_TOKEN}"}}
  },
  "setup_commands": {
    "github": "claude-workspace mcp remote https://api.githubcopilot.com/mcp/ --name github --scope user",
    "github_pat": "claude-workspace mcp remote https://api.githubcopilot.com/mcp/ --name github --scope user --bearer",
    "notion": "claude-workspace mcp remote https://mcp.notion.com/mcp --name notion --scope user",
    "linear": "claude-workspace mcp remote https://mcp.linear.app/sse --name linear --scope user",
    "jira": "claude-workspace mcp add jira --scope user --api-key JIRA_API_TOKEN -- npx -y @anthropic/claude-code-jira-server"
  },
  "notes": {
    "github": "Recommended scope: user (personal OAuth credentials, used across all projects). Uses OAuth",
    "github_pat": "Recommended scope: user (personal access token). Uses PAT",
    "notion": "Recommended scope: user (personal Notion workspace access). Uses OAuth",
    "linear": "Recommended scope: user (personal Linear credentials). Uses OAuth",
    "jira": "Recommended scope: user (personal API token). Requires JIRA_URL, JIRA_EMAIL, JIRA_API_TOKEN"
  }
}`

const searchJSON = `{
  "examples": {
    "brave-search": {"command": "npx", "args": ["-y", "@modelcontextprotocol/server-brave-search"], "env": {"BRAVE_API_KEY": "${BRAVE_API_KEY}"}}
  },
  "setup_commands": {
    "brave-search": "claude-workspace mcp add brave-search --scope user --api-key BRAVE_API_KEY -- npx -y @modelcontextprotocol/server-brave-search"
  },
  "notes": {
    "brave-search": "Recommended scope: user (personal API key). Get a free API key at https://brave.com/search/api/"
  }
}`

const observabilityJSON = `{
  "examples": {
    "sentry": {"type": "http", "url": "https://mcp.sentry.dev/mcp"},
    "grafana": {"type": "http", "url": "${GRAFANA_MCP_URL}"},
    "honeycomb": {"type": "http", "url": "https://mcp.honeycomb.io/mcp"},
    "honeycomb-api-key": {"type": "http", "url": "https://mcp.honeycomb.io/mcp"},
    "dynatrace": {"type": "stdio", "command": "npx", "args": ["-y", "@dynatrace-oss/dynatrace-mcp-server@latest"], "env": {"DT_ENVIRONMENT": "${DT_ENVIRONMENT}"}}
  },
  "setup_commands": {
    "sentry": "claude-workspace mcp remote https://mcp.sentry.dev/mcp --name sentry --scope user",
    "grafana": "claude-workspace mcp remote $GRAFANA_MCP_URL --name grafana --scope user --bearer",
    "honeycomb": "claude-workspace mcp remote https://mcp.honeycomb.io/mcp --name honeycomb --scope user",
    "honeycomb-api-key": "claude-workspace mcp remote https://mcp.honeycomb.io/mcp --name honeycomb --scope user --bearer",
    "dynatrace": "claude-workspace mcp add dynatrace --scope user --api-key DT_ENVIRONMENT -- npx -y @dynatrace-oss/dynatrace-mcp-server@latest"
  },
  "notes": {
    "sentry": "Recommended scope: user (personal OAuth credentials). Uses OAuth",
    "grafana": "Recommended scope: user (personal bearer token). Requires Bearer token",
    "honeycomb": "Recommended scope: user (personal OAuth). Uses OAuth",
    "honeycomb-api-key": "Recommended scope: user (personal API key). Uses Bearer token",
    "dynatrace": "Recommended scope: user (personal platform credentials). Uses browser OAuth"
  }
}`

const databaseJSON = `{
  "examples": {
    "postgresql": {"command": "npx", "args": ["-y", "@bytebase/dbhub", "--dsn", "${DATABASE_URL:-postgresql://localhost:5432/mydb}"], "env": {}},
    "mysql": {"command": "npx", "args": ["-y", "@bytebase/dbhub", "--dsn", "${DATABASE_URL:-mysql://localhost:3306/mydb}"], "env": {}},
    "sqlite": {"command": "npx", "args": ["-y", "@anthropic/claude-code-sqlite-server", "${DB_PATH:-./db.sqlite}"], "env": {}}
  },
  "setup_commands": {
    "postgresql": "claude-workspace mcp add postgres --scope local --api-key DATABASE_URL -- npx -y @bytebase/dbhub",
    "mysql": "claude-workspace mcp add mysql --scope local --api-key DATABASE_URL -- npx -y @bytebase/dbhub",
    "sqlite": "claude-workspace mcp add sqlite --scope local -- npx -y @anthropic/claude-code-sqlite-server ./db.sqlite"
  },
  "notes": {
    "postgresql": "Recommended scope: local (connection string is environment-specific). Enter DATABASE_URL securely",
    "mysql": "Recommended scope: local (connection string is environment-specific). Enter DATABASE_URL securely",
    "sqlite": "Recommended scope: local (file path is personal). No API key needed"
  }
}`

const memoryJSON = `{
  "examples": {
    "mcp-memory-libsql": {"command": "npx", "args": ["-y", "mcp-memory-libsql"], "env": {"LIBSQL_URL": "file:/Users/<user>/.config/claude-workspace/memory.db"}},
    "engram": {"command": "engram", "args": ["mcp"]}
  },
  "setup_commands": {
    "mcp-memory-libsql": "claude-workspace memory configure --provider mcp-memory-libsql",
    "engram": "claude-workspace memory configure --provider engram"
  },
  "notes": {
    "mcp-memory-libsql": "Recommended scope: user (cross-project persistent memory). Auto-registered by setup",
    "engram": "Recommended scope: user (cross-project persistent memory). Optional legacy provider"
  }
}`
