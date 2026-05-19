"use client";

import React from "react";
import { useTranslations } from "next-intl";
import { Spinner } from "@/components/ui/spinner";
import { ConfigForm } from "@/components/ide/ConfigForm";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { RunnerSelect } from "./RunnerSelect";
import { CredentialSelect } from "./CredentialSelect";
import { RepositorySelect, BranchInput } from "./RepositorySelect";
import { AdvancedOptions } from "./AdvancedOptions";
import { AgentfileLayerEditor } from "./AgentfileLayerEditor";
import type { CreatePodFormState } from "../hooks";
import type { RunnerData, RepositoryData, ConfigField } from "@/lib/api";

interface AdvancedFormSectionProps {
  form: CreatePodFormState;
  runners: RunnerData[];
  repositories: RepositoryData[];
  selectedRunner: { id: number } | null;
  setSelectedRunnerId: (id: number | null) => void;
  configFields: ConfigField[];
  loadingConfig: boolean;
  configValues: Record<string, unknown>;
  handleConfigChange: (key: string, value: unknown) => void;
  showPerpetual?: boolean;
}

export function AdvancedFormSection({
  form,
  runners,
  repositories,
  selectedRunner,
  setSelectedRunnerId,
  configFields,
  loadingConfig,
  configValues,
  handleConfigChange,
  showPerpetual,
}: AdvancedFormSectionProps) {
  const t = useTranslations();

  const hideFormSections = form.rawLayerMode;

  return (
    <AdvancedOptions t={t}>
      {/* Pod Alias (optional display name) — always visible */}
      <div>
        <label htmlFor="pod-alias" className="block text-sm font-medium mb-1">
          {t("ide.createPod.alias")}
        </label>
        <Input
          id="pod-alias"
          value={form.alias}
          onChange={(e) => form.setAlias(e.target.value)}
          placeholder={t("ide.createPod.aliasPlaceholder")}
          maxLength={100}
        />
      </div>

      {/* Perpetual Pod (auto-restart on exit) — workspace only */}
      {showPerpetual && (
        <div className="flex items-center justify-between">
          <div>
            <label htmlFor="pod-perpetual" className="text-sm font-medium">
              {t("ide.createPod.perpetual")}
            </label>
            <p className="text-xs text-muted-foreground">
              {t("ide.createPod.perpetualDescription")}
            </p>
          </div>
          <Switch
            id="pod-perpetual"
            checked={form.perpetual}
            onCheckedChange={form.setPerpetual}
          />
        </div>
      )}

      {/* Runner Select (manual override, optional) — always visible */}
      <RunnerSelect
        runners={runners}
        selectedRunnerId={selectedRunner?.id ?? null}
        onSelect={setSelectedRunnerId}
        error={form.validationErrors.runner}
        t={t}
      />

      {/* Form-mode-only sections (hidden when source mode is ON) */}
      {!hideFormSections && (
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

      {/* AgentFile Layer Editor — always visible */}
      <AgentfileLayerEditor
        generatedLayer={form.agentfileLayer}
        rawMode={form.rawLayerMode}
        rawText={form.rawLayerText}
        onRawModeChange={form.setRawLayerMode}
        onRawTextChange={form.setRawLayerText}
        configFields={configFields}
        repositories={repositories}
        credentialProfiles={form.credentialProfiles}
        t={t}
      />
    </AdvancedOptions>
  );
}
