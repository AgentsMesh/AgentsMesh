"use client";

import { useTranslations } from "next-intl";

export function FullExample() {
  const t = useTranslations();

  return (
    <section className="mb-12">
      <h2 className="text-2xl font-semibold mb-4">
        {t("docs.concepts.agentfile.fullExample.title")}
      </h2>
      <p className="text-muted-foreground leading-relaxed mb-6">
        {t("docs.concepts.agentfile.fullExample.description")}
      </p>
      <pre className="bg-muted rounded-lg p-4 text-sm overflow-x-auto">
        <code>{`# Claude Code AgentFile
AGENT "claude-code"
EXECUTABLE "claude"

# Execution mode: PTY terminal
MODE pty

# User-configurable options
CONFIG model STRING = "sonnet"
CONFIG permission_mode SELECT("default","bypassPermissions") = "default"
CONFIG verbose BOOL = false

# Environment variables
ENV ANTHROPIC_API_KEY SECRET
ENV CLAUDE_CODE_USE_BEDROCK TEXT OPTIONAL

# MCP tools enabled
MCP ON

# Build the CLI arguments
arg "--model " + config.model

if config.permission_mode == "bypassPermissions" {
  arg "--dangerously-skip-permissions"
}

when config.verbose arg "--verbose"

# Inject MCP server configuration
for server in mcp.servers {
  arg "--mcp-config " + server
}

# Setup script runs before the agent starts
SETUP timeout=30 <<INIT
echo "Workspace ready at $(pwd)"
INIT`}</code>
      </pre>
    </section>
  );
}
