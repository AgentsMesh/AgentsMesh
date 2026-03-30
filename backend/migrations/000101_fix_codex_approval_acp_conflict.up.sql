-- Fix Codex CLI: approval arg conflicts with app-server subcommand in ACP mode
-- Wrap approval arg in mode check so it only applies in PTY mode
UPDATE agents SET podfile_source = REPLACE(
    podfile_source,
    E'arg "--ask-for-approval" config.approval_mode when config.approval_mode != ""',
    E'if mode != "acp" {\n  arg "--ask-for-approval" config.approval_mode when config.approval_mode != ""\n}'
) WHERE slug = 'codex-cli';
