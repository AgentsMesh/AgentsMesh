"use client";

import { useState, useCallback, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { useTranslations } from "next-intl";
import { getRepositoryService } from "@/lib/wasm-core";
import type { WebhookStatus, RepositoryData, WebhookSecretResponse } from "@/lib/api/repositoryTypes";
import { cn } from "@/lib/utils";
import { getLocalizedErrorMessage } from "@/lib/api/errors";
import { toast } from "sonner";
import { RefreshCw, Loader2 } from "lucide-react";
import {
  WebhookStatusDisplay,
  WebhookActiveInfo,
  WebhookManualSetup,
  WebhookActions,
  type WebhookSecretInfo,
  type WebhookActionType,
  type CopiedField,
} from "./webhook";

interface WebhookSettingsCardProps {
  repository: RepositoryData;
  onStatusChange?: () => void;
}

export function WebhookSettingsCard({
  repository,
  onStatusChange,
}: WebhookSettingsCardProps) {
  const t = useTranslations();
  const [status, setStatus] = useState<WebhookStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState<WebhookActionType>(null);
  const [error, setError] = useState<string | null>(null);
  const [secretInfo, setSecretInfo] = useState<WebhookSecretInfo | null>(null);
  const [showSecret, setShowSecret] = useState(false);
  const [copied, setCopied] = useState<CopiedField>(null);

  const loadStatus = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const res: { webhook_status: WebhookStatus } = JSON.parse(
        await getRepositoryService().get_webhook_status(BigInt(repository.id))
      );
      setStatus(res.webhook_status);
    } catch (err) {
      const msg = getLocalizedErrorMessage(err, t, t("repositories.webhook.retry"));
      setError(msg);
      toast.error(msg);
    } finally {
      setLoading(false);
    }
  }, [repository.id, t]);

  useEffect(() => {
    loadStatus();
  }, [loadStatus]);

  const handleRegister = useCallback(async () => {
    try {
      setActionLoading("register");
      setError(null);
      await getRepositoryService().register_webhook(BigInt(repository.id));
      await loadStatus();
      onStatusChange?.();
    } catch (err) {
      const msg = getLocalizedErrorMessage(err, t, t("repositories.detail.webhookFailed"));
      setError(msg);
      toast.error(msg);
    } finally {
      setActionLoading(null);
    }
  }, [repository.id, loadStatus, onStatusChange, t]);

  const handleDelete = useCallback(async () => {
    try {
      setActionLoading("delete");
      setError(null);
      await getRepositoryService().delete_webhook(BigInt(repository.id));
      setSecretInfo(null);
      setShowSecret(false);
      await loadStatus();
      onStatusChange?.();
    } catch (err) {
      const msg = getLocalizedErrorMessage(err, t, t("repositories.detail.webhookFailed"));
      setError(msg);
      toast.error(msg);
    } finally {
      setActionLoading(null);
    }
  }, [repository.id, loadStatus, onStatusChange, t]);

  const handleMarkConfigured = useCallback(async () => {
    try {
      setActionLoading("markConfigured");
      setError(null);
      await getRepositoryService().mark_webhook_configured(BigInt(repository.id));
      setShowSecret(false);
      await loadStatus();
      onStatusChange?.();
    } catch (err) {
      const msg = getLocalizedErrorMessage(err, t, t("repositories.detail.webhookFailed"));
      setError(msg);
      toast.error(msg);
    } finally {
      setActionLoading(null);
    }
  }, [repository.id, loadStatus, onStatusChange, t]);

  const handleGetSecret = useCallback(async () => {
    try {
      setActionLoading("getSecret");
      setError(null);
      const res: WebhookSecretResponse = JSON.parse(
        await getRepositoryService().get_webhook_secret(BigInt(repository.id))
      );
      // Transform API response to component format
      setSecretInfo({
        url: res.webhook_url,
        secret: res.webhook_secret,
        events: res.events,
      });
      setShowSecret(true);
    } catch (err) {
      const msg = getLocalizedErrorMessage(err, t, t("repositories.detail.webhookFailed"));
      setError(msg);
      toast.error(msg);
    } finally {
      setActionLoading(null);
    }
  }, [repository.id, t]);

  const handleCopy = useCallback(
    async (text: string, type: "url" | "secret") => {
      try {
        await navigator.clipboard.writeText(text);
        setCopied(type);
        setTimeout(() => setCopied(null), 2000);
      } catch {
      }
    },
    []
  );

  if (loading) {
    return (
      <div className="border border-border rounded-lg p-6 md:col-span-2">
        <h3 className="font-semibold mb-4">{t("repositories.webhook.title")}</h3>
        <div className="flex items-center justify-center py-8">
          <Loader2 className="w-5 h-5 animate-spin mr-2" />
          <span className="text-muted-foreground text-sm">
            {t("repositories.webhook.loading")}
          </span>
        </div>
      </div>
    );
  }

  return (
    <div className="border border-border rounded-lg p-6 md:col-span-2">
      <div className="flex items-center justify-between mb-4">
        <h3 className="font-semibold">{t("repositories.webhook.title")}</h3>
        <Button
          variant="ghost"
          size="sm"
          onClick={loadStatus}
          disabled={!!actionLoading}
        >
          <RefreshCw className={cn("w-4 h-4", loading && "animate-spin")} />
        </Button>
      </div>

      {error && (
        <div className="bg-destructive/10 text-destructive text-sm px-3 py-2 rounded mb-4">
          {error}
        </div>
      )}

      {/* Status display */}
      <WebhookStatusDisplay status={status} t={t} />

      {/* Active webhook info */}
      {status && <WebhookActiveInfo status={status} t={t} />}

      {/* Manual setup instructions */}
      {status?.needs_manual_setup && (
        <WebhookManualSetup
          repository={repository}
          secretInfo={secretInfo}
          showSecret={showSecret}
          copied={copied}
          actionLoading={actionLoading}
          onGetSecret={handleGetSecret}
          onCopy={handleCopy}
          t={t}
        />
      )}

      {/* Not registered state */}
      {!status?.registered && !status?.needs_manual_setup && (
        <div className="text-sm text-muted-foreground mb-4">
          {t("repositories.webhook.notRegisteredDescription")}
        </div>
      )}

      {/* Action buttons */}
      <WebhookActions
        status={status}
        actionLoading={actionLoading}
        onRegister={handleRegister}
        onDelete={handleDelete}
        onMarkConfigured={handleMarkConfigured}
        t={t}
      />
    </div>
  );
}

export default WebhookSettingsCard;
