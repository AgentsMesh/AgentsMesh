-- Fix Codex CLI PodFile: update arg name and option values for Rust version (0.106+)
-- Old (Node.js): --approval-mode suggest|auto-edit|full-auto
-- New (Rust):    --ask-for-approval untrusted|on-request|never

UPDATE agents SET podfile_source = REPLACE(
    podfile_source,
    E'CONFIG approval_mode SELECT("suggest", "auto-edit", "full-auto") = "suggest"',
    E'CONFIG approval_mode SELECT("untrusted", "on-request", "never") = "untrusted"'
) WHERE slug = 'codex-cli';

UPDATE agents SET podfile_source = REPLACE(
    podfile_source,
    E'arg "--approval-mode" config.approval_mode when config.approval_mode != ""',
    E'arg "--ask-for-approval" config.approval_mode when config.approval_mode != ""'
) WHERE slug = 'codex-cli';
