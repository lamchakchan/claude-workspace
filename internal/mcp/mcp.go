// Package mcp implements the "mcp" command for adding, listing, and connecting
// to MCP (Model Context Protocol) servers with support for stdio, HTTP, and SSE
// transports and secure credential handling.
package mcp

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/lamchakchan/claude-workspace/internal/platform"
	"golang.org/x/term"
)

const (
	flagHeader    = "--header"
	flagClientSec = "--client-secret"

	transportStdio = "stdio"
	transportHTTP  = "http"
	transportSSE   = "sse"
)

type addConfig struct {
	Name               string
	Scope              string
	Transport          string
	EnvVars            map[string]string
	Headers            []string
	CommandArgs        []string
	McpURL             string
	APIKeyEnvVar       string
	PromptBearer       bool
	UseOAuth           bool
	ClientID           string
	PromptClientSecret bool
	ClientSecret       string
}

type remoteConfig struct {
	Name               string
	McpURL             string
	Scope              string
	Headers            []string
	PromptBearer       bool
	UseOAuth           bool
	ClientID           string
	PromptClientSecret bool
	ClientSecret       string
	Transport          string
}

func promptSecret(prompt string) (string, error) {
	platform.PrintPrompt(os.Stdout, prompt)

	fd := int(syscall.Stdin)
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		// Fall back to hidden input if raw mode is unavailable.
		password, err := term.ReadPassword(fd)
		fmt.Println()
		if err != nil {
			return "", fmt.Errorf("reading secret: %w", err)
		}
		return strings.TrimSpace(string(password)), nil
	}
	defer func() { _ = term.Restore(fd, oldState) }()

	var buf []byte
	b := make([]byte, 1)
	for {
		_, err := os.Stdin.Read(b)
		if err != nil {
			fmt.Println()
			return "", fmt.Errorf("reading secret: %w", err)
		}
		switch b[0] {
		case '\r', '\n': // Enter
			fmt.Println()
			return strings.TrimSpace(string(buf)), nil
		case '\x03': // Ctrl+C
			fmt.Println()
			return "", fmt.Errorf("interrupted")
		case '\x04': // Ctrl+D / EOF
			fmt.Println()
			return strings.TrimSpace(string(buf)), nil
		case '\x7f', '\x08': // Backspace / Delete
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
				fmt.Print("\b \b")
			}
		default:
			if b[0] >= 0x20 && b[0] < 0x7f { // printable ASCII
				buf = append(buf, b[0])
				fmt.Print("*")
			}
		}
	}
}

func parseAddArgs(args []string) (*addConfig, error) {
	if len(args) < 1 {
		printMcpAddHelp()
		return nil, fmt.Errorf("server name is required")
	}

	cfg := &addConfig{
		Name:    args[0],
		Scope:   "local",
		EnvVars: map[string]string{},
	}

	i := 1
	for i < len(args) {
		if args[i] == "--" {
			cfg.CommandArgs = args[i+1:]
			break
		}

		switch args[i] {
		case "--scope":
			i++
			if i < len(args) {
				cfg.Scope = args[i]
			}
		case "--transport":
			i++
			if i < len(args) {
				cfg.Transport = args[i]
			}
		case "--env":
			i++
			if i < len(args) {
				envPair := args[i]
				eqIdx := strings.Index(envPair, "=")
				if eqIdx > 0 {
					cfg.EnvVars[envPair[:eqIdx]] = envPair[eqIdx+1:]
				}
			}
		case "--api-key":
			i++
			if i < len(args) {
				cfg.APIKeyEnvVar = args[i]
			}
		case flagHeader:
			i++
			if i < len(args) {
				cfg.Headers = append(cfg.Headers, args[i])
			}
		case "--bearer":
			cfg.PromptBearer = true
		case "--oauth":
			cfg.UseOAuth = true
		case "--client-id":
			i++
			if i < len(args) {
				cfg.ClientID = args[i]
			}
		case flagClientSec:
			cfg.PromptClientSecret = true
		default:
			if strings.HasPrefix(args[i], "http://") || strings.HasPrefix(args[i], "https://") {
				cfg.McpURL = args[i]
			} else {
				cfg.CommandArgs = args[i:]
				i = len(args)
				continue
			}
		}
		i++
	}

	// Determine transport
	if cfg.Transport == "" {
		if cfg.McpURL != "" {
			cfg.Transport = transportHTTP
		} else {
			cfg.Transport = transportStdio
		}
	}

	return cfg, nil
}

func parseRemoteArgs(mcpURL string, extraArgs []string) (*remoteConfig, error) {
	if mcpURL == "" {
		printMcpRemoteHelp()
		return nil, fmt.Errorf("URL is required")
	}

	cfg := &remoteConfig{
		McpURL: mcpURL,
		Scope:  "user",
	}

	for i := 0; i < len(extraArgs); i++ {
		switch extraArgs[i] {
		case "--name":
			i++
			if i < len(extraArgs) {
				cfg.Name = extraArgs[i]
			}
		case "--scope":
			i++
			if i < len(extraArgs) {
				cfg.Scope = extraArgs[i]
			}
		case flagHeader:
			i++
			if i < len(extraArgs) {
				cfg.Headers = append(cfg.Headers, extraArgs[i])
			}
		case "--bearer":
			cfg.PromptBearer = true
		case "--oauth":
			cfg.UseOAuth = true
		case "--client-id":
			i++
			if i < len(extraArgs) {
				cfg.ClientID = extraArgs[i]
			}
		case flagClientSec:
			cfg.PromptClientSecret = true
		}
	}

	if cfg.Name == "" {
		cfg.Name = deriveServerName(mcpURL)
	}

	cfg.Transport = transportHTTP
	if strings.HasSuffix(mcpURL, "/sse") {
		cfg.Transport = transportSSE
	}

	return cfg, nil
}

func deriveServerName(mcpURL string) string {
	u, err := url.Parse(mcpURL)
	if err != nil {
		return "remote-gateway"
	}
	name := u.Hostname()
	name = strings.TrimPrefix(name, "mcp-")
	name = strings.TrimPrefix(name, "mcp.")
	name = strings.TrimSuffix(name, ".com")
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	return b.String()
}

func buildAddClaudeArgs(cfg *addConfig) ([]string, error) {
	claudeArgs := []string{"mcp", "add", "--transport", cfg.Transport, "--scope", cfg.Scope}

	if cfg.ClientID != "" {
		claudeArgs = append(claudeArgs, "--client-id", cfg.ClientID)
	}
	if cfg.ClientSecret != "" {
		claudeArgs = append(claudeArgs, flagClientSec, cfg.ClientSecret)
	}

	claudeArgs = append(claudeArgs, cfg.Name)

	// -e must come after <name> (so the variadic flag can't consume the server name)
	// but before -- <cmd> (so claude treats it as a subprocess env var, not a cmd arg).
	for key, value := range cfg.EnvVars {
		claudeArgs = append(claudeArgs, "-e", key+"="+value)
	}

	if cfg.Transport == transportHTTP || cfg.Transport == transportSSE {
		if cfg.McpURL == "" {
			return nil, fmt.Errorf("URL required for http/sse transport")
		}
		claudeArgs = append(claudeArgs, cfg.McpURL)
	} else if len(cfg.CommandArgs) > 0 {
		claudeArgs = append(claudeArgs, "--")
		claudeArgs = append(claudeArgs, cfg.CommandArgs...)
	}

	// --header is variadic in the Claude CLI and must come after positional args
	// to prevent it from consuming <name> and <url> as header values.
	for _, header := range cfg.Headers {
		claudeArgs = append(claudeArgs, flagHeader, header)
	}

	return claudeArgs, nil
}

func buildRemoteClaudeArgs(cfg *remoteConfig) []string {
	claudeArgs := []string{"mcp", "add", "--transport", cfg.Transport, "--scope", cfg.Scope}

	if cfg.ClientID != "" {
		claudeArgs = append(claudeArgs, "--client-id", cfg.ClientID)
	}
	if cfg.ClientSecret != "" {
		claudeArgs = append(claudeArgs, flagClientSec, cfg.ClientSecret)
	}

	claudeArgs = append(claudeArgs, cfg.Name, cfg.McpURL)

	// --header is variadic in the Claude CLI and must come after positional args
	// to prevent it from consuming <name> and <url> as header values.
	for _, header := range cfg.Headers {
		claudeArgs = append(claudeArgs, flagHeader, header)
	}

	return claudeArgs
}

func maskSensitiveArgs(args []string) []string {
	safeArgs := make([]string, len(args))
	for idx, arg := range args {
		if idx > 0 {
			prev := args[idx-1]
			if prev == "-e" && strings.Contains(arg, "=") {
				eqIdx := strings.Index(arg, "=")
				safeArgs[idx] = arg[:eqIdx] + "=****"
				continue
			}
			if prev == flagHeader && strings.Contains(strings.ToLower(arg), "bearer") {
				safeArgs[idx] = "Authorization: Bearer ****"
				continue
			}
			if prev == flagClientSec {
				safeArgs[idx] = "****"
				continue
			}
		}
		safeArgs[idx] = arg
	}
	return safeArgs
}

// Add adds a local or remote MCP server to the project or user config.
func Add(args []string) error {
	cfg, err := parseAddArgs(args)
	if err != nil {
		return err
	}

	// Secure API key prompting
	if cfg.APIKeyEnvVar != "" {
		fmt.Printf("\nAPI key required for '%s' server.\n", cfg.Name)
		fmt.Printf("The key will be stored as env var: %s\n", cfg.APIKeyEnvVar)
		fmt.Println("Stored in your Claude config (~/.claude.json), NOT in project files.")
		fmt.Println()

		keyValue, err := promptSecret(fmt.Sprintf("Enter %s: ", cfg.APIKeyEnvVar))
		if err != nil {
			return err
		}
		if keyValue == "" {
			return fmt.Errorf("no API key provided")
		}
		cfg.EnvVars[cfg.APIKeyEnvVar] = keyValue
	}

	if cfg.PromptBearer {
		fmt.Printf("\nBearer token required for '%s' server.\n", cfg.Name)
		fmt.Println("Stored in your Claude config (~/.claude.json), NOT in project files.")
		fmt.Println()

		token, err := promptSecret("Enter Bearer token: ")
		if err != nil {
			return err
		}
		if token == "" {
			return fmt.Errorf("no token provided")
		}
		cfg.Headers = append(cfg.Headers, "Authorization: Bearer "+token)
	}

	if cfg.PromptClientSecret {
		fmt.Printf("\nOAuth client secret required for '%s' server.\n", cfg.Name)
		fmt.Println("Stored in your Claude config (~/.claude.json), NOT in project files.")
		fmt.Println()

		secret, err := promptSecret("Enter OAuth client secret: ")
		if err != nil {
			return err
		}
		if secret == "" {
			return fmt.Errorf("no client secret provided")
		}
		cfg.ClientSecret = secret
	}

	claudeArgs, err := buildAddClaudeArgs(cfg)
	if err != nil {
		return err
	}

	fmt.Printf("Adding MCP server '%s' (%s, scope: %s)...\n", cfg.Name, cfg.Transport, cfg.Scope)

	safeArgs := maskSensitiveArgs(claudeArgs)
	fmt.Printf("  > claude %s\n\n", strings.Join(safeArgs, " "))

	exitCode, err := platform.RunSpawn("claude", claudeArgs...)
	if err != nil {
		return fmt.Errorf("could not run 'claude' command. Is Claude Code installed?")
	}

	if exitCode == 0 {
		fmt.Fprintf(os.Stdout, "\n%s\n", platform.Green(fmt.Sprintf("MCP server '%s' added successfully.", cfg.Name)))

		if cfg.UseOAuth {
			fmt.Println("Next: Run '/mcp' in Claude Code to complete OAuth authentication.")
		} else {
			fmt.Println("Run '/mcp' in Claude Code to verify the connection.")
		}

		if cfg.Scope == "project" && len(cfg.EnvVars) > 0 {
			fmt.Println("\n  NOTE: Server added to .mcp.json (project scope).")
			fmt.Println("  API keys are in your LOCAL Claude config, not in .mcp.json.")
			fmt.Println("  Team members must set these env vars in their own environment:")
			for key := range cfg.EnvVars {
				fmt.Printf("    export %s=<value>\n", key)
			}
		}
	} else {
		fmt.Fprintf(os.Stderr, "\n%s\n", platform.Red(fmt.Sprintf("Failed to add MCP server. Exit code: %d", exitCode)))
	}

	return nil
}

// Remote connects to a remote MCP gateway.
func Remote(mcpURL string, extraArgs []string) error {
	cfg, err := parseRemoteArgs(mcpURL, extraArgs)
	if err != nil {
		return err
	}

	if cfg.PromptBearer {
		fmt.Printf("\nBearer token required for '%s'.\n", cfg.Name)
		fmt.Println("Stored securely in your Claude config.")
		fmt.Println()

		token, err := promptSecret("Enter Bearer token: ")
		if err != nil {
			return err
		}
		if token == "" {
			return fmt.Errorf("no token provided")
		}
		cfg.Headers = append(cfg.Headers, "Authorization: Bearer "+token)
	}

	if cfg.PromptClientSecret {
		fmt.Printf("\nOAuth client secret required for '%s'.\n", cfg.Name)
		fmt.Println("Stored securely in your Claude config.")
		fmt.Println()

		secret, err := promptSecret("Enter OAuth client secret: ")
		if err != nil {
			return err
		}
		if secret == "" {
			return fmt.Errorf("no client secret provided")
		}
		cfg.ClientSecret = secret
	}

	claudeArgs := buildRemoteClaudeArgs(cfg)

	fmt.Printf("\nConnecting to remote MCP server '%s'...\n", cfg.Name)
	fmt.Printf("  URL:       %s\n", cfg.McpURL)
	fmt.Printf("  Transport: %s\n", cfg.Transport)
	fmt.Printf("  Scope:     %s\n", cfg.Scope)
	if cfg.UseOAuth || cfg.ClientID != "" {
		fmt.Println("  Auth:      OAuth 2.0")
	} else {
		hasAuth := false
		for _, h := range cfg.Headers {
			if strings.HasPrefix(strings.ToLower(h), "authorization") {
				hasAuth = true
				break
			}
		}
		if hasAuth {
			fmt.Println("  Auth:      Bearer token")
		} else {
			fmt.Println("  Auth:      OAuth (via /mcp in session)")
		}
	}
	fmt.Println()

	exitCode, err := platform.RunSpawn("claude", claudeArgs...)
	if err != nil {
		return fmt.Errorf("could not run 'claude' command. Is Claude Code installed?")
	}

	if exitCode == 0 {
		fmt.Fprintf(os.Stdout, "\n%s\n", platform.Green(fmt.Sprintf("Remote MCP server '%s' connected.", cfg.Name)))
		if cfg.UseOAuth || cfg.ClientID != "" || !cfg.PromptBearer {
			fmt.Printf("Next: Run '/mcp' in Claude Code → select '%s' → Authenticate\n", cfg.Name)
		}
	} else {
		fmt.Fprintf(os.Stderr, "\n%s\n", platform.Red(fmt.Sprintf("Failed to connect. Exit code: %d", exitCode)))
	}

	return nil
}

// List lists all configured MCP servers.
func List() error {
	platform.PrintBanner(os.Stdout, "Configured MCP Servers")
	fmt.Println()

	_, _ = platform.RunSpawn("claude", "mcp", "list")

	mcpJSONPath := filepath.Join(".", ".mcp.json")
	if platform.FileExists(mcpJSONPath) {
		platform.PrintSection(os.Stdout, "Project .mcp.json")
		var mcpConfig struct {
			MCPServers map[string]json.RawMessage `json:"mcpServers"`
		}
		if err := platform.ReadJSONFile(mcpJSONPath, &mcpConfig); err == nil {
			for name, raw := range mcpConfig.MCPServers {
				var cfg struct {
					Type    string            `json:"type"`
					URL     string            `json:"url"`
					Command string            `json:"command"`
					Args    []string          `json:"args"`
					Env     map[string]string `json:"env"`
				}
				if json.Unmarshal(raw, &cfg) != nil {
					continue
				}

				var envKeys []string
				for k, v := range cfg.Env {
					if v != "" {
						envKeys = append(envKeys, k)
					}
				}
				envNote := ""
				if len(envKeys) > 0 {
					envNote = fmt.Sprintf(" (env: %s)", strings.Join(envKeys, ", "))
				}

				if cfg.Type == transportHTTP || cfg.URL != "" {
					urlOrType := cfg.URL
					if urlOrType == "" {
						urlOrType = cfg.Type
					}
					fmt.Printf("  %s: %s (remote)%s\n", name, urlOrType, envNote)
				} else {
					fmt.Printf("  %s: %s %s (local)%s\n", name, cfg.Command, strings.Join(cfg.Args, " "), envNote)
				}
			}
		} else {
			fmt.Fprintf(os.Stderr, "  %s\n", platform.Red("Could not parse .mcp.json"))
		}
	}

	platform.PrintSection(os.Stdout, "Quick Add Commands")
	fmt.Println("  Local server (no auth):     claude-workspace mcp add <name> -- <cmd>")
	fmt.Println("  Local server (API key):     claude-workspace mcp add <name> --api-key API_KEY -- <cmd>")
	fmt.Println("  Remote server (OAuth):      claude-workspace mcp remote <url>")
	fmt.Println("  Remote server (Bearer):     claude-workspace mcp remote <url> --bearer")
	fmt.Println("  Remote server (client creds): claude-workspace mcp remote <url> --oauth --client-id <id> --client-secret")
	fmt.Println()

	return nil
}

func printMcpAddHelp() {
	fmt.Print(`Usage: claude-workspace mcp add <name> [options] [-- <command> [args...]]

Add a local or remote MCP server with secure API key handling.

Authentication Options:
  --api-key ENV_VAR_NAME        Prompt for API key (masked input), stored as env var
  --bearer                      Prompt for Bearer token (masked input), added as header
  --oauth                       Use OAuth 2.0 (authenticate via /mcp in Claude Code)
  --client-id <id>              OAuth client ID (for pre-registered apps)
  --client-secret               Prompt for OAuth client secret (masked input)

Other Options:
  --scope local|project|user    Where to save config (default: local)
  --transport stdio|http|sse    Transport type (default: auto-detected)
  --env KEY=VALUE               Set environment variable (repeatable, visible)
  --header 'Key: Value'         Add HTTP header

Security:
  - --api-key and --bearer use masked input (characters not shown)
  - Secrets are stored in ~/.claude.json, NEVER in .mcp.json
  - When using --scope project, only the server definition goes in .mcp.json
  - .mcp.json supports ${VAR} syntax for team members to supply their own keys

Examples:

  # Server requiring an API key (prompted securely)
  claude-workspace mcp add brave-search --api-key BRAVE_API_KEY \
    -- npx -y @modelcontextprotocol/server-brave-search

  # Database with connection string as secret
  claude-workspace mcp add postgres --api-key DATABASE_URL \
    -- npx -y @bytebase/dbhub

  # Remote server with OAuth (GitHub, Sentry, Notion, etc.)
  claude-workspace mcp add github --transport http \
    https://api.githubcopilot.com/mcp/

  # Remote server with Bearer token
  claude-workspace mcp add my-api --bearer --transport http \
    https://api.example.com/mcp

  # Share server with team (key stays local)
  claude-workspace mcp add sentry --scope project \
    --transport http https://mcp.sentry.dev/mcp
`)
}

func printMcpRemoteHelp() {
	fmt.Print(`Usage: claude-workspace mcp remote <url> [options]

Connect to a remote MCP server or gateway.

Authentication Options:
  --bearer                        Prompt for Bearer token (masked input)
  --oauth                         Use OAuth 2.0 flow
  --client-id <id>                OAuth client ID
  --client-secret                 Prompt for OAuth client secret

Other Options:
  --name <name>                   Server name (default: derived from URL)
  --scope local|project|user      Where to save (default: user)
  --header 'Key: Value'           Add custom HTTP header

Examples:

  # OAuth servers (most cloud services - authenticate via /mcp)
  claude-workspace mcp remote https://mcp.sentry.dev/mcp --name sentry
  claude-workspace mcp remote https://api.githubcopilot.com/mcp/ --name github
  claude-workspace mcp remote https://mcp.notion.com/mcp --name notion
  claude-workspace mcp remote https://mcp.linear.app/mcp --name linear

  # Bearer token (prompted securely)
  claude-workspace mcp remote https://mcp.example.com --bearer

  # Pre-registered OAuth credentials
  claude-workspace mcp remote https://mcp.example.com \
    --oauth --client-id my-client-id --client-secret

  # Organization gateway
  claude-workspace mcp remote https://mcp-gateway.company.com --name company
  claude-workspace mcp remote https://mcp-gateway.company.com --bearer --name company
`)
}
