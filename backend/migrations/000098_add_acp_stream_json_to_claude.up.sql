-- Add --output-format stream-json for ACP mode in Claude Code PodFile
UPDATE agents SET podfile_source = REPLACE(
    podfile_source,
    E'PROMPT_POSITION prepend',
    E'PROMPT_POSITION prepend\n\nif mode == "acp" {\n  arg "--output-format" "stream-json"\n}'
) WHERE slug = 'claude-code' AND podfile_source LIKE '%PROMPT_POSITION prepend%';
