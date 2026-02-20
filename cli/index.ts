#!/usr/bin/env bun

/**
 * Claude Platform CLI
 *
 * Bun-based tooling for configuring, installing, and managing
 * the Claude Code Platform Engineering Kit.
 *
 * Commands:
 *   setup              - First-time setup with API key provisioning
 *   attach <path>      - Attach platform config to a project
 *   sandbox <path> <n> - Create sandboxed parallel branches
 *   mcp add <args>     - Add a local MCP server
 *   mcp remote <url>   - Connect to a remote MCP gateway
 *   mcp list           - List configured MCP servers
 *   doctor             - Diagnose configuration issues
 */

import { setup } from "./commands/setup";
import { attach } from "./commands/attach";
import { sandbox } from "./commands/sandbox";
import { mcpAdd, mcpRemote, mcpList } from "./commands/mcp";
import { doctor } from "./commands/doctor";

const args = process.argv.slice(2);
const command = args[0];

const HELP = `
claude-platform - Claude Code Platform Engineering Kit CLI

Usage:
  bun run cli/index.ts <command> [options]

Commands:
  setup                          First-time setup & API key provisioning
  attach <project-path>          Attach platform config to a project
  sandbox <project-path> <name>  Create a sandboxed branch worktree
  mcp add <name> [-- <cmd>]      Add a local MCP server
  mcp remote <url>               Connect to a remote MCP gateway
  mcp list                       List all configured MCP servers
  doctor                         Check platform configuration health

Options:
  --help, -h    Show this help message

Examples:
  bun run cli/index.ts setup
  bun run cli/index.ts attach /path/to/my-project
  bun run cli/index.ts sandbox /path/to/my-project feature-auth
  bun run cli/index.ts mcp add postgres -- npx -y @bytebase/dbhub
  bun run cli/index.ts mcp remote https://mcp-gateway.company.com
`;

async function main() {
  if (!command || command === "--help" || command === "-h") {
    console.log(HELP);
    process.exit(0);
  }

  try {
    switch (command) {
      case "setup":
        await setup();
        break;
      case "attach":
        await attach(args[1]);
        break;
      case "sandbox":
        await sandbox(args[1], args[2]);
        break;
      case "mcp": {
        const subcommand = args[1];
        if (subcommand === "add") {
          await mcpAdd(args.slice(2));
        } else if (subcommand === "remote") {
          await mcpRemote(args[2], args.slice(3));
        } else if (subcommand === "list") {
          await mcpList();
        } else {
          console.error(`Unknown mcp subcommand: ${subcommand}`);
          console.log("Available: add, remote, list");
          process.exit(1);
        }
        break;
      }
      case "doctor":
        await doctor();
        break;
      default:
        console.error(`Unknown command: ${command}`);
        console.log(HELP);
        process.exit(1);
    }
  } catch (error) {
    console.error(`Error: ${error instanceof Error ? error.message : error}`);
    process.exit(1);
  }
}

main();
