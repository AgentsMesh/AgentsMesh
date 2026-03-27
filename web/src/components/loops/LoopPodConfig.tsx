"use client";

import React from "react";
import { RunnerSelect } from "@/components/pod/CreatePodForm/RunnerSelect";
import { CredentialSelect } from "@/components/pod/CreatePodForm/CredentialSelect";
import { RepositorySelect, BranchInput } from "@/components/pod/CreatePodForm/RepositorySelect";
import { AdvancedOptions } from "@/components/pod/CreatePodForm/AdvancedOptions";
import { ConfigForm } from "@/components/ide/ConfigForm";
import { Spinner } from "@/components/ui/spinner";
import type { RunnerData, RepositoryData, CredentialProfileData, ConfigField } from "@/lib/api";

interface LoopPodConfigProps {
  selectedAgentSlug: string | null;
  runners: RunnerData[];
  selectedRunnerId: number | null;
  setSelectedRunnerId: (id: number | null) => void;
  credentialProfiles: CredentialProfileData[];
  selectedCredentialProfileId: number;
  setSelectedCredentialProfileId: (id: number) => void;
  loadingCredentials: boolean;
  repositories: RepositoryData[];
  selectedRepositoryId: number | null;
  setSelectedRepositoryId: (id: number | null) => void;
  selectedBranch: string;
  setSelectedBranch: (v: string) => void;
  configFields: ConfigField[];
  loadingConfig: boolean;
  configValues: Record<string, unknown>;
  handleConfigChange: (key: string, value: unknown) => void;
  t: (key: string) => string;
}

export function LoopPodConfig({
  selectedAgentSlug,
  runners,
  selectedRunnerId,
  setSelectedRunnerId,
  credentialProfiles,
  selectedCredentialProfileId,
  setSelectedCredentialProfileId,
  loadingCredentials,
  repositories,
  selectedRepositoryId,
  setSelectedRepositoryId,
  selectedBranch,
  setSelectedBranch,
  configFields,
  loadingConfig,
  configValues,
  handleConfigChange,
  t,
}: LoopPodConfigProps) {
  if (!selectedAgentSlug) return null;

  return (
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
  );
}
