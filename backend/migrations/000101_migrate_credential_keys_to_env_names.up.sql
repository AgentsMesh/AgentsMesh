-- Migrate credential JSONB keys from short names to full ENV names
-- (AgentFile SSOT: credential keys must match ENV declaration names for eval injection)
-- Values are encrypted — only renaming keys, no re-encryption needed.

-- Claude Code: api_key → ANTHROPIC_API_KEY, auth_token → ANTHROPIC_AUTH_TOKEN, base_url → ANTHROPIC_BASE_URL
UPDATE user_agent_credential_profiles
SET credentials_encrypted = (
    SELECT COALESCE(
        jsonb_object_agg(
            CASE key
                WHEN 'api_key'    THEN 'ANTHROPIC_API_KEY'
                WHEN 'auth_token' THEN 'ANTHROPIC_AUTH_TOKEN'
                WHEN 'base_url'   THEN 'ANTHROPIC_BASE_URL'
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

-- Codex CLI: api_key → OPENAI_API_KEY
UPDATE user_agent_credential_profiles
SET credentials_encrypted = (
    SELECT COALESCE(
        jsonb_object_agg(
            CASE key
                WHEN 'api_key' THEN 'OPENAI_API_KEY'
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

-- Gemini CLI: api_key → GOOGLE_API_KEY
UPDATE user_agent_credential_profiles
SET credentials_encrypted = (
    SELECT COALESCE(
        jsonb_object_agg(
            CASE key
                WHEN 'api_key' THEN 'GOOGLE_API_KEY'
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

-- Aider: api_key → OPENAI_API_KEY, anthropic_api_key → ANTHROPIC_API_KEY
UPDATE user_agent_credential_profiles
SET credentials_encrypted = (
    SELECT COALESCE(
        jsonb_object_agg(
            CASE key
                WHEN 'api_key'           THEN 'OPENAI_API_KEY'
                WHEN 'anthropic_api_key' THEN 'ANTHROPIC_API_KEY'
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
