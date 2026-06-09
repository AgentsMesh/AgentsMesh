"use client";

import { useState, useCallback } from "react";
import { CenteredSpinner } from "@/components/ui/spinner";
import { AlertMessage } from "@/components/ui/alert-message";
import { ConfirmDialog, useConfirmDialog } from "@/components/ui/confirm-dialog";
import { useTranslations } from "next-intl";
import { Bot, AlertCircle } from "lucide-react";
import type { CredentialProfileViewModel } from "../_shared/credentialViewModel";
import { useAgentConfig } from "./useAgentConfig";
import { CredentialsSection } from "./CredentialsSection";
import { RuntimeConfigSection } from "./RuntimeConfigSection";
import { RuntimeBundlesSection } from "./RuntimeBundlesSection";
import { CredentialFormDialog } from "../CredentialFormDialog";
import { RuntimeBundleDialog } from "./RuntimeBundleDialog";
import type {
  AgentConfigPageProps,
  CredentialFormData,
  RuntimeBundleViewModel,
} from "./types";

/**
 * AgentConfigPage - Unified configuration page for a single agent
 *
 * Combines credentials management and runtime configuration in one place.
 * Acts as the coordinator for the extracted sub-components.
 */
export function AgentConfigPage({ agentSlug }: AgentConfigPageProps) {
  const t = useTranslations();

  // Dialog state
  const [showCredentialDialog, setShowCredentialDialog] = useState(false);
  const [editingProfile, setEditingProfile] = useState<CredentialProfileViewModel | null>(null);
  const [showRuntimeDialog, setShowRuntimeDialog] = useState(false);
  const [editingRuntime, setEditingRuntime] = useState<RuntimeBundleViewModel | null>(null);

  // Use the custom hook for data and actions
  const {
    loading,
    savingConfig,
    agent,
    configFields,
    configValues,
    credentialProfiles,
    noPrimaryBundle,
    runtimeBundles,
    error,
    success,
    handleConfigChange,
    handleSaveConfig,
    handleClearPrimaryBundle,
    handleSetDefault,
    handleDeleteProfile,
    handleSaveProfile,
    handleSetRuntimePrimary,
    handleClearRuntimePrimary,
    handleDeleteRuntimeBundle,
    handleSaveRuntimeBundle,
    setError,
    setSuccess,
  } = useAgentConfig(agentSlug, t);

  // Confirm dialog for delete
  const { dialogProps, confirm } = useConfirmDialog();

  // Open credential add dialog
  const handleOpenAddDialog = useCallback(() => {
    setEditingProfile(null);
    setShowCredentialDialog(true);
  }, []);

  // Open credential edit dialog
  const handleOpenEditDialog = useCallback((profile: CredentialProfileViewModel) => {
    setEditingProfile(profile);
    setShowCredentialDialog(true);
  }, []);

  // Handle credential form submission
  const handleCredentialSubmit = useCallback(async (
    data: CredentialFormData,
    profile: CredentialProfileViewModel | null
  ) => {
    await handleSaveProfile(data, profile);
    setShowCredentialDialog(false);
  }, [handleSaveProfile]);

  // Handle delete with confirmation
  const handleDeleteWithConfirm = useCallback(async (profileId: number) => {
    const confirmed = await confirm({
      title: t("common.confirmDelete"),
      description: t("settings.agentCredentials.confirmDelete"),
      variant: "destructive",
      confirmText: t("common.delete"),
      cancelText: t("common.cancel"),
    });
    if (confirmed) {
      await handleDeleteProfile(profileId);
    }
  }, [confirm, handleDeleteProfile, t]);

  // Runtime bundle dialog handlers
  const handleOpenAddRuntime = useCallback(() => {
    setEditingRuntime(null);
    setShowRuntimeDialog(true);
  }, []);

  const handleOpenEditRuntime = useCallback((b: RuntimeBundleViewModel) => {
    setEditingRuntime(b);
    setShowRuntimeDialog(true);
  }, []);

  const handleDeleteRuntimeWithConfirm = useCallback(async (id: number) => {
    const confirmed = await confirm({
      title: t("common.confirmDelete"),
      description: t("settings.agentConfig.runtimeBundles.confirmDelete"),
      variant: "destructive",
      confirmText: t("common.delete"),
      cancelText: t("common.cancel"),
    });
    if (confirmed) {
      await handleDeleteRuntimeBundle(id);
    }
  }, [confirm, handleDeleteRuntimeBundle, t]);

  if (loading) {
    return <CenteredSpinner className="py-12" />;
  }

  if (!agent) {
    return (
      <div className="flex flex-col items-center justify-center py-12">
        <AlertCircle className="w-12 h-12 text-muted-foreground mb-4" />
        <p className="text-muted-foreground">{error || t("settings.agentConfig.agentNotFound")}</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Bot className="w-8 h-8 text-primary" />
        <div>
          <h2 className="text-xl font-semibold">{agent.name}</h2>
          {agent.description && (
            <p className="text-sm text-muted-foreground">{agent.description}</p>
          )}
        </div>
      </div>

      {/* Error/Success messages */}
      {error && <AlertMessage type="error" message={error} onDismiss={() => setError(null)} />}
      {success && <AlertMessage type="success" message={success} onDismiss={() => setSuccess(null)} />}

      {/* Credentials Section */}
      <CredentialsSection
        agentSlug={agentSlug}
        noPrimaryBundle={noPrimaryBundle}
        credentialProfiles={credentialProfiles}
        onClearPrimary={handleClearPrimaryBundle}
        onSetDefault={handleSetDefault}
        onEdit={handleOpenEditDialog}
        onDelete={handleDeleteWithConfirm}
        onAdd={handleOpenAddDialog}
        t={t}
      />

      {/* Runtime Config Section */}
      <RuntimeConfigSection
        configFields={configFields}
        configValues={configValues}
        agentSlug={agentSlug}
        saving={savingConfig}
        onChange={handleConfigChange}
        onSave={handleSaveConfig}
        t={t}
      />

      {/* Runtime EnvBundles Section — plaintext KV preferences attached
          to this agent (model overrides, log levels, etc.). */}
      <RuntimeBundlesSection
        bundles={runtimeBundles}
        onSetDefault={handleSetRuntimePrimary}
        onClearDefault={handleClearRuntimePrimary}
        onEdit={handleOpenEditRuntime}
        onDelete={handleDeleteRuntimeWithConfirm}
        onAdd={handleOpenAddRuntime}
        t={t}
      />

      {/* Add/Edit Credential Dialog */}
      <CredentialFormDialog
        open={showCredentialDialog}
        onOpenChange={setShowCredentialDialog}
        agentSlug={agentSlug}
        editingProfile={editingProfile}
        onSubmit={handleCredentialSubmit}
        t={t}
      />

      {/* Add/Edit Runtime Bundle Dialog */}
      <RuntimeBundleDialog
        open={showRuntimeDialog}
        onOpenChange={setShowRuntimeDialog}
        editingBundle={editingRuntime}
        onSubmit={handleSaveRuntimeBundle}
        t={t}
      />

      {/* Confirm Dialog */}
      <ConfirmDialog {...dialogProps} />
    </div>
  );
}

export default AgentConfigPage;
