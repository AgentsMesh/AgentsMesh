"use client";

import { useState, useEffect, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { ConfirmDialog, useConfirmDialog } from "@/components/ui/confirm-dialog";
import type { APIKeyData, UpdateAPIKeyRequest } from "@/lib/api/apikeyTypes";
import { listApiKeys, createApiKey, updateApiKey, revokeApiKey } from "@/lib/api/apikey";
import { useCurrentOrg } from "@/stores/auth";
import { APIKeyCard, CreateAPIKeyDialog, APIKeySecretDialog, EditAPIKeyDialog } from "./apikeys";
import type { TranslationFn } from "./GeneralSettings";

interface APIKeysSettingsProps {
  t: TranslationFn;
}

export function APIKeysSettings({ t }: APIKeysSettingsProps) {
  const currentOrg = useCurrentOrg();
  const [apiKeys, setApiKeys] = useState<APIKeyData[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Dialog states
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [createdRawKey, setCreatedRawKey] = useState<string | null>(null);
  const [editingKey, setEditingKey] = useState<APIKeyData | null>(null);

  // Confirm dialog for revoke
  const { dialogProps, confirm } = useConfirmDialog();

  const fetchKeys = useCallback(async () => {
    if (!currentOrg) return;
    setLoading(true);
    setError(null);
    try {
      const response = await listApiKeys(currentOrg.slug);
      setApiKeys(response.items);
    } catch (err) {
      console.error("Failed to load API keys:", err);
      setError(t("settings.apiKeys.loadError"));
    } finally {
      setLoading(false);
    }
  }, [currentOrg, t]);

  useEffect(() => {
    fetchKeys();
  }, [fetchKeys]);

  const handleCreate = useCallback(async (data: {
    name: string;
    description?: string;
    scopes: string[];
    expires_in?: number;
  }) => {
    if (!currentOrg) return;
    try {
      const response = await createApiKey(currentOrg.slug, data);
      setCreatedRawKey(response.raw_key);
      setShowCreateDialog(false);
      fetchKeys();
    } catch (err) {
      console.error("Failed to create API key:", err);
      setError(t("settings.apiKeys.createFailed"));
      throw err;
    }
  }, [currentOrg, fetchKeys, t]);

  const handleUpdate = useCallback(async (id: number, data: UpdateAPIKeyRequest) => {
    if (!currentOrg) return;
    try {
      await updateApiKey(currentOrg.slug, id, data);
      fetchKeys();
    } catch (err) {
      console.error("Failed to update API key:", err);
      setError(t("settings.apiKeys.updateFailed"));
      throw err;
    }
  }, [currentOrg, fetchKeys, t]);

  const handleRevoke = useCallback(
    async (id: number) => {
      if (!currentOrg) return;
      const confirmed = await confirm({
        title: t("settings.apiKeys.revokeDialog.title"),
        description: t("settings.apiKeys.revokeDialog.description"),
        variant: "destructive",
        confirmText: t("settings.apiKeys.revokeDialog.revoke"),
        cancelText: t("settings.apiKeys.revokeDialog.cancel"),
      });
      if (confirmed) {
        try {
          await revokeApiKey(currentOrg.slug, id);
          fetchKeys();
        } catch (err) {
          console.error("Failed to revoke API key:", err);
          setError(t("settings.apiKeys.revokeFailed"));
        }
      }
    },
    [currentOrg, confirm, fetchKeys, t]
  );

  return (
    <div className="space-y-6">
      {error && (
        <div role="alert" className="bg-destructive/10 border border-destructive text-destructive px-4 py-3 rounded-lg flex items-center justify-between">
          <span>{error}</span>
          <button onClick={() => setError(null)} className="text-sm underline">
            {t("settings.apiKeys.dismiss")}
          </button>
        </div>
      )}

      <div className="border border-border rounded-lg p-6">
        <div className="mb-4 flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold">{t("settings.apiKeys.title")}</h2>
            <p className="text-sm text-muted-foreground">
              {t("settings.apiKeys.description")}
            </p>
          </div>
          <Button onClick={() => setShowCreateDialog(true)}>
            {t("settings.apiKeys.createKey")}
          </Button>
        </div>

        {loading ? (
          <div className="text-center py-4 text-muted-foreground">
            {t("settings.apiKeys.loading")}
          </div>
        ) : apiKeys.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">
            {t("settings.apiKeys.noKeys")}
          </div>
        ) : (
          <div className="space-y-3">
            {apiKeys.map((key) => (
              <APIKeyCard
                key={key.id}
                apiKey={key}
                onEdit={setEditingKey}
                onRevoke={handleRevoke}
                t={t}
              />
            ))}
          </div>
        )}

        <ConfirmDialog {...dialogProps} />
      </div>

      {/* Create Dialog */}
      <CreateAPIKeyDialog
        open={showCreateDialog}
        onOpenChange={setShowCreateDialog}
        onCreate={handleCreate}
        t={t}
      />

      {/* Secret Display Dialog */}
      <APIKeySecretDialog
        rawKey={createdRawKey || ""}
        open={createdRawKey !== null}
        onOpenChange={(open) => { if (!open) setCreatedRawKey(null); }}
        t={t}
      />

      {/* Edit Dialog */}
      {editingKey && (
        <EditAPIKeyDialog
          key={editingKey.id}
          apiKey={editingKey}
          open={true}
          onOpenChange={(open) => { if (!open) setEditingKey(null); }}
          onSave={handleUpdate}
          t={t}
        />
      )}
    </div>
  );
}
