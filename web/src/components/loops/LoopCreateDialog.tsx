"use client";

import React from "react";
import {
  ResponsiveDialog,
  ResponsiveDialogContent,
  ResponsiveDialogHeader,
  ResponsiveDialogTitle,
  ResponsiveDialogBody,
  ResponsiveDialogFooter,
} from "@/components/ui/responsive-dialog";
import { Button } from "@/components/ui/button";
import { Loader2 } from "lucide-react";
import { useTranslations } from "next-intl";
import type { LoopData } from "@/lib/api/loop";

// Reuse Pod creation data hooks
import { usePodCreationData } from "@/components/pod/hooks";
import { useConfigOptions } from "@/components/ide/hooks";

// Extracted sub-components and hooks
import { useLoopForm } from "./useLoopForm";
import { useCredentialLoader } from "./useLoopFormEffects";
import { useLoopSubmit } from "./useLoopSubmit";
import { LoopBasicFields } from "./LoopBasicFields";
import { LoopPodConfig } from "./LoopPodConfig";
import { LoopSettings } from "./LoopSettings";

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
  const isEdit = !!editLoop;

  // Form state
  const form = useLoopForm(open, editLoop);

  // Pod creation data (agents, runners, repositories)
  const { availableAgents, runners, repositories } = usePodCreationData(open);

  // Agent config (config fields based on selected agent)
  const { fields: configFields, loading: loadingConfig, config: configValues, updateConfig: handleConfigChange } =
    useConfigOptions(form.selectedRunnerId, form.selectedAgentSlug);

  // Credential profiles
  const {
    credentialProfiles,
    loadingCredentials,
    selectedCredentialProfileId,
    setSelectedCredentialProfileId,
  } = useCredentialLoader(open, form.selectedAgentSlug, editLoop);

  // Submit handler
  const { handleSubmit } = useLoopSubmit({
    form,
    editLoop,
    onCreated,
    configValues,
    selectedCredentialProfileId,
  });

  const canSubmit = form.name.trim() && form.promptTemplate.trim() && form.selectedAgentSlug;

  return (
    <ResponsiveDialog open={open} onOpenChange={onOpenChange}>
      <ResponsiveDialogContent className="max-w-xl">
        <ResponsiveDialogHeader>
          <ResponsiveDialogTitle>
            {isEdit ? t("loops.editLoop") : t("loops.createLoop")}
          </ResponsiveDialogTitle>
        </ResponsiveDialogHeader>

        <ResponsiveDialogBody className="space-y-4 py-4">
          {/* Basic Fields: name, description, agent, prompt */}
          <LoopBasicFields
            name={form.name} setName={form.setName}
            description={form.description} setDescription={form.setDescription}
            promptTemplate={form.promptTemplate} setPromptTemplate={form.setPromptTemplate}
            selectedAgentSlug={form.selectedAgentSlug}
            setSelectedAgentSlug={form.setSelectedAgentSlug}
            availableAgents={availableAgents}
            t={t}
          />

          {/* Pod Configuration: runner, credentials, repository, config */}
          <LoopPodConfig
            selectedAgentSlug={form.selectedAgentSlug}
            runners={runners}
            selectedRunnerId={form.selectedRunnerId}
            setSelectedRunnerId={form.setSelectedRunnerId}
            credentialProfiles={credentialProfiles}
            selectedCredentialProfileId={selectedCredentialProfileId}
            setSelectedCredentialProfileId={setSelectedCredentialProfileId}
            loadingCredentials={loadingCredentials}
            repositories={repositories}
            selectedRepositoryId={form.selectedRepositoryId}
            setSelectedRepositoryId={form.setSelectedRepositoryId}
            selectedBranch={form.selectedBranch}
            setSelectedBranch={form.setSelectedBranch}
            configFields={configFields}
            loadingConfig={loadingConfig}
            configValues={configValues}
            handleConfigChange={handleConfigChange}
            t={t}
          />

          {/* Loop Settings: cron, sandbox, concurrency, timeout, etc. */}
          <LoopSettings
            executionMode={form.executionMode} setExecutionMode={form.setExecutionMode}
            cronEnabled={form.cronEnabled} setCronEnabled={form.setCronEnabled}
            cronExpression={form.cronExpression} setCronExpression={form.setCronExpression}
            sandboxStrategy={form.sandboxStrategy} setSandboxStrategy={form.setSandboxStrategy}
            concurrencyPolicy={form.concurrencyPolicy} setConcurrencyPolicy={form.setConcurrencyPolicy}
            timeoutMinutes={form.timeoutMinutes} setTimeoutMinutes={form.setTimeoutMinutes}
            maxConcurrentRuns={form.maxConcurrentRuns} setMaxConcurrentRuns={form.setMaxConcurrentRuns}
            maxRetainedRuns={form.maxRetainedRuns} setMaxRetainedRuns={form.setMaxRetainedRuns}
            sessionPersistence={form.sessionPersistence} setSessionPersistence={form.setSessionPersistence}
            callbackUrl={form.callbackUrl} setCallbackUrl={form.setCallbackUrl}
            t={t}
          />
        </ResponsiveDialogBody>

        <ResponsiveDialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("common.cancel")}
          </Button>
          <Button onClick={handleSubmit} disabled={!canSubmit || form.loading}>
            {form.loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            {isEdit ? t("loops.update") : t("loops.create")}
          </Button>
        </ResponsiveDialogFooter>
      </ResponsiveDialogContent>
    </ResponsiveDialog>
  );
}
