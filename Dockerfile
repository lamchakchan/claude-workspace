# Claude Code Platform Engineering Kit
# Pre-built image with all system dependencies for running Claude Code agents
#
# Build:  docker build -t claude-platform .
# Run:    docker run -it -v /path/to/project:/workspace claude-platform
#
# For your org's internal registry:
#   docker build -t registry.company.com/platform/claude-code:latest .
#   docker push registry.company.com/platform/claude-code:latest

FROM ubuntu:24.04 AS base

# Prevent interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# Install system dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    # Core tools
    curl \
    wget \
    git \
    jq \
    ca-certificates \
    gnupg \
    # Agent teams support
    tmux \
    # Shell script validation
    shellcheck \
    # Process management
    procps \
    # Network tools
    openssh-client \
    # File utilities
    unzip \
    zip \
    && rm -rf /var/lib/apt/lists/*

# Install Node.js (LTS) via NodeSource
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# Install Bun
RUN curl -fsSL https://bun.sh/install | bash
ENV PATH="/root/.bun/bin:${PATH}"

# Install Claude Code CLI
RUN npm install -g @anthropic-ai/claude-code

# Install common formatters (optional, used by auto-format hook)
RUN npm install -g prettier

# ---- Platform Setup ----

# Copy platform configuration
WORKDIR /opt/claude-platform
COPY . .

# Install platform Bun dependencies
RUN bun install

# Make hooks executable
RUN chmod +x .claude/hooks/*.sh

# Create global Claude config directory
RUN mkdir -p /root/.claude/agents

# Copy global settings and agents to user-level
RUN cp -r .claude/agents/* /root/.claude/agents/ 2>/dev/null || true

# ---- Runtime Configuration ----

# Default working directory (mount your project here)
WORKDIR /workspace

# Environment variables with safe defaults
ENV CLAUDE_CODE_ENABLE_TELEMETRY=1 \
    CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1 \
    CLAUDE_CODE_ENABLE_TASKS=true \
    CLAUDE_CODE_SUBAGENT_MODEL=sonnet \
    CLAUDE_AUTOCOMPACT_PCT_OVERRIDE=80 \
    # tmux for agent teams
    TERM=xterm-256color

# Entrypoint script handles setup and launches claude
COPY docker/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["claude"]
