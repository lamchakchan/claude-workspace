# Plugins

Claude Code plugins extend the agent's capabilities with reusable tools, skills, and integrations. The `claude-workspace plugins` command manages plugin installation, removal, and discovery.

## Overview

Plugins are distributed through **marketplaces** — Git repositories containing curated collections of plugins. Each plugin lives in its own directory with a `.claude-plugin/plugin.json` manifest that declares its name, description, and capabilities.

### Plugin Scopes

| Scope | Location | Effect |
|-------|----------|--------|
| `user` | `~/.claude/plugins/` | Available in all projects for the current user |
| `project` | `.claude/plugins/` | Available only in the current project, shared with the team |

## Managing Plugins

### List Installed Plugins

```bash
claude-workspace plugins
# or
claude-workspace plugins list
```

Shows all installed plugins with their scope, version, enabled state, and description.

### Install a Plugin

```bash
claude-workspace plugins add <plugin[@marketplace]> [--scope user|project]
```

Installs a plugin from a configured marketplace. The default scope is `user`.

**Examples:**

```bash
# Install from the official marketplace
claude-workspace plugins add skill-creator@claude-plugins-official

# Install with project scope (shared with team via Git)
claude-workspace plugins add my-tool@my-marketplace --scope project
```

### Remove a Plugin

```bash
claude-workspace plugins remove <plugin[@marketplace]> [--scope user|project]
```

Removes an installed plugin.

**Examples:**

```bash
claude-workspace plugins remove skill-creator@claude-plugins-official
```

### Browse Available Plugins

```bash
claude-workspace plugins available
```

Lists all plugins from configured marketplaces, grouped by marketplace name. Installed plugins are marked with a checkmark.

## Marketplaces

A marketplace is a Git repository that contains plugins in a standard directory structure. Claude Code ships with support for the official Anthropic marketplace.

### Managing Marketplaces via CLI

```bash
# List configured marketplaces
claude-workspace plugins marketplace list

# Add a marketplace by owner/repo
claude-workspace plugins marketplace add anthropics/claude-plugins-official

# Remove a configured marketplace
claude-workspace plugins marketplace remove claude-plugins-official
```

Adding a marketplace clones the repository to `~/.claude/plugins/marketplaces/` and makes its plugins available for installation.

### Curated Registry

`claude-workspace` ships with a curated registry of known marketplaces (embedded in `docs/plugin-marketplaces/marketplaces.json`). The TUI marketplace picker displays these curated entries alongside an option to add custom marketplaces by `owner/repo`. Already-configured marketplaces appear dimmed.

### Marketplace Structure

```
marketplace-repo/
  plugins/
    plugin-a/
      .claude-plugin/
        plugin.json
      skills/
        ...
    plugin-b/
      .claude-plugin/
        plugin.json
      ...
```

## Platform-Managed Plugins

The `claude-workspace setup` command automatically installs recommended plugins:

- **skill-creator** — Create and improve Claude Code skills with iterative evaluation. See [Skills - Using skill-creator](SKILLS.md#using-skill-creator) for details.

## TUI Integration

The interactive TUI (`claude-workspace` with no arguments) includes a **Plugins** group with six actions:

| Item | Description |
|------|-------------|
| Install Plugin | Browse available marketplace plugins and install |
| List Plugins | View all installed plugins |
| Remove Plugin | Select and remove an installed plugin with confirmation |
| Add Marketplace | Browse curated marketplaces or add a custom one by owner/repo |
| List Marketplaces | View all configured marketplaces with repo and plugin count |
| Remove Marketplace | Select and remove a configured marketplace with confirmation |

## Plugin Configuration

### Enabled State

Plugins can be enabled or disabled in `~/.claude/settings.json` via the `enabledPlugins` map:

```json
{
  "enabledPlugins": {
    "skill-creator@claude-plugins-official": true,
    "my-plugin@my-marketplace": false
  }
}
```

Disabled plugins remain installed but are not loaded by Claude Code.

### Plugin Manifest

Each plugin contains a `.claude-plugin/plugin.json` manifest:

```json
{
  "name": "skill-creator",
  "description": "Create new skills, modify and improve existing skills, and measure skill performance."
}
```

## See Also

- [CLI Reference - plugins](CLI.md#claude-workspace-plugins) — Full command reference
- [Skills](SKILLS.md) — Built-in skills and creating custom skills
- [Getting Started](GETTING-STARTED.md) — Installation and first-time setup
