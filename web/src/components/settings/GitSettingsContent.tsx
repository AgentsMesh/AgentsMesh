"use client";

import { useState, useEffect, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  userRepositoryProviderApi,
  RepositoryProviderData,
  userGitCredentialApi,
  GitCredentialData,
  RunnerLocalCredentialData,
  CredentialType,
  CredentialTypeValue,
  getCredentialTypeLabel,
} from "@/lib/api/client";
import { useTranslations } from "@/lib/i18n/client";
import { ChevronLeft, Plus, Settings, Key, GitBranch, Check, Trash2, TestTube } from "lucide-react";

// Provider icons component
const ProviderIcon = ({ provider }: { provider: string }) => {
  switch (provider) {
    case "github":
      return (
        <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
          <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
        </svg>
      );
    case "gitlab":
      return (
        <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
          <path d="m23.6 9.593-.033-.086L20.3.98a.851.851 0 0 0-.336-.405.875.875 0 0 0-1.004.054.868.868 0 0 0-.29.44l-2.208 6.763H7.538L5.33 1.07a.857.857 0 0 0-.29-.441.875.875 0 0 0-1.004-.053.851.851 0 0 0-.336.404L.433 9.502l-.032.086a6.066 6.066 0 0 0 2.012 7.01l.011.008.028.02 4.97 3.722 2.458 1.86 1.496 1.131a1.008 1.008 0 0 0 1.22 0l1.496-1.131 2.458-1.86 5-3.743.012-.01a6.068 6.068 0 0 0 2.009-7.002Z" />
        </svg>
      );
    case "gitee":
      return (
        <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
          <path d="M11.984 0A12 12 0 0 0 0 12a12 12 0 0 0 12 12 12 12 0 0 0 12-12A12 12 0 0 0 12 0a12 12 0 0 0-.016 0zm6.09 5.333c.328 0 .593.266.592.593v1.482a.594.594 0 0 1-.593.592H9.777c-.982 0-1.778.796-1.778 1.778v5.63c0 .327.266.592.593.592h5.63c.982 0 1.778-.796 1.778-1.778v-.296a.593.593 0 0 0-.592-.593h-4.15a.592.592 0 0 1-.592-.592v-1.482a.593.593 0 0 1 .593-.592h6.815c.327 0 .593.265.593.592v3.408a4 4 0 0 1-4 4H5.926a.593.593 0 0 1-.593-.593V9.778a4.444 4.444 0 0 1 4.445-4.444h8.296Z" />
        </svg>
      );
    default:
      return (
        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
        </svg>
      );
  }
};

// Credential type icon
const CredentialTypeIcon = ({ type }: { type: string }) => {
  switch (type) {
    case CredentialType.RUNNER_LOCAL:
      return <Settings className="w-4 h-4" />;
    case CredentialType.OAUTH:
      return <GitBranch className="w-4 h-4" />;
    case CredentialType.PAT:
      return <Key className="w-4 h-4" />;
    case CredentialType.SSH_KEY:
      return (
        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
        </svg>
      );
    default:
      return <Key className="w-4 h-4" />;
  }
};

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
        <div className="mb-4 p-4 bg-green-500/10 text-green-600 rounded-lg flex items-center justify-between">
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
                    <ProviderIcon provider={provider.provider_type} />
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
                        <span className="px-2 py-0.5 text-xs bg-yellow-500/10 text-yellow-600 rounded-full">
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

      {/* Add Provider Dialog */}
      {showAddProviderDialog && (
        <AddProviderDialog
          onClose={() => setShowAddProviderDialog(false)}
          onSuccess={() => {
            setShowAddProviderDialog(false);
            loadData();
          }}
        />
      )}

      {/* Edit Provider Dialog */}
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

      {/* Add Credential Dialog */}
      {showAddCredentialDialog && (
        <AddCredentialDialog
          providers={providers}
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

// Add Provider Dialog
function AddProviderDialog({
  onClose,
  onSuccess,
}: {
  onClose: () => void;
  onSuccess: () => void;
}) {
  const t = useTranslations();
  const [step, setStep] = useState<"type" | "details">("type");
  const [providerType, setProviderType] = useState("");
  const [name, setName] = useState("");
  const [baseUrl, setBaseUrl] = useState("");
  const [botToken, setBotToken] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const providers = [
    { type: "github", name: "GitHub", defaultUrl: "https://github.com" },
    { type: "gitlab", name: "GitLab", defaultUrl: "https://gitlab.com" },
    { type: "gitee", name: "Gitee", defaultUrl: "https://gitee.com" },
  ];

  const selectType = (type: string) => {
    const provider = providers.find(p => p.type === type);
    setProviderType(type);
    setName(provider?.name || "");
    setBaseUrl(provider?.defaultUrl || "");
    setStep("details");
  };

  const handleSubmit = async () => {
    if (!name || !baseUrl || !botToken) {
      setError(t("settings.gitSettings.providers.dialog.fillAll"));
      return;
    }

    setSaving(true);
    setError(null);

    try {
      await userRepositoryProviderApi.create({
        provider_type: providerType,
        name,
        base_url: baseUrl,
        bot_token: botToken,
      });
      onSuccess();
    } catch (err) {
      console.error("Failed to create provider:", err);
      setError(t("settings.gitSettings.providers.dialog.failed"));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-background rounded-lg shadow-lg w-full max-w-md mx-4">
        <div className="flex items-center justify-between p-4 border-b border-border">
          <h2 className="text-lg font-semibold">{t("settings.gitSettings.providers.dialog.title")}</h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground">
            ✕
          </button>
        </div>

        <div className="p-4">
          {error && (
            <div className="mb-4 p-3 bg-destructive/10 text-destructive text-sm rounded-lg">
              {error}
            </div>
          )}

          {step === "type" && (
            <div className="space-y-3">
              <p className="text-sm text-muted-foreground mb-4">
                {t("settings.gitSettings.providers.dialog.selectType")}
              </p>
              {providers.map((provider) => (
                <button
                  key={provider.type}
                  onClick={() => selectType(provider.type)}
                  className="w-full flex items-center gap-4 p-4 border border-border rounded-lg hover:bg-muted/50 transition-colors"
                >
                  <ProviderIcon provider={provider.type} />
                  <span className="font-medium">{provider.name}</span>
                </button>
              ))}
            </div>
          )}

          {step === "details" && (
            <div className="space-y-4">
              <button
                onClick={() => setStep("type")}
                className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
              >
                <ChevronLeft className="w-4 h-4" />
                {t("common.back")}
              </button>

              <div>
                <label className="block text-sm font-medium mb-2">
                  {t("settings.gitSettings.providers.dialog.name")}
                </label>
                <Input
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="My GitHub"
                />
              </div>

              <div>
                <label className="block text-sm font-medium mb-2">
                  {t("settings.gitSettings.providers.dialog.baseUrl")}
                </label>
                <Input
                  value={baseUrl}
                  onChange={(e) => setBaseUrl(e.target.value)}
                  placeholder="https://github.com"
                />
                <p className="text-xs text-muted-foreground mt-1">
                  {t("settings.gitSettings.providers.dialog.baseUrlHint")}
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium mb-2">
                  {t("settings.gitSettings.providers.dialog.token")}
                </label>
                <Input
                  type="password"
                  value={botToken}
                  onChange={(e) => setBotToken(e.target.value)}
                  placeholder="ghp_xxx or glpat-xxx"
                />
                <p className="text-xs text-muted-foreground mt-1">
                  {t("settings.gitSettings.providers.dialog.tokenHint")}
                </p>
              </div>
            </div>
          )}
        </div>

        {step === "details" && (
          <div className="flex justify-end gap-3 p-4 border-t border-border">
            <Button variant="outline" onClick={onClose}>
              {t("common.cancel")}
            </Button>
            <Button onClick={handleSubmit} disabled={saving}>
              {saving ? t("common.loading") : t("common.save")}
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}

// Edit Provider Dialog
function EditProviderDialog({
  provider,
  onClose,
  onSuccess,
}: {
  provider: RepositoryProviderData;
  onClose: () => void;
  onSuccess: () => void;
}) {
  const t = useTranslations();
  const [name, setName] = useState(provider.name);
  const [baseUrl, setBaseUrl] = useState(provider.base_url);
  const [botToken, setBotToken] = useState("");
  const [isActive, setIsActive] = useState(provider.is_active);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async () => {
    setSaving(true);
    setError(null);

    try {
      await userRepositoryProviderApi.update(provider.id, {
        name: name || undefined,
        base_url: baseUrl || undefined,
        bot_token: botToken || undefined,
        is_active: isActive,
      });
      onSuccess();
    } catch (err) {
      console.error("Failed to update provider:", err);
      setError(t("settings.gitSettings.providers.dialog.failed"));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-background rounded-lg shadow-lg w-full max-w-md mx-4">
        <div className="flex items-center justify-between p-4 border-b border-border">
          <h2 className="text-lg font-semibold">{t("settings.gitSettings.providers.dialog.editTitle")}</h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground">
            ✕
          </button>
        </div>

        <div className="p-4 space-y-4">
          {error && (
            <div className="p-3 bg-destructive/10 text-destructive text-sm rounded-lg">
              {error}
            </div>
          )}

          <div>
            <label className="block text-sm font-medium mb-2">
              {t("settings.gitSettings.providers.dialog.name")}
            </label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-2">
              {t("settings.gitSettings.providers.dialog.baseUrl")}
            </label>
            <Input
              value={baseUrl}
              onChange={(e) => setBaseUrl(e.target.value)}
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-2">
              {t("settings.gitSettings.providers.dialog.token")}
            </label>
            <Input
              type="password"
              value={botToken}
              onChange={(e) => setBotToken(e.target.value)}
              placeholder={t("settings.gitSettings.providers.dialog.tokenUpdateHint")}
            />
          </div>

          <div className="flex items-center justify-between">
            <label className="text-sm font-medium">
              {t("settings.gitSettings.providers.dialog.active")}
            </label>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                className="sr-only peer"
                checked={isActive}
                onChange={(e) => setIsActive(e.target.checked)}
              />
              <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"></div>
            </label>
          </div>
        </div>

        <div className="flex justify-end gap-3 p-4 border-t border-border">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button onClick={handleSubmit} disabled={saving}>
            {saving ? t("common.loading") : t("common.save")}
          </Button>
        </div>
      </div>
    </div>
  );
}

// Add Credential Dialog
function AddCredentialDialog({
  providers,
  onClose,
  onSuccess,
}: {
  providers: RepositoryProviderData[];
  onClose: () => void;
  onSuccess: () => void;
}) {
  const t = useTranslations();
  const [credentialType, setCredentialType] = useState<"pat" | "ssh_key">("pat");
  const [name, setName] = useState("");
  const [pat, setPat] = useState("");
  const [privateKey, setPrivateKey] = useState("");
  const [hostPattern, setHostPattern] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async () => {
    if (!name) {
      setError(t("settings.gitSettings.credentials.dialog.nameRequired"));
      return;
    }

    if (credentialType === "pat" && !pat) {
      setError(t("settings.gitSettings.credentials.dialog.patRequired"));
      return;
    }

    if (credentialType === "ssh_key" && !privateKey) {
      setError(t("settings.gitSettings.credentials.dialog.sshRequired"));
      return;
    }

    setSaving(true);
    setError(null);

    try {
      await userGitCredentialApi.create({
        name,
        credential_type: credentialType,
        pat: credentialType === "pat" ? pat : undefined,
        private_key: credentialType === "ssh_key" ? privateKey : undefined,
        host_pattern: hostPattern || undefined,
      });
      onSuccess();
    } catch (err) {
      console.error("Failed to create credential:", err);
      setError(t("settings.gitSettings.credentials.dialog.failed"));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-background rounded-lg shadow-lg w-full max-w-md mx-4 max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between p-4 border-b border-border">
          <h2 className="text-lg font-semibold">{t("settings.gitSettings.credentials.dialog.title")}</h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground">
            ✕
          </button>
        </div>

        <div className="p-4 space-y-4">
          {error && (
            <div className="p-3 bg-destructive/10 text-destructive text-sm rounded-lg">
              {error}
            </div>
          )}

          <div>
            <label className="block text-sm font-medium mb-2">
              {t("settings.gitSettings.credentials.dialog.type")}
            </label>
            <div className="flex gap-4">
              <label className="flex items-center gap-2">
                <input
                  type="radio"
                  checked={credentialType === "pat"}
                  onChange={() => setCredentialType("pat")}
                  className="w-4 h-4"
                />
                <span className="text-sm">Personal Access Token</span>
              </label>
              <label className="flex items-center gap-2">
                <input
                  type="radio"
                  checked={credentialType === "ssh_key"}
                  onChange={() => setCredentialType("ssh_key")}
                  className="w-4 h-4"
                />
                <span className="text-sm">SSH Key</span>
              </label>
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium mb-2">
              {t("settings.gitSettings.credentials.dialog.name")}
            </label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={t("settings.gitSettings.credentials.dialog.namePlaceholder")}
            />
          </div>

          {credentialType === "pat" && (
            <div>
              <label className="block text-sm font-medium mb-2">
                Personal Access Token
              </label>
              <Input
                type="password"
                value={pat}
                onChange={(e) => setPat(e.target.value)}
                placeholder="ghp_xxx or glpat-xxx"
              />
              <p className="text-xs text-muted-foreground mt-1">
                {t("settings.gitSettings.credentials.dialog.patHint")}
              </p>
            </div>
          )}

          {credentialType === "ssh_key" && (
            <div>
              <label className="block text-sm font-medium mb-2">
                {t("settings.gitSettings.credentials.dialog.privateKey")}
              </label>
              <textarea
                value={privateKey}
                onChange={(e) => setPrivateKey(e.target.value)}
                placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
                className="flex min-h-[120px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm font-mono placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              />
            </div>
          )}

          <div>
            <label className="block text-sm font-medium mb-2">
              {t("settings.gitSettings.credentials.dialog.hostPattern")}
            </label>
            <Input
              value={hostPattern}
              onChange={(e) => setHostPattern(e.target.value)}
              placeholder="github.com, gitlab.company.com"
            />
            <p className="text-xs text-muted-foreground mt-1">
              {t("settings.gitSettings.credentials.dialog.hostPatternHint")}
            </p>
          </div>
        </div>

        <div className="flex justify-end gap-3 p-4 border-t border-border">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button onClick={handleSubmit} disabled={saving}>
            {saving ? t("common.loading") : t("common.save")}
          </Button>
        </div>
      </div>
    </div>
  );
}
