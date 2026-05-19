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

export function CreatePodForm({
  config,
  enabled = true,
  className,
}: CreatePodFormProps) {
  const t = useTranslations();
  const prevEnabledRef = useRef(enabled);
  const promptInitializedRef = useRef(false);
  const repoInitializedRef = useRef(false);

  const mergedConfig = useMemo(() => mergeConfig(config), [config]);

  const { context, promptGenerator, onSuccess, onError, onCancel } = mergedConfig;

  const [selectedAgentSlug, setSelectedAgentSlug] = useState<string | null>(null);

  const {
    runners,
    repositories,
    loading: loadingData,
    selectedRunner,
    setSelectedRunnerId,
    availableAgents,
  } = usePodCreationData(enabled);

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

  const form = useCreatePodForm(
    availableAgents,
    repositories,
    onSuccess,
    configValues,
    { repositoryId: context?.ticket?.repositoryId ?? null },
  );

  useEffect(() => {
    setSelectedAgentSlug(form.selectedAgent);
  }, [form.selectedAgent]);

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

  const defaultPrompt = useMemo(() => {
    if (promptGenerator && context) {
      return promptGenerator(context);
    }
    return "";
  }, [promptGenerator, context]);

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

  useEffect(() => {
    if (enabled && defaultPrompt && !form.prompt && !promptInitializedRef.current) {
      form.setPrompt(defaultPrompt);
      promptInitializedRef.current = true;
    }
    // form is a stable object from custom hook, only track specific values
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [enabled, defaultPrompt, form.prompt, form.setPrompt]);

  const handleCreate = useCreatePodSubmitHandler(
    form, selectedRunner, configValues, context, onError,
  );

  return (
    <div className={className}>
      {loadingData ? (
        <CenteredSpinner className="py-8" />
      ) : (
        <div className="space-y-4">
          <AgentSelect
            agents={availableAgents}
            selectedAgentSlug={form.selectedAgent}
            onSelect={form.setSelectedAgent}
            error={form.validationErrors.agent}
            t={t}
          />

          {form.selectedAgent && !form.rawLayerMode && (
            <InteractionModeToggle
              supportedModes={form.supportedModes}
              interactionMode={form.interactionMode}
              onModeChange={form.setInteractionMode}
            />
          )}

          {form.selectedAgent && (
            <PromptInput
              value={form.prompt}
              onChange={form.setPrompt}
              placeholder={mergedConfig.promptPlaceholder}
              t={t}
            />
          )}

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
              showPerpetual={mergedConfig.scenario === "workspace"}
            />
          )}

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

export * from "./types";
export * from "./presets";
