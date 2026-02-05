/**
 * Git Provider Icons
 * Shared icon components for GitHub, GitLab, Gitee and other git providers.
 */

import { Settings, Key, GitBranch } from "lucide-react";
import { CredentialType } from "@/lib/api";

interface GitProviderIconProps {
  provider: string;
  className?: string;
}

/**
 * GitProviderIcon - Icon component for git providers (GitHub, GitLab, Gitee)
 */
export function GitProviderIcon({ provider, className = "w-5 h-5" }: GitProviderIconProps) {
  switch (provider) {
    case "github":
      return (
        <svg className={className} fill="currentColor" viewBox="0 0 24 24">
          <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
        </svg>
      );
    case "gitlab":
      return (
        <svg className={className} fill="currentColor" viewBox="0 0 24 24">
          <path d="m23.6 9.593-.033-.086L20.3.98a.851.851 0 0 0-.336-.405.875.875 0 0 0-1.004.054.868.868 0 0 0-.29.44l-2.208 6.763H7.538L5.33 1.07a.857.857 0 0 0-.29-.441.875.875 0 0 0-1.004-.053.851.851 0 0 0-.336.404L.433 9.502l-.032.086a6.066 6.066 0 0 0 2.012 7.01l.011.008.028.02 4.97 3.722 2.458 1.86 1.496 1.131a1.008 1.008 0 0 0 1.22 0l1.496-1.131 2.458-1.86 5-3.743.012-.01a6.068 6.068 0 0 0 2.009-7.002Z" />
        </svg>
      );
    case "gitee":
      return (
        <svg className={className} fill="currentColor" viewBox="0 0 24 24">
          <path d="M11.984 0A12 12 0 0 0 0 12a12 12 0 0 0 12 12 12 12 0 0 0 12-12A12 12 0 0 0 12 0a12 12 0 0 0-.016 0zm6.09 5.333c.328 0 .593.266.592.593v1.482a.594.594 0 0 1-.593.592H9.777c-.982 0-1.778.796-1.778 1.778v5.63c0 .327.266.592.593.592h5.63c.982 0 1.778-.796 1.778-1.778v-.296a.593.593 0 0 0-.592-.593h-4.15a.592.592 0 0 1-.592-.592v-1.482a.593.593 0 0 1 .593-.592h6.815c.327 0 .593.265.593.592v3.408a4 4 0 0 1-4 4H5.926a.593.593 0 0 1-.593-.593V9.778a4.444 4.444 0 0 1 4.445-4.444h8.296Z" />
        </svg>
      );
    default:
      return (
        <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
        </svg>
      );
  }
}

interface CredentialTypeIconProps {
  type: string;
  className?: string;
}

/**
 * CredentialTypeIcon - Icon component for git credential types (OAuth, PAT, SSH)
 */
export function CredentialTypeIcon({ type, className = "w-4 h-4" }: CredentialTypeIconProps) {
  switch (type) {
    case CredentialType.RUNNER_LOCAL:
      return <Settings className={className} />;
    case CredentialType.OAUTH:
      return <GitBranch className={className} />;
    case CredentialType.PAT:
      return <Key className={className} />;
    case CredentialType.SSH_KEY:
      return (
        <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
        </svg>
      );
    default:
      return <Key className={className} />;
  }
}
