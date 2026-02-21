import { existsSync } from "fs";
import { join } from "path";
import { $ } from "bun";
import { homedir } from "os";

/**
 * Diagnoses the Claude Code Platform configuration.
 * Checks all components are properly set up and reports issues.
 */
export async function doctor() {
  console.log("\n=== Claude Platform Health Check ===\n");

  let issues = 0;
  let warnings = 0;

  // 1. Check Claude Code CLI
  console.log("[Claude Code CLI]");
  try {
    const version = (await $`claude --version`.text()).trim();
    pass(`Installed: ${version}`);
  } catch {
    fail("Claude Code CLI not found");
    console.log("    Install: npm install -g @anthropic-ai/claude-code");
    issues++;
  }

  // 2. Check Bun
  console.log("\n[Bun Runtime]");
  try {
    const bunVersion = (await $`bun --version`.text()).trim();
    pass(`Installed: ${bunVersion}`);
  } catch {
    fail("Bun not found");
    console.log("    Install: https://bun.sh");
    issues++;
  }

  // 3. Check Git
  console.log("\n[Git]");
  try {
    const gitVersion = (await $`git --version`.text()).trim();
    pass(gitVersion);
  } catch {
    fail("Git not found");
    issues++;
  }

  // 4. Check Global Settings
  console.log("\n[Global Configuration]");
  const globalSettingsPath = join(homedir(), ".claude", "settings.json");
  if (existsSync(globalSettingsPath)) {
    pass("~/.claude/settings.json exists");
    try {
      const settings = JSON.parse(await Bun.file(globalSettingsPath).text());
      if (settings.env?.CLAUDE_CODE_SUBAGENT_MODEL) {
        pass(`Subagent model: ${settings.env.CLAUDE_CODE_SUBAGENT_MODEL}`);
      }
      if (settings.env?.CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS === "1") {
        pass("Agent teams: enabled");
      }
    } catch {
      warn("Could not parse global settings");
      warnings++;
    }
  } else {
    warn("~/.claude/settings.json not found. Run 'claude-platform setup'");
    warnings++;
  }

  const globalClaudeMd = join(homedir(), ".claude", "CLAUDE.md");
  if (existsSync(globalClaudeMd)) {
    pass("~/.claude/CLAUDE.md exists");
  } else {
    warn("~/.claude/CLAUDE.md not found");
    warnings++;
  }

  // 5. Check Project Configuration
  console.log("\n[Project Configuration]");
  const cwd = process.cwd();

  const checks = [
    {
      path: ".claude/settings.json",
      label: "Project settings",
      required: true,
    },
    { path: ".claude/CLAUDE.md", label: "Project CLAUDE.md", required: true },
    { path: ".mcp.json", label: "MCP configuration", required: false },
    { path: ".claude/agents", label: "Agents directory", required: false },
    { path: ".claude/skills", label: "Skills directory", required: false },
    { path: ".claude/hooks", label: "Hooks directory", required: false },
    { path: "plans", label: "Plans directory", required: false },
  ];

  for (const check of checks) {
    const fullPath = join(cwd, check.path);
    if (existsSync(fullPath)) {
      pass(`${check.label}: ${check.path}`);
    } else if (check.required) {
      fail(`${check.label} not found: ${check.path}`);
      issues++;
    } else {
      warn(`${check.label} not found: ${check.path}`);
      warnings++;
    }
  }

  // 6. Check Agents
  console.log("\n[Agents]");
  const agentsDir = join(cwd, ".claude/agents");
  if (existsSync(agentsDir)) {
    const glob = new Bun.Glob("*.md");
    const agents: string[] = [];
    for await (const file of glob.scan({ cwd: agentsDir })) {
      agents.push(file.replace(".md", ""));
    }
    if (agents.length > 0) {
      pass(`Found ${agents.length} agents: ${agents.join(", ")}`);
    } else {
      warn("No agent definitions found");
      warnings++;
    }
  }

  // 7. Check Skills
  console.log("\n[Skills]");
  const skillsDir = join(cwd, ".claude/skills");
  if (existsSync(skillsDir)) {
    const glob = new Bun.Glob("**/SKILL.md");
    const skills: string[] = [];
    for await (const file of glob.scan({ cwd: skillsDir })) {
      skills.push(file.replace("/SKILL.md", ""));
    }
    if (skills.length > 0) {
      pass(`Found ${skills.length} skills: ${skills.join(", ")}`);
    } else {
      warn("No skill definitions found");
      warnings++;
    }
  }

  // 8. Check Hooks
  console.log("\n[Hooks]");
  const hooksDir = join(cwd, ".claude/hooks");
  if (existsSync(hooksDir)) {
    const glob = new Bun.Glob("*.sh");
    for await (const file of glob.scan({ cwd: hooksDir })) {
      const hookPath = join(hooksDir, file);
      try {
        await $`test -x ${hookPath}`.quiet();
        pass(`${file}: executable`);
      } catch {
        fail(`${file}: not executable. Run: chmod +x ${hookPath}`);
        issues++;
      }
    }
  }

  // 9. Check settings.json hooks reference valid scripts
  console.log("\n[Hook Configuration]");
  const settingsPath = join(cwd, ".claude/settings.json");
  if (existsSync(settingsPath)) {
    try {
      const settings = JSON.parse(await Bun.file(settingsPath).text());
      if (settings.hooks) {
        let hookCount = 0;
        for (const [event, matchers] of Object.entries(settings.hooks)) {
          for (const matcher of matchers as any[]) {
            for (const hook of matcher.hooks || []) {
              if (hook.type === "command") {
                hookCount++;
              }
            }
          }
        }
        pass(`${hookCount} hook commands configured`);
      }
    } catch {
      warn("Could not validate hook configuration");
      warnings++;
    }
  }

  // 10. Check MCP servers
  console.log("\n[MCP Servers]");
  const mcpPath = join(cwd, ".mcp.json");
  if (existsSync(mcpPath)) {
    try {
      const mcpConfig = JSON.parse(await Bun.file(mcpPath).text());
      const servers = Object.keys(mcpConfig.mcpServers || {});
      if (servers.length > 0) {
        pass(`${servers.length} MCP servers configured: ${servers.join(", ")}`);
      } else {
        warn("No MCP servers configured in .mcp.json");
        warnings++;
      }
    } catch {
      fail("Could not parse .mcp.json");
      issues++;
    }
  }

  // 11. Check API key / authentication
  console.log("\n[Authentication]");
  if (process.env.ANTHROPIC_API_KEY) {
    pass("ANTHROPIC_API_KEY is set");
  } else {
    const claudeConfig = join(homedir(), ".claude.json");
    if (existsSync(claudeConfig)) {
      try {
        const config = JSON.parse(await Bun.file(claudeConfig).text());
        if (config.oauthAccount) {
          pass("OAuth authentication configured");
        } else {
          warn("No API key or OAuth found. Run: claude-platform setup");
          warnings++;
        }
      } catch {
        warn("Could not read authentication config");
        warnings++;
      }
    } else {
      warn("No authentication configured. Run: claude-platform setup");
      warnings++;
    }
  }

  // Summary
  console.log("\n=== Summary ===");
  if (issues === 0 && warnings === 0) {
    console.log("All checks passed. Platform is healthy.");
  } else {
    if (issues > 0) console.log(`Issues: ${issues} (must fix)`);
    if (warnings > 0) console.log(`Warnings: ${warnings} (optional)`);
  }
  console.log("");
}

function pass(msg: string) {
  console.log(`  [OK] ${msg}`);
}

function fail(msg: string) {
  console.log(`  [FAIL] ${msg}`);
}

function warn(msg: string) {
  console.log(`  [WARN] ${msg}`);
}
