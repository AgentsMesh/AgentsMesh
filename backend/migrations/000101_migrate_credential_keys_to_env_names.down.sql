-- Revert credential JSONB keys from full ENV names back to short names

-- Claude Code
UPDATE user_agent_credential_profiles
SET credentials_encrypted = (
    SELECT COALESCE(
        jsonb_object_agg(
            CASE key
                WHEN 'ANTHROPIC_API_KEY'    THEN 'api_key'
                WHEN 'ANTHROPIC_AUTH_TOKEN'  THEN 'auth_token'
                WHEN 'ANTHROPIC_BASE_URL'    THEN 'base_url'
                ELSE key
            END,
            value
        ),
        '{}'::jsonb
    )
    FROM jsonb_each(credentials_encrypted)
)
WHERE agent_slug = 'claude-code'
  AND credentials_encrypted IS NOT NULL
  AND credentials_encrypted != '{}'::jsonb;

-- Codex CLI
UPDATE user_agent_credential_profiles
SET credentials_encrypted = (
    SELECT COALESCE(
        jsonb_object_agg(
            CASE key
                WHEN 'OPENAI_API_KEY' THEN 'api_key'
                ELSE key
            END,
            value
        ),
        '{}'::jsonb
    )
    FROM jsonb_each(credentials_encrypted)
)
WHERE agent_slug = 'codex-cli'
  AND credentials_encrypted IS NOT NULL
  AND credentials_encrypted != '{}'::jsonb;

-- Gemini CLI
UPDATE user_agent_credential_profiles
SET credentials_encrypted = (
    SELECT COALESCE(
        jsonb_object_agg(
            CASE key
                WHEN 'GOOGLE_API_KEY' THEN 'api_key'
                ELSE key
            END,
            value
        ),
        '{}'::jsonb
    )
    FROM jsonb_each(credentials_encrypted)
)
WHERE agent_slug = 'gemini-cli'
  AND credentials_encrypted IS NOT NULL
  AND credentials_encrypted != '{}'::jsonb;

-- Aider
UPDATE user_agent_credential_profiles
SET credentials_encrypted = (
    SELECT COALESCE(
        jsonb_object_agg(
            CASE key
                WHEN 'OPENAI_API_KEY'    THEN 'api_key'
                WHEN 'ANTHROPIC_API_KEY'  THEN 'anthropic_api_key'
                ELSE key
            END,
            value
        ),
        '{}'::jsonb
    )
    FROM jsonb_each(credentials_encrypted)
)
WHERE agent_slug = 'aider'
  AND credentials_encrypted IS NOT NULL
  AND credentials_encrypted != '{}'::jsonb;
