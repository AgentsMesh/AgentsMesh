export const CredentialType = {
  RUNNER_LOCAL: "runner_local",
  OAUTH: "oauth",
  PAT: "pat",
  SSH_KEY: "ssh_key",
} as const;

export type CredentialTypeValue =
  (typeof CredentialType)[keyof typeof CredentialType];

export interface GitCredentialData {
  id: number;
  user_id?: number;
  name: string;
  credential_type: CredentialTypeValue;
  repository_provider_id?: number;
  repository_provider?: {
    id: number;
    name: string;
    provider_type: string;
    base_url: string;
  };
  provider_name?: string;
  public_key?: string;
  fingerprint?: string;
  host_pattern?: string;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export interface RunnerLocalCredentialData {
  id: string;
  name: string;
  credential_type: "runner_local";
  is_default: boolean;
}

export interface CreateGitCredentialRequest {
  name: string;
  credential_type: CredentialTypeValue;
  repository_provider_id?: number;
  pat?: string;
  private_key?: string;
  host_pattern?: string;
}

export interface UpdateGitCredentialRequest {
  name?: string;
  pat?: string;
  private_key?: string;
  host_pattern?: string;
}

export interface SetDefaultRequest {
  credential_id?: number | null;
}

export function getCredentialTypeLabel(type: CredentialTypeValue): string {
  switch (type) {
    case CredentialType.RUNNER_LOCAL:
      return "Runner Local";
    case CredentialType.OAUTH:
      return "OAuth";
    case CredentialType.PAT:
      return "Personal Access Token";
    case CredentialType.SSH_KEY:
      return "SSH Key";
    default:
      return type;
  }
}

export function isRunnerLocalCredential(
  credential: GitCredentialData | RunnerLocalCredentialData
): credential is RunnerLocalCredentialData {
  return credential.credential_type === CredentialType.RUNNER_LOCAL;
}
