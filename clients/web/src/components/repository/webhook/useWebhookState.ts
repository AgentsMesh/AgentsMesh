"use client";

import { useState, useCallback } from "react";
import { WebhookStatus, WebhookSecretResponse } from "@/lib/api";
import { getRepositoryService } from "@/lib/wasm-core";
import { WebhookState, WebhookSettingsState, WebhookSettingsActions } from "./types";

export interface UseWebhookStateResult extends WebhookSettingsState, WebhookSettingsActions {}

export function useWebhookState(repositoryId: number, onUpdate?: () => void): UseWebhookStateResult {
  const [state, setState] = useState<WebhookState>("loading");
  const [status, setStatus] = useState<WebhookStatus | null>(null);
  const [secretData, setSecretData] = useState<WebhookSecretResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const loadStatus = useCallback(async () => {
    setState("loading");
    setError(null);
    try {
      const res = JSON.parse(await getRepositoryService().get_webhook_status(BigInt(repositoryId)));
      setStatus(res.webhook_status);

      if (res.webhook_status.registered && res.webhook_status.is_active) {
        setState("registered");
      } else if (res.webhook_status.needs_manual_setup) {
        setState("needs_manual_setup");
        try {
          const secretRes = JSON.parse(await getRepositoryService().get_webhook_secret(BigInt(repositoryId)));
          setSecretData(secretRes);
        } catch {
        }
      } else {
        setState("not_registered");
      }
    } catch (err) {
      console.error("Failed to load webhook status:", err);
      setError("Failed to load webhook status");
      setState("error");
    }
  }, [repositoryId]);

  const handleRegister = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await getRepositoryService().register_webhook(BigInt(repositoryId));
      onUpdate?.();
      await loadStatus();
    } catch (err) {
      console.error("Failed to register webhook:", err);
      setError("Failed to register webhook");
    } finally {
      setLoading(false);
    }
  }, [repositoryId, onUpdate, loadStatus]);

  const handleDelete = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await getRepositoryService().delete_webhook(BigInt(repositoryId));
      setState("not_registered");
      setStatus(null);
      setSecretData(null);
      onUpdate?.();
    } catch (err) {
      console.error("Failed to delete webhook:", err);
      setError("Failed to delete webhook");
    } finally {
      setLoading(false);
    }
  }, [repositoryId, onUpdate]);

  const handleMarkConfigured = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await getRepositoryService().mark_webhook_configured(BigInt(repositoryId));
      setState("registered");
      onUpdate?.();
      await loadStatus();
    } catch (err) {
      console.error("Failed to mark webhook as configured:", err);
      setError("Failed to mark webhook as configured");
    } finally {
      setLoading(false);
    }
  }, [repositoryId, onUpdate, loadStatus]);

  return {
    state,
    status,
    secretData,
    error,
    loading,
    handleRegister,
    handleDelete,
    handleMarkConfigured,
    loadStatus,
  };
}
