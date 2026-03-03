-- Add base_url (text, visible on edit) and auth_token (secret, hidden on edit) to Claude Code credential schema.
-- api_key and auth_token are mutually exclusive authentication methods, both set to required:false.
-- Frontend enforces mutual exclusivity; backend stores whichever the user provides.
UPDATE agent_types
SET credential_schema = '[
  {"name":"api_key","type":"secret","env_var":"ANTHROPIC_API_KEY","required":false},
  {"name":"auth_token","type":"secret","env_var":"ANTHROPIC_AUTH_TOKEN","required":false},
  {"name":"base_url","type":"text","env_var":"ANTHROPIC_BASE_URL","required":false}
]'
WHERE slug = 'claude-code';
