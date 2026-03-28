-- Add ACP mode support for Codex CLI.
-- Codex CLI >= 0.100 (Rust rewrite) supports the app-server subcommand
-- which provides a JSON-RPC 2.0 protocol for programmatic interaction.
UPDATE agent_types SET supported_modes = 'pty,acp' WHERE slug = 'codex-cli';
