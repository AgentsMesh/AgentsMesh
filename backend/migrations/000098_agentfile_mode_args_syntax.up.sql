-- AgentFile syntax upgrade: MODE args + Codex Rust version + when mode conditions
--
-- 1. Claude Code: MODE acp with -p --input-format --output-format stream-json
-- 2. Codex CLI: MODE acp with app-server; Rust version approval args; when mode != "acp"
-- 3. Gemini CLI: MODE acp with --experimental-acp
-- 4. OpenCode: MODE acp with acp subcommand

-- Claude Code: replace if-block with declarative MODE args
UPDATE agents SET agentfile_source = REPLACE(
    agentfile_source,
    E'PROMPT_POSITION prepend',
    E'PROMPT_POSITION prepend\n\nMODE acp "-p" "--input-format" "stream-json" "--output-format" "stream-json"'
) WHERE slug = 'claude-code' AND agentfile_source NOT LIKE '%MODE acp%';

-- Codex CLI: update approval args to Rust version + add MODE acp + when condition
UPDATE agents SET agentfile_source = REPLACE(
    agentfile_source,
    E'CONFIG approval_mode SELECT("suggest", "auto-edit", "full-auto") = "suggest"',
    E'CONFIG approval_mode SELECT("untrusted", "on-request", "never") = "untrusted"'
) WHERE slug = 'codex-cli';

UPDATE agents SET agentfile_source = REPLACE(
    agentfile_source,
    E'arg "--approval-mode" config.approval_mode when config.approval_mode != ""',
    E'arg "--ask-for-approval" config.approval_mode when config.approval_mode != "" and mode != "acp"'
) WHERE slug = 'codex-cli';

UPDATE agents SET agentfile_source = REPLACE(
    agentfile_source,
    E'PROMPT_POSITION prepend',
    E'PROMPT_POSITION prepend\n\nMODE acp "app-server"'
) WHERE slug = 'codex-cli' AND agentfile_source NOT LIKE '%MODE acp%';

-- Gemini CLI: add MODE acp args
UPDATE agents SET agentfile_source = REPLACE(
    agentfile_source,
    E'PROMPT_POSITION append',
    E'PROMPT_POSITION append\n\nMODE acp "--experimental-acp"'
) WHERE slug = 'gemini-cli' AND agentfile_source NOT LIKE '%MODE acp%';

-- OpenCode: add MODE acp args
UPDATE agents SET agentfile_source = REPLACE(
    agentfile_source,
    E'PROMPT_POSITION prepend',
    E'PROMPT_POSITION prepend\n\nMODE acp "acp"'
) WHERE slug = 'opencode' AND agentfile_source NOT LIKE '%MODE acp%';
