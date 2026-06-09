import type { CredentialFormSpec } from "./types";

// AgentFile source: backend/migrations/000088_add_agentfile_source.up.sql
//   ENV ANTHROPIC_API_KEY SECRET OPTIONAL
//   ENV ANTHROPIC_AUTH_TOKEN SECRET OPTIONAL
//   ENV ANTHROPIC_BASE_URL TEXT OPTIONAL
export const claudeCodeFormSpec: CredentialFormSpec = {
  agentSlug: "claude-code",
  fields: [
    {
      kind: "text",
      envKey: "ANTHROPIC_BASE_URL",
      label: "settings.credentialForm.anthropic.baseUrl",
      placeholder: "https://api.anthropic.com",
      securityHint: "settings.agentCredentials.baseUrlSecurityHint",
    },
    {
      kind: "oneof",
      group: "anthropic_auth",
      label: "settings.credentialForm.anthropic.authMethod",
      description: "settings.credentialForm.anthropic.authMethodHint",
      options: [
        {
          kind: "secret",
          envKey: "ANTHROPIC_API_KEY",
          label: "settings.credentialForm.anthropic.apiKey",
          placeholder: "sk-ant-...",
        },
        {
          kind: "secret",
          envKey: "ANTHROPIC_AUTH_TOKEN",
          label: "settings.credentialForm.anthropic.authToken",
        },
      ],
    },
  ],
  allowCustomEnv: false,
};
