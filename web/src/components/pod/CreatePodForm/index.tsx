"use client";

import React, { useState, useMemo, useEffect, useRef } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { CenteredSpinner } from "@/components/ui/spinner";
import { usePodCreationData, useCreatePodForm } from "../hooks";
import { useConfigOptions } from "@/components/ide/hooks";
import { CreatePodFormProps } from "./types";
import { mergeConfig } from "./presets";
import { AgentSelect } from "./AgentSelect";
import { PromptInput } from "./PromptInput";
import { InteractionModeToggle } from "./InteractionModeToggle";
import { AdvancedFormSection } from "./AdvancedFormSection";
import { useCreatePodSubmitHandler } from "./useCreatePodSubmitHandler";

/**
 * Shared Pod creation form component
 * Agent-first layout with advanced options collapsed
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

  // Track selected agent at parent level so both hooks can consume it
  const [selectedAgentSlug, setSelectedAgentSlug] = useState<string | null>(null);

  // Load base data (runners, agents, repositories)
  const {
    runners,
    repositories,
    loading: loadingData,
    selectedRunner,
    setSelectedRunnerId,
    availableAgents,
  } = usePodCreationData(enabled);

  // Config options management (loads from Backend ConfigSchema)
  // Placed before useCreatePodForm so configValues is available
  const {
    fields: configFields,
    loading: loadingConfig,
    config: configValues,
    updateConfig: handleConfigChange,
    resetConfig: resetConfig,
  } = useConfigOptions(
    selectedRunner?.id || null,
    selectedAgentSlug
  );

  // Form state management (receives configValues for AgentFile Layer generation)
  const form = useCreatePodForm(availableAgents, repositories, onSuccess, configValues);

  // Sync selected agent from form to local state for useConfigOptions
  useEffect(() => {
    setSelectedAgentSlug(form.selectedAgent);
  }, [form.selectedAgent]);

  // Reset form when enabled changes from true to false (e.g., modal closes)
  useEffect(() => {
    if (prevEnabledRef.current && !enabled) {
      form.reset();
      resetConfig();
      setSelectedRunnerId(null);
      setSelectedAgentSlug(null);
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
  const handleCreate = useCreatePodSubmitHandler(
    form, selectedRunner, configValues, context, onError,
  );

  return (
    <div className={className}>
      {loadingData ? (
        <CenteredSpinner className="py-8" />
      ) : (
        <div className="space-y-4">
          {/* Agent Select (shown first) */}
          <AgentSelect
            agents={availableAgents}
            selectedAgentSlug={form.selectedAgent}
            onSelect={form.setSelectedAgent}
            error={form.validationErrors.agent}
            t={t}
          />

          {/* Interaction Mode Toggle (only when agent supports multiple modes and source mode is off) */}
          {form.selectedAgent && !form.rawLayerMode && (
            <InteractionModeToggle
              supportedModes={form.supportedModes}
              interactionMode={form.interactionMode}
              onModeChange={form.setInteractionMode}
            />
          )}

          {/* Initial Prompt (visible at top level) */}
          {form.selectedAgent && (
            <PromptInput
              value={form.prompt}
              onChange={form.setPrompt}
              placeholder={mergedConfig.promptPlaceholder}
              t={t}
            />
          )}

          {/* Advanced Options (collapsed by default) */}
          {form.selectedAgent && (
            <AdvancedFormSection
              form={form}
              runners={runners}
              repositories={repositories}
              selectedRunner={selectedRunner}
              setSelectedRunnerId={setSelectedRunnerId}
              configFields={configFields}
              loadingConfig={loadingConfig}
              configValues={configValues}
              handleConfigChange={handleConfigChange}
            />
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
          disabled={!form.selectedAgent || form.loading || loadingData}
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
