"use client";

import { Button } from "@/components/ui/button";
import { Copy, CheckCircle, ExternalLink, Loader2 } from "lucide-react";
import type { WebhookManualSetupProps } from "./types";
import type { RepositoryData } from "@/lib/api";

/**
 * Helper function to generate provider webhook settings URL
 */
function getProviderWebhookSettingsUrl(repository: RepositoryData): string {
  const baseUrl = repository.provider_base_url || "https://github.com";
  const slug = repository.slug;

  switch (repository.provider_type) {
    case "github":
      return `${baseUrl}/${slug}/settings/hooks`;
    case "gitlab":
      return `${baseUrl}/${slug}/-/hooks`;
    case "gitee":
      return `${baseUrl}/${slug}/hooks`;
    default:
      return `${baseUrl}/${slug}`;
  }
}

/**
 * Displays manual setup instructions for webhook
 */
export function WebhookManualSetup({
  repository,
  secretInfo,
  showSecret,
  copied,
  actionLoading,
  onGetSecret,
  onCopy,
  t,
}: WebhookManualSetupProps) {
  return (
    <div className="bg-yellow-500/10 border border-yellow-500/30 rounded-lg p-4 mb-4">
      <p className="text-sm mb-3">
        {t("repositories.webhook.manualSetupInstructions")}
      </p>

      {showSecret && secretInfo ? (
        <div className="space-y-3">
          {/* Webhook URL */}
          <div>
            <label className="text-xs text-muted-foreground block mb-1">
              Webhook URL
            </label>
            <div className="flex items-center gap-2">
              <code className="bg-background px-2 py-1 rounded text-xs flex-1 overflow-auto">
                {secretInfo.url}
              </code>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onCopy(secretInfo.url, "url")}
              >
                {copied === "url" ? (
                  <CheckCircle className="w-4 h-4 text-green-500" />
                ) : (
                  <Copy className="w-4 h-4" />
                )}
              </Button>
            </div>
          </div>

          {/* Secret */}
          <div>
            <label className="text-xs text-muted-foreground block mb-1">
              Secret
            </label>
            <div className="flex items-center gap-2">
              <code className="bg-background px-2 py-1 rounded text-xs flex-1 font-mono">
                {secretInfo.secret}
              </code>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onCopy(secretInfo.secret, "secret")}
              >
                {copied === "secret" ? (
                  <CheckCircle className="w-4 h-4 text-green-500" />
                ) : (
                  <Copy className="w-4 h-4" />
                )}
              </Button>
            </div>
          </div>

          {/* Events */}
          <div>
            <label className="text-xs text-muted-foreground block mb-1">
              {t("repositories.webhook.events")}
            </label>
            <div className="text-sm">
              {secretInfo.events.map((event) => (
                <span
                  key={event}
                  className="bg-muted px-2 py-0.5 rounded text-xs mr-2"
                >
                  {event}
                </span>
              ))}
            </div>
          </div>

          {/* Provider settings link */}
          <div className="pt-2">
            <a
              href={getProviderWebhookSettingsUrl(repository)}
              target="_blank"
              rel="noopener noreferrer"
              className="text-sm text-primary hover:underline inline-flex items-center gap-1"
            >
              Open {repository.provider_type} webhook settings
              <ExternalLink className="w-3 h-3" />
            </a>
          </div>
        </div>
      ) : (
        <Button
          variant="outline"
          size="sm"
          onClick={onGetSecret}
          disabled={actionLoading === "getSecret"}
        >
          {actionLoading === "getSecret" ? (
            <Loader2 className="w-4 h-4 animate-spin mr-2" />
          ) : null}
          Show Webhook Configuration
        </Button>
      )}
    </div>
  );
}
