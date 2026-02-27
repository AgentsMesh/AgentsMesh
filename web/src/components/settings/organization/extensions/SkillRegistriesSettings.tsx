"use client";

import { useState, useEffect, useCallback, useMemo } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { extensionApi, SkillRegistry, SkillRegistryOverride } from "@/lib/api";
import type { SkillRegistryAuthType } from "@/lib/api/extension";
import { getLocalizedErrorMessage } from "@/lib/api/errors";
import { toast } from "sonner";
import { RefreshCw, Trash2, Plus, Lock, Globe } from "lucide-react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogBody, DialogFooter } from "@/components/ui/dialog";
import type { TranslationFn } from "../GeneralSettings";

// Supported agent types for the compatible_agents field
const SUPPORTED_AGENTS = [
  { slug: "claude-code", label: "Claude Code" },
  { slug: "gemini-cli", label: "Gemini CLI" },
  { slug: "codex-cli", label: "Codex CLI" },
  { slug: "aider", label: "Aider" },
] as const;

interface SkillRegistriesSettingsProps {
  t: TranslationFn;
}

export function SkillRegistriesSettings({ t }: SkillRegistriesSettingsProps) {
  const [registries, setRegistries] = useState<SkillRegistry[]>([]);
  const [overrides, setOverrides] = useState<SkillRegistryOverride[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAdd, setShowAdd] = useState(false);
  const [addUrl, setAddUrl] = useState("");
  const [addBranch, setAddBranch] = useState("");
  const [addType, setAddType] = useState("");
  const [addCompatibleAgents, setAddCompatibleAgents] = useState<string[]>(["claude-code"]);
  const [addAuthType, setAddAuthType] = useState<SkillRegistryAuthType>("none");
  const [addAuthCredential, setAddAuthCredential] = useState("");
  const [adding, setAdding] = useState(false);
  const [syncingId, setSyncingId] = useState<number | null>(null);
  const [togglingId, setTogglingId] = useState<number | null>(null);

  const loadRegistries = useCallback(async (signal?: AbortSignal) => {
    try {
      const [registriesRes, overridesRes] = await Promise.all([
        extensionApi.listSkillRegistries(),
        extensionApi.listSkillRegistryOverrides(),
      ]);
      if (signal?.aborted) return;
      setRegistries(registriesRes.skill_registries || []);
      setOverrides(overridesRes.overrides || []);
    } catch (error) {
      if (signal?.aborted) return;
      console.error("Failed to load skill registries:", error);
    } finally {
      if (!signal?.aborted) {
        setLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    loadRegistries(controller.signal);
    return () => controller.abort();
  }, [loadRegistries]);

  // Split registries into platform vs org
  // organization_id == null covers both null and undefined (omitempty in Go)
  const platformRegistries = useMemo(
    () => registries.filter((r) => r.organization_id == null),
    [registries]
  );
  const orgRegistries = useMemo(
    () => registries.filter((r) => r.organization_id != null),
    [registries]
  );

  // Build a set of disabled registry IDs for quick lookup
  const disabledRegistryIds = useMemo(() => {
    const ids = new Set<number>();
    for (const o of overrides) {
      if (o.is_disabled) ids.add(o.registry_id);
    }
    return ids;
  }, [overrides]);

  const handleTogglePlatformRegistry = useCallback(
    async (registryId: number, currentlyDisabled: boolean) => {
      setTogglingId(registryId);
      try {
        const res = await extensionApi.togglePlatformRegistry(registryId, !currentlyDisabled);
        setOverrides(res.overrides || []);
        toast.success(t("extensions.skillRegistries.toggleSuccess"));
      } catch (error) {
        toast.error(getLocalizedErrorMessage(error, t, t("extensions.skillRegistries.failedToToggle")));
      } finally {
        setTogglingId(null);
      }
    },
    [t]
  );

  const resetAddForm = useCallback(() => {
    setAddUrl("");
    setAddBranch("");
    setAddType("");
    setAddCompatibleAgents(["claude-code"]);
    setAddAuthType("none");
    setAddAuthCredential("");
  }, []);

  const handleAdd = useCallback(async () => {
    if (!addUrl.trim()) return;
    setAdding(true);
    try {
      await extensionApi.createSkillRegistry({
        repository_url: addUrl.trim(),
        branch: addBranch.trim() || undefined,
        source_type: addType.trim() || undefined,
        compatible_agents: addCompatibleAgents.length > 0 ? addCompatibleAgents : undefined,
        auth_type: addAuthType !== "none" ? addAuthType : undefined,
        auth_credential: addAuthCredential.trim() || undefined,
      });
      toast.success(t("extensions.sourceAdded"));
      setShowAdd(false);
      resetAddForm();
      loadRegistries();
    } catch (error) {
      toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToAddSource")));
    } finally {
      setAdding(false);
    }
  }, [addUrl, addBranch, addType, addCompatibleAgents, addAuthType, addAuthCredential, t, loadRegistries, resetAddForm]);

  const handleSync = useCallback(async (id: number) => {
    setSyncingId(id);
    try {
      await extensionApi.syncSkillRegistry(id);
      toast.success(t("extensions.syncStarted"));
      loadRegistries();
    } catch (error) {
      toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToSync")));
    } finally {
      setSyncingId(null);
    }
  }, [t, loadRegistries]);

  const handleDelete = useCallback(async (id: number) => {
    if (!window.confirm(t("extensions.confirmDeleteSource"))) return;
    try {
      await extensionApi.deleteSkillRegistry(id);
      toast.success(t("extensions.sourceDeleted"));
      loadRegistries();
    } catch (error) {
      toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToDeleteSource")));
    }
  }, [t, loadRegistries]);

  const getSyncStatusVariant = (status: string): "default" | "secondary" | "destructive" | "outline" => {
    switch (status) {
      case "success": return "default";
      case "syncing": return "secondary";
      case "failed": return "destructive";
      default: return "outline";
    }
  };

  return (
    <div className="space-y-6">
      {/* Platform Registries Section */}
      <div className="border border-border rounded-lg p-6">
        <div className="mb-4">
          <h2 className="text-lg font-semibold">{t("extensions.skillRegistries.platformSources")}</h2>
          <p className="text-sm text-muted-foreground">
            {t("extensions.skillRegistries.platformSourcesDescription")}
          </p>
        </div>

        {loading ? (
          <div className="text-center py-4 text-muted-foreground">
            {t("extensions.loading")}
          </div>
        ) : platformRegistries.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">
            {t("extensions.skillRegistries.noPlatformSources")}
          </div>
        ) : (
          <div className="space-y-3">
            {platformRegistries.map((registry) => {
              const isDisabled = disabledRegistryIds.has(registry.id);
              return (
                <div
                  key={registry.id}
                  className={`border border-border rounded-lg p-4 flex items-center justify-between gap-4 ${
                    isDisabled ? "opacity-60" : ""
                  }`}
                >
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <span className="font-medium truncate">{registry.repository_url}</span>
                      <Badge variant={getSyncStatusVariant(registry.sync_status)} className="text-xs shrink-0">
                        {registry.sync_status}
                      </Badge>
                      <Badge variant="secondary" className="text-xs shrink-0">
                        {t("extensions.skillRegistries.platform")}
                      </Badge>
                    </div>
                    <div className="flex items-center gap-4 text-xs text-muted-foreground">
                      <span>{t("extensions.skillRegistries.skillCount")}: {registry.skill_count}</span>
                      {registry.branch && <span>{t("extensions.branch")}: {registry.branch}</span>}
                      {registry.last_synced_at && (
                        <span>{t("extensions.skillRegistries.lastSynced")}: {new Date(registry.last_synced_at).toLocaleString()}</span>
                      )}
                    </div>
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    <Button
                      variant={isDisabled ? "outline" : "default"}
                      size="sm"
                      disabled={togglingId === registry.id}
                      onClick={() => handleTogglePlatformRegistry(registry.id, isDisabled)}
                    >
                      {isDisabled
                        ? t("extensions.skillRegistries.disabled")
                        : t("extensions.skillRegistries.enabled")}
                    </Button>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Organization Registries Section */}
      <div className="border border-border rounded-lg p-6">
        <div className="mb-4 flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold">{t("extensions.skillRegistries.orgSources")}</h2>
            <p className="text-sm text-muted-foreground">
              {t("extensions.skillRegistries.description")}
            </p>
          </div>
          <Button onClick={() => setShowAdd(true)}>
            <Plus className="w-4 h-4 mr-1" />
            {t("extensions.skillRegistries.addSource")}
          </Button>
        </div>

        {loading ? (
          <div className="text-center py-4 text-muted-foreground">
            {t("extensions.loading")}
          </div>
        ) : orgRegistries.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">
            {t("extensions.skillRegistries.noSources")}
          </div>
        ) : (
          <div className="space-y-3">
            {orgRegistries.map((registry) => (
              <div
                key={registry.id}
                className="border border-border rounded-lg p-4 flex items-center justify-between gap-4"
              >
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <span className="font-medium truncate">{registry.repository_url}</span>
                    <Badge variant={getSyncStatusVariant(registry.sync_status)} className="text-xs shrink-0">
                      {registry.sync_status}
                    </Badge>
                    {registry.source_type && (
                      <Badge variant="outline" className="text-xs shrink-0">
                        {registry.source_type}
                      </Badge>
                    )}
                    {registry.auth_type && registry.auth_type !== "none" ? (
                      <Badge variant="secondary" className="text-xs shrink-0">
                        <Lock className="w-3 h-3 mr-1" />
                        {registry.auth_type.replace("_", " ").toUpperCase()}
                      </Badge>
                    ) : (
                      <Badge variant="outline" className="text-xs shrink-0">
                        <Globe className="w-3 h-3 mr-1" />
                        {t("extensions.skillRegistries.public")}
                      </Badge>
                    )}
                  </div>
                  <div className="flex items-center gap-4 text-xs text-muted-foreground">
                    <span>{t("extensions.skillRegistries.skillCount")}: {registry.skill_count}</span>
                    {registry.branch && <span>{t("extensions.branch")}: {registry.branch}</span>}
                    {registry.last_synced_at && (
                      <span>{t("extensions.skillRegistries.lastSynced")}: {new Date(registry.last_synced_at).toLocaleString()}</span>
                    )}
                  </div>
                  {registry.compatible_agents && registry.compatible_agents.length > 0 && (
                    <div className="flex items-center gap-1 mt-1">
                      <span className="text-xs text-muted-foreground">{t("extensions.skillRegistries.compatibleAgents")}:</span>
                      {registry.compatible_agents.map((agent) => (
                        <Badge key={agent} variant="outline" className="text-xs">
                          {agent}
                        </Badge>
                      ))}
                    </div>
                  )}
                  {registry.sync_error && (
                    <p className="text-xs text-destructive mt-1">{registry.sync_error}</p>
                  )}
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  <Button
                    variant="ghost"
                    size="sm"
                    disabled={syncingId === registry.id}
                    onClick={() => handleSync(registry.id)}
                  >
                    <RefreshCw className={`w-4 h-4 ${syncingId === registry.id ? "animate-spin" : ""}`} />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleDelete(registry.id)}
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

      {/* Add Registry Dialog */}
      <Dialog open={showAdd} onOpenChange={(open) => { setShowAdd(open); if (!open) resetAddForm(); }}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>{t("extensions.skillRegistries.addSource")}</DialogTitle>
          </DialogHeader>
          <DialogBody>
            <div className="space-y-4">
              {/* Repository URL */}
              <div>
                <label className="text-sm font-medium mb-1 block">
                  {t("extensions.repoUrl")} <span className="text-destructive">*</span>
                </label>
                <Input
                  placeholder="https://github.com/owner/skills-repo"
                  value={addUrl}
                  onChange={(e) => setAddUrl(e.target.value)}
                />
              </div>

              {/* Branch */}
              <div>
                <label className="text-sm font-medium mb-1 block">
                  {t("extensions.branch")}
                </label>
                <Input
                  placeholder="main"
                  value={addBranch}
                  onChange={(e) => setAddBranch(e.target.value)}
                />
              </div>

              {/* Source Type */}
              <div>
                <label className="text-sm font-medium mb-1 block">
                  {t("extensions.skillRegistries.sourceType")}
                </label>
                <Input
                  placeholder={t("extensions.skillRegistries.sourceTypePlaceholder")}
                  value={addType}
                  onChange={(e) => setAddType(e.target.value)}
                />
              </div>

              {/* Compatible Agents */}
              <div>
                <label className="text-sm font-medium mb-1 block">
                  {t("extensions.skillRegistries.compatibleAgents")}
                </label>
                <p className="text-xs text-muted-foreground mb-2">
                  {t("extensions.skillRegistries.compatibleAgentsHint")}
                </p>
                <div className="flex flex-wrap gap-2">
                  {SUPPORTED_AGENTS.map((agent) => {
                    const isSelected = addCompatibleAgents.includes(agent.slug);
                    return (
                      <Button
                        key={agent.slug}
                        type="button"
                        variant={isSelected ? "default" : "outline"}
                        size="sm"
                        onClick={() => {
                          setAddCompatibleAgents((prev) =>
                            isSelected
                              ? prev.filter((s) => s !== agent.slug)
                              : [...prev, agent.slug]
                          );
                        }}
                      >
                        {agent.label}
                      </Button>
                    );
                  })}
                </div>
              </div>

              {/* Authentication */}
              <div className="border-t border-border pt-4">
                <label className="text-sm font-medium mb-1 block">
                  {t("extensions.skillRegistries.authentication")}
                </label>
                <p className="text-xs text-muted-foreground mb-2">
                  {t("extensions.skillRegistries.authenticationHint")}
                </p>
                <select
                  className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                  value={addAuthType}
                  onChange={(e) => {
                    setAddAuthType(e.target.value as SkillRegistryAuthType);
                    setAddAuthCredential("");
                  }}
                >
                  <option value="none">{t("extensions.skillRegistries.authNone")}</option>
                  <option value="github_pat">{t("extensions.skillRegistries.authGitHubPAT")}</option>
                  <option value="gitlab_pat">{t("extensions.skillRegistries.authGitLabPAT")}</option>
                  <option value="ssh_key">{t("extensions.skillRegistries.authSSHKey")}</option>
                </select>

                {addAuthType !== "none" && (
                  <div className="mt-3">
                    <label className="text-sm font-medium mb-1 block">
                      {addAuthType === "ssh_key"
                        ? t("extensions.skillRegistries.sshKeyLabel")
                        : t("extensions.skillRegistries.patLabel")}
                    </label>
                    {addAuthType === "ssh_key" ? (
                      <textarea
                        className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm font-mono min-h-[100px] resize-y"
                        placeholder={t("extensions.skillRegistries.sshKeyPlaceholder")}
                        value={addAuthCredential}
                        onChange={(e) => setAddAuthCredential(e.target.value)}
                        autoComplete="off"
                      />
                    ) : (
                      <Input
                        type="password"
                        placeholder={t("extensions.skillRegistries.patPlaceholder")}
                        value={addAuthCredential}
                        onChange={(e) => setAddAuthCredential(e.target.value)}
                        autoComplete="off"
                      />
                    )}
                  </div>
                )}
              </div>
            </div>
          </DialogBody>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAdd(false)}>
              {t("common.cancel")}
            </Button>
            <Button disabled={adding || !addUrl.trim()} onClick={handleAdd}>
              {adding ? t("extensions.adding") : t("extensions.skillRegistries.addSource")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
