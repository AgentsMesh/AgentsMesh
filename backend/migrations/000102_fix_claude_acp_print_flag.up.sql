-- Fix Claude Code ACP: add required --print and --input-format flags
-- --output-format stream-json only works with -p/--print
-- --input-format stream-json enables bidirectional stream-json protocol
UPDATE agents SET podfile_source = REPLACE(
    podfile_source,
    E'if mode == "acp" {\n  arg "--output-format" "stream-json"\n}',
    E'if mode == "acp" {\n  arg "-p"\n  arg "--input-format" "stream-json"\n  arg "--output-format" "stream-json"\n}'
) WHERE slug = 'claude-code';
