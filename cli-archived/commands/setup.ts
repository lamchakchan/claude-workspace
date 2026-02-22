import { $ } from "bun";
import { existsSync } from "fs";
import { join } from "path";
import { homedir } from "os";

const CLAUDE_HOME = join(homedir(), ".claude");
const CLAUDE_CONFIG = join(homedir(), ".claude.json");

/**
 * First-time setup for Claude Code Platform.
 * Handles:
 *   1. Verify Claude Code is installed
 *   2. API key provisioning (self-service via Option 2)
 *   3. Create global user settings
 *   4. Create global CLAUDE.md
 *   5. Install dependencies and register CLI
 *   6. Check for optional system tools
 */
export async function setup() {
  console.log("\n=== Claude Code Platform Setup ===\n");

  // Step 1: Check if Claude Code CLI is installed
  console.log("[1/6] Checking Claude Code installation...");
  const claudeInstalled = await checkClaudeInstalled();
  if (!claudeInstalled) {
    console.log("  Claude Code CLI not found. Installing...");
    await installClaude();
  } else {
    const version = await getClaudeVersion();
    console.log(`  Claude Code CLI found: ${version}`);
  }

  // Step 2: API Key provisioning (Option 2 - self-provision)
  console.log("\n[2/6] API Key provisioning...");
  await provisionApiKey();

  // Step 3: Create global user settings
  console.log("\n[3/6] Setting up global user configuration...");
  await setupGlobalSettings();

  // Step 4: Create global CLAUDE.md
  console.log("\n[4/6] Setting up global CLAUDE.md...");
  await setupGlobalClaudeMd();

  // Step 5: Install dependencies and register CLI
  console.log("\n[5/6] Installing dependencies and registering CLI...");
  await installDependencies();

  // Step 6: Check optional system tools
  console.log("\n[6/6] Checking optional system tools...");
  await checkOptionalTools();

  console.log("\n=== Setup Complete ===");
  console.log("\nNext steps:");
  console.log(
    "  1. Attach to a project:  claude-workspace attach /path/to/project",
  );
  console.log("  2. Start Claude Code:    cd /path/to/project && claude");
  console.log(
    "  3. Add MCP servers:      claude-workspace mcp add <name> -- <command>",
  );
  console.log("");
}

async function checkClaudeInstalled(): Promise<boolean> {
  try {
    await $`which claude`.quiet();
    return true;
  } catch {
    return false;
  }
}

async function getClaudeVersion(): Promise<string> {
  try {
    const result = await $`claude --version`.text();
    return result.trim();
  } catch {
    return "unknown";
  }
}

async function installClaude() {
  console.log("  Installing Claude Code via npm...");
  try {
    await $`npm install -g @anthropic-ai/claude-code`.quiet();
    console.log("  Claude Code installed successfully.");
  } catch (error) {
    console.error("  Failed to install Claude Code automatically.");
    console.log(
      "  Please install manually: npm install -g @anthropic-ai/claude-code",
    );
    console.log("  Or visit: https://docs.anthropic.com/en/docs/claude-code");
    process.exit(1);
  }
}

async function provisionApiKey() {
  // Check if already authenticated
  if (existsSync(CLAUDE_CONFIG)) {
    try {
      const config = JSON.parse(await Bun.file(CLAUDE_CONFIG).text());
      if (config.oauthAccount || config.primaryApiKey) {
        console.log("  Already authenticated. Skipping API key provisioning.");
        return;
      }
    } catch {
      // Config exists but can't be parsed - continue with setup
    }
  }

  console.log("  Starting self-service API key provisioning (Option 2)...");
  console.log("  This will open Claude Code's interactive login flow.");
  console.log("  Select 'Use an API key' when prompted.\n");

  // Launch Claude Code which will trigger the login flow
  const proc = Bun.spawn(["claude", "--print-api-key-config"], {
    stdin: "inherit",
    stdout: "inherit",
    stderr: "inherit",
  });

  await proc.exited;

  if (proc.exitCode !== 0) {
    console.log("\n  API key provisioning requires interactive setup.");
    console.log("  Run 'claude' directly to complete the login flow.");
    console.log(
      "  You can set ANTHROPIC_API_KEY in your environment as an alternative.",
    );
  }
}

async function setupGlobalSettings() {
  const settingsPath = join(CLAUDE_HOME, "settings.json");

  if (existsSync(settingsPath)) {
    console.log(
      "  Global settings already exist. Merging platform defaults...",
    );
    try {
      const existing = JSON.parse(await Bun.file(settingsPath).text());
      const merged = mergeSettings(existing, getDefaultGlobalSettings());
      await Bun.write(settingsPath, JSON.stringify(merged, null, 2));
      console.log("  Global settings updated.");
    } catch (error) {
      console.log(
        "  Could not merge settings. Skipping global settings update.",
      );
    }
    return;
  }

  // Create ~/.claude/ directory if needed
  await $`mkdir -p ${CLAUDE_HOME}`.quiet();

  await Bun.write(
    settingsPath,
    JSON.stringify(getDefaultGlobalSettings(), null, 2),
  );
  console.log("  Global settings created at ~/.claude/settings.json");
}

function getDefaultGlobalSettings() {
  return {
    $schema: "https://json.schemastore.org/claude-code-settings.json",
    env: {
      CLAUDE_CODE_ENABLE_TELEMETRY: "1",
      CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS: "1",
      CLAUDE_CODE_ENABLE_TASKS: "true",
      CLAUDE_CODE_SUBAGENT_MODEL: "sonnet",
      CLAUDE_AUTOCOMPACT_PCT_OVERRIDE: "80",
    },
    permissions: {
      deny: [
        "Bash(rm -rf /)",
        "Bash(rm -rf /*)",
        "Bash(git push --force * main)",
        "Bash(git push --force * master)",
        "Bash(git push -f * main)",
        "Bash(git push -f * master)",
        "Read(./.env)",
        "Read(./.env.*)",
        "Read(./secrets/**)",
      ],
    },
    alwaysThinkingEnabled: true,
    showTurnDuration: true,
  };
}

function mergeSettings(
  existing: Record<string, any>,
  defaults: Record<string, any>,
): Record<string, any> {
  const merged = { ...existing };

  // Merge env vars (don't overwrite existing)
  if (defaults.env) {
    merged.env = { ...defaults.env, ...existing.env };
  }

  // Merge deny permissions (union)
  if (defaults.permissions?.deny) {
    const existingDeny = existing.permissions?.deny || [];
    const newDeny = defaults.permissions.deny.filter(
      (rule: string) => !existingDeny.includes(rule),
    );
    merged.permissions = {
      ...existing.permissions,
      deny: [...existingDeny, ...newDeny],
    };
  }

  // Set boolean flags only if not already set
  for (const key of ["alwaysThinkingEnabled", "showTurnDuration"]) {
    if (merged[key] === undefined && defaults[key] !== undefined) {
      merged[key] = defaults[key];
    }
  }

  return merged;
}

async function setupGlobalClaudeMd() {
  const claudeMdPath = join(CLAUDE_HOME, "CLAUDE.md");

  if (existsSync(claudeMdPath)) {
    console.log("  Global CLAUDE.md already exists. Skipping.");
    return;
  }

  const globalClaudeMd = `# Global Claude Code Instructions

## Identity
You are an AI coding agent operating within a governed platform environment.
Follow the platform conventions, use subagents for delegation, and plan before implementing.

## Defaults
- Always use TodoWrite for multi-step tasks
- Prefer Sonnet for coding, Haiku for exploration
- Read files before modifying them
- Run tests after making changes
- Never commit secrets or credentials

## Git Conventions
- Work on feature branches, never main/master
- Commit messages: imperative mood, explain "why"
- Create PRs with clear descriptions
`;

  await Bun.write(claudeMdPath, globalClaudeMd);
  console.log("  Global CLAUDE.md created at ~/.claude/CLAUDE.md");
}

async function installDependencies() {
  const platformDir = import.meta.dir.replace("/cli/commands", "");

  try {
    // Check if bun is available
    await $`which bun`.quiet();
    console.log("  Bun is available. Installing dependencies...");
    await $`cd ${platformDir} && bun install`.quiet();
    console.log("  Dependencies installed.");

    // Register claude-workspace as a global command via bun link
    console.log("  Registering claude-workspace command...");
    await $`cd ${platformDir} && bun link`.quiet();
    console.log("  Registered: claude-workspace is now available globally.");
  } catch {
    console.log("  Bun not found. Install Bun: https://bun.sh");
    console.log("  Then run: cd ~/claude-workspace && bun install && bun link");
  }
}

async function checkOptionalTools() {
  const tools: Array<{
    name: string;
    check: string;
    purpose: string;
    install: string;
  }> = [
    {
      name: "shellcheck",
      check: "shellcheck",
      purpose: "Hook script validation",
      install:
        "brew install shellcheck (macOS) / apt install shellcheck (Linux)",
    },
    {
      name: "jq",
      check: "jq",
      purpose: "JSON processing in hooks",
      install: "brew install jq (macOS) / apt install jq (Linux)",
    },
    {
      name: "prettier",
      check: "prettier",
      purpose: "Auto-format hook (JS/TS/JSON/CSS)",
      install: "npm install -g prettier",
    },
    {
      name: "tmux",
      check: "tmux",
      purpose:
        "Agent teams split-pane mode (optional — in-process mode works without it)",
      install: "brew install tmux (macOS) / apt install tmux (Linux)",
    },
  ];

  const missing: typeof tools = [];
  const found: string[] = [];

  for (const tool of tools) {
    try {
      await $`which ${tool.check}`.quiet();
      found.push(tool.name);
    } catch {
      missing.push(tool);
    }
  }

  if (found.length > 0) {
    console.log(`  Found: ${found.join(", ")}`);
  }

  if (missing.length > 0) {
    console.log(`\n  Optional tools not found (not required, but useful):`);
    for (const tool of missing) {
      console.log(`    - ${tool.name}: ${tool.purpose}`);
      console.log(`      Install: ${tool.install}`);
    }
  } else {
    console.log("  All optional tools are available.");
  }
}
