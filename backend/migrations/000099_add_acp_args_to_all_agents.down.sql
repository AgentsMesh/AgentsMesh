-- Remove ACP mode startup arguments from Codex, Gemini, and OpenCode

UPDATE agents SET podfile_source = REPLACE(
    podfile_source,
    E'# --- build logic ---\n\nif mode == "acp" {\n  arg "app-server"\n}',
    E'# --- build logic ---'
) WHERE slug = 'codex-cli';

UPDATE agents SET podfile_source = REPLACE(
    podfile_source,
    E'# --- build logic ---\n\nif mode == "acp" {\n  arg "--experimental-acp"\n}',
    E'# --- build logic ---'
) WHERE slug = 'gemini-cli';

UPDATE agents SET podfile_source = REPLACE(
    podfile_source,
    E'# --- build logic ---\n\nif mode == "acp" {\n  arg "acp"\n}',
    E'# --- build logic ---'
) WHERE slug = 'opencode';
