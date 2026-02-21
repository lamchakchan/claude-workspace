package mcp

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/lamchakchan/claude-platform/internal/platform"
	"golang.org/x/term"
)

func promptSecret(prompt string) (string, error) {
	fmt.Print(prompt)
	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("reading secret: %w", err)
	}
	return strings.TrimSpace(string(password)), nil
}

// Add adds a local or remote MCP server to the project or user config.
func Add(args []string) error {
	if len(args) < 1 {
		printMcpAddHelp()
		os.Exit(1)
	}

	name := args[0]
	scope := "local"
	transport := ""
	envVars := map[string]string{}
	var headers []string
	var commandArgs []string
	mcpURL := ""
	apiKeyEnvVar := ""
	promptBearer := false
	useOAuth := false
	clientId := ""
	promptClientSecret := false

	i := 1
	seenDoubleDash := false

	for i < len(args) {
		if args[i] == "--" && !seenDoubleDash {
			seenDoubleDash = true
			commandArgs = args[i+1:]
			break
		}

		switch args[i] {
		case "--scope":
			i++
			if i < len(args) {
				scope = args[i]
			}
		case "--transport":
			i++
			if i < len(args) {
				transport = args[i]
			}
		case "--env":
			i++
			if i < len(args) {
				envPair := args[i]
				eqIdx := strings.Index(envPair, "=")
				if eqIdx > 0 {
					envVars[envPair[:eqIdx]] = envPair[eqIdx+1:]
				}
			}
		case "--api-key":
			i++
			if i < len(args) {
				apiKeyEnvVar = args[i]
			}
		case "--header":
			i++
			if i < len(args) {
				headers = append(headers, args[i])
			}
		case "--bearer":
			promptBearer = true
		case "--oauth":
			useOAuth = true
		case "--client-id":
			i++
			if i < len(args) {
				clientId = args[i]
			}
		case "--client-secret":
			promptClientSecret = true
		default:
			if strings.HasPrefix(args[i], "http://") || strings.HasPrefix(args[i], "https://") {
				mcpURL = args[i]
			} else {
				commandArgs = args[i:]
				i = len(args)
				continue
			}
		}
		i++
	}

	// Secure API key prompting
	if apiKeyEnvVar != "" {
		fmt.Printf("\nAPI key required for '%s' server.\n", name)
		fmt.Printf("The key will be stored as env var: %s\n", apiKeyEnvVar)
		fmt.Println("Stored in your Claude config (~/.claude.json), NOT in project files.")
		fmt.Println()

		keyValue, err := promptSecret(fmt.Sprintf("Enter %s: ", apiKeyEnvVar))
		if err != nil {
			return err
		}
		if keyValue == "" {
			return fmt.Errorf("no API key provided")
		}
		envVars[apiKeyEnvVar] = keyValue
	}

	if promptBearer {
		fmt.Printf("\nBearer token required for '%s' server.\n", name)
		fmt.Println("Stored in your Claude config (~/.claude.json), NOT in project files.")
		fmt.Println()

		token, err := promptSecret("Enter Bearer token: ")
		if err != nil {
			return err
		}
		if token == "" {
			return fmt.Errorf("no token provided")
		}
		headers = append(headers, "Authorization: Bearer "+token)
	}

	// Determine transport
	if transport == "" {
		if mcpURL != "" {
			transport = "http"
		} else {
			transport = "stdio"
		}
	}

	// Build the claude mcp add command
	claudeArgs := []string{"mcp", "add", "--transport", transport, "--scope", scope}

	for key, value := range envVars {
		claudeArgs = append(claudeArgs, "--env", key+"="+value)
	}

	for _, header := range headers {
		claudeArgs = append(claudeArgs, "--header", header)
	}

	if clientId != "" {
		claudeArgs = append(claudeArgs, "--client-id", clientId)
	}
	if promptClientSecret {
		claudeArgs = append(claudeArgs, "--client-secret")
	}

	claudeArgs = append(claudeArgs, name)

	if transport == "http" || transport == "sse" {
		if mcpURL == "" {
			return fmt.Errorf("URL required for http/sse transport")
		}
		claudeArgs = append(claudeArgs, mcpURL)
	} else if len(commandArgs) > 0 {
		claudeArgs = append(claudeArgs, "--")
		claudeArgs = append(claudeArgs, commandArgs...)
	}

	fmt.Printf("Adding MCP server '%s' (%s, scope: %s)...\n", name, transport, scope)

	// Mask sensitive values in log output
	safeArgs := make([]string, len(claudeArgs))
	for idx, arg := range claudeArgs {
		if idx > 0 {
			prev := claudeArgs[idx-1]
			if prev == "--env" && strings.Contains(arg, "=") {
				eqIdx := strings.Index(arg, "=")
				safeArgs[idx] = arg[:eqIdx] + "=****"
				continue
			}
			if prev == "--header" && strings.Contains(strings.ToLower(arg), "bearer") {
				safeArgs[idx] = "Authorization: Bearer ****"
				continue
			}
		}
		safeArgs[idx] = arg
	}
	fmt.Printf("  > claude %s\n\n", strings.Join(safeArgs, " "))

	exitCode, err := platform.RunSpawn("claude", claudeArgs...)
	if err != nil {
		return fmt.Errorf("could not run 'claude' command. Is Claude Code installed?")
	}

	if exitCode == 0 {
		fmt.Printf("\nMCP server '%s' added successfully.\n", name)

		if useOAuth {
			fmt.Println("Next: Run '/mcp' in Claude Code to complete OAuth authentication.")
		} else {
			fmt.Println("Run '/mcp' in Claude Code to verify the connection.")
		}

		if scope == "project" && len(envVars) > 0 {
			fmt.Println("\n  NOTE: Server added to .mcp.json (project scope).")
			fmt.Println("  API keys are in your LOCAL Claude config, not in .mcp.json.")
			fmt.Println("  Team members must set these env vars in their own environment:")
			for key := range envVars {
				fmt.Printf("    export %s=<value>\n", key)
			}
		}
	} else {
		fmt.Fprintf(os.Stderr, "\nFailed to add MCP server. Exit code: %d\n", exitCode)
	}

	return nil
}

// Remote connects to a remote MCP gateway.
func Remote(mcpURL string, extraArgs []string) error {
	if mcpURL == "" {
		printMcpRemoteHelp()
		os.Exit(1)
	}

	name := ""
	scope := "user"
	var headers []string
	promptBearerFlag := false
	useOAuth := false
	clientId := ""
	promptClientSecret := false

	for i := 0; i < len(extraArgs); i++ {
		switch extraArgs[i] {
		case "--name":
			i++
			if i < len(extraArgs) {
				name = extraArgs[i]
			}
		case "--scope":
			i++
			if i < len(extraArgs) {
				scope = extraArgs[i]
			}
		case "--header":
			i++
			if i < len(extraArgs) {
				headers = append(headers, extraArgs[i])
			}
		case "--bearer":
			promptBearerFlag = true
		case "--oauth":
			useOAuth = true
		case "--client-id":
			i++
			if i < len(extraArgs) {
				clientId = extraArgs[i]
			}
		case "--client-secret":
			promptClientSecret = true
		}
	}

	if name == "" {
		if u, err := url.Parse(mcpURL); err == nil {
			name = u.Hostname()
			name = strings.TrimPrefix(name, "mcp-")
			name = strings.TrimPrefix(name, "mcp.")
			name = strings.TrimSuffix(name, ".com")
			// Replace non-alphanumeric with dash
			var b strings.Builder
			for _, r := range name {
				if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
					b.WriteRune(r)
				} else {
					b.WriteRune('-')
				}
			}
			name = b.String()
		} else {
			name = "remote-gateway"
		}
	}

	if promptBearerFlag {
		fmt.Printf("\nBearer token required for '%s'.\n", name)
		fmt.Println("Stored securely in your Claude config.")
		fmt.Println()

		token, err := promptSecret("Enter Bearer token: ")
		if err != nil {
			return err
		}
		if token == "" {
			return fmt.Errorf("no token provided")
		}
		headers = append(headers, "Authorization: Bearer "+token)
	}

	transport := "http"
	if strings.HasSuffix(mcpURL, "/sse") {
		transport = "sse"
	}

	claudeArgs := []string{"mcp", "add", "--transport", transport, "--scope", scope}

	for _, header := range headers {
		claudeArgs = append(claudeArgs, "--header", header)
	}

	if clientId != "" {
		claudeArgs = append(claudeArgs, "--client-id", clientId)
	}
	if promptClientSecret {
		claudeArgs = append(claudeArgs, "--client-secret")
	}

	claudeArgs = append(claudeArgs, name, mcpURL)

	fmt.Printf("\nConnecting to remote MCP server '%s'...\n", name)
	fmt.Printf("  URL:       %s\n", mcpURL)
	fmt.Printf("  Transport: %s\n", transport)
	fmt.Printf("  Scope:     %s\n", scope)
	if useOAuth || clientId != "" {
		fmt.Println("  Auth:      OAuth 2.0")
	} else {
		hasAuth := false
		for _, h := range headers {
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
		fmt.Printf("\nRemote MCP server '%s' connected.\n", name)
		if useOAuth || clientId != "" || !promptBearerFlag {
			fmt.Printf("Next: Run '/mcp' in Claude Code → select '%s' → Authenticate\n", name)
		}
	} else {
		fmt.Fprintf(os.Stderr, "\nFailed to connect. Exit code: %d\n", exitCode)
	}

	return nil
}

// List lists all configured MCP servers.
func List() error {
	fmt.Println("\n=== Configured MCP Servers ===")
	fmt.Println()

	platform.RunSpawn("claude", "mcp", "list")

	mcpJsonPath := filepath.Join(".", ".mcp.json")
	if platform.FileExists(mcpJsonPath) {
		fmt.Println("\n--- Project .mcp.json ---")
		var mcpConfig struct {
			MCPServers map[string]json.RawMessage `json:"mcpServers"`
		}
		if err := platform.ReadJSONFile(mcpJsonPath, &mcpConfig); err == nil {
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

				if cfg.Type == "http" || cfg.URL != "" {
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
			fmt.Println("  Could not parse .mcp.json")
		}
	}

	fmt.Println("\n--- Quick Add Commands ---")
	fmt.Println("  Local server (no auth):     claude-platform mcp add <name> -- <cmd>")
	fmt.Println("  Local server (API key):     claude-platform mcp add <name> --api-key API_KEY -- <cmd>")
	fmt.Println("  Remote server (OAuth):      claude-platform mcp remote <url>")
	fmt.Println("  Remote server (Bearer):     claude-platform mcp remote <url> --bearer")
	fmt.Println("  Remote server (client creds): claude-platform mcp remote <url> --oauth --client-id <id> --client-secret")
	fmt.Println()

	return nil
}

func printMcpAddHelp() {
	fmt.Print(`Usage: claude-platform mcp add <name> [options] [-- <command> [args...]]

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
  claude-platform mcp add brave-search --api-key BRAVE_API_KEY \
    -- npx -y @modelcontextprotocol/server-brave-search

  # Database with connection string as secret
  claude-platform mcp add postgres --api-key DATABASE_URL \
    -- npx -y @bytebase/dbhub

  # Remote server with OAuth (GitHub, Sentry, Notion, etc.)
  claude-platform mcp add github --transport http \
    https://api.githubcopilot.com/mcp/

  # Remote server with Bearer token
  claude-platform mcp add my-api --bearer --transport http \
    https://api.example.com/mcp

  # Share server with team (key stays local)
  claude-platform mcp add sentry --scope project \
    --transport http https://mcp.sentry.dev/mcp
`)
}

func printMcpRemoteHelp() {
	fmt.Print(`Usage: claude-platform mcp remote <url> [options]

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
  claude-platform mcp remote https://mcp.sentry.dev/mcp --name sentry
  claude-platform mcp remote https://api.githubcopilot.com/mcp/ --name github
  claude-platform mcp remote https://mcp.notion.com/mcp --name notion
  claude-platform mcp remote https://mcp.linear.app/mcp --name linear

  # Bearer token (prompted securely)
  claude-platform mcp remote https://mcp.example.com --bearer

  # Pre-registered OAuth credentials
  claude-platform mcp remote https://mcp.example.com \
    --oauth --client-id my-client-id --client-secret

  # Organization gateway
  claude-platform mcp remote https://mcp-gateway.company.com --name company
  claude-platform mcp remote https://mcp-gateway.company.com --bearer --name company
`)
}
