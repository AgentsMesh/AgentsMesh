"use client";

import { useState, useCallback } from "react";
import { CenteredSpinner } from "@/components/ui/spinner";
import { AlertMessage } from "@/components/ui/alert-message";
import { ConfirmDialog, useConfirmDialog } from "@/components/ui/confirm-dialog";
import { useTranslations } from "next-intl";
import type { CredentialProfileData } from "@/lib/api";
import { useAgentCredentials } from "./useAgentCredentials";
import { AgentTypeItem } from "./AgentTypeItem";
import { CredentialProfileDialog } from "./CredentialProfileDialog";
import type { CredentialFormData } from "./types";

/**
 * AgentCredentialsSettings - Manages credential profiles for all agent types
 *
 * Displays a collapsible list of agent types, each with RunnerHost as the
 * default option and custom credential profiles below.
 */
export function AgentCredentialsSettings() {
  const t = useTranslations();

  // Dialog state
  const [showDialog, setShowDialog] = useState(false);
  const [editingProfile, setEditingProfile] = useState<CredentialProfileData | null>(null);
  const [selectedAgentTypeId, setSelectedAgentTypeId] = useState<number | null>(null);

  // Use the custom hook for data and actions
  const {
    loading,
    error,
    success,
    agentTypes,
    expandedAgentTypes,
    runnerHostDefaults,
    toggleAgentType,
    handleSetRunnerHostDefault,
    handleSetDefault,
    handleDelete,
    handleSaveProfile,
    getProfilesForAgentType,
    setError,
    setSuccess,
  } = useAgentCredentials(t);

  // Confirm dialog for delete
  const { dialogProps, confirm } = useConfirmDialog();

  // Open add dialog
  const handleOpenAddDialog = useCallback((agentTypeId: number) => {
    setSelectedAgentTypeId(agentTypeId);
    setEditingProfile(null);
    setShowDialog(true);
  }, []);

  // Open edit dialog
  const handleOpenEditDialog = useCallback((profile: CredentialProfileData) => {
    setSelectedAgentTypeId(profile.agent_type_id);
    setEditingProfile(profile);
    setShowDialog(true);
  }, []);

  // Handle dialog submit
  const handleDialogSubmit = useCallback(async (data: CredentialFormData) => {
    if (!selectedAgentTypeId) return;
    await handleSaveProfile(selectedAgentTypeId, data, editingProfile);
    setShowDialog(false);
  }, [selectedAgentTypeId, editingProfile, handleSaveProfile]);

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
      await handleDelete(profileId);
    }
  }, [confirm, handleDelete, t]);

  if (loading) {
    return <CenteredSpinner className="py-12" />;
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h2 className="text-lg font-semibold">{t("settings.agentCredentials.title")}</h2>
        <p className="text-sm text-muted-foreground mt-1">
          {t("settings.agentCredentials.description")}
        </p>
      </div>

      {/* Error/Success messages */}
      {error && <AlertMessage type="error" message={error} onDismiss={() => setError(null)} />}
      {success && <AlertMessage type="success" message={success} onDismiss={() => setSuccess(null)} />}

      {/* Agent Types List */}
      <div className="space-y-2">
        {agentTypes.map((agentType) => {
          const profiles = getProfilesForAgentType(agentType.id);
          const isExpanded = expandedAgentTypes.has(agentType.id);
          const isRunnerHostDefault = runnerHostDefaults.has(agentType.id);

          return (
            <AgentTypeItem
              key={agentType.id}
              agentType={agentType}
              profiles={profiles}
              isExpanded={isExpanded}
              isRunnerHostDefault={isRunnerHostDefault}
              onToggle={() => toggleAgentType(agentType.id)}
              onSetRunnerHostDefault={() => handleSetRunnerHostDefault(agentType.id)}
              onSetDefault={handleSetDefault}
              onEdit={handleOpenEditDialog}
              onDelete={handleDeleteWithConfirm}
              onAdd={() => handleOpenAddDialog(agentType.id)}
              t={t}
            />
          );
        })}

        {agentTypes.length === 0 && (
          <div className="text-center py-12 text-muted-foreground">
            {t("settings.agentCredentials.noAgentTypes")}
          </div>
        )}
      </div>

      {/* Add/Edit Dialog */}
      <CredentialProfileDialog
        open={showDialog}
        onOpenChange={setShowDialog}
        editingProfile={editingProfile}
        onSubmit={handleDialogSubmit}
        t={t}
      />

      {/* Confirm Dialog */}
      <ConfirmDialog {...dialogProps} />
    </div>
  );
}

export default AgentCredentialsSettings;
