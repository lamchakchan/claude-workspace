import { existsSync } from "fs";
import { join, resolve, relative } from "path";
import { $ } from "bun";

const PLATFORM_DIR = resolve(import.meta.dir, "../..");

/**
 * Attaches the Claude Code Platform configuration to a target project.
 *
 * This copies/symlinks the platform's agents, skills, hooks, settings,
 * and MCP configuration into the target project directory.
 *
 * Options:
 *   --symlink  Use symlinks instead of copies (keeps config in sync)
 *   --force    Overwrite existing configuration
 */
export async function attach(targetPath?: string) {
  if (!targetPath) {
    console.error(
      "Usage: claude-workspace attach <project-path> [--symlink] [--force]",
    );
    process.exit(1);
  }

  const projectDir = resolve(targetPath);
  const useSymlinks = process.argv.includes("--symlink");
  const force = process.argv.includes("--force");

  if (!existsSync(projectDir)) {
    console.error(`Project directory not found: ${projectDir}`);
    process.exit(1);
  }

  console.log(`\n=== Attaching Claude Platform to: ${projectDir} ===\n`);

  const claudeDir = join(projectDir, ".claude");

  // Create .claude directory
  await $`mkdir -p ${claudeDir}`.quiet();
  await $`mkdir -p ${join(claudeDir, "agents")}`.quiet();
  await $`mkdir -p ${join(claudeDir, "skills")}`.quiet();
  await $`mkdir -p ${join(claudeDir, "hooks")}`.quiet();
  await $`mkdir -p ${join(projectDir, "plans")}`.quiet();

  // Copy or symlink agents
  console.log("[1/6] Setting up agents...");
  await copyOrLink(
    join(PLATFORM_DIR, ".claude/agents"),
    join(claudeDir, "agents"),
    useSymlinks,
    force,
  );

  // Copy or symlink skills
  console.log("[2/6] Setting up skills...");
  await copyOrLink(
    join(PLATFORM_DIR, ".claude/skills"),
    join(claudeDir, "skills"),
    useSymlinks,
    force,
  );

  // Copy or symlink hooks
  console.log("[3/6] Setting up hooks...");
  await copyOrLink(
    join(PLATFORM_DIR, ".claude/hooks"),
    join(claudeDir, "hooks"),
    useSymlinks,
    force,
  );

  // Create or merge settings.json
  console.log("[4/6] Setting up settings...");
  await setupProjectSettings(claudeDir, force);

  // Create or merge .mcp.json
  console.log("[5/6] Setting up MCP configuration...");
  await setupMcpConfig(projectDir, force);

  // Create project CLAUDE.md if it doesn't exist
  console.log("[6/6] Setting up CLAUDE.md...");
  await setupProjectClaudeMd(projectDir, claudeDir, force);

  // Setup gitignore
  await setupGitignore(claudeDir);

  console.log("\n=== Attachment Complete ===");
  console.log(`\nPlatform attached to: ${projectDir}`);
  console.log("\nTo start Claude Code:");
  console.log(`  cd ${projectDir}`);
  console.log("  claude");
  console.log("\nTo customize for this project:");
  console.log(`  Edit ${join(claudeDir, "CLAUDE.md")} for team instructions`);
  console.log(
    `  Copy .claude/settings.local.json.example to .claude/settings.local.json for personal overrides`,
  );
  console.log("");
}

async function copyOrLink(
  src: string,
  dest: string,
  useSymlinks: boolean,
  force: boolean,
) {
  if (!existsSync(src)) {
    console.log(`  Skipping: ${src} does not exist`);
    return;
  }

  // Get all files from source
  const glob = new Bun.Glob("**/*");
  const sourceFiles: string[] = [];
  for await (const file of glob.scan({ cwd: src, onlyFiles: true })) {
    sourceFiles.push(file);
  }

  for (const file of sourceFiles) {
    const srcFile = join(src, file);
    const destFile = join(dest, file);
    const destDir = join(dest, file.split("/").slice(0, -1).join("/"));

    // Create intermediate directories
    await $`mkdir -p ${destDir}`.quiet();

    if (existsSync(destFile) && !force) {
      console.log(`  Skipping (exists): ${relative(process.cwd(), destFile)}`);
      continue;
    }

    if (useSymlinks) {
      // Remove existing file/link before creating symlink
      if (existsSync(destFile)) {
        await $`rm -f ${destFile}`.quiet();
      }
      await $`ln -s ${srcFile} ${destFile}`.quiet();
      console.log(`  Linked: ${file}`);
    } else {
      await $`cp ${srcFile} ${destFile}`.quiet();
      console.log(`  Copied: ${file}`);
    }
  }
}

async function setupProjectSettings(claudeDir: string, force: boolean) {
  const settingsPath = join(claudeDir, "settings.json");

  if (existsSync(settingsPath) && !force) {
    console.log("  Project settings already exist. Use --force to overwrite.");
    return;
  }

  // Read platform settings as base
  const platformSettings = JSON.parse(
    await Bun.file(join(PLATFORM_DIR, ".claude/settings.json")).text(),
  );

  await Bun.write(settingsPath, JSON.stringify(platformSettings, null, 2));
  console.log("  Created .claude/settings.json");

  // Copy the local settings example
  const examplePath = join(PLATFORM_DIR, ".claude/settings.local.json.example");
  if (existsSync(examplePath)) {
    const destExample = join(claudeDir, "settings.local.json.example");
    if (!existsSync(destExample) || force) {
      await $`cp ${examplePath} ${destExample}`.quiet();
      console.log("  Created .claude/settings.local.json.example");
    }
  }
}

async function setupMcpConfig(projectDir: string, force: boolean) {
  const mcpPath = join(projectDir, ".mcp.json");

  if (existsSync(mcpPath) && !force) {
    console.log("  MCP config already exists. Use --force to overwrite.");
    return;
  }

  const platformMcp = JSON.parse(
    await Bun.file(join(PLATFORM_DIR, ".mcp.json")).text(),
  );

  await Bun.write(mcpPath, JSON.stringify(platformMcp, null, 2));
  console.log("  Created .mcp.json");
}

async function setupProjectClaudeMd(
  projectDir: string,
  claudeDir: string,
  force: boolean,
) {
  const claudeMdPath = join(claudeDir, "CLAUDE.md");

  if (existsSync(claudeMdPath) && !force) {
    console.log(
      "  Project CLAUDE.md already exists. Use --force to overwrite.",
    );
    return;
  }

  // Detect project info for a better initial CLAUDE.md
  const projectName = projectDir.split("/").pop() || "project";
  let techStack = "Unknown";
  let buildCmd = "";
  let testCmd = "";

  // Try to detect from package.json
  const pkgPath = join(projectDir, "package.json");
  if (existsSync(pkgPath)) {
    try {
      const pkg = JSON.parse(await Bun.file(pkgPath).text());
      techStack = detectTechStack(pkg);
      buildCmd = pkg.scripts?.build ? `npm run build` : "";
      testCmd = pkg.scripts?.test ? `npm test` : "";
    } catch {}
  }

  // Try Cargo.toml
  if (existsSync(join(projectDir, "Cargo.toml"))) {
    techStack = "Rust";
    buildCmd = "cargo build";
    testCmd = "cargo test";
  }

  // Try go.mod
  if (existsSync(join(projectDir, "go.mod"))) {
    techStack = "Go";
    buildCmd = "go build ./...";
    testCmd = "go test ./...";
  }

  // Try pyproject.toml
  if (existsSync(join(projectDir, "pyproject.toml"))) {
    techStack = "Python";
    testCmd = "pytest";
  }

  const claudeMd = `# Project Instructions

## Project
Name: ${projectName}
Tech Stack: ${techStack}
${buildCmd ? `Build: \`${buildCmd}\`` : ""}
${testCmd ? `Test: \`${testCmd}\`` : ""}

## Conventions
<!-- Add your team's coding conventions here -->

## Key Directories
<!-- Map your project's important directories -->
<!-- Example:
- src/          - Application source code
- tests/        - Test files
- docs/         - Documentation
-->

## Important Notes
<!-- Add project-specific notes for Claude -->
`;

  await Bun.write(claudeMdPath, claudeMd);
  console.log("  Created .claude/CLAUDE.md (customize for your project)");

  // Copy local example
  const examplePath = join(PLATFORM_DIR, ".claude/CLAUDE.local.md.example");
  if (existsSync(examplePath)) {
    const destExample = join(claudeDir, "CLAUDE.local.md.example");
    if (!existsSync(destExample) || force) {
      await $`cp ${examplePath} ${destExample}`.quiet();
      console.log("  Created .claude/CLAUDE.local.md.example");
    }
  }
}

function detectTechStack(pkg: any): string {
  const deps = {
    ...pkg.dependencies,
    ...pkg.devDependencies,
  };

  const stack: string[] = [];

  if (deps?.next) stack.push("Next.js");
  else if (deps?.react) stack.push("React");
  else if (deps?.vue) stack.push("Vue");
  else if (deps?.["@angular/core"]) stack.push("Angular");
  else if (deps?.svelte) stack.push("Svelte");

  if (deps?.express) stack.push("Express");
  if (deps?.fastify) stack.push("Fastify");
  if (deps?.hono) stack.push("Hono");
  if (deps?.typescript) stack.push("TypeScript");
  if (deps?.prisma || deps?.["@prisma/client"]) stack.push("Prisma");
  if (deps?.drizzle) stack.push("Drizzle");

  return stack.length > 0 ? stack.join(", ") : "Node.js";
}

async function setupGitignore(claudeDir: string) {
  const gitignorePath = join(claudeDir, ".gitignore");

  const gitignoreContent = `# Personal local settings (not shared)
settings.local.json
CLAUDE.local.md

# Agent memory (personal)
agent-memory-local/

# Example files are tracked
!*.example
`;

  // Only create if it doesn't exist
  if (!existsSync(gitignorePath)) {
    await Bun.write(gitignorePath, gitignoreContent);
    console.log("  Created .claude/.gitignore");
  }
}
