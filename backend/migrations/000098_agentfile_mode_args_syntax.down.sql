-- Revert AgentFile MODE args syntax upgrade

-- Claude Code: remove MODE acp line
UPDATE agents SET agentfile_source = REPLACE(
    agentfile_source,
    E'\n\nMODE acp "-p" "--input-format" "stream-json" "--output-format" "stream-json"',
    E''
) WHERE slug = 'claude-code';

-- Codex CLI: revert approval args and remove MODE acp
UPDATE agents SET agentfile_source = REPLACE(
    agentfile_source,
    E'CONFIG approval_mode SELECT("untrusted", "on-request", "never") = "untrusted"',
    E'CONFIG approval_mode SELECT("suggest", "auto-edit", "full-auto") = "suggest"'
) WHERE slug = 'codex-cli';

UPDATE agents SET agentfile_source = REPLACE(
    agentfile_source,
    E'arg "--ask-for-approval" config.approval_mode when config.approval_mode != "" and mode != "acp"',
    E'arg "--approval-mode" config.approval_mode when config.approval_mode != ""'
) WHERE slug = 'codex-cli';

UPDATE agents SET agentfile_source = REPLACE(
    agentfile_source,
    E'\n\nMODE acp "app-server"',
    E''
) WHERE slug = 'codex-cli';

-- Gemini CLI: remove MODE acp line
UPDATE agents SET agentfile_source = REPLACE(
    agentfile_source,
    E'\n\nMODE acp "--experimental-acp"',
    E''
) WHERE slug = 'gemini-cli';

-- OpenCode: remove MODE acp line
UPDATE agents SET agentfile_source = REPLACE(
    agentfile_source,
    E'\n\nMODE acp "acp"',
    E''
) WHERE slug = 'opencode';
