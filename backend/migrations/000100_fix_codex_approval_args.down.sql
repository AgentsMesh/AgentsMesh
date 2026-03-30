-- Revert: restore old Node.js style Codex approval args
UPDATE agents SET podfile_source = REPLACE(
    podfile_source,
    E'CONFIG approval_mode SELECT("untrusted", "on-request", "never") = "untrusted"',
    E'CONFIG approval_mode SELECT("suggest", "auto-edit", "full-auto") = "suggest"'
) WHERE slug = 'codex-cli';

UPDATE agents SET podfile_source = REPLACE(
    podfile_source,
    E'arg "--ask-for-approval" config.approval_mode when config.approval_mode != ""',
    E'arg "--approval-mode" config.approval_mode when config.approval_mode != ""'
) WHERE slug = 'codex-cli';
