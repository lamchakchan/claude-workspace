import { existsSync } from "fs";
import { join, resolve } from "path";
import { $ } from "bun";
import { homedir } from "os";

/**
 * MCP server management commands.
 * Provides easy ways to add local and remote MCP servers.
 */

/**
 * Add a local MCP server to the project or user config.
 *
 * Usage:
 *   bun run cli/index.ts mcp add <name> -- <command> [args...]
 *   bun run cli/index.ts mcp add <name> --transport http <url>
 *   bun run cli/index.ts mcp add <name> --env KEY=VALUE -- <command>
 *
 * Options:
 *   --scope local|project|user   Where to save (default: local)
 *   --transport stdio|http|sse   Transport type (default: stdio for commands, http for URLs)
 *   --env KEY=VALUE              Environment variables (repeatable)
 */
export async function mcpAdd(args: string[]) {
  if (args.length < 2) {
    console.log("Usage: bun run cli/index.ts mcp add <name> [options] -- <command> [args...]");
    console.log("       bun run cli/index.ts mcp add <name> --transport http <url>");
    console.log("\nOptions:");
    console.log("  --scope local|project|user   Where to save the config (default: local)");
    console.log("  --transport stdio|http|sse   Transport type");
    console.log("  --env KEY=VALUE              Environment variables");
    console.log("\nExamples:");
    console.log('  bun run cli/index.ts mcp add postgres -- npx -y @bytebase/dbhub --dsn "postgres://..."');
    console.log("  bun run cli/index.ts mcp add sentry --transport http https://mcp.sentry.dev/mcp");
    console.log("  bun run cli/index.ts mcp add notion --transport http https://mcp.notion.com/mcp");
    console.log("  bun run cli/index.ts mcp add github --transport http https://api.githubcopilot.com/mcp/");
    process.exit(1);
  }

  // Parse arguments
  const name = args[0];
  let scope = "local";
  let transport = "";
  const envVars: Record<string, string> = {};
  let commandArgs: string[] = [];
  let url = "";

  let i = 1;
  let seenDoubleDash = false;

  while (i < args.length) {
    if (args[i] === "--" && !seenDoubleDash) {
      seenDoubleDash = true;
      commandArgs = args.slice(i + 1);
      break;
    }

    if (args[i] === "--scope" && i + 1 < args.length) {
      scope = args[++i];
    } else if (args[i] === "--transport" && i + 1 < args.length) {
      transport = args[++i];
    } else if (args[i] === "--env" && i + 1 < args.length) {
      const envPair = args[++i];
      const eqIdx = envPair.indexOf("=");
      if (eqIdx > 0) {
        envVars[envPair.slice(0, eqIdx)] = envPair.slice(eqIdx + 1);
      }
    } else if (args[i].startsWith("http://") || args[i].startsWith("https://")) {
      url = args[i];
    } else {
      // Unknown arg, might be part of the command
      commandArgs = args.slice(i);
      break;
    }
    i++;
  }

  // Determine transport
  if (!transport) {
    transport = url ? "http" : "stdio";
  }

  // Build the claude mcp add command
  const claudeArgs = ["mcp", "add", "--transport", transport, "--scope", scope];

  // Add env vars
  for (const [key, value] of Object.entries(envVars)) {
    claudeArgs.push("--env", `${key}=${value}`);
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

  try {
    const proc = Bun.spawn(["claude", ...claudeArgs], {
      stdin: "inherit",
      stdout: "inherit",
      stderr: "inherit",
    });
    await proc.exited;

    if (proc.exitCode === 0) {
      console.log(`\nMCP server '${name}' added successfully.`);
      console.log("Run '/mcp' in Claude Code to verify the connection.");
    } else {
      console.error(`\nFailed to add MCP server. Exit code: ${proc.exitCode}`);
    }
  } catch (error) {
    console.error("Error: Could not run 'claude' command. Is Claude Code installed?");
    process.exit(1);
  }
}

/**
 * Connect to a remote MCP gateway.
 *
 * Usage:
 *   bun run cli/index.ts mcp remote <gateway-url> [--name <name>] [--scope <scope>]
 *   bun run cli/index.ts mcp remote <gateway-url> --header "Authorization: Bearer <token>"
 */
export async function mcpRemote(url?: string, extraArgs: string[] = []) {
  if (!url) {
    console.log("Usage: bun run cli/index.ts mcp remote <gateway-url> [options]");
    console.log("\nOptions:");
    console.log("  --name <name>                   Server name (default: derived from URL)");
    console.log("  --scope local|project|user      Where to save (default: user)");
    console.log("  --header 'Key: Value'           Add authentication header");
    console.log("\nExamples:");
    console.log("  bun run cli/index.ts mcp remote https://mcp-gateway.company.com");
    console.log('  bun run cli/index.ts mcp remote https://mcp.example.com --header "Authorization: Bearer token"');
    console.log("  bun run cli/index.ts mcp remote https://mcp.sentry.dev/mcp --name sentry");
    process.exit(1);
  }

  // Parse options
  let name = "";
  let scope = "user";
  const headers: string[] = [];

  for (let i = 0; i < extraArgs.length; i++) {
    if (extraArgs[i] === "--name" && i + 1 < extraArgs.length) {
      name = extraArgs[++i];
    } else if (extraArgs[i] === "--scope" && i + 1 < extraArgs.length) {
      scope = extraArgs[++i];
    } else if (extraArgs[i] === "--header" && i + 1 < extraArgs.length) {
      headers.push(extraArgs[++i]);
    }
  }

  // Derive name from URL if not provided
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

  // Determine transport (prefer http, fallback to sse)
  const transport = url.endsWith("/sse") ? "sse" : "http";

  const claudeArgs = ["mcp", "add", "--transport", transport, "--scope", scope];

  for (const header of headers) {
    claudeArgs.push("--header", header);
  }

  claudeArgs.push(name, url);

  console.log(`Connecting to remote MCP gateway '${name}'...`);
  console.log(`URL: ${url}`);
  console.log(`Transport: ${transport}`);
  console.log(`Scope: ${scope}`);

  try {
    const proc = Bun.spawn(["claude", ...claudeArgs], {
      stdin: "inherit",
      stdout: "inherit",
      stderr: "inherit",
    });
    await proc.exited;

    if (proc.exitCode === 0) {
      console.log(`\nRemote MCP gateway '${name}' connected.`);
      console.log("Run '/mcp' in Claude Code to authenticate if needed.");
    } else {
      console.error(`\nFailed to connect. Exit code: ${proc.exitCode}`);
    }
  } catch (error) {
    console.error("Error: Could not run 'claude' command. Is Claude Code installed?");
    process.exit(1);
  }
}

/**
 * List all configured MCP servers.
 */
export async function mcpList() {
  console.log("\n=== Configured MCP Servers ===\n");

  // List via claude CLI
  try {
    const proc = Bun.spawn(["claude", "mcp", "list"], {
      stdout: "inherit",
      stderr: "inherit",
    });
    await proc.exited;
  } catch {
    console.log("Could not list servers via 'claude' CLI.\n");
  }

  // Also show .mcp.json if it exists
  const mcpJsonPath = join(process.cwd(), ".mcp.json");
  if (existsSync(mcpJsonPath)) {
    console.log("\n--- Project .mcp.json ---");
    try {
      const mcpConfig = JSON.parse(await Bun.file(mcpJsonPath).text());
      for (const [name, config] of Object.entries(mcpConfig.mcpServers || {})) {
        const cfg = config as any;
        if (cfg.type === "http" || cfg.url) {
          console.log(`  ${name}: ${cfg.url || cfg.type} (remote)`);
        } else {
          console.log(`  ${name}: ${cfg.command} ${(cfg.args || []).join(" ")} (local)`);
        }
      }
    } catch {
      console.log("  Could not parse .mcp.json");
    }
  }

  console.log("\n--- Quick Add Commands ---");
  console.log("  bun run cli/index.ts mcp add <name> -- <command>");
  console.log("  bun run cli/index.ts mcp remote <url>");
  console.log("");
}
