-- Codex Rust reads MCP configuration from $CODEX_HOME/config.toml.
-- Keep CODEX_HOME pod-local so runner can merge platform MCP config into the
-- isolated Codex home before creating files.
UPDATE agents SET agentfile_source = E'# === Identity ===\nAGENT codex\nEXECUTABLE codex\n\n# === Mode ===\nMODE pty\nMODE acp "app-server"\n\n# === Configuration ===\nCONFIG approval_mode SELECT("untrusted", "on-request", "never") = "untrusted"\n\n# === Environment ===\nENV OPENAI_API_KEY SECRET OPTIONAL\nENV CODEX_HOME = sandbox.root + "/codex-home"\n\n# === Prompt ===\nPROMPT_POSITION prepend\n\n# === Capabilities ===\nMCP ON\n\n# === Build Logic ===\narg "--ask-for-approval" config.approval_mode when config.approval_mode != "" and mode != "acp"\n\nif mcp.enabled {\n  file sandbox.root + "/codex-home/config.toml" codex_mcp_toml(mcp.servers)\n}\n'
WHERE slug = 'codex-cli';
