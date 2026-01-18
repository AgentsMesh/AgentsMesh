"use client";

import React, { useEffect, useRef } from "react";
import { PodData } from "@/lib/api/client";
import { useTranslations } from "@/lib/i18n/client";
import { Button } from "@/components/ui/button";
import { ConfigForm } from "./ConfigForm";
import {
  usePodCreationData,
  useConfigOptions,
  useFocusTrap,
  useCreatePodForm,
  RUNNER_HOST_PROFILE_ID,
} from "./hooks";

/**
 * Ticket context for pre-filling prompt and associating pod with ticket
 */
export interface TicketContext {
  id: number;
  identifier: string;
  title: string;
  description?: string;
}

interface CreatePodModalProps {
  open: boolean;
  onClose: () => void;
  onCreated: (pod?: PodData) => void;
  /** Optional ticket context for creating pod from ticket */
  ticketContext?: TicketContext;
}

export function CreatePodModal({ open, onClose, onCreated, ticketContext }: CreatePodModalProps) {
  const t = useTranslations();

  // Load base data (runners, agents, repositories)
  // Runner selection is now managed here
  const {
    runners,
    repositories,
    loading: loadingData,
    selectedRunner,
    setSelectedRunnerId,
    availableAgentTypes,
  } = usePodCreationData(open);

  // Form state management - now receives filtered agent types
  const form = useCreatePodForm(availableAgentTypes, repositories, onCreated);

  // Config options management (loads from Backend ConfigSchema)
  const {
    fields: configFields,
    loading: loadingConfig,
    config: configValues,
    updateConfig: handleConfigChange,
    resetConfig: resetConfig,
  } = useConfigOptions(selectedRunner?.id || null, form.selectedAgentSlug, form.selectedAgent);

  // Focus trap for modal accessibility
  const modalRef = useFocusTrap<HTMLDivElement>(open, onClose);

  // Track previous open state to detect close transition
  const prevOpenRef = useRef(open);

  // Reset form when modal closes (transition from open to closed)
  useEffect(() => {
    if (prevOpenRef.current && !open) {
      // Modal just closed
      form.reset();
      resetConfig();
    }
    prevOpenRef.current = open;
  }, [open]); // eslint-disable-line react-hooks/exhaustive-deps
  // Note: form.reset and resetConfig are intentionally excluded from deps

  // Auto-fill prompt with ticket context when modal opens
  // Track if we've already set the prompt for this modal session
  const hasSetPromptRef = React.useRef(false);

  useEffect(() => {
    // Reset the ref when modal closes
    if (!open) {
      hasSetPromptRef.current = false;
      return;
    }

    // Only set prompt once per modal open session
    if (ticketContext && open && !hasSetPromptRef.current) {
      const ticketPrompt = `Work on ticket ${ticketContext.identifier}: ${ticketContext.title}${
        ticketContext.description ? `\n\n${ticketContext.description}` : ""
      }`;
      form.setPrompt(ticketPrompt);
      hasSetPromptRef.current = true;
    }
  }, [ticketContext, open, form.setPrompt]); // eslint-disable-line react-hooks/exhaustive-deps
  // Note: form is excluded to avoid infinite loops, only setPrompt is needed

  // Handle runner selection change
  const handleRunnerChange = (runnerId: number | null) => {
    setSelectedRunnerId(runnerId);
    // Agent selection will be reset automatically by useCreatePodForm
    // when availableAgentTypes changes
  };

  // Handle form submission
  const handleCreate = async () => {
    await form.submit(
      selectedRunner?.id || null,
      configValues,
      ticketContext ? { ticketId: ticketContext.id } : undefined
    );
  };

  // Helper flags for conditional rendering
  const hasSelectedRunner = selectedRunner !== null;
  const hasAvailableAgents = availableAgentTypes.length > 0;

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="create-pod-title"
    >
      <div
        ref={modalRef}
        className="bg-background border border-border rounded-lg w-full max-w-md p-4 md:p-6 max-h-[90vh] overflow-y-auto"
      >
        <h2 id="create-pod-title" className="text-lg md:text-xl font-semibold mb-4">
          {t("ide.createPod.title")}
        </h2>

        {loadingData ? (
          <div className="flex items-center justify-center py-8">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
          </div>
        ) : (
          <div className="space-y-4">
            {/* Step 1: Runner Select */}
            <div>
              <label htmlFor="runner-select" className="block text-sm font-medium mb-2">
                {t("ide.createPod.selectRunner")}
              </label>
              <select
                id="runner-select"
                className={`w-full px-3 py-2 border rounded-md bg-background ${
                  form.validationErrors.runner ? "border-destructive" : "border-border"
                }`}
                value={selectedRunner?.id || ""}
                onChange={(e) => handleRunnerChange(e.target.value ? Number(e.target.value) : null)}
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

            {/* Step 2: Agent Type Select (only shown after runner is selected) */}
            {hasSelectedRunner && (
              <div>
                <label htmlFor="agent-type-select" className="block text-sm font-medium mb-2">
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
                        form.validationErrors.agent ? "border-destructive" : "border-border"
                      }`}
                      value={form.selectedAgent || ""}
                      onChange={(e) => form.setSelectedAgent(e.target.value ? Number(e.target.value) : null)}
                      aria-required="true"
                      aria-invalid={!!form.validationErrors.agent}
                      aria-describedby={form.validationErrors.agent ? "agent-error" : undefined}
                    >
                      <option value="">{t("ide.createPod.selectAgentPlaceholder")}</option>
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

            {/* Step 3: Agent-specific Configuration (only shown after agent is selected) */}
            {form.selectedAgent && (
              <>
                {/* Credential Profile Select */}
                <div>
                  <label htmlFor="credential-select" className="block text-sm font-medium mb-2">
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
                        onChange={(e) => form.setSelectedCredentialProfile(Number(e.target.value))}
                      >
                        <option value={RUNNER_HOST_PROFILE_ID}>
                          RunnerHost ({t("ide.createPod.runnerHostDescription")})
                        </option>
                        {form.credentialProfiles.map((profile) => (
                          <option key={profile.id} value={profile.id}>
                            {profile.name}
                            {profile.is_default ? ` (${t("settings.agentCredentials.default")})` : ""}
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
                  <label htmlFor="repository-select" className="block text-sm font-medium mb-2">
                    {t("ide.createPod.selectRepository")}
                  </label>
                  <select
                    id="repository-select"
                    className="w-full px-3 py-2 border border-border rounded-md bg-background"
                    value={form.selectedRepository || ""}
                    onChange={(e) => form.setSelectedRepository(e.target.value ? Number(e.target.value) : null)}
                  >
                    <option value="">{t("ide.createPod.selectRepositoryPlaceholder")}</option>
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
                    <label htmlFor="branch-input" className="block text-sm font-medium mb-2">
                      {t("ide.createPod.branch")}
                    </label>
                    <input
                      id="branch-input"
                      type="text"
                      className={`w-full px-3 py-2 border rounded-md bg-background ${
                        form.validationErrors.branch ? "border-destructive" : "border-border"
                      }`}
                      placeholder={t("ide.createPod.branchPlaceholder")}
                      value={form.selectedBranch}
                      onChange={(e) => form.setSelectedBranch(e.target.value)}
                      aria-invalid={!!form.validationErrors.branch}
                      aria-describedby={form.validationErrors.branch ? "branch-error" : undefined}
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
                  <label htmlFor="prompt-input" className="block text-sm font-medium mb-2">
                    {t("ide.createPod.initialPrompt")}
                  </label>
                  <textarea
                    id="prompt-input"
                    className="w-full px-3 py-2 border border-border rounded-md bg-background resize-none"
                    rows={3}
                    placeholder={t("ide.createPod.initialPromptPlaceholder")}
                    value={form.prompt}
                    onChange={(e) => form.setPrompt(e.target.value)}
                  />
                </div>

                {/* Agent Configuration Section */}
                {loadingConfig ? (
                  <div className="flex items-center justify-center py-4">
                    <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-primary mr-2"></div>
                    <span className="text-sm text-muted-foreground">{t("ide.createPod.loadingPlugins")}</span>
                  </div>
                ) : (
                  configFields.length > 0 && (
                    <div>
                      <label className="block text-sm font-medium mb-2">{t("ide.createPod.pluginConfig")}</label>
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
          <Button variant="outline" onClick={onClose} className="w-full sm:w-auto">
            {t("ide.createPod.cancel")}
          </Button>
          <Button
            onClick={handleCreate}
            disabled={!selectedRunner || !form.selectedAgent || form.loading || loadingData}
            className="w-full sm:w-auto"
          >
            {form.loading ? t("ide.createPod.creating") : t("ide.createPod.create")}
          </Button>
        </div>
      </div>
    </div>
  );
}

export default CreatePodModal;
