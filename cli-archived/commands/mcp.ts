import { existsSync } from "fs";
import { join, resolve } from "path";
import { $ } from "bun";
import { homedir } from "os";

/**
 * MCP server management commands.
 * Provides easy ways to add local and remote MCP servers,
 * with secure API key handling.
 */

/**
 * Prompt for a value with masked input (for secrets).
 */
async function promptSecret(prompt: string): Promise<string> {
  process.stdout.write(prompt);
  const proc = Bun.spawn(["bash", "-c", 'read -s val && echo "$val"'], {
    stdin: "inherit",
    stdout: "pipe",
    stderr: "inherit",
  });
  const output = await new Response(proc.stdout).text();
  await proc.exited;
  process.stdout.write("\n");
  return output.trim();
}

/**
 * Add a local or remote MCP server to the project or user config.
 *
 * Supports three authentication methods:
 *   --api-key ENV_VAR_NAME   Prompt for API key (masked), store as env var
 *   --bearer                 Prompt for Bearer token (masked), add as header
 *   --oauth / --client-id    Use OAuth 2.0 flow (authenticate via /mcp)
 *
 * Usage:
 *   claude-workspace mcp add <name> -- <command> [args...]
 *   claude-workspace mcp add <name> --api-key API_KEY -- <command>
 *   claude-workspace mcp add <name> --transport http <url>
 *   claude-workspace mcp add <name> --bearer --transport http <url>
 *   claude-workspace mcp add <name> --oauth --client-id <id> --transport http <url>
 *
 * Options:
 *   --scope local|project|user    Where to save (default: local)
 *   --transport stdio|http|sse    Transport type (default: auto-detected)
 *   --env KEY=VALUE               Environment variable (repeatable)
 *   --api-key ENV_VAR_NAME        Securely prompt for API key, store as env var
 *   --header 'Key: Value'         Add HTTP header
 *   --bearer                      Securely prompt for Bearer token
 *   --oauth                       Use OAuth 2.0 (authenticate via /mcp)
 *   --client-id <id>              OAuth client ID
 *   --client-secret               Prompt for OAuth client secret
 */
export async function mcpAdd(args: string[]) {
  if (args.length < 1) {
    printMcpAddHelp();
    process.exit(1);
  }

  // Parse arguments
  const name = args[0];
  let scope = "local";
  let transport = "";
  const envVars: Record<string, string> = {};
  const headers: string[] = [];
  let commandArgs: string[] = [];
  let url = "";
  let apiKeyEnvVar = "";
  let promptBearer = false;
  let useOAuth = false;
  let clientId = "";
  let promptClientSecret = false;

  let i = 1;
  let seenDoubleDash = false;

  while (i < args.length) {
    if (args[i] === "--" && !seenDoubleDash) {
      seenDoubleDash = true;
      commandArgs = args.slice(i + 1);
      break;
    }

    switch (args[i]) {
      case "--scope":
        scope = args[++i];
        break;
      case "--transport":
        transport = args[++i];
        break;
      case "--env": {
        const envPair = args[++i];
        const eqIdx = envPair.indexOf("=");
        if (eqIdx > 0) {
          envVars[envPair.slice(0, eqIdx)] = envPair.slice(eqIdx + 1);
        }
        break;
      }
      case "--api-key":
        apiKeyEnvVar = args[++i];
        break;
      case "--header":
        headers.push(args[++i]);
        break;
      case "--bearer":
        promptBearer = true;
        break;
      case "--oauth":
        useOAuth = true;
        break;
      case "--client-id":
        clientId = args[++i];
        break;
      case "--client-secret":
        promptClientSecret = true;
        break;
      default:
        if (args[i].startsWith("http://") || args[i].startsWith("https://")) {
          url = args[i];
        } else {
          commandArgs = args.slice(i);
          i = args.length;
          continue;
        }
        break;
    }
    i++;
  }

  // --- Secure API key prompting ---

  if (apiKeyEnvVar) {
    console.log(`\nAPI key required for '${name}' server.`);
    console.log(`The key will be stored as env var: ${apiKeyEnvVar}`);
    console.log(
      `Stored in your Claude config (~/.claude.json), NOT in project files.\n`,
    );

    const keyValue = await promptSecret(`Enter ${apiKeyEnvVar}: `);
    if (!keyValue) {
      console.error("No API key provided. Aborting.");
      process.exit(1);
    }
    envVars[apiKeyEnvVar] = keyValue;
  }

  if (promptBearer) {
    console.log(`\nBearer token required for '${name}' server.`);
    console.log(
      "Stored in your Claude config (~/.claude.json), NOT in project files.\n",
    );

    const token = await promptSecret("Enter Bearer token: ");
    if (!token) {
      console.error("No token provided. Aborting.");
      process.exit(1);
    }
    headers.push(`Authorization: Bearer ${token}`);
  }

  // Determine transport
  if (!transport) {
    transport = url ? "http" : "stdio";
  }

  // Build the claude mcp add command
  const claudeArgs = ["mcp", "add", "--transport", transport, "--scope", scope];

  for (const [key, value] of Object.entries(envVars)) {
    claudeArgs.push("--env", `${key}=${value}`);
  }

  for (const header of headers) {
    claudeArgs.push("--header", header);
  }

  if (clientId) {
    claudeArgs.push("--client-id", clientId);
  }
  if (promptClientSecret) {
    claudeArgs.push("--client-secret");
  }

  claudeArgs.push(name);

  if (transport === "http" || transport === "sse") {
    if (!url) {
      console.error("URL required for http/sse transport");
      process.exit(1);
    }
    claudeArgs.push(url);
  } else if (commandArgs.length > 0) {
    claudeArgs.push("--");
    claudeArgs.push(...commandArgs);
  }

  console.log(`Adding MCP server '${name}' (${transport}, scope: ${scope})...`);

  // Mask sensitive values in log output
  const safeArgs = claudeArgs.map((arg, idx) => {
    const prev = claudeArgs[idx - 1];
    if (prev === "--env" && arg.includes("=")) {
      const eqIdx = arg.indexOf("=");
      return `${arg.slice(0, eqIdx)}=****`;
    }
    if (prev === "--header" && arg.toLowerCase().includes("bearer")) {
      return "Authorization: Bearer ****";
    }
    return arg;
  });
  console.log(`  > claude ${safeArgs.join(" ")}\n`);

  try {
    const proc = Bun.spawn(["claude", ...claudeArgs], {
      stdin: "inherit",
      stdout: "inherit",
      stderr: "inherit",
    });
    await proc.exited;

    if (proc.exitCode === 0) {
      console.log(`\nMCP server '${name}' added successfully.`);

      if (useOAuth) {
        console.log(
          "Next: Run '/mcp' in Claude Code to complete OAuth authentication.",
        );
      } else {
        console.log("Run '/mcp' in Claude Code to verify the connection.");
      }

      if (scope === "project" && Object.keys(envVars).length > 0) {
        console.log("\n  NOTE: Server added to .mcp.json (project scope).");
        console.log(
          "  API keys are in your LOCAL Claude config, not in .mcp.json.",
        );
        console.log(
          "  Team members must set these env vars in their own environment:",
        );
        for (const key of Object.keys(envVars)) {
          console.log(`    export ${key}=<value>`);
        }
      }
    } else {
      console.error(`\nFailed to add MCP server. Exit code: ${proc.exitCode}`);
    }
  } catch {
    console.error(
      "Error: Could not run 'claude' command. Is Claude Code installed?",
    );
    process.exit(1);
  }
}

/**
 * Connect to a remote MCP gateway.
 *
 * Supports authentication via:
 *   --bearer        Prompt for Bearer token (masked input)
 *   --oauth         OAuth 2.0 flow (authenticate via /mcp in session)
 *   --client-id     Pre-registered OAuth credentials
 *   --header        Custom HTTP headers
 */
export async function mcpRemote(url?: string, extraArgs: string[] = []) {
  if (!url) {
    printMcpRemoteHelp();
    process.exit(1);
  }

  let name = "";
  let scope = "user";
  const headers: string[] = [];
  let promptBearer = false;
  let useOAuth = false;
  let clientId = "";
  let promptClientSecret = false;

  for (let i = 0; i < extraArgs.length; i++) {
    switch (extraArgs[i]) {
      case "--name":
        name = extraArgs[++i];
        break;
      case "--scope":
        scope = extraArgs[++i];
        break;
      case "--header":
        headers.push(extraArgs[++i]);
        break;
      case "--bearer":
        promptBearer = true;
        break;
      case "--oauth":
        useOAuth = true;
        break;
      case "--client-id":
        clientId = extraArgs[++i];
        break;
      case "--client-secret":
        promptClientSecret = true;
        break;
    }
  }

  if (!name) {
    try {
      const urlObj = new URL(url);
      name = urlObj.hostname
        .replace(/^mcp[-.]/, "")
        .replace(/\.com$/, "")
        .replace(/[^a-z0-9]/g, "-");
    } catch {
      name = "remote-gateway";
    }
  }

  if (promptBearer) {
    console.log(`\nBearer token required for '${name}'.`);
    console.log("Stored securely in your Claude config.\n");

    const token = await promptSecret("Enter Bearer token: ");
    if (!token) {
      console.error("No token provided. Aborting.");
      process.exit(1);
    }
    headers.push(`Authorization: Bearer ${token}`);
  }

  const transport = url.endsWith("/sse") ? "sse" : "http";
  const claudeArgs = ["mcp", "add", "--transport", transport, "--scope", scope];

  for (const header of headers) {
    claudeArgs.push("--header", header);
  }

  if (clientId) {
    claudeArgs.push("--client-id", clientId);
  }
  if (promptClientSecret) {
    claudeArgs.push("--client-secret");
  }

  claudeArgs.push(name, url);

  console.log(`\nConnecting to remote MCP server '${name}'...`);
  console.log(`  URL:       ${url}`);
  console.log(`  Transport: ${transport}`);
  console.log(`  Scope:     ${scope}`);
  if (useOAuth || clientId) console.log(`  Auth:      OAuth 2.0`);
  else if (headers.some((h) => h.toLowerCase().startsWith("authorization")))
    console.log(`  Auth:      Bearer token`);
  else console.log(`  Auth:      OAuth (via /mcp in session)`);
  console.log("");

  try {
    const proc = Bun.spawn(["claude", ...claudeArgs], {
      stdin: "inherit",
      stdout: "inherit",
      stderr: "inherit",
    });
    await proc.exited;

    if (proc.exitCode === 0) {
      console.log(`\nRemote MCP server '${name}' connected.`);
      if (useOAuth || clientId || !promptBearer) {
        console.log(
          "Next: Run '/mcp' in Claude Code → select '${name}' → Authenticate",
        );
      }
    } else {
      console.error(`\nFailed to connect. Exit code: ${proc.exitCode}`);
    }
  } catch {
    console.error(
      "Error: Could not run 'claude' command. Is Claude Code installed?",
    );
    process.exit(1);
  }
}

/**
 * List all configured MCP servers.
 */
export async function mcpList() {
  console.log("\n=== Configured MCP Servers ===\n");

  try {
    const proc = Bun.spawn(["claude", "mcp", "list"], {
      stdout: "inherit",
      stderr: "inherit",
    });
    await proc.exited;
  } catch {
    console.log("Could not list servers via 'claude' CLI.\n");
  }

  const mcpJsonPath = join(process.cwd(), ".mcp.json");
  if (existsSync(mcpJsonPath)) {
    console.log("\n--- Project .mcp.json ---");
    try {
      const mcpConfig = JSON.parse(await Bun.file(mcpJsonPath).text());
      for (const [name, config] of Object.entries(mcpConfig.mcpServers || {})) {
        const cfg = config as any;
        const envKeys = cfg.env
          ? Object.keys(cfg.env).filter((k) => cfg.env[k])
          : [];
        const envNote =
          envKeys.length > 0 ? ` (env: ${envKeys.join(", ")})` : "";

        if (cfg.type === "http" || cfg.url) {
          console.log(`  ${name}: ${cfg.url || cfg.type} (remote)${envNote}`);
        } else {
          console.log(
            `  ${name}: ${cfg.command} ${(cfg.args || []).join(" ")} (local)${envNote}`,
          );
        }
      }
    } catch {
      console.log("  Could not parse .mcp.json");
    }
  }

  console.log("\n--- Quick Add Commands ---");
  console.log(
    "  Local server (no auth):     claude-workspace mcp add <name> -- <cmd>",
  );
  console.log(
    "  Local server (API key):     claude-workspace mcp add <name> --api-key API_KEY -- <cmd>",
  );
  console.log("  Remote server (OAuth):      claude-workspace mcp remote <url>");
  console.log(
    "  Remote server (Bearer):     claude-workspace mcp remote <url> --bearer",
  );
  console.log(
    "  Remote server (client creds): claude-workspace mcp remote <url> --oauth --client-id <id> --client-secret",
  );
  console.log("");
}

function printMcpAddHelp() {
  console.log(`Usage: claude-workspace mcp add <name> [options] [-- <command> [args...]]

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
  - .mcp.json supports \${VAR} syntax for team members to supply their own keys

Examples:

  # Server requiring an API key (prompted securely)
  claude-workspace mcp add brave-search --api-key BRAVE_API_KEY \\
    -- npx -y @modelcontextprotocol/server-brave-search

  # Database with connection string as secret
  claude-workspace mcp add postgres --api-key DATABASE_URL \\
    -- npx -y @bytebase/dbhub

  # Airtable with explicit env var (key visible in command)
  claude-workspace mcp add airtable \\
    --env AIRTABLE_API_KEY=patXXXXXXXX \\
    -- npx -y airtable-mcp-server

  # Remote server with OAuth (GitHub, Sentry, Notion, etc.)
  claude-workspace mcp add github --transport http \\
    https://api.githubcopilot.com/mcp/
  # Then: /mcp in Claude Code → Authenticate

  # Remote server with Bearer token
  claude-workspace mcp add my-api --bearer --transport http \\
    https://api.example.com/mcp

  # Remote with pre-registered OAuth credentials
  claude-workspace mcp add my-server --oauth --client-id abc123 \\
    --client-secret --transport http https://mcp.example.com/mcp

  # Share server with team (key stays local)
  claude-workspace mcp add sentry --scope project \\
    --transport http https://mcp.sentry.dev/mcp
`);
}

function printMcpRemoteHelp() {
  console.log(`Usage: claude-workspace mcp remote <url> [options]

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
  claude-workspace mcp remote https://mcp.example.com \\
    --oauth --client-id my-client-id --client-secret

  # Organization gateway
  claude-workspace mcp remote https://mcp-gateway.company.com --name company
  claude-workspace mcp remote https://mcp-gateway.company.com --bearer --name company
`);
}
