"use client";

import { useState, useCallback } from "react";
import { CenteredSpinner } from "@/components/ui/spinner";
import { AlertMessage } from "@/components/ui/alert-message";
import { ConfirmDialog, useConfirmDialog } from "@/components/ui/confirm-dialog";
import { useTranslations } from "next-intl";
import { Bot, AlertCircle } from "lucide-react";
import type { CredentialProfileData } from "@/lib/api";
import { useAgentConfig } from "./useAgentConfig";
import { CredentialsSection } from "./CredentialsSection";
import { RuntimeConfigSection } from "./RuntimeConfigSection";
import { CredentialDialog } from "./CredentialDialog";
import type { AgentConfigPageProps, CredentialFormData } from "./types";

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
  const [editingProfile, setEditingProfile] = useState<CredentialProfileData | null>(null);

  // Use the custom hook for data and actions
  const {
    loading,
    savingConfig,
    agent,
    configFields,
    configValues,
    credentialProfiles,
    isRunnerHostDefault,
    error,
    success,
    handleConfigChange,
    handleSaveConfig,
    handleSetRunnerHostDefault,
    handleSetDefault,
    handleDeleteProfile,
    handleSaveProfile,
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
  const handleOpenEditDialog = useCallback((profile: CredentialProfileData) => {
    setEditingProfile(profile);
    setShowCredentialDialog(true);
  }, []);

  // Handle credential form submission
  const handleCredentialSubmit = useCallback(async (
    data: CredentialFormData,
    profile: CredentialProfileData | null
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
        isRunnerHostDefault={isRunnerHostDefault}
        credentialProfiles={credentialProfiles}
        onSetRunnerHostDefault={handleSetRunnerHostDefault}
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

      {/* Add/Edit Credential Dialog */}
      <CredentialDialog
        open={showCredentialDialog}
        onOpenChange={setShowCredentialDialog}
        editingProfile={editingProfile}
        onSubmit={handleCredentialSubmit}
        t={t}
      />

      {/* Confirm Dialog */}
      <ConfirmDialog {...dialogProps} />
    </div>
  );
}

export default AgentConfigPage;
