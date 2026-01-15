"use client";

import React, { useMemo, useEffect, useRef } from "react";
import { useTranslations } from "@/lib/i18n/client";
import { Button } from "@/components/ui/button";
import { ConfigForm } from "@/components/ide/ConfigForm";
import {
  usePodCreationData,
  useCreatePodForm,
  RUNNER_HOST_PROFILE_ID,
} from "../hooks";
import { useConfigOptions } from "@/components/ide/hooks";
import { CreatePodFormProps } from "./types";
import { mergeConfig } from "./presets";

/**
 * 共享的 Pod 创建表单组件
 * 支持 workspace 和 ticket 两种场景
 */
export function CreatePodForm({
  config,
  enabled = true,
  className,
}: CreatePodFormProps) {
  const t = useTranslations();
  const prevEnabledRef = useRef(enabled);
  const promptInitializedRef = useRef(false);

  // 合并预设配置和用户配置
  const mergedConfig = useMemo(() => mergeConfig(config), [config]);

  const { context, promptGenerator, onSuccess, onError, onCancel } = mergedConfig;

  // 加载基础数据 (runners, agents, repositories)
  const {
    runners,
    repositories,
    loading: loadingData,
    selectedRunner,
    setSelectedRunnerId,
    availableAgentTypes,
  } = usePodCreationData(enabled);

  // 表单状态管理
  const form = useCreatePodForm(availableAgentTypes, repositories, onSuccess);

  // Config options management (loads from Backend ConfigSchema)
  const {
    fields: configFields,
    loading: loadingConfig,
    config: configValues,
    updateConfig: handleConfigChange,
    resetConfig: resetConfig,
  } = useConfigOptions(
    selectedRunner?.id || null,
    form.selectedAgentSlug,
    form.selectedAgent
  );

  // 当 enabled 从 true 变为 false 时重置表单（如 Modal 关闭）
  useEffect(() => {
    if (prevEnabledRef.current && !enabled) {
      form.reset();
      resetConfig();
      setSelectedRunnerId(null);
      promptInitializedRef.current = false;
    }
    prevEnabledRef.current = enabled;
  }, [enabled]); // eslint-disable-line react-hooks/exhaustive-deps

  // 计算默认 prompt
  const defaultPrompt = useMemo(() => {
    if (promptGenerator && context) {
      return promptGenerator(context);
    }
    return "";
  }, [promptGenerator, context]);

  // 当有默认 prompt 且表单 prompt 为空时，初始化一次
  useEffect(() => {
    if (enabled && defaultPrompt && !form.prompt && !promptInitializedRef.current) {
      form.setPrompt(defaultPrompt);
      promptInitializedRef.current = true;
    }
  }, [enabled, defaultPrompt, form.prompt, form.setPrompt]);

  // 处理 Runner 选择变更
  const handleRunnerChange = (runnerId: number | null) => {
    setSelectedRunnerId(runnerId);
  };

  // 处理表单提交
  const handleCreate = async () => {
    if (!selectedRunner || !form.selectedAgent) return;

    try {
      // onSuccess 回调已在 useCreatePodForm.submit 中处理
      await form.submit(selectedRunner.id, configValues, {
        ticketId: context?.ticket?.id,
        initialPrompt: form.prompt,
      });
    } catch (err) {
      const error = err instanceof Error ? err : new Error("Unknown error");
      onError?.(error);
    }
  };

  const hasSelectedRunner = selectedRunner !== null;
  const hasAvailableAgents = availableAgentTypes.length > 0;

  return (
    <div className={className}>
      {loadingData ? (
        <div className="flex items-center justify-center py-8">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
        </div>
      ) : (
        <div className="space-y-4">
          {/* Step 1: Runner Select */}
          <div>
            <label
              htmlFor="runner-select"
              className="block text-sm font-medium mb-2"
            >
              {t("ide.createPod.selectRunner")}
            </label>
            <select
              id="runner-select"
              className={`w-full px-3 py-2 border rounded-md bg-background ${
                form.validationErrors.runner
                  ? "border-destructive"
                  : "border-border"
              }`}
              value={selectedRunner?.id || ""}
              onChange={(e) =>
                handleRunnerChange(e.target.value ? Number(e.target.value) : null)
              }
              aria-required="true"
              aria-invalid={!!form.validationErrors.runner}
              aria-describedby={
                form.validationErrors.runner
                  ? "runner-error"
                  : runners.length === 0
                  ? "runner-help"
                  : undefined
              }
            >
              <option value="">{t("ide.createPod.selectRunnerPlaceholder")}</option>
              {runners.map((runner) => (
                <option key={runner.id} value={runner.id}>
                  {runner.node_id} ({runner.current_pods}/{runner.max_concurrent_pods})
                </option>
              ))}
            </select>
            {form.validationErrors.runner && (
              <p id="runner-error" className="text-xs text-destructive mt-1">
                {form.validationErrors.runner}
              </p>
            )}
            {!form.validationErrors.runner && runners.length === 0 && (
              <p id="runner-help" className="text-xs text-muted-foreground mt-1">
                {t("ide.createPod.noRunnersAvailable")}
              </p>
            )}
          </div>

          {/* Step 2: Agent Type Select */}
          {hasSelectedRunner && (
            <div>
              <label
                htmlFor="agent-type-select"
                className="block text-sm font-medium mb-2"
              >
                {t("ide.createPod.selectAgent")}
              </label>
              {!hasAvailableAgents ? (
                <p className="text-sm text-muted-foreground py-2">
                  {t("ide.createPod.noAgentsForRunner")}
                </p>
              ) : (
                <>
                  <select
                    id="agent-type-select"
                    className={`w-full px-3 py-2 border rounded-md bg-background ${
                      form.validationErrors.agent
                        ? "border-destructive"
                        : "border-border"
                    }`}
                    value={form.selectedAgent || ""}
                    onChange={(e) =>
                      form.setSelectedAgent(
                        e.target.value ? Number(e.target.value) : null
                      )
                    }
                    aria-required="true"
                    aria-invalid={!!form.validationErrors.agent}
                    aria-describedby={
                      form.validationErrors.agent ? "agent-error" : undefined
                    }
                  >
                    <option value="">
                      {t("ide.createPod.selectAgentPlaceholder")}
                    </option>
                    {availableAgentTypes.map((agent) => (
                      <option key={agent.id} value={agent.id}>
                        {agent.name}
                      </option>
                    ))}
                  </select>
                  {form.validationErrors.agent && (
                    <p id="agent-error" className="text-xs text-destructive mt-1">
                      {form.validationErrors.agent}
                    </p>
                  )}
                </>
              )}
            </div>
          )}

          {/* Step 3: Agent-specific Configuration */}
          {form.selectedAgent && (
            <>
              {/* Credential Profile Select */}
              <div>
                <label
                  htmlFor="credential-select"
                  className="block text-sm font-medium mb-2"
                >
                  {t("ide.createPod.selectCredential")}
                </label>
                {form.loadingCredentials ? (
                  <div className="flex items-center text-sm text-muted-foreground py-2">
                    <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-primary mr-2"></div>
                    {t("common.loading")}
                  </div>
                ) : (
                  <>
                    <select
                      id="credential-select"
                      className="w-full px-3 py-2 border border-border rounded-md bg-background"
                      value={form.selectedCredentialProfile}
                      onChange={(e) =>
                        form.setSelectedCredentialProfile(Number(e.target.value))
                      }
                    >
                      <option value={RUNNER_HOST_PROFILE_ID}>
                        RunnerHost ({t("ide.createPod.runnerHostDescription")})
                      </option>
                      {form.credentialProfiles.map((profile) => (
                        <option key={profile.id} value={profile.id}>
                          {profile.name}
                          {profile.is_default
                            ? ` (${t("settings.agentCredentials.default")})`
                            : ""}
                        </option>
                      ))}
                    </select>
                    <p className="text-xs text-muted-foreground mt-1">
                      {form.selectedCredentialProfile === RUNNER_HOST_PROFILE_ID
                        ? t("ide.createPod.runnerHostHint")
                        : t("ide.createPod.customCredentialHint")}
                    </p>
                  </>
                )}
              </div>

              {/* Repository Select */}
              <div>
                <label
                  htmlFor="repository-select"
                  className="block text-sm font-medium mb-2"
                >
                  {t("ide.createPod.selectRepository")}
                </label>
                <select
                  id="repository-select"
                  className="w-full px-3 py-2 border border-border rounded-md bg-background"
                  value={form.selectedRepository || ""}
                  onChange={(e) =>
                    form.setSelectedRepository(
                      e.target.value ? Number(e.target.value) : null
                    )
                  }
                >
                  <option value="">
                    {t("ide.createPod.selectRepositoryPlaceholder")}
                  </option>
                  {repositories.map((repo) => (
                    <option key={repo.id} value={repo.id}>
                      {repo.full_path}
                    </option>
                  ))}
                </select>
              </div>

              {/* Branch Input */}
              {form.selectedRepository && (
                <div>
                  <label
                    htmlFor="branch-input"
                    className="block text-sm font-medium mb-2"
                  >
                    {t("ide.createPod.branch")}
                  </label>
                  <input
                    id="branch-input"
                    type="text"
                    className={`w-full px-3 py-2 border rounded-md bg-background ${
                      form.validationErrors.branch
                        ? "border-destructive"
                        : "border-border"
                    }`}
                    placeholder={t("ide.createPod.branchPlaceholder")}
                    value={form.selectedBranch}
                    onChange={(e) => form.setSelectedBranch(e.target.value)}
                    aria-invalid={!!form.validationErrors.branch}
                    aria-describedby={
                      form.validationErrors.branch ? "branch-error" : undefined
                    }
                  />
                  {form.validationErrors.branch && (
                    <p id="branch-error" className="text-xs text-destructive mt-1">
                      {form.validationErrors.branch}
                    </p>
                  )}
                </div>
              )}

              {/* Initial Prompt */}
              <div>
                <label
                  htmlFor="prompt-input"
                  className="block text-sm font-medium mb-2"
                >
                  {t("ide.createPod.initialPrompt")}
                </label>
                <textarea
                  id="prompt-input"
                  className="w-full px-3 py-2 border border-border rounded-md bg-background resize-none"
                  rows={3}
                  placeholder={
                    mergedConfig.promptPlaceholder ||
                    t("ide.createPod.initialPromptPlaceholder")
                  }
                  value={form.prompt}
                  onChange={(e) => form.setPrompt(e.target.value)}
                />
              </div>

              {/* Agent Configuration Section */}
              {loadingConfig ? (
                <div className="flex items-center justify-center py-4">
                  <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-primary mr-2"></div>
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
                      agentSlug={form.selectedAgentSlug}
                    />
                  </div>
                )
              )}
            </>
          )}

          {/* Error Display */}
          {form.error && (
            <div
              role="alert"
              aria-live="assertive"
              className="bg-destructive/10 border border-destructive/30 rounded-md p-3"
            >
              <p className="text-sm text-destructive">{form.error}</p>
            </div>
          )}
        </div>
      )}

      {/* Action Buttons */}
      <div className="flex flex-col-reverse sm:flex-row justify-end gap-3 mt-6">
        {onCancel && (
          <Button variant="outline" onClick={onCancel} className="w-full sm:w-auto">
            {t("ide.createPod.cancel")}
          </Button>
        )}
        <Button
          onClick={handleCreate}
          disabled={
            !selectedRunner || !form.selectedAgent || form.loading || loadingData
          }
          className="w-full sm:w-auto"
        >
          {form.loading ? t("ide.createPod.creating") : t("ide.createPod.create")}
        </Button>
      </div>
    </div>
  );
}

export default CreatePodForm;

// Re-export types
export * from "./types";
export * from "./presets";
