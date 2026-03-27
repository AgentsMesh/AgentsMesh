"use client";

import React, { useState, useCallback, useMemo, useEffect } from "react";
import {
  ResponsiveDialog,
  ResponsiveDialogContent,
  ResponsiveDialogHeader,
  ResponsiveDialogTitle,
  ResponsiveDialogBody,
  ResponsiveDialogFooter,
} from "@/components/ui/responsive-dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Loader2 } from "lucide-react";
import { useLoopStore } from "@/stores/loop";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

// Reuse Pod creation components
import { usePodCreationData } from "@/components/pod/hooks";
import { useConfigOptions } from "@/components/ide/hooks";
import { AgentSelect } from "@/components/pod/CreatePodForm/AgentSelect";
import { RunnerSelect } from "@/components/pod/CreatePodForm/RunnerSelect";
import { CredentialSelect } from "@/components/pod/CreatePodForm/CredentialSelect";
import { RepositorySelect, BranchInput } from "@/components/pod/CreatePodForm/RepositorySelect";
import { PromptInput } from "@/components/pod/CreatePodForm/PromptInput";
import { AdvancedOptions } from "@/components/pod/CreatePodForm/AdvancedOptions";
import { ConfigForm } from "@/components/ide/ConfigForm";
import { Spinner } from "@/components/ui/spinner";
import { userAgentCredentialApi, CredentialProfileData } from "@/lib/api";
import type { LoopData } from "@/lib/api/loop";

// Special value for RunnerHost credential
const RUNNER_HOST_PROFILE_ID = 0;

interface LoopCreateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: (createdLoop?: LoopData) => void;
  editLoop?: LoopData;
}

export function LoopCreateDialog({
  open,
  onOpenChange,
  onCreated,
  editLoop,
}: LoopCreateDialogProps) {
  const t = useTranslations();
  const createLoop = useLoopStore((s) => s.createLoop);
  const updateLoop = useLoopStore((s) => s.updateLoop);
  const isEdit = !!editLoop;

  const [loading, setLoading] = useState(false);

  // --- Basic fields ---
  const [name, setName] = useState(editLoop?.name || "");
  const [description, setDescription] = useState(editLoop?.description || "");
  const [promptTemplate, setPromptTemplate] = useState(editLoop?.prompt_template || "");

  // --- Pod configuration fields ---
  const [selectedAgentSlug, setSelectedAgentSlug] = useState<string | null>(editLoop?.agent_slug || null);
  const [selectedRunnerId, setSelectedRunnerId] = useState<number | null>(editLoop?.runner_id || null);
  const [selectedRepositoryId, setSelectedRepositoryId] = useState<number | null>(editLoop?.repository_id || null);
  const [selectedBranch, setSelectedBranch] = useState(editLoop?.branch_name || "");
  const [selectedCredentialProfileId, setSelectedCredentialProfileId] = useState<number>(
    editLoop?.credential_profile_id || RUNNER_HOST_PROFILE_ID
  );
  const [credentialProfiles, setCredentialProfiles] = useState<CredentialProfileData[]>([]);
  const [loadingCredentials, setLoadingCredentials] = useState(false);

  // --- Loop-specific fields ---
  const [executionMode, setExecutionMode] = useState<string>(editLoop?.execution_mode || "autopilot");
  const [cronEnabled, setCronEnabled] = useState(!!editLoop?.cron_expression);
  const [cronExpression, setCronExpression] = useState(editLoop?.cron_expression || "");
  const [sandboxStrategy, setSandboxStrategy] = useState<string>(editLoop?.sandbox_strategy || "persistent");
  const [concurrencyPolicy, setConcurrencyPolicy] = useState<string>(editLoop?.concurrency_policy || "skip");
  const [timeoutMinutes, setTimeoutMinutes] = useState(editLoop?.timeout_minutes || 60);
  const [callbackUrl, setCallbackUrl] = useState(editLoop?.callback_url || "");
  const [sessionPersistence, setSessionPersistence] = useState(editLoop?.session_persistence ?? true);
  const [maxConcurrentRuns, setMaxConcurrentRuns] = useState(editLoop?.max_concurrent_runs || 1);
  const [maxRetainedRuns, setMaxRetainedRuns] = useState(editLoop?.max_retained_runs || 0);

  // Sync form state when dialog opens or editLoop changes
  useEffect(() => {
    if (!open) return;
    setName(editLoop?.name || "");
    setDescription(editLoop?.description || "");
    setPromptTemplate(editLoop?.prompt_template || "");
    setSelectedAgentSlug(editLoop?.agent_slug || null);
    setSelectedRunnerId(editLoop?.runner_id || null);
    setSelectedRepositoryId(editLoop?.repository_id || null);
    setSelectedBranch(editLoop?.branch_name || "");
    setSelectedCredentialProfileId(editLoop?.credential_profile_id || RUNNER_HOST_PROFILE_ID);
    setExecutionMode(editLoop?.execution_mode || "autopilot");
    setCronEnabled(!!editLoop?.cron_expression);
    setCronExpression(editLoop?.cron_expression || "");
    setSandboxStrategy(editLoop?.sandbox_strategy || "persistent");
    setConcurrencyPolicy(editLoop?.concurrency_policy || "skip");
    setTimeoutMinutes(editLoop?.timeout_minutes || 60);
    setCallbackUrl(editLoop?.callback_url || "");
    setSessionPersistence(editLoop?.session_persistence ?? true);
    setMaxConcurrentRuns(editLoop?.max_concurrent_runs || 1);
    setMaxRetainedRuns(editLoop?.max_retained_runs || 0);
    setLoading(false);
  }, [open, editLoop]);

  // --- Load Pod creation data (runners, agents, repositories) ---
  const {
    runners,
    repositories,
    selectedRunner,
    setSelectedRunnerId: setPodSelectedRunnerId,
    availableAgents,
  } = usePodCreationData(open);

  // Sync runner selection with Pod creation data hook
  useEffect(() => {
    setPodSelectedRunnerId(selectedRunnerId);
  }, [selectedRunnerId, setPodSelectedRunnerId]);


  // Load agent config schema
  const {
    fields: configFields,
    loading: loadingConfig,
    config: configValues,
    updateConfig: handleConfigChange,
  } = useConfigOptions(
    selectedRunner?.id || null,
    selectedAgentSlug,
    );

  // Restore config_overrides from editLoop in edit mode once config fields have loaded
  const [configOverridesRestored, setConfigOverridesRestored] = useState(false);
  useEffect(() => {
    if (!open) {
      setConfigOverridesRestored(false);
      return;
    }
    if (editLoop?.config_overrides && configFields.length > 0 && !configOverridesRestored) {
      Object.entries(editLoop.config_overrides).forEach(([key, value]) => {
        handleConfigChange(key, value);
      });
      setConfigOverridesRestored(true);
    }
  }, [open, editLoop, configFields, configOverridesRestored, handleConfigChange]);

  // Load credential profiles when agent changes.
  // In edit mode, preserve the editLoop's credential_profile_id on first load.
  const [credentialInitialized, setCredentialInitialized] = useState(false);

  useEffect(() => {
    // Reset initialization flag when dialog re-opens
    if (!open) {
      setCredentialInitialized(false);
      return;
    }
  }, [open]);

  useEffect(() => {
    if (!selectedAgentSlug) {
      setCredentialProfiles([]);
      setSelectedCredentialProfileId(RUNNER_HOST_PROFILE_ID);
      setCredentialInitialized(false);
      return;
    }

    const loadCredentials = async () => {
      setLoadingCredentials(true);
      try {
        const res = await userAgentCredentialApi.listForAgent(selectedAgentSlug);
        const profiles = res.profiles || [];
        setCredentialProfiles(profiles);

        // In edit mode, preserve editLoop's credential on initial load
        if (editLoop?.credential_profile_id && !credentialInitialized) {
          setSelectedCredentialProfileId(editLoop.credential_profile_id);
          setCredentialInitialized(true);
        } else {
          const defaultProfile = profiles.find((p) => p.is_default);
          if (defaultProfile) {
            setSelectedCredentialProfileId(defaultProfile.id);
          } else {
            setSelectedCredentialProfileId(RUNNER_HOST_PROFILE_ID);
          }
        }
      } catch {
        setCredentialProfiles([]);
        setSelectedCredentialProfileId(RUNNER_HOST_PROFILE_ID);
      } finally {
        setLoadingCredentials(false);
      }
    };

    loadCredentials();
  }, [selectedAgentSlug, editLoop, credentialInitialized]);

  // Auto-fill branch when repository changes
  useEffect(() => {
    if (!selectedRepositoryId) {
      setSelectedBranch("");
      return;
    }
    const repo = repositories.find((r) => r.id === selectedRepositoryId);
    if (repo?.default_branch) {
      setSelectedBranch(repo.default_branch);
    }
  }, [selectedRepositoryId, repositories]);

  // Reset agent if not available in current runner's agents (only after agents loaded)
  useEffect(() => {
    if (availableAgents.length > 0 && selectedAgentSlug && !availableAgents.find((a) => a.slug === selectedAgentSlug)) {
      setSelectedAgentSlug(null);
    }
  }, [availableAgents, selectedAgentSlug]);

  const handleSubmit = useCallback(async () => {
    if (!name.trim() || !promptTemplate.trim() || !selectedAgentSlug) return;

    setLoading(true);
    try {
      const data = {
        name: name.trim(),
        description: description || undefined,
        agent_slug: selectedAgentSlug,
        prompt_template: promptTemplate,
        runner_id: selectedRunnerId || undefined,
        repository_id: selectedRepositoryId || undefined,
        branch_name: selectedBranch || undefined,
        credential_profile_id: selectedCredentialProfileId > 0 ? selectedCredentialProfileId : undefined,
        config_overrides: Object.keys(configValues).length > 0 ? configValues : undefined,
        execution_mode: executionMode,
        cron_expression: cronEnabled && cronExpression ? cronExpression : "",
        sandbox_strategy: sandboxStrategy,
        concurrency_policy: concurrencyPolicy,
        timeout_minutes: timeoutMinutes,
        callback_url: callbackUrl || undefined,
        session_persistence: sessionPersistence,
        max_concurrent_runs: maxConcurrentRuns,
        max_retained_runs: maxRetainedRuns,
      };

      if (isEdit && editLoop) {
        await updateLoop(editLoop.slug, data);
        toast.success(t("loops.updated"));
        onCreated();
      } else {
        const res = await createLoop(data);
        toast.success(t("loops.created"));
        onCreated(res.loop);
      }
    } catch (err) {
      toast.error(isEdit ? t("loops.updateFailed") : t("loops.createFailed"), {
        description: (err as Error).message,
      });
    } finally {
      setLoading(false);
    }
  }, [
    name, description, promptTemplate, selectedAgentSlug, selectedRunnerId,
    selectedRepositoryId, selectedBranch, selectedCredentialProfileId, configValues,
    executionMode, cronEnabled, cronExpression, sandboxStrategy,
    concurrencyPolicy, timeoutMinutes, callbackUrl, sessionPersistence,
    maxConcurrentRuns, maxRetainedRuns, isEdit, editLoop, createLoop, updateLoop, onCreated, t,
  ]);

  const dialogTitle = isEdit ? t("loops.editLoop") : t("loops.createLoop");

  return (
    <ResponsiveDialog open={open} onOpenChange={onOpenChange}>
      <ResponsiveDialogContent className="max-w-lg">
        <ResponsiveDialogHeader onClose={() => onOpenChange(false)}>
          <ResponsiveDialogTitle>{dialogTitle}</ResponsiveDialogTitle>
        </ResponsiveDialogHeader>

        <ResponsiveDialogBody className="space-y-4">
          {/* Name */}
          <div className="space-y-1.5">
            <Label>{t("loops.name")}</Label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="daily-code-review"
            />
          </div>

          {/* Description */}
          <div className="space-y-1.5">
            <Label>{t("loops.description")}</Label>
            <Input
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder={t("loops.descriptionPlaceholder")}
            />
          </div>

          {/* Agent Select */}
          <AgentSelect
            agents={availableAgents}
            selectedAgentSlug={selectedAgentSlug}
            onSelect={setSelectedAgentSlug}
            t={t}
          />

          {/* Prompt Template (shown when agent selected) */}
          {selectedAgentSlug && (
            <PromptInput
              value={promptTemplate}
              onChange={setPromptTemplate}
              placeholder={t("loops.promptPlaceholder")}
              t={t}
            />
          )}

          {/* Pod Configuration (Advanced, collapsed) */}
          {selectedAgentSlug && (
            <AdvancedOptions t={t}>
              <RunnerSelect
                runners={runners}
                selectedRunnerId={selectedRunnerId}
                onSelect={setSelectedRunnerId}
                t={t}
              />

              <CredentialSelect
                profiles={credentialProfiles}
                selectedProfileId={selectedCredentialProfileId}
                onSelect={setSelectedCredentialProfileId}
                loading={loadingCredentials}
                t={t}
              />

              <RepositorySelect
                repositories={repositories}
                selectedRepositoryId={selectedRepositoryId}
                onSelect={setSelectedRepositoryId}
                t={t}
              />

              {selectedRepositoryId && (
                <BranchInput
                  value={selectedBranch}
                  onChange={setSelectedBranch}
                  t={t}
                />
              )}

              {loadingConfig ? (
                <div className="flex items-center justify-center py-4">
                  <Spinner size="sm" className="mr-2" />
                  <span className="text-sm text-muted-foreground">
                    {t("ide.createPod.loadingPlugins")}
                  </span>
                </div>
              ) : (
                configFields.length > 0 && (
                  <div>
                    <label className="block text-sm font-medium mb-2">
                      {t("ide.createPod.pluginConfig")}
                    </label>
                    <ConfigForm
                      fields={configFields}
                      values={configValues}
                      onChange={handleConfigChange}
                      agentSlug={selectedAgentSlug}
                    />
                  </div>
                )
              )}
            </AdvancedOptions>
          )}

          {/* --- Loop settings --- */}
          <div className="border-t border-border pt-4 space-y-4">
            {/* Cron scheduling (optional, API trigger is always available) */}
            <div className="flex items-center justify-between">
              <div>
                <Label>{t("loops.enableCron")}</Label>
                <p className="text-xs text-muted-foreground">{t("loops.apiAlwaysAvailable")}</p>
              </div>
              <Switch checked={cronEnabled} onCheckedChange={setCronEnabled} />
            </div>

            {cronEnabled && (
              <div className="space-y-1.5">
                <Label>{t("loops.cronExpression")}</Label>
                <Input
                  value={cronExpression}
                  onChange={(e) => setCronExpression(e.target.value)}
                  placeholder="0 9 * * *"
                  className="font-mono"
                />
                <p className="text-xs text-muted-foreground">
                  {t("loops.cronHelp")}
                </p>
              </div>
            )}

            {/* Execution Mode */}
            <div className="space-y-1.5">
              <Label>{t("loops.executionMode")}</Label>
              <Select value={executionMode} onValueChange={setExecutionMode}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="autopilot">{t("loops.modeAutopilot")}</SelectItem>
                  <SelectItem value="direct">{t("loops.modeDirect")}</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {/* Sandbox & Concurrency */}
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <Label>{t("loops.sandboxStrategy")}</Label>
                <Select value={sandboxStrategy} onValueChange={setSandboxStrategy}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="persistent">{t("loops.sandboxPersistent")}</SelectItem>
                    <SelectItem value="fresh">{t("loops.sandboxFresh")}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-1.5">
                <Label>{t("loops.concurrency")}</Label>
                <Select value={concurrencyPolicy} onValueChange={setConcurrencyPolicy}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="skip">{t("loops.policySkip")}</SelectItem>
                    <SelectItem value="queue" disabled>
                      {t("loops.policyQueue")} ({t("loops.comingSoon")})
                    </SelectItem>
                    <SelectItem value="replace" disabled>
                      {t("loops.policyReplace")} ({t("loops.comingSoon")})
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>

            {/* Timeout & Max Concurrent */}
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <Label>{t("loops.timeout")}</Label>
                <div className="flex items-center gap-2">
                  <Input
                    type="number"
                    min={1}
                    max={1440}
                    value={timeoutMinutes}
                    onChange={(e) => setTimeoutMinutes(Math.max(1, parseInt(e.target.value) || 60))}
                    className="w-24"
                  />
                  <span className="text-sm text-muted-foreground">{t("loops.minutes")}</span>
                </div>
              </div>
              <div className="space-y-1.5">
                <Label>{t("loops.maxConcurrentRuns")}</Label>
                <Input
                  type="number"
                  min={1}
                  max={10}
                  value={maxConcurrentRuns}
                  onChange={(e) => setMaxConcurrentRuns(parseInt(e.target.value) || 1)}
                  className="w-24"
                />
              </div>
            </div>

            {/* Run History Limit */}
            <div className="space-y-1.5">
              <Label>{t("loops.maxRetainedRuns")}</Label>
              <div className="flex items-center gap-2">
                <Input
                  type="number"
                  min={0}
                  max={10000}
                  value={maxRetainedRuns}
                  onChange={(e) => setMaxRetainedRuns(Math.max(0, parseInt(e.target.value) || 0))}
                  className="w-24"
                />
                <span className="text-sm text-muted-foreground">
                  {maxRetainedRuns === 0 ? t("loops.unlimited") : ""}
                </span>
              </div>
              <p className="text-xs text-muted-foreground">{t("loops.maxRetainedRunsHelp")}</p>
            </div>

            {/* Session Persistence */}
            {sandboxStrategy === "persistent" && (
              <div className="flex items-center justify-between">
                <div>
                  <Label>{t("loops.sessionPersistence")}</Label>
                  <p className="text-xs text-muted-foreground">{t("loops.sessionPersistenceHelp")}</p>
                </div>
                <Switch checked={sessionPersistence} onCheckedChange={setSessionPersistence} />
              </div>
            )}

            {/* Webhook URL */}
            <div className="space-y-1.5">
              <Label>{t("loops.callbackUrl")}</Label>
              <Input
                type="url"
                value={callbackUrl}
                onChange={(e) => setCallbackUrl(e.target.value)}
                placeholder={t("loops.callbackUrlPlaceholder")}
              />
              <p className="text-xs text-muted-foreground">{t("loops.callbackUrlHelp")}</p>
            </div>
          </div>
        </ResponsiveDialogBody>

        <ResponsiveDialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("common.cancel")}
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={loading || !name.trim() || !promptTemplate.trim() || !selectedAgentSlug}
          >
            {loading && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
            {isEdit ? t("common.save") : t("loops.createLoop")}
          </Button>
        </ResponsiveDialogFooter>
      </ResponsiveDialogContent>
    </ResponsiveDialog>
  );
}
