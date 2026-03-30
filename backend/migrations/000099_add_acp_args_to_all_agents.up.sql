-- Add ACP mode startup arguments for Codex, Gemini, and OpenCode
-- Codex needs "app-server" subcommand
-- Gemini needs "--experimental-acp" flag
-- OpenCode needs "acp" subcommand

-- Codex CLI: add "app-server" as first arg in ACP mode
UPDATE agents SET podfile_source = REPLACE(
    podfile_source,
    E'# --- build logic ---',
    E'# --- build logic ---\n\nif mode == "acp" {\n  arg "app-server"\n}'
) WHERE slug = 'codex-cli' AND podfile_source LIKE '%# --- build logic ---%';

-- Gemini CLI: add "--experimental-acp" flag in ACP mode
UPDATE agents SET podfile_source = REPLACE(
    podfile_source,
    E'# --- build logic ---',
    E'# --- build logic ---\n\nif mode == "acp" {\n  arg "--experimental-acp"\n}'
) WHERE slug = 'gemini-cli' AND podfile_source LIKE '%# --- build logic ---%';

-- OpenCode: add "acp" subcommand in ACP mode
UPDATE agents SET podfile_source = REPLACE(
    podfile_source,
    E'# --- build logic ---',
    E'# --- build logic ---\n\nif mode == "acp" {\n  arg "acp"\n}'
) WHERE slug = 'opencode' AND podfile_source LIKE '%# --- build logic ---%';
