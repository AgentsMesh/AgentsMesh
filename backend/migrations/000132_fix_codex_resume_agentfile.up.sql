-- Codex CLI does not resume a conversation by simply starting in the same
-- sandbox directory. In resume mode, run `codex resume --last` so Codex loads
-- the latest recorded session from the pod-local CODEX_HOME for that cwd.
--
-- PROMPT_POSITION append keeps normal launches as:
--   codex [options] <prompt>
-- and resume launches as:
--   codex resume --last [options] <prompt>
UPDATE agents SET agentfile_source = E'# === Identity ===\nAGENT codex\nEXECUTABLE codex\n\n# === Mode ===\nMODE pty\nMODE acp "app-server"\n\n# === Configuration ===\nCONFIG approval_mode SELECT("untrusted", "on-request", "never") = "untrusted"\n\n# === Environment ===\nENV OPENAI_API_KEY SECRET OPTIONAL\nENV CODEX_HOME = sandbox.root + "/codex-home"\n\n# === Prompt ===\nPROMPT_POSITION append\n\n# === Capabilities ===\nMCP ON\n\n# === Build Logic ===\narg "resume" "--last" when config.resume_enabled and mode != "acp"\narg "--ask-for-approval" config.approval_mode when config.approval_mode != "" and mode != "acp"\n\nif mcp.enabled {\n  file sandbox.root + "/codex-home/config.toml" codex_mcp_toml(mcp.servers)\n\n  mkdir sandbox.work_dir + "/.codex"\n  file sandbox.work_dir + "/.codex/mcp.json" json({ mcpServers: mcp.servers })\n}\n'
WHERE slug = 'codex-cli';
