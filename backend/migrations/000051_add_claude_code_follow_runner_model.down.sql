-- Revert: restore original model options (opus, sonnet) with sonnet as default
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
                            'default', 'sonnet',
                            'options', '[{"value": "opus"}, {"value": "sonnet"}]'::jsonb
                        )
                    ELSE field
                END
            )
            FROM jsonb_array_elements(config_schema->'fields') AS field
        )
    )
WHERE slug = 'claude-code'
  AND config_schema->'fields' IS NOT NULL;
