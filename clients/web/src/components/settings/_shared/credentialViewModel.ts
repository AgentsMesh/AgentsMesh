/**
 * Settings-shared ViewModel for the per-agent credentials UI. Adapts the
 * generic `EnvBundle` wire type (kind=credential) into a per-agent grouped
 * shape that the two settings sub-pages (AgentCredentialsSettings and
 * AgentConfigPage) already speak. Lives in `settings/_shared/` rather than
 * in `@/lib/api` because nothing outside the settings tree should think in
 * terms of "credential profiles" anymore — Pod creation and Loop binding
 * both consume `EnvBundle` directly.
 */
export interface CredentialProfileViewModel {
  id: number;
  agent_slug: string;
  name: string;
  description?: string;
  is_default: boolean;
  is_active: boolean;
  configured_fields?: string[];
  configured_values?: Record<string, string>;
  agent_name?: string;
  created_at: string;
  updated_at: string;
}

export interface CredentialProfilesByAgent {
  agent_slug: string;
  agent_name: string;
  profiles: CredentialProfileViewModel[];
}

/**
 * Keys the profile has configured: secret names (configured_fields, never their
 * values) plus non-secret keys whose plaintext round-trips (configured_values).
 * A bundle with only a non-secret value (e.g. just a base URL) is still
 * configured — judging by configured_fields alone would mislabel it empty.
 */
export function getConfiguredKeys(profile: {
  configured_fields?: string[];
  configured_values?: Record<string, string>;
}): string[] {
  // Deduped + sorted: the backend keeps the two slots disjoint, but unioning
  // without a Set would double-list a key if that regressed, and the
  // configured_values half arrives in Go-map (nondeterministic) order — sort so
  // the "Configured: …" summary and rebuilt custom-env rows stay stable.
  return [
    ...new Set([
      ...(profile.configured_fields ?? []),
      ...Object.keys(profile.configured_values ?? {}),
    ]),
  ].sort();
}
