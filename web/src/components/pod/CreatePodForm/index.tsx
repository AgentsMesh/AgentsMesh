"use client";

import React, { useMemo, useEffect, useRef } from "react";
import { useTranslations } from "@/lib/i18n/client";
import { Button } from "@/components/ui/button";
import { Spinner, CenteredSpinner } from "@/components/ui/spinner";
import { ConfigForm } from "@/components/ide/ConfigForm";
import { usePodCreationData, useCreatePodForm } from "../hooks";
import { useConfigOptions } from "@/components/ide/hooks";
import { CreatePodFormProps } from "./types";
import { mergeConfig } from "./presets";
import { RunnerSelect } from "./RunnerSelect";
import { AgentSelect } from "./AgentSelect";
import { CredentialSelect } from "./CredentialSelect";
import { RepositorySelect, BranchInput } from "./RepositorySelect";
import { PromptInput } from "./PromptInput";
import { estimateWorkspaceTerminalSize } from "@/lib/terminal-size";

/**
 * Shared Pod creation form component
 * Supports workspace and ticket scenarios
 */
export function CreatePodForm({
  config,
  enabled = true,
  className,
}: CreatePodFormProps) {
  const t = useTranslations();
  const prevEnabledRef = useRef(enabled);
  const promptInitializedRef = useRef(false);
  const repoInitializedRef = useRef(false);

  // Merge preset config with user config
  const mergedConfig = useMemo(() => mergeConfig(config), [config]);

  const { context, promptGenerator, onSuccess, onError, onCancel } = mergedConfig;

  // Load base data (runners, agents, repositories)
  const {
    runners,
    repositories,
    loading: loadingData,
    selectedRunner,
    setSelectedRunnerId,
    availableAgentTypes,
  } = usePodCreationData(enabled);

  // Form state management
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

  // Reset form when enabled changes from true to false (e.g., modal closes)
  useEffect(() => {
    if (prevEnabledRef.current && !enabled) {
      form.reset();
      resetConfig();
      setSelectedRunnerId(null);
      promptInitializedRef.current = false;
      repoInitializedRef.current = false;
    }
    prevEnabledRef.current = enabled;
  }, [enabled]); // eslint-disable-line react-hooks/exhaustive-deps

  // Calculate default prompt
  const defaultPrompt = useMemo(() => {
    if (promptGenerator && context) {
      return promptGenerator(context);
    }
    return "";
  }, [promptGenerator, context]);

  // Initialize repository from ticket context when available
  useEffect(() => {
    if (
      enabled &&
      context?.ticket?.repositoryId &&
      !form.selectedRepository &&
      !repoInitializedRef.current &&
      repositories.length > 0
    ) {
      form.setSelectedRepository(context.ticket.repositoryId);
      repoInitializedRef.current = true;
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [enabled, context?.ticket?.repositoryId, form.selectedRepository, form.setSelectedRepository, repositories]);

  // Initialize prompt once when default is available and form is empty
  useEffect(() => {
    if (enabled && defaultPrompt && !form.prompt && !promptInitializedRef.current) {
      form.setPrompt(defaultPrompt);
      promptInitializedRef.current = true;
    }
    // form is a stable object from custom hook, only track specific values
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [enabled, defaultPrompt, form.prompt, form.setPrompt]);

  // Handle form submission
  const handleCreate = async () => {
    if (!selectedRunner || !form.selectedAgent) return;

    try {
      // Estimate terminal size based on current window/device dimensions
      // This provides better initial PTY dimensions than hardcoded values
      // Terminal will still resize after connection via fit addon if needed
      const { cols, rows } = estimateWorkspaceTerminalSize();

      // onSuccess callback is handled in useCreatePodForm.submit
      await form.submit(selectedRunner.id, configValues, {
        ticketId: context?.ticket?.id,
        initialPrompt: form.prompt,
        cols,
        rows,
      });
    } catch (err) {
      const error = err instanceof Error ? err : new Error("Unknown error");
      onError?.(error);
    }
  };

  const hasSelectedRunner = selectedRunner !== null;

  return (
    <div className={className}>
      {loadingData ? (
        <CenteredSpinner className="py-8" />
      ) : (
        <div className="space-y-4">
          {/* Step 1: Runner Select */}
          <RunnerSelect
            runners={runners}
            selectedRunnerId={selectedRunner?.id ?? null}
            onSelect={setSelectedRunnerId}
            error={form.validationErrors.runner}
            t={t}
          />

          {/* Step 2: Agent Type Select */}
          {hasSelectedRunner && (
            <AgentSelect
              agents={availableAgentTypes}
              selectedAgentId={form.selectedAgent}
              onSelect={form.setSelectedAgent}
              error={form.validationErrors.agent}
              t={t}
            />
          )}

          {/* Step 3: Agent-specific Configuration */}
          {form.selectedAgent && (
            <>
              {/* Credential Profile Select */}
              <CredentialSelect
                profiles={form.credentialProfiles}
                selectedProfileId={form.selectedCredentialProfile}
                onSelect={form.setSelectedCredentialProfile}
                loading={form.loadingCredentials}
                t={t}
              />

              {/* Repository Select */}
              <RepositorySelect
                repositories={repositories}
                selectedRepositoryId={form.selectedRepository}
                onSelect={form.setSelectedRepository}
                t={t}
              />

              {/* Branch Input */}
              {form.selectedRepository && (
                <BranchInput
                  value={form.selectedBranch}
                  onChange={form.setSelectedBranch}
                  error={form.validationErrors.branch}
                  t={t}
                />
              )}

              {/* Initial Prompt */}
              <PromptInput
                value={form.prompt}
                onChange={form.setPrompt}
                placeholder={mergedConfig.promptPlaceholder}
                t={t}
              />

              {/* Agent Configuration Section */}
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
