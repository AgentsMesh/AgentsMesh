-- Revert: unwrap approval arg from mode check
UPDATE agents SET podfile_source = REPLACE(
    podfile_source,
    E'if mode != "acp" {\n  arg "--ask-for-approval" config.approval_mode when config.approval_mode != ""\n}',
    E'arg "--ask-for-approval" config.approval_mode when config.approval_mode != ""'
) WHERE slug = 'codex-cli';
