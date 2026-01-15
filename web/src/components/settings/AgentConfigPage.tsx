"use client";

import { useState, useEffect, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  agentApi,
  userAgentConfigApi,
  userAgentCredentialApi,
  ConfigField,
  AgentTypeData,
  UserAgentConfigData,
  CredentialProfileData,
} from "@/lib/api";
import { ConfigForm } from "@/components/ide/ConfigForm";
import { useTranslations } from "@/lib/i18n/client";
import {
  Bot,
  Plus,
  Check,
  Trash2,
  Edit2,
  Server,
  Key,
  Star,
  Settings2,
  AlertCircle,
} from "lucide-react";

interface AgentConfigPageProps {
  agentSlug: string;
}

// Special constant for RunnerHost virtual profile
const RUNNER_HOST_ID = -1;

/**
 * AgentConfigPage - Unified configuration page for a single agent type
 * Combines credentials management and runtime configuration in one place
 */
export function AgentConfigPage({ agentSlug }: AgentConfigPageProps) {
  const t = useTranslations();

  // Loading states
  const [loading, setLoading] = useState(true);
  const [savingConfig, setSavingConfig] = useState(false);

  // Data states
  const [agentType, setAgentType] = useState<AgentTypeData | null>(null);
  const [configFields, setConfigFields] = useState<ConfigField[]>([]);
  const [configValues, setConfigValues] = useState<Record<string, unknown>>({});
  const [credentialProfiles, setCredentialProfiles] = useState<CredentialProfileData[]>([]);
  const [isRunnerHostDefault, setIsRunnerHostDefault] = useState(true);

  // UI states
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  // Credential dialog state
  const [showCredentialDialog, setShowCredentialDialog] = useState(false);
  const [editingProfile, setEditingProfile] = useState<CredentialProfileData | null>(null);
  const [formName, setFormName] = useState("");
  const [formDescription, setFormDescription] = useState("");
  const [formBaseUrl, setFormBaseUrl] = useState("");
  const [formApiKey, setFormApiKey] = useState("");
  const [formSubmitting, setFormSubmitting] = useState(false);

  // Load all data
  const loadData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      // Load agent types to find the one matching the slug
      const typesRes = await agentApi.listTypes();
      const foundAgentType = typesRes.agent_types?.find(
        (t: AgentTypeData) => t.slug === agentSlug
      );

      if (!foundAgentType) {
        setError(t("settings.agentConfig.agentNotFound"));
        setLoading(false);
        return;
      }

      setAgentType(foundAgentType);

      // Load data in parallel
      const [schemaRes, credentialsRes] = await Promise.all([
        agentApi.getConfigSchema(foundAgentType.id).catch(() => ({ schema: { fields: [] } })),
        userAgentCredentialApi.list().catch(() => ({ items: [] })),
      ]);

      // Set config schema fields
      const fields = schemaRes.schema?.fields || [];
      setConfigFields(fields);

      // Initialize config values with defaults from schema
      const defaultValues: Record<string, unknown> = {};
      for (const field of fields) {
        if (field.default !== undefined) {
          defaultValues[field.name] = field.default;
        }
      }

      // Try to load user's saved config
      try {
        const userConfigRes = await userAgentConfigApi.get(foundAgentType.id);
        if (userConfigRes.config?.config_values) {
          // Merge user config over defaults
          setConfigValues({ ...defaultValues, ...userConfigRes.config.config_values });
        } else {
          setConfigValues(defaultValues);
        }
      } catch {
        // No saved config, use defaults
        setConfigValues(defaultValues);
      }

      // Extract credential profiles for this agent type
      const agentCredentials = credentialsRes.items?.find(
        (item: { agent_type_id: number }) => item.agent_type_id === foundAgentType.id
      );
      const profiles = agentCredentials?.profiles || [];
      setCredentialProfiles(profiles);

      // Check if RunnerHost is default (no custom profile is default)
      const hasCustomDefault = profiles.some((p: CredentialProfileData) => p.is_default);
      setIsRunnerHostDefault(!hasCustomDefault);
    } catch (err) {
      console.error("Failed to load agent config:", err);
      setError(t("settings.agentConfig.loadFailed"));
    } finally {
      setLoading(false);
    }
  }, [agentSlug, t]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  // Handle config field change
  const handleConfigChange = useCallback((fieldName: string, value: unknown) => {
    setConfigValues((prev) => ({
      ...prev,
      [fieldName]: value,
    }));
  }, []);

  // Save runtime config
  const handleSaveConfig = async () => {
    if (!agentType) return;

    try {
      setSavingConfig(true);
      setError(null);

      // Filter out undefined/empty values, but keep false for booleans
      const cleanedConfig: Record<string, unknown> = {};
      for (const [key, value] of Object.entries(configValues)) {
        if (value !== undefined && value !== "") {
          cleanedConfig[key] = value;
        }
      }

      await userAgentConfigApi.set(agentType.id, cleanedConfig);
      setSuccess(t("settings.agentConfig.configSaved"));
      setTimeout(() => setSuccess(null), 3000);
    } catch (err) {
      console.error("Failed to save config:", err);
      setError(t("settings.agentConfig.configSaveFailed"));
    } finally {
      setSavingConfig(false);
    }
  };

  // Open credential add dialog
  const handleOpenAddDialog = () => {
    setFormName("");
    setFormDescription("");
    setFormBaseUrl("");
    setFormApiKey("");
    setEditingProfile(null);
    setShowCredentialDialog(true);
  };

  // Open credential edit dialog
  const handleOpenEditDialog = (profile: CredentialProfileData) => {
    setFormName(profile.name);
    setFormDescription(profile.description || "");
    setFormBaseUrl("");
    setFormApiKey("");
    setEditingProfile(profile);
    setShowCredentialDialog(true);
  };

  // Submit credential form
  const handleCredentialSubmit = async () => {
    if (!agentType) return;

    try {
      setFormSubmitting(true);
      setError(null);

      const credentials: Record<string, string> = {};
      if (formBaseUrl) credentials.base_url = formBaseUrl;
      if (formApiKey) credentials.api_key = formApiKey;

      if (editingProfile) {
        await userAgentCredentialApi.update(editingProfile.id, {
          name: formName,
          description: formDescription || undefined,
          is_runner_host: false,
          credentials: Object.keys(credentials).length > 0 ? credentials : undefined,
        });
        setSuccess(t("settings.agentCredentials.profileUpdated"));
      } else {
        await userAgentCredentialApi.create(agentType.id, {
          name: formName,
          description: formDescription || undefined,
          is_runner_host: false,
          credentials: credentials,
        });
        setSuccess(t("settings.agentCredentials.profileCreated"));
      }

      setShowCredentialDialog(false);
      await loadData();
      setTimeout(() => setSuccess(null), 3000);
    } catch (err) {
      console.error("Failed to save profile:", err);
      setError(t("settings.agentCredentials.failedToSave"));
    } finally {
      setFormSubmitting(false);
    }
  };

  // Set RunnerHost as default
  const handleSetRunnerHostDefault = async () => {
    try {
      setError(null);
      const currentDefault = credentialProfiles.find((p) => p.is_default);
      if (currentDefault) {
        await userAgentCredentialApi.update(currentDefault.id, { is_default: false });
      }
      setSuccess(t("settings.agentCredentials.defaultSet"));
      await loadData();
      setTimeout(() => setSuccess(null), 3000);
    } catch (err) {
      console.error("Failed to set RunnerHost as default:", err);
      setError(t("settings.agentCredentials.failedToSetDefault"));
    }
  };

  // Set custom profile as default
  const handleSetDefault = async (profileId: number) => {
    try {
      setError(null);
      await userAgentCredentialApi.setDefault(profileId);
      setSuccess(t("settings.agentCredentials.defaultSet"));
      await loadData();
      setTimeout(() => setSuccess(null), 3000);
    } catch (err) {
      console.error("Failed to set default:", err);
      setError(t("settings.agentCredentials.failedToSetDefault"));
    }
  };

  // Delete credential profile
  const handleDeleteProfile = async (profileId: number) => {
    if (!confirm(t("settings.agentCredentials.confirmDelete"))) return;
    try {
      setError(null);
      await userAgentCredentialApi.delete(profileId);
      setSuccess(t("settings.agentCredentials.profileDeleted"));
      await loadData();
      setTimeout(() => setSuccess(null), 3000);
    } catch (err) {
      console.error("Failed to delete profile:", err);
      setError(t("settings.agentCredentials.failedToDelete"));
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>
    );
  }

  if (!agentType) {
    return (
      <div className="flex flex-col items-center justify-center py-12">
        <AlertCircle className="w-12 h-12 text-muted-foreground mb-4" />
        <p className="text-muted-foreground">{error || t("settings.agentConfig.agentNotFound")}</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Bot className="w-8 h-8 text-primary" />
        <div>
          <h2 className="text-xl font-semibold">{agentType.name}</h2>
          {agentType.description && (
            <p className="text-sm text-muted-foreground">{agentType.description}</p>
          )}
        </div>
      </div>

      {/* Error/Success messages */}
      {error && (
        <div className="bg-destructive/15 text-destructive text-sm p-3 rounded-md flex items-center justify-between">
          <span>{error}</span>
          <button onClick={() => setError(null)} className="text-xs underline">
            {t("common.dismiss")}
          </button>
        </div>
      )}
      {success && (
        <div className="bg-green-500/15 text-green-600 text-sm p-3 rounded-md flex items-center justify-between">
          <span>{success}</span>
          <button onClick={() => setSuccess(null)} className="text-xs underline">
            {t("common.dismiss")}
          </button>
        </div>
      )}

      {/* Credentials Section */}
      <div className="border border-border rounded-lg p-6">
        <div className="flex items-center gap-2 mb-4">
          <Key className="w-5 h-5 text-muted-foreground" />
          <h3 className="text-lg font-semibold">{t("settings.agentConfig.credentials.title")}</h3>
        </div>
        <p className="text-sm text-muted-foreground mb-4">
          {t("settings.agentConfig.credentials.description")}
        </p>

        {/* RunnerHost - always shown as first option */}
        <div className="space-y-2">
          <div className="flex items-center justify-between p-3 border border-border rounded-lg hover:bg-muted/50">
            <div className="flex items-center gap-3">
              <Server className="w-4 h-4 text-muted-foreground" />
              <div>
                <div className="flex items-center gap-2">
                  <span className="font-medium">RunnerHost</span>
                  {isRunnerHostDefault && (
                    <span className="inline-flex items-center px-1.5 py-0.5 rounded text-xs bg-primary/10 text-primary">
                      <Star className="w-3 h-3 mr-0.5" />
                      {t("settings.agentCredentials.default")}
                    </span>
                  )}
                </div>
                <div className="text-xs text-muted-foreground">
                  {t("settings.agentCredentials.runnerHostHint")}
                </div>
              </div>
            </div>
            {!isRunnerHostDefault && (
              <Button
                variant="ghost"
                size="sm"
                onClick={handleSetRunnerHostDefault}
                title={t("settings.agentCredentials.setAsDefault")}
              >
                <Check className="w-4 h-4" />
              </Button>
            )}
          </div>

          {/* Custom credential profiles */}
          {credentialProfiles.map((profile) => (
            <div
              key={profile.id}
              className="flex items-center justify-between p-3 border border-border rounded-lg hover:bg-muted/50"
            >
              <div className="flex items-center gap-3">
                <Key className="w-4 h-4 text-muted-foreground" />
                <div>
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{profile.name}</span>
                    {profile.is_default && (
                      <span className="inline-flex items-center px-1.5 py-0.5 rounded text-xs bg-primary/10 text-primary">
                        <Star className="w-3 h-3 mr-0.5" />
                        {t("settings.agentCredentials.default")}
                      </span>
                    )}
                  </div>
                  <div className="text-xs text-muted-foreground">
                    {profile.configured_fields?.length
                      ? `${t("settings.agentCredentials.configured")}: ${profile.configured_fields.join(", ")}`
                      : t("settings.agentCredentials.notConfigured")}
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-1">
                {!profile.is_default && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleSetDefault(profile.id)}
                    title={t("settings.agentCredentials.setAsDefault")}
                  >
                    <Check className="w-4 h-4" />
                  </Button>
                )}
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleOpenEditDialog(profile)}
                >
                  <Edit2 className="w-4 h-4" />
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleDeleteProfile(profile.id)}
                  className="text-destructive hover:text-destructive"
                >
                  <Trash2 className="w-4 h-4" />
                </Button>
              </div>
            </div>
          ))}

          {/* Add credential button */}
          <Button
            variant="outline"
            size="sm"
            onClick={handleOpenAddDialog}
            className="mt-2"
          >
            <Plus className="w-4 h-4 mr-1" />
            {t("settings.agentCredentials.addProfile")}
          </Button>
        </div>
      </div>

      {/* Runtime Config Section */}
      <div className="border border-border rounded-lg p-6">
        <div className="flex items-center gap-2 mb-4">
          <Settings2 className="w-5 h-5 text-muted-foreground" />
          <h3 className="text-lg font-semibold">{t("settings.agentConfig.runtime.title")}</h3>
        </div>
        <p className="text-sm text-muted-foreground mb-4">
          {t("settings.agentConfig.runtime.description")}
        </p>

        {configFields.length > 0 ? (
          <>
            <ConfigForm
              fields={configFields}
              values={configValues}
              onChange={handleConfigChange}
              agentSlug={agentSlug}
            />
            <div className="mt-4">
              <Button onClick={handleSaveConfig} disabled={savingConfig}>
                {savingConfig ? t("common.saving") : t("common.saveChanges")}
              </Button>
            </div>
          </>
        ) : (
          <div className="text-center py-8 text-muted-foreground">
            {t("settings.agentConfig.noConfigFields")}
          </div>
        )}
      </div>

      {/* Add/Edit Credential Dialog */}
      <Dialog open={showCredentialDialog} onOpenChange={setShowCredentialDialog}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>
              {editingProfile
                ? t("settings.agentCredentials.editProfile")
                : t("settings.agentCredentials.addProfile")}
            </DialogTitle>
            <DialogDescription>
              {t("settings.agentCredentials.customProfileDescription")}
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="name">{t("settings.agentCredentials.name")}</Label>
              <Input
                id="name"
                value={formName}
                onChange={(e) => setFormName(e.target.value)}
                placeholder={t("settings.agentCredentials.namePlaceholder")}
              />
            </div>

            <div className="grid gap-2">
              <Label htmlFor="description">{t("settings.agentCredentials.descriptionLabel")}</Label>
              <Textarea
                id="description"
                value={formDescription}
                onChange={(e) => setFormDescription(e.target.value)}
                placeholder={t("settings.agentCredentials.descriptionPlaceholder")}
                rows={2}
              />
            </div>

            <div className="grid gap-2">
              <Label htmlFor="base_url">
                {t("settings.agentCredentials.baseUrl")}
                <span className="text-xs text-muted-foreground ml-1">
                  ({t("common.optional")})
                </span>
              </Label>
              <Input
                id="base_url"
                value={formBaseUrl}
                onChange={(e) => setFormBaseUrl(e.target.value)}
                placeholder="https://api.anthropic.com"
              />
            </div>

            <div className="grid gap-2">
              <Label htmlFor="api_key">{t("settings.agentCredentials.apiKey")}</Label>
              <Input
                id="api_key"
                type="password"
                value={formApiKey}
                onChange={(e) => setFormApiKey(e.target.value)}
                placeholder={editingProfile ? t("settings.agentCredentials.apiKeyPlaceholder") : "sk-..."}
              />
              {editingProfile && (
                <p className="text-xs text-muted-foreground">
                  {t("settings.agentCredentials.apiKeyEditHint")}
                </p>
              )}
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setShowCredentialDialog(false)}>
              {t("common.cancel")}
            </Button>
            <Button onClick={handleCredentialSubmit} disabled={formSubmitting || !formName}>
              {formSubmitting
                ? t("common.saving")
                : editingProfile
                ? t("common.save")
                : t("common.create")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

export default AgentConfigPage;
