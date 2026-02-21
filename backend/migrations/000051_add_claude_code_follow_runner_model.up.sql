-- Add "Follow Runner" (empty string) as default model option for Claude Code
-- When model is empty, no --model flag is passed to Claude CLI,
-- so it uses the default model configured on the runner's machine.

-- Update config_schema: add empty string option as first choice, set default to ""
UPDATE agent_types SET
    config_schema = jsonb_set(
        config_schema,
        '{fields}',
        (
            SELECT jsonb_agg(
                CASE
                    WHEN field->>'name' = 'model' THEN
                        jsonb_build_object(
                            'name', 'model',
                            'type', 'select',
                            'default', '',
                            'options', '[{"value": ""}, {"value": "sonnet"}, {"value": "opus"}]'::jsonb
                        )
                    ELSE field
                END
            )
            FROM jsonb_array_elements(config_schema->'fields') AS field
        )
    )
WHERE slug = 'claude-code'
  AND config_schema->'fields' IS NOT NULL;
