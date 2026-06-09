export type CredentialFieldKind = "text" | "secret";

export interface SimpleCredentialField {
  kind: CredentialFieldKind;
  envKey: string;
  label: string;
  description?: string;
  placeholder?: string;
  // i18n key for a security caveat rendered under the input, e.g. warning not
  // to embed secrets in a base URL whose plaintext round-trips to the wire.
  securityHint?: string;
}

export interface OneOfOption {
  kind: CredentialFieldKind;
  envKey: string;
  label: string;
  placeholder?: string;
}

export interface OneOfCredentialField {
  kind: "oneof";
  group: string;
  label: string;
  description?: string;
  options: OneOfOption[];
}

export type CredentialFieldSpec = SimpleCredentialField | OneOfCredentialField;

export interface CredentialFormSpec {
  agentSlug: string;
  fields: CredentialFieldSpec[];
  allowCustomEnv: boolean;
  customEnvHint?: string;
}

export interface CustomEnvEntry {
  id: string;
  key: string;
  value: string;
}

// Credential add/edit dialog submission shape. credentials key = full ENV name
// (e.g. "ANTHROPIC_API_KEY"), value = user input. SSOT — the two settings
// feature types re-export this.
export interface CredentialFormData {
  name: string;
  description: string;
  credentials: Record<string, string>;
}
