import { existsSync } from "fs";
import { join, resolve, basename } from "path";
import { $ } from "bun";

/**
 * Creates sandboxed parallel branch worktrees for a project.
 *
 * This enables multiple Claude Code instances to work on the same
 * project simultaneously, each in its own isolated branch and directory.
 *
 * Uses git worktrees under the hood for true filesystem isolation
 * while sharing the same git history.
 */
export async function sandbox(projectPath?: string, branchName?: string) {
  if (!projectPath || !branchName) {
    console.error(
      "Usage: claude-workspace sandbox <project-path> <branch-name>",
    );
    console.log("\nExamples:");
    console.log("  claude-workspace sandbox ./my-project feature-auth");
    console.log("  claude-workspace sandbox ./my-project feature-api");
    console.log("  claude-workspace sandbox ./my-project bugfix-login");
    process.exit(1);
  }

  const projectDir = resolve(projectPath);

  if (!existsSync(projectDir)) {
    console.error(`Project directory not found: ${projectDir}`);
    process.exit(1);
  }

  // Verify it's a git repo
  try {
    await $`git -C ${projectDir} rev-parse --git-dir`.quiet();
  } catch {
    console.error(`Not a git repository: ${projectDir}`);
    console.error("Initialize git first: git init");
    process.exit(1);
  }

  const projectName = basename(projectDir);
  const worktreeBase = resolve(projectDir, "..", `${projectName}-worktrees`);
  const worktreeDir = join(worktreeBase, branchName);

  console.log(`\n=== Creating Sandboxed Branch: ${branchName} ===\n`);

  // Create worktrees directory
  await $`mkdir -p ${worktreeBase}`.quiet();

  // Check if worktree already exists
  if (existsSync(worktreeDir)) {
    console.log(`Worktree already exists at: ${worktreeDir}`);
    console.log(`To use it: cd ${worktreeDir} && claude`);
    return;
  }

  // Check if branch already exists
  let branchExists = false;
  try {
    await $`git -C ${projectDir} rev-parse --verify ${branchName}`.quiet();
    branchExists = true;
  } catch {
    branchExists = false;
  }

  // Create the worktree
  console.log("[1/4] Creating git worktree...");
  if (branchExists) {
    await $`git -C ${projectDir} worktree add ${worktreeDir} ${branchName}`;
  } else {
    await $`git -C ${projectDir} worktree add -b ${branchName} ${worktreeDir}`;
  }
  console.log(`  Worktree created at: ${worktreeDir}`);

  // Copy .claude configuration to worktree if it exists in main project
  console.log("[2/4] Setting up Claude configuration...");
  const claudeDir = join(projectDir, ".claude");
  if (existsSync(claudeDir)) {
    // The worktree will have .claude from git if it's tracked
    // Copy any local-only files
    const localSettings = join(claudeDir, "settings.local.json");
    if (existsSync(localSettings)) {
      const destLocal = join(worktreeDir, ".claude", "settings.local.json");
      await $`mkdir -p ${join(worktreeDir, ".claude")}`.quiet();
      await $`cp ${localSettings} ${destLocal}`.quiet();
      console.log("  Copied local settings to worktree");
    }

    const localClaudeMd = join(claudeDir, "CLAUDE.local.md");
    if (existsSync(localClaudeMd)) {
      await $`cp ${localClaudeMd} ${join(worktreeDir, ".claude/CLAUDE.local.md")}`.quiet();
      console.log("  Copied local CLAUDE.md to worktree");
    }
  }

  // Copy .mcp.json if not tracked by git
  console.log("[3/4] Setting up MCP configuration...");
  const mcpJson = join(projectDir, ".mcp.json");
  const worktreeMcp = join(worktreeDir, ".mcp.json");
  if (existsSync(mcpJson) && !existsSync(worktreeMcp)) {
    await $`cp ${mcpJson} ${worktreeMcp}`.quiet();
    console.log("  Copied .mcp.json to worktree");
  }

  // Install dependencies if needed
  console.log("[4/4] Setting up dependencies...");
  await installWorktreeDeps(worktreeDir);

  console.log(`\n=== Sandbox Ready ===`);
  console.log(`\nBranch:    ${branchName}`);
  console.log(`Directory: ${worktreeDir}`);
  console.log(`\nTo start working:`);
  console.log(`  cd ${worktreeDir}`);
  console.log(`  claude`);
  console.log(`\nTo list all worktrees:`);
  console.log(`  git -C ${projectDir} worktree list`);
  console.log(`\nTo remove this sandbox when done:`);
  console.log(`  git -C ${projectDir} worktree remove ${worktreeDir}`);
  console.log("");
}

async function installWorktreeDeps(worktreeDir: string) {
  // Check for package manager lock files and install deps
  if (
    existsSync(join(worktreeDir, "bun.lockb")) ||
    existsSync(join(worktreeDir, "bun.lock"))
  ) {
    try {
      await $`cd ${worktreeDir} && bun install`.quiet();
      console.log("  Dependencies installed (bun)");
    } catch {
      console.log("  Warning: Could not install bun dependencies");
    }
  } else if (existsSync(join(worktreeDir, "package-lock.json"))) {
    try {
      await $`cd ${worktreeDir} && npm ci`.quiet();
      console.log("  Dependencies installed (npm)");
    } catch {
      console.log("  Warning: Could not install npm dependencies");
    }
  } else if (existsSync(join(worktreeDir, "yarn.lock"))) {
    try {
      await $`cd ${worktreeDir} && yarn install --frozen-lockfile`.quiet();
      console.log("  Dependencies installed (yarn)");
    } catch {
      console.log("  Warning: Could not install yarn dependencies");
    }
  } else if (existsSync(join(worktreeDir, "pnpm-lock.yaml"))) {
    try {
      await $`cd ${worktreeDir} && pnpm install --frozen-lockfile`.quiet();
      console.log("  Dependencies installed (pnpm)");
    } catch {
      console.log("  Warning: Could not install pnpm dependencies");
    }
  } else if (existsSync(join(worktreeDir, "package.json"))) {
    console.log(
      "  No lockfile found. Run your package manager to install dependencies.",
    );
  } else {
    console.log("  No package.json found. Skipping dependency installation.");
  }
}
