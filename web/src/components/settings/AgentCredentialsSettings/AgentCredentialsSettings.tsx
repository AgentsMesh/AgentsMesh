"use client";

import { useState, useCallback } from "react";
import { CenteredSpinner } from "@/components/ui/spinner";
import { AlertMessage } from "@/components/ui/alert-message";
import { ConfirmDialog, useConfirmDialog } from "@/components/ui/confirm-dialog";
import { useTranslations } from "next-intl";
import type { CredentialProfileData } from "@/lib/api";
import { useAgentCredentials } from "./useAgentCredentials";
import { AgentItem } from "./AgentItem";
import { CredentialProfileDialog } from "./CredentialProfileDialog";
import type { CredentialFormData } from "./types";

/**
 * AgentCredentialsSettings - Manages credential profiles for all agents
 *
 * Displays a collapsible list of agents, each with RunnerHost as the
 * default option and custom credential profiles below.
 */
export function AgentCredentialsSettings() {
  const t = useTranslations();

  // Dialog state
  const [showDialog, setShowDialog] = useState(false);
  const [editingProfile, setEditingProfile] = useState<CredentialProfileData | null>(null);
  const [selectedAgentSlug, setSelectedAgentSlug] = useState<string | null>(null);

  // Use the custom hook for data and actions
  const {
    loading,
    error,
    success,
    agents,
    expandedAgents,
    runnerHostDefaults,
    credentialFieldsByAgent,
    toggleAgent,
    handleSetRunnerHostDefault,
    handleSetDefault,
    handleDelete,
    handleSaveProfile,
    getProfilesForAgent,
    setError,
    setSuccess,
  } = useAgentCredentials(t);

  // Confirm dialog for delete
  const { dialogProps, confirm } = useConfirmDialog();

  // Open add dialog
  const handleOpenAddDialog = useCallback((agentSlug: string) => {
    setSelectedAgentSlug(agentSlug);
    setEditingProfile(null);
    setShowDialog(true);
  }, []);

  // Open edit dialog
  const handleOpenEditDialog = useCallback((profile: CredentialProfileData) => {
    setSelectedAgentSlug(profile.agent_slug);
    setEditingProfile(profile);
    setShowDialog(true);
  }, []);

  // Handle dialog submit
  const handleDialogSubmit = useCallback(async (data: CredentialFormData) => {
    if (!selectedAgentSlug) {
      throw new Error("No agent selected");
    }
    await handleSaveProfile(selectedAgentSlug, data, editingProfile);
    setShowDialog(false);
  }, [selectedAgentSlug, editingProfile, handleSaveProfile]);

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

      {/* Agents List */}
      <div className="space-y-2">
        {agents.map((agent) => {
          const profiles = getProfilesForAgent(agent.slug);
          const isExpanded = expandedAgents.has(agent.slug);
          const isRunnerHostDefault = runnerHostDefaults.has(agent.slug);

          return (
            <AgentItem
              key={agent.slug}
              agent={agent}
              profiles={profiles}
              isExpanded={isExpanded}
              isRunnerHostDefault={isRunnerHostDefault}
              onToggle={() => toggleAgent(agent.slug)}
              onSetRunnerHostDefault={() => handleSetRunnerHostDefault(agent.slug)}
              onSetDefault={handleSetDefault}
              onEdit={handleOpenEditDialog}
              onDelete={handleDeleteWithConfirm}
              onAdd={() => handleOpenAddDialog(agent.slug)}
              t={t}
            />
          );
        })}

        {agents.length === 0 && (
          <div className="text-center py-12 text-muted-foreground">
            {t("settings.agentCredentials.noAgents")}
          </div>
        )}
      </div>

      {/* Add/Edit Dialog */}
      <CredentialProfileDialog
        open={showDialog}
        onOpenChange={setShowDialog}
        credentialFields={selectedAgentSlug ? credentialFieldsByAgent.get(selectedAgentSlug) || [] : []}
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
