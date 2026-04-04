-- Add agentfile_source column to agent_types
ALTER TABLE agent_types ADD COLUMN IF NOT EXISTS agentfile_source TEXT;

-- Claude Code AgentFile
UPDATE agent_types SET agentfile_source = '
AGENT claude
EXECUTABLE claude

CONFIG mcp_enabled BOOL = true
CONFIG model SELECT("", "sonnet", "opus") = ""
CONFIG permission_mode SELECT("default", "plan", "bypassPermissions") = "default"

ENV ANTHROPIC_API_KEY SECRET OPTIONAL
ENV ANTHROPIC_AUTH_TOKEN SECRET OPTIONAL
ENV ANTHROPIC_BASE_URL TEXT OPTIONAL

MCP ON
SKILLS am-delegate, am-channel

# --- build logic ---

arg "--model" config.model when config.model != ""

if config.permission_mode == "plan" {
  arg "--permission-mode" "plan"
}
if config.permission_mode == "bypassPermissions" {
  arg "--dangerously-skip-permissions"
}

prompt prepend

if mcp.enabled {
  mcp_cfg = json_merge(mcp.builtin, mcp.installed)
  plugin_dir = sandbox.root + "/agentsmesh-plugin"

  mkdir plugin_dir
  mkdir plugin_dir + "/.claude-plugin"

  file plugin_dir + "/.claude-plugin/plugin.json" json({
    name: "agentsmesh",
    description: "AgentsMesh collaboration plugin for Claude Code",
    version: "1.0.0"
  })

  file plugin_dir + "/.mcp.json" json({ mcpServers: mcp_cfg })

  arg "--plugin-dir" plugin_dir
}
' WHERE slug = 'claude-code';

-- Gemini CLI AgentFile
UPDATE agent_types SET agentfile_source = '
AGENT gemini
EXECUTABLE gemini

CONFIG mcp_enabled BOOL = true
CONFIG sandbox_mode BOOL = false

ENV GOOGLE_API_KEY SECRET OPTIONAL

MCP ON

# --- build logic ---

arg "--sandbox" when config.sandbox_mode

prompt append

if mcp.enabled {
  mcp_cfg = mcp_transform(json_merge(mcp.builtin, mcp.installed), "gemini")
  mkdir sandbox.work_dir + "/.gemini"
  file sandbox.work_dir + "/.gemini/settings.json" json({ mcpServers: mcp_cfg })
}
' WHERE slug = 'gemini-cli';

-- Codex CLI AgentFile
UPDATE agent_types SET agentfile_source = '
AGENT codex
EXECUTABLE codex

CONFIG mcp_enabled BOOL = true
CONFIG approval_mode SELECT("suggest", "auto-edit", "full-auto") = "suggest"

ENV OPENAI_API_KEY SECRET OPTIONAL

MCP ON

# --- build logic ---

arg "--approval-mode" config.approval_mode when config.approval_mode != ""

prompt prepend

if mcp.enabled {
  mcp_cfg = json_merge(mcp.builtin, mcp.installed)
  mkdir sandbox.work_dir + "/.codex"
  file sandbox.work_dir + "/.codex/mcp.json" json({ mcpServers: mcp_cfg })
}
' WHERE slug = 'codex-cli';

-- Aider AgentFile
UPDATE agent_types SET agentfile_source = '
AGENT aider
EXECUTABLE aider

CONFIG model STRING = ""
CONFIG edit_format SELECT("", "whole", "diff", "udiff") = ""

ENV OPENAI_API_KEY SECRET OPTIONAL
ENV ANTHROPIC_API_KEY SECRET OPTIONAL

MCP OFF

# --- build logic ---

arg "--model" config.model when config.model != ""
arg "--edit-format" config.edit_format when config.edit_format != ""

prompt none
' WHERE slug = 'aider';

-- OpenCode AgentFile
UPDATE agent_types SET agentfile_source = '
AGENT opencode
EXECUTABLE opencode

CONFIG mcp_enabled BOOL = true

MCP ON

# --- build logic ---

prompt prepend

if mcp.enabled {
  mcp_cfg = mcp_transform(json_merge(mcp.builtin, mcp.installed), "opencode")
  file sandbox.work_dir + "/opencode.json" json({ mcp: mcp_cfg })
}
' WHERE slug = 'opencode';
