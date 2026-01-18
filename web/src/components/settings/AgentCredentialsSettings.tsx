"use client";

import { useState, useEffect, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  userAgentCredentialApi,
  CredentialProfileData,
  CredentialProfilesByAgentType,
  agentApi,
  AgentTypeData,
} from "@/lib/api";
import { useTranslations } from "@/lib/i18n/client";
import {
  Bot,
  Plus,
  Check,
  Trash2,
  Edit2,
  Server,
  Key,
  ChevronDown,
  ChevronRight,
  Star,
} from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";

// Agent type icon based on slug
function AgentIcon({ slug }: { slug: string }) {
  // Return different icons based on agent type
  return <Bot className="w-5 h-5" />;
}

// Special constant for RunnerHost virtual profile
const RUNNER_HOST_ID = -1;

export function AgentCredentialsSettings() {
  const t = useTranslations();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  // Data state
  const [profilesByAgentType, setProfilesByAgentType] = useState<CredentialProfilesByAgentType[]>([]);
  const [agentTypes, setAgentTypes] = useState<AgentTypeData[]>([]);
  const [expandedAgentTypes, setExpandedAgentTypes] = useState<Set<number>>(new Set());
  // Track which agent types have RunnerHost as default (no custom default profile)
  const [runnerHostDefaults, setRunnerHostDefaults] = useState<Set<number>>(new Set());

  // Dialog state
  const [showAddDialog, setShowAddDialog] = useState(false);
  const [editingProfile, setEditingProfile] = useState<CredentialProfileData | null>(null);
  const [selectedAgentTypeId, setSelectedAgentTypeId] = useState<number | null>(null);

  // Form state
  const [formName, setFormName] = useState("");
  const [formDescription, setFormDescription] = useState("");
  const [formBaseUrl, setFormBaseUrl] = useState("");
  const [formApiKey, setFormApiKey] = useState("");
  const [formSubmitting, setFormSubmitting] = useState(false);

  // Load data
  const loadData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      const [profilesRes, typesRes] = await Promise.all([
        userAgentCredentialApi.list(),
        agentApi.listTypes(),
      ]);

      setProfilesByAgentType(profilesRes.items || []);
      setAgentTypes(typesRes.agent_types || []);

      // Determine which agent types have RunnerHost as default
      // (i.e., no custom profile is set as default)
      const runnerHostDefaultSet = new Set<number>();
      const agentTypeIds = new Set(typesRes.agent_types?.map((t: AgentTypeData) => t.id) || []);

      // Start by assuming all agent types default to RunnerHost
      agentTypeIds.forEach((id: number) => runnerHostDefaultSet.add(id));

      // Remove from set if there's a custom default profile
      profilesRes.items?.forEach((item) => {
        const hasCustomDefault = item.profiles.some((p) => p.is_default);
        if (hasCustomDefault) {
          runnerHostDefaultSet.delete(item.agent_type_id);
        }
      });

      setRunnerHostDefaults(runnerHostDefaultSet);

      // Auto-expand first agent type or those with profiles
      const expandedIds = new Set<number>();
      if (typesRes.agent_types?.length > 0) {
        expandedIds.add(typesRes.agent_types[0].id);
      }
      profilesRes.items?.forEach((item) => {
        if (item.profiles.length > 0) {
          expandedIds.add(item.agent_type_id);
        }
      });
      setExpandedAgentTypes(expandedIds);
    } catch (err) {
      console.error("Failed to load agent credentials:", err);
      setError(t("settings.agentCredentials.failedToLoad"));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  // Toggle agent type expansion
  const toggleAgentType = (agentTypeId: number) => {
    setExpandedAgentTypes((prev) => {
      const next = new Set(prev);
      if (next.has(agentTypeId)) {
        next.delete(agentTypeId);
      } else {
        next.add(agentTypeId);
      }
      return next;
    });
  };

  // Open add dialog - for adding custom credential profiles (not RunnerHost)
  const handleOpenAddDialog = (agentTypeId: number) => {
    setSelectedAgentTypeId(agentTypeId);
    setFormName("");
    setFormDescription("");
    setFormBaseUrl("");
    setFormApiKey("");
    setEditingProfile(null);
    setShowAddDialog(true);
  };

  // Open edit dialog
  const handleOpenEditDialog = (profile: CredentialProfileData) => {
    setSelectedAgentTypeId(profile.agent_type_id);
    setFormName(profile.name);
    setFormDescription(profile.description || "");
    setFormBaseUrl("");
    setFormApiKey("");
    setEditingProfile(profile);
    setShowAddDialog(true);
  };

  // Submit form - always creates custom credential profile (not RunnerHost)
  const handleSubmit = async () => {
    if (!selectedAgentTypeId) return;

    try {
      setFormSubmitting(true);
      setError(null);

      const credentials: Record<string, string> = {};
      if (formBaseUrl) credentials.base_url = formBaseUrl;
      if (formApiKey) credentials.api_key = formApiKey;

      if (editingProfile) {
        // Update existing profile
        await userAgentCredentialApi.update(editingProfile.id, {
          name: formName,
          description: formDescription || undefined,
          is_runner_host: false,
          credentials: Object.keys(credentials).length > 0 ? credentials : undefined,
        });
        setSuccess(t("settings.agentCredentials.profileUpdated"));
      } else {
        // Create new profile - always custom credentials (not RunnerHost)
        await userAgentCredentialApi.create(selectedAgentTypeId, {
          name: formName,
          description: formDescription || undefined,
          is_runner_host: false,
          credentials: credentials,
        });
        setSuccess(t("settings.agentCredentials.profileCreated"));
      }

      setShowAddDialog(false);
      await loadData();
      setTimeout(() => setSuccess(null), 3000);
    } catch (err) {
      console.error("Failed to save profile:", err);
      setError(t("settings.agentCredentials.failedToSave"));
    } finally {
      setFormSubmitting(false);
    }
  };

  // Set RunnerHost as default for an agent type (clear any custom default)
  const handleSetRunnerHostDefault = async (agentTypeId: number) => {
    try {
      setError(null);
      // Find current default profile and unset it
      const group = profilesByAgentType.find((g) => g.agent_type_id === agentTypeId);
      const currentDefault = group?.profiles.find((p) => p.is_default);
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

  // Set default profile
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

  // Delete profile
  const handleDelete = async (profileId: number) => {
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

  // Get profiles for a specific agent type
  const getProfilesForAgentType = (agentTypeId: number): CredentialProfileData[] => {
    const group = profilesByAgentType.find((g) => g.agent_type_id === agentTypeId);
    return group?.profiles || [];
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h2 className="text-lg font-semibold">{t("settings.agentCredentials.title")}</h2>
        <p className="text-sm text-muted-foreground mt-1">
          {t("settings.agentCredentials.description")}
        </p>
      </div>

      {/* Error/Success messages */}
      {error && (
        <div className="bg-destructive/15 text-destructive text-sm p-3 rounded-md">
          {error}
        </div>
      )}
      {success && (
        <div className="bg-green-500/15 text-green-600 dark:text-green-400 text-sm p-3 rounded-md">
          {success}
        </div>
      )}

      {/* Agent Types List */}
      <div className="space-y-2">
        {agentTypes.map((agentType) => {
          const profiles = getProfilesForAgentType(agentType.id);
          const isExpanded = expandedAgentTypes.has(agentType.id);
          const isRunnerHostDefault = runnerHostDefaults.has(agentType.id);

          return (
            <div
              key={agentType.id}
              className="border border-border rounded-lg overflow-hidden"
            >
              {/* Agent Type Header */}
              <button
                className="w-full flex items-center justify-between p-4 hover:bg-muted/50 transition-colors"
                onClick={() => toggleAgentType(agentType.id)}
              >
                <div className="flex items-center gap-3">
                  {isExpanded ? (
                    <ChevronDown className="w-4 h-4 text-muted-foreground" />
                  ) : (
                    <ChevronRight className="w-4 h-4 text-muted-foreground" />
                  )}
                  <AgentIcon slug={agentType.slug} />
                  <div className="text-left">
                    <div className="font-medium">{agentType.name}</div>
                    {agentType.description && (
                      <div className="text-xs text-muted-foreground">
                        {agentType.description}
                      </div>
                    )}
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-sm text-muted-foreground">
                    {profiles.length} {t("settings.agentCredentials.profiles")}
                  </span>
                </div>
              </button>

              {/* Profiles List */}
              {isExpanded && (
                <div className="border-t border-border bg-muted/20">
                  {/* RunnerHost - always shown as first option, not deletable */}
                  <div className="px-4 py-3 flex items-center justify-between hover:bg-muted/50 border-b border-border">
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
                    <div className="flex items-center gap-1">
                      {!isRunnerHostDefault && (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleSetRunnerHostDefault(agentType.id)}
                          title={t("settings.agentCredentials.setAsDefault")}
                        >
                          <Check className="w-4 h-4" />
                        </Button>
                      )}
                      {/* RunnerHost cannot be edited or deleted */}
                    </div>
                  </div>

                  {/* Custom credential profiles */}
                  {profiles.length > 0 && (
                    <div className="divide-y divide-border">
                      {profiles.map((profile) => (
                        <div
                          key={profile.id}
                          className="px-4 py-3 flex items-center justify-between hover:bg-muted/50"
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
                              onClick={() => handleDelete(profile.id)}
                              className="text-destructive hover:text-destructive"
                            >
                              <Trash2 className="w-4 h-4" />
                            </Button>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}

                  {/* Add button */}
                  <div className="px-4 py-3 border-t border-border">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleOpenAddDialog(agentType.id)}
                    >
                      <Plus className="w-4 h-4 mr-1" />
                      {t("settings.agentCredentials.addProfile")}
                    </Button>
                  </div>
                </div>
              )}
            </div>
          );
        })}

        {agentTypes.length === 0 && (
          <div className="text-center py-12 text-muted-foreground">
            {t("settings.agentCredentials.noAgentTypes")}
          </div>
        )}
      </div>

      {/* Add/Edit Dialog - for custom credential profiles only */}
      <Dialog open={showAddDialog} onOpenChange={setShowAddDialog}>
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
            {/* Name */}
            <div className="grid gap-2">
              <Label htmlFor="name">{t("settings.agentCredentials.name")}</Label>
              <Input
                id="name"
                value={formName}
                onChange={(e) => setFormName(e.target.value)}
                placeholder={t("settings.agentCredentials.namePlaceholder")}
              />
            </div>

            {/* Description */}
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

            {/* Base URL */}
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

            {/* API Key */}
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
            <Button variant="outline" onClick={() => setShowAddDialog(false)}>
              {t("common.cancel")}
            </Button>
            <Button onClick={handleSubmit} disabled={formSubmitting || !formName}>
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

export default AgentCredentialsSettings;
