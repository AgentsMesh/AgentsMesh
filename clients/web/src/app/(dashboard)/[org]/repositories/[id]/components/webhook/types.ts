import type { WebhookStatus, RepositoryData } from "@/lib/api";

export interface WebhookSettingsCardProps {
  repository: RepositoryData;
  onStatusChange?: () => void;
}

/**
 * Webhook secret info for manual setup
 * Maps from API response (webhook_url, webhook_secret) to display format
 */
export interface WebhookSecretInfo {
  url: string;          // webhook_url from API
  secret: string;       // webhook_secret from API
  events: string[];
}

export type WebhookActionType =
  | "register"
  | "delete"
  | "markConfigured"
  | "getSecret"
  | null;

export type CopiedField = "url" | "secret" | null;

export interface WebhookStatusDisplayProps {
  status: WebhookStatus | null;
  t: (key: string) => string;
}

export interface WebhookActiveInfoProps {
  status: WebhookStatus;
  t: (key: string) => string;
}

export interface WebhookManualSetupProps {
  repository: RepositoryData;
  secretInfo: WebhookSecretInfo | null;
  showSecret: boolean;
  copied: CopiedField;
  actionLoading: WebhookActionType;
  onGetSecret: () => void;
  onCopy: (text: string, type: "url" | "secret") => void;
  t: (key: string) => string;
}

export interface WebhookActionsProps {
  status: WebhookStatus | null;
  actionLoading: WebhookActionType;
  onRegister: () => void;
  onDelete: () => void;
  onMarkConfigured: () => void;
  t: (key: string) => string;
}
