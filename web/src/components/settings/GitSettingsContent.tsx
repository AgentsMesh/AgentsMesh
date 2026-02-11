"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { CenteredSpinner } from "@/components/ui/spinner";
import { RepositoryProviderData, CredentialType } from "@/lib/api";
import { AlertMessage } from "@/components/ui/alert-message";
import { useTranslations } from "next-intl";
import { Plus } from "lucide-react";
import { useConfirmDialog, ConfirmDialog } from "@/components/ui/confirm-dialog";
import {
  AddProviderDialog,
  EditProviderDialog,
  AddCredentialDialog,
  GitProviderCard,
  GitCredentialCard,
  DefaultCredentialSection,
  useGitSettings,
  getAllSelectableCredentials,
} from "./git";

/**
 * GitSettingsContent - Shared Git settings component
 * Used by both user settings page and organization settings page.
 *
 * Refactored to use:
 * - useGitSettings hook for data management
 * - Reusable card components for providers and credentials
 * - ConfirmDialog for delete confirmations
 */
export function GitSettingsContent() {
  const t = useTranslations();

  // Data and actions from hook
  const {
    data,
    loading,
    successMessage,
    errorMessage,
    setSuccessMessage,
    setErrorMessage,
    refetch,
    handleSetDefault,
    handleDeleteProvider,
    handleDeleteCredential,
    handleTestConnection,
  } = useGitSettings(t);

  // Dialog states
  const [showAddProviderDialog, setShowAddProviderDialog] = useState(false);
  const [showAddCredentialDialog, setShowAddCredentialDialog] = useState(false);
  const [editingProvider, setEditingProvider] = useState<RepositoryProviderData | null>(null);

  // Delete confirmation dialogs
  const deleteProviderDialog = useConfirmDialog({
    title: t("settings.gitSettings.providers.deleteDialog.title"),
    description: t("settings.gitSettings.providers.deleteDialog.description"),
    confirmText: t("common.delete"),
    variant: "destructive",
  });

  const deleteCredentialDialog = useConfirmDialog({
    title: t("settings.gitSettings.credentials.deleteDialog.title"),
    description: t("settings.gitSettings.credentials.deleteDialog.description"),
    confirmText: t("common.delete"),
    variant: "destructive",
  });

  // Handle provider delete with confirmation
  const onDeleteProvider = async (id: number) => {
    const confirmed = await deleteProviderDialog.confirm();
    if (confirmed) {
      await handleDeleteProvider(id);
    }
  };

  // Handle credential delete with confirmation
  const onDeleteCredential = async (id: number) => {
    const confirmed = await deleteCredentialDialog.confirm();
    if (confirmed) {
      await handleDeleteCredential(id);
    }
  };

  if (loading) {
    return (
      <div className="p-6 max-w-4xl mx-auto">
        <CenteredSpinner className="py-12" />
      </div>
    );
  }

  if (!data) {
    return null;
  }

  const selectableCredentials = getAllSelectableCredentials(data);
  const nonOAuthCredentials = data.credentials.filter(
    (c) => c.credential_type !== CredentialType.OAUTH
  );

  return (
    <div className="space-y-6">
      {/* Error/Success messages */}
      {errorMessage && (
        <AlertMessage
          type="error"
          message={errorMessage}
          onDismiss={() => setErrorMessage(null)}
          className="mb-4"
        />
      )}
      {successMessage && (
        <AlertMessage
          type="success"
          message={successMessage}
          onDismiss={() => setSuccessMessage(null)}
          className="mb-4"
        />
      )}

      {/* Section 1: Default Git Credential */}
      <DefaultCredentialSection
        credentials={selectableCredentials}
        onSetDefault={handleSetDefault}
        t={t}
      />

      {/* Section 2: Repository Providers */}
      <div className="border border-border rounded-lg p-6 mb-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h2 className="text-lg font-semibold">
              {t("settings.gitSettings.providers.title")}
            </h2>
            <p className="text-sm text-muted-foreground">
              {t("settings.gitSettings.providers.description")}
            </p>
          </div>
          <Button onClick={() => setShowAddProviderDialog(true)}>
            <Plus className="w-4 h-4 mr-2" />
            {t("settings.gitSettings.providers.add")}
          </Button>
        </div>

        {data.providers.length === 0 ? (
          <p className="text-sm text-muted-foreground py-4 text-center">
            {t("settings.gitSettings.providers.empty")}
          </p>
        ) : (
          <div className="space-y-3">
            {data.providers.map((provider) => (
              <GitProviderCard
                key={provider.id}
                provider={provider}
                onEdit={() => setEditingProvider(provider)}
                onDelete={() => onDeleteProvider(provider.id)}
                onTestConnection={() => handleTestConnection(provider.id)}
                t={t}
              />
            ))}
          </div>
        )}
      </div>

      {/* Section 3: Git Credentials */}
      <div className="border border-border rounded-lg p-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h2 className="text-lg font-semibold">
              {t("settings.gitSettings.credentials.title")}
            </h2>
            <p className="text-sm text-muted-foreground">
              {t("settings.gitSettings.credentials.description")}
            </p>
          </div>
          <Button onClick={() => setShowAddCredentialDialog(true)}>
            <Plus className="w-4 h-4 mr-2" />
            {t("settings.gitSettings.credentials.add")}
          </Button>
        </div>

        {nonOAuthCredentials.length === 0 ? (
          <p className="text-sm text-muted-foreground py-4 text-center">
            {t("settings.gitSettings.credentials.empty")}
          </p>
        ) : (
          <div className="space-y-3">
            {nonOAuthCredentials.map((cred) => (
              <GitCredentialCard
                key={cred.id}
                credential={cred}
                onDelete={() => onDeleteCredential(cred.id)}
                t={t}
              />
            ))}
          </div>
        )}
      </div>

      {/* Dialogs */}
      {showAddProviderDialog && (
        <AddProviderDialog
          onClose={() => setShowAddProviderDialog(false)}
          onSuccess={() => {
            setShowAddProviderDialog(false);
            refetch();
          }}
        />
      )}

      {editingProvider && (
        <EditProviderDialog
          provider={editingProvider}
          onClose={() => setEditingProvider(null)}
          onSuccess={() => {
            setEditingProvider(null);
            refetch();
          }}
        />
      )}

      {showAddCredentialDialog && (
        <AddCredentialDialog
          onClose={() => setShowAddCredentialDialog(false)}
          onSuccess={() => {
            setShowAddCredentialDialog(false);
            refetch();
          }}
        />
      )}

      {/* Delete confirmation dialogs */}
      <ConfirmDialog {...deleteProviderDialog.dialogProps} />
      <ConfirmDialog {...deleteCredentialDialog.dialogProps} />
    </div>
  );
}
