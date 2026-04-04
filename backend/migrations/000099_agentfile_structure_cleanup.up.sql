-- AgentFile final structure: MCP ON FORMAT + mcp.servers auto-population
-- MCP ON is the single source of truth; mcp.servers auto-computed from builtin+installed+transform

-- Claude Code (no transform needed)
UPDATE agents SET agentfile_source = E'# === Identity ===\nAGENT claude\nEXECUTABLE claude\n\n# === Mode ===\nMODE pty\nMODE acp "-p" "--input-format" "stream-json" "--output-format" "stream-json"\n\n# === Configuration ===\nCONFIG model SELECT("", "sonnet", "opus") = ""\nCONFIG permission_mode SELECT("default", "plan", "bypassPermissions") = "default"\n\n# === Environment ===\nENV ANTHROPIC_API_KEY SECRET OPTIONAL\nENV ANTHROPIC_AUTH_TOKEN SECRET OPTIONAL\nENV ANTHROPIC_BASE_URL TEXT OPTIONAL\n\n# === Prompt ===\nPROMPT_POSITION prepend\n\n# === Capabilities ===\nMCP ON\nSKILLS am-delegate, am-channel\n\n# === Build Logic ===\narg "--model" config.model when config.model != ""\n\nif config.permission_mode == "plan" {\n  arg "--permission-mode" "plan"\n}\nif config.permission_mode == "bypassPermissions" {\n  arg "--dangerously-skip-permissions"\n}\n\nif mcp.enabled {\n  plugin_dir = sandbox.root + "/agentsmesh-plugin"\n\n  mkdir plugin_dir\n  mkdir plugin_dir + "/.claude-plugin"\n\n  file plugin_dir + "/.claude-plugin/plugin.json" json({\n    name: "agentsmesh",\n    description: "AgentsMesh collaboration plugin for Claude Code",\n    version: "1.0.0"\n  })\n\n  file plugin_dir + "/.mcp.json" json({ mcpServers: mcp.servers })\n\n  arg "--plugin-dir" plugin_dir\n}\n'
WHERE slug = 'claude-code';

-- Codex CLI (no transform needed)
UPDATE agents SET agentfile_source = E'# === Identity ===\nAGENT codex\nEXECUTABLE codex\n\n# === Mode ===\nMODE pty\nMODE acp "app-server"\n\n# === Configuration ===\nCONFIG approval_mode SELECT("untrusted", "on-request", "never") = "untrusted"\n\n# === Environment ===\nENV OPENAI_API_KEY SECRET OPTIONAL\n\n# === Prompt ===\nPROMPT_POSITION prepend\n\n# === Capabilities ===\nMCP ON\n\n# === Build Logic ===\narg "--ask-for-approval" config.approval_mode when config.approval_mode != "" and mode != "acp"\n\nif mcp.enabled {\n  mkdir sandbox.work_dir + "/.codex"\n  file sandbox.work_dir + "/.codex/mcp.json" json({ mcpServers: mcp.servers })\n}\n'
WHERE slug = 'codex-cli';

-- Gemini CLI (gemini transform)
UPDATE agents SET agentfile_source = E'# === Identity ===\nAGENT gemini\nEXECUTABLE gemini\n\n# === Mode ===\nMODE pty\nMODE acp "--experimental-acp"\n\n# === Configuration ===\nCONFIG sandbox_mode BOOL = false\n\n# === Environment ===\nENV GOOGLE_API_KEY SECRET OPTIONAL\n\n# === Prompt ===\nPROMPT_POSITION append\n\n# === Capabilities ===\nMCP ON FORMAT gemini\n\n# === Build Logic ===\narg "--sandbox" when config.sandbox_mode\n\nif mcp.enabled {\n  mkdir sandbox.work_dir + "/.gemini"\n  file sandbox.work_dir + "/.gemini/settings.json" json({ mcpServers: mcp.servers })\n}\n'
WHERE slug = 'gemini-cli';

-- OpenCode (opencode transform)
UPDATE agents SET agentfile_source = E'# === Identity ===\nAGENT opencode\nEXECUTABLE opencode\n\n# === Mode ===\nMODE pty\nMODE acp "acp"\n\n# === Prompt ===\nPROMPT_POSITION prepend\n\n# === Capabilities ===\nMCP ON FORMAT opencode\n\n# === Build Logic ===\nif mcp.enabled {\n  file sandbox.work_dir + "/opencode.json" json({ mcp: mcp.servers })\n}\n'
WHERE slug = 'opencode';
