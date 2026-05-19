import type { WebhookActiveInfoProps } from "./types";

export function WebhookActiveInfo({ status, t }: WebhookActiveInfoProps) {
  if (!status.is_active || !status.registered || status.needs_manual_setup) {
    return null;
  }

  return (
    <div className="space-y-3 mb-4">
      {status.webhook_url && (
        <div className="text-sm">
          <span className="text-muted-foreground">URL: </span>
          <code className="bg-muted px-1 py-0.5 rounded text-xs">
            {status.webhook_url}
          </code>
        </div>
      )}
      {status.events && status.events.length > 0 && (
        <div className="text-sm">
          <span className="text-muted-foreground">
            {t("repositories.webhook.events")}:{" "}
          </span>
          <span className="text-foreground">{status.events.join(", ")}</span>
        </div>
      )}
      {status.registered_at && (
        <div className="text-sm text-muted-foreground">
          Registered: {new Date(status.registered_at).toLocaleString()}
        </div>
      )}
    </div>
  );
}
