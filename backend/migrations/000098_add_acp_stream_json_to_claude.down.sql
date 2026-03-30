-- Remove ACP stream-json argument from Claude Code PodFile
UPDATE agents SET podfile_source = REPLACE(
    podfile_source,
    E'PROMPT_POSITION prepend\n\nif mode == "acp" {\n  arg "--output-format" "stream-json"\n}',
    E'PROMPT_POSITION prepend'
) WHERE slug = 'claude-code';
