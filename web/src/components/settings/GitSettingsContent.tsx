"use client";

import { useState, useEffect, useCallback } from "react";
import { Button } from "@/components/ui/button";
import {
  userRepositoryProviderApi,
  RepositoryProviderData,
  userGitCredentialApi,
  GitCredentialData,
  RunnerLocalCredentialData,
  CredentialType,
  CredentialTypeValue,
  getCredentialTypeLabel,
} from "@/lib/api";
import { useTranslations } from "@/lib/i18n/client";
import { Plus, Settings, Check, Trash2, TestTube } from "lucide-react";
import { GitProviderIcon, CredentialTypeIcon } from "@/components/icons/GitProviderIcon";
import { AddProviderDialog, EditProviderDialog, AddCredentialDialog } from "./git";

/**
 * GitSettingsContent - Shared Git settings component
 * Used by both user settings page and organization settings page.
 */
export function GitSettingsContent() {
  const t = useTranslations();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  // Data state
  const [providers, setProviders] = useState<RepositoryProviderData[]>([]);
  const [credentials, setCredentials] = useState<GitCredentialData[]>([]);
  const [runnerLocal, setRunnerLocal] = useState<RunnerLocalCredentialData | null>(null);
  const [defaultCredentialId, setDefaultCredentialId] = useState<number | null | "runner_local">(null);

  // Dialog states
  const [showAddProviderDialog, setShowAddProviderDialog] = useState(false);
  const [showAddCredentialDialog, setShowAddCredentialDialog] = useState(false);
  const [editingProvider, setEditingProvider] = useState<RepositoryProviderData | null>(null);

  // Load data
  const loadData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      const [providersRes, credentialsRes] = await Promise.all([
        userRepositoryProviderApi.list(),
        userGitCredentialApi.list(),
      ]);

      setProviders(providersRes.providers || []);
      setCredentials(credentialsRes.credentials || []);
      setRunnerLocal(credentialsRes.runner_local);

      // Determine default credential
      if (credentialsRes.runner_local.is_default) {
        setDefaultCredentialId("runner_local");
      } else {
        const defaultCred = credentialsRes.credentials.find(c => c.is_default);
        setDefaultCredentialId(defaultCred?.id || "runner_local");
      }
    } catch (err) {
      console.error("Failed to load data:", err);
      setError(t("settings.gitSettings.failedToLoad"));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  // Set default credential
  const handleSetDefault = async (credentialId: number | null) => {
    try {
      setError(null);
      await userGitCredentialApi.setDefault({ credential_id: credentialId });
      setDefaultCredentialId(credentialId || "runner_local");
      setSuccess(t("settings.gitSettings.defaultSet"));
      setTimeout(() => setSuccess(null), 3000);
    } catch (err) {
      console.error("Failed to set default:", err);
      setError(t("settings.gitSettings.failedToSetDefault"));
    }
  };

  // Delete provider
  const handleDeleteProvider = async (id: number) => {
    if (!confirm(t("settings.gitSettings.confirmDeleteProvider"))) return;
    try {
      await userRepositoryProviderApi.delete(id);
      await loadData();
    } catch (err) {
      console.error("Failed to delete provider:", err);
      setError(t("settings.gitSettings.failedToDeleteProvider"));
    }
  };

  // Delete credential
  const handleDeleteCredential = async (id: number) => {
    if (!confirm(t("settings.gitSettings.confirmDeleteCredential"))) return;
    try {
      await userGitCredentialApi.delete(id);
      await loadData();
    } catch (err) {
      console.error("Failed to delete credential:", err);
      setError(t("settings.gitSettings.failedToDeleteCredential"));
    }
  };

  // Test provider connection
  const handleTestConnection = async (id: number) => {
    try {
      setError(null);
      const result = await userRepositoryProviderApi.testConnection(id);
      if (result.success) {
        setSuccess(t("settings.gitSettings.connectionSuccess"));
      } else {
        setError(result.error || t("settings.gitSettings.connectionFailed"));
      }
      setTimeout(() => {
        setSuccess(null);
        setError(null);
      }, 3000);
    } catch (err) {
      console.error("Failed to test connection:", err);
      setError(t("settings.gitSettings.connectionFailed"));
    }
  };

  // Get all selectable credentials for default picker
  const getAllCredentials = () => {
    const items: Array<{
      id: number | "runner_local";
      name: string;
      type: string;
      isDefault: boolean;
    }> = [];

    // Add runner local first
    if (runnerLocal) {
      items.push({
        id: "runner_local",
        name: runnerLocal.name,
        type: CredentialType.RUNNER_LOCAL,
        isDefault: defaultCredentialId === "runner_local",
      });
    }

    // Add OAuth credentials from providers
    credentials
      .filter(c => c.credential_type === CredentialType.OAUTH)
      .forEach(c => {
        items.push({
          id: c.id,
          name: c.name,
          type: c.credential_type,
          isDefault: defaultCredentialId === c.id,
        });
      });

    // Add PAT and SSH credentials
    credentials
      .filter(c => c.credential_type === CredentialType.PAT || c.credential_type === CredentialType.SSH_KEY)
      .forEach(c => {
        items.push({
          id: c.id,
          name: c.name,
          type: c.credential_type,
          isDefault: defaultCredentialId === c.id,
        });
      });

    return items;
  };

  if (loading) {
    return (
      <div className="p-6 max-w-4xl mx-auto">
        <div className="flex items-center justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Error/Success messages */}
      {error && (
        <div className="mb-4 p-4 bg-destructive/10 text-destructive rounded-lg flex items-center justify-between">
          {error}
          <button onClick={() => setError(null)} className="text-sm underline">
            {t("common.close")}
          </button>
        </div>
      )}
      {success && (
        <div className="mb-4 p-4 bg-green-500/10 text-green-600 dark:text-green-400 rounded-lg flex items-center justify-between">
          {success}
          <button onClick={() => setSuccess(null)} className="text-sm underline">
            {t("common.close")}
          </button>
        </div>
      )}

      {/* Section 1: Default Git Credential */}
      <div className="border border-border rounded-lg p-6 mb-6">
        <h2 className="text-lg font-semibold mb-2">{t("settings.gitSettings.defaultCredential.title")}</h2>
        <p className="text-sm text-muted-foreground mb-4">
          {t("settings.gitSettings.defaultCredential.description")}
        </p>

        <div className="space-y-2">
          {getAllCredentials().map((cred) => (
            <button
              key={cred.id}
              onClick={() => handleSetDefault(cred.id === "runner_local" ? null : cred.id as number)}
              className={`w-full flex items-center gap-3 p-3 rounded-lg border transition-colors text-left ${
                cred.isDefault
                  ? "border-primary bg-primary/5"
                  : "border-border hover:bg-muted/50"
              }`}
            >
              <div className={`w-8 h-8 rounded-full flex items-center justify-center ${
                cred.isDefault ? "bg-primary text-primary-foreground" : "bg-muted"
              }`}>
                <CredentialTypeIcon type={cred.type} />
              </div>
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <span className="font-medium">{cred.name}</span>
                  <span className="text-xs px-2 py-0.5 rounded bg-muted text-muted-foreground">
                    {getCredentialTypeLabel(cred.type as CredentialTypeValue)}
                  </span>
                </div>
                {cred.type === CredentialType.RUNNER_LOCAL && (
                  <p className="text-xs text-muted-foreground">
                    {t("settings.gitSettings.defaultCredential.runnerLocalHint")}
                  </p>
                )}
              </div>
              {cred.isDefault && (
                <Check className="w-5 h-5 text-primary" />
              )}
            </button>
          ))}
        </div>
      </div>

      {/* Section 2: Repository Providers */}
      <div className="border border-border rounded-lg p-6 mb-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h2 className="text-lg font-semibold">{t("settings.gitSettings.providers.title")}</h2>
            <p className="text-sm text-muted-foreground">
              {t("settings.gitSettings.providers.description")}
            </p>
          </div>
          <Button onClick={() => setShowAddProviderDialog(true)}>
            <Plus className="w-4 h-4 mr-2" />
            {t("settings.gitSettings.providers.add")}
          </Button>
        </div>

        {providers.length === 0 ? (
          <p className="text-sm text-muted-foreground py-4 text-center">
            {t("settings.gitSettings.providers.empty")}
          </p>
        ) : (
          <div className="space-y-3">
            {providers.map((provider) => (
              <div
                key={provider.id}
                className={`flex items-center justify-between p-4 rounded-lg border ${
                  !provider.is_active ? "opacity-60 bg-muted/30" : "bg-muted/50"
                }`}
              >
                <div className="flex items-center gap-4">
                  <div className="w-10 h-10 rounded-full bg-background flex items-center justify-center">
                    <GitProviderIcon provider={provider.provider_type} />
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="font-medium">{provider.name}</span>
                      {provider.is_default && (
                        <span className="px-2 py-0.5 text-xs bg-primary/10 text-primary rounded-full">
                          {t("settings.gitSettings.providers.default")}
                        </span>
                      )}
                      {!provider.is_active && (
                        <span className="px-2 py-0.5 text-xs bg-yellow-500/10 text-yellow-600 dark:text-yellow-400 rounded-full">
                          {t("settings.gitSettings.providers.disabled")}
                        </span>
                      )}
                    </div>
                    <p className="text-sm text-muted-foreground">{provider.base_url}</p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleTestConnection(provider.id)}
                    title={t("settings.gitSettings.providers.test")}
                  >
                    <TestTube className="w-4 h-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setEditingProvider(provider)}
                  >
                    <Settings className="w-4 h-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleDeleteProvider(provider.id)}
                    className="text-destructive hover:text-destructive"
                  >
                    <Trash2 className="w-4 h-4" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Section 3: Git Credentials */}
      <div className="border border-border rounded-lg p-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h2 className="text-lg font-semibold">{t("settings.gitSettings.credentials.title")}</h2>
            <p className="text-sm text-muted-foreground">
              {t("settings.gitSettings.credentials.description")}
            </p>
          </div>
          <Button onClick={() => setShowAddCredentialDialog(true)}>
            <Plus className="w-4 h-4 mr-2" />
            {t("settings.gitSettings.credentials.add")}
          </Button>
        </div>

        {credentials.filter(c => c.credential_type !== CredentialType.OAUTH).length === 0 ? (
          <p className="text-sm text-muted-foreground py-4 text-center">
            {t("settings.gitSettings.credentials.empty")}
          </p>
        ) : (
          <div className="space-y-3">
            {credentials
              .filter(c => c.credential_type !== CredentialType.OAUTH)
              .map((cred) => (
                <div
                  key={cred.id}
                  className="flex items-center justify-between p-4 rounded-lg bg-muted/50"
                >
                  <div className="flex items-center gap-4">
                    <div className="w-10 h-10 rounded-full bg-background flex items-center justify-center">
                      <CredentialTypeIcon type={cred.credential_type} />
                    </div>
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="font-medium">{cred.name}</span>
                        <span className="px-2 py-0.5 text-xs bg-muted text-muted-foreground rounded">
                          {getCredentialTypeLabel(cred.credential_type as CredentialTypeValue)}
                        </span>
                      </div>
                      {cred.fingerprint && (
                        <p className="text-xs text-muted-foreground font-mono">
                          {cred.fingerprint}
                        </p>
                      )}
                      {cred.host_pattern && (
                        <p className="text-xs text-muted-foreground">
                          {t("settings.gitSettings.credentials.hostPattern")}: {cred.host_pattern}
                        </p>
                      )}
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleDeleteCredential(cred.id)}
                      className="text-destructive hover:text-destructive"
                    >
                      <Trash2 className="w-4 h-4" />
                    </Button>
                  </div>
                </div>
              ))}
          </div>
        )}
      </div>

      {/* Dialogs */}
      {showAddProviderDialog && (
        <AddProviderDialog
          onClose={() => setShowAddProviderDialog(false)}
          onSuccess={() => {
            setShowAddProviderDialog(false);
            loadData();
          }}
        />
      )}

      {editingProvider && (
        <EditProviderDialog
          provider={editingProvider}
          onClose={() => setEditingProvider(null)}
          onSuccess={() => {
            setEditingProvider(null);
            loadData();
          }}
        />
      )}

      {showAddCredentialDialog && (
        <AddCredentialDialog
          onClose={() => setShowAddCredentialDialog(false)}
          onSuccess={() => {
            setShowAddCredentialDialog(false);
            loadData();
          }}
        />
      )}
    </div>
  );
}
