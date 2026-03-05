"use client";

import { useState, useEffect, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { ConfirmDialog, useConfirmDialog } from "@/components/ui/confirm-dialog";
import { useRunnerStore, Runner } from "@/stores/runner";
import { RunnerCard, TokenDialog, EditRunnerDialog } from "./runners";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { getLocalizedErrorMessage } from "@/lib/api/errors";
import type { TranslationFn } from "./GeneralSettings";

interface RunnersSettingsProps {
  t: TranslationFn;
}

export function RunnersSettings({ t }: RunnersSettingsProps) {
  const i18n = useTranslations();
  const runners = useRunnerStore((s) => s.runners);
  const loading = useRunnerStore((s) => s.loading);
  const error = useRunnerStore((s) => s.error);
  const fetchRunners = useRunnerStore((s) => s.fetchRunners);
  const updateRunner = useRunnerStore((s) => s.updateRunner);
  const deleteRunner = useRunnerStore((s) => s.deleteRunner);
  const createToken = useRunnerStore((s) => s.createToken);
  const clearError = useRunnerStore((s) => s.clearError);

  const [editingRunner, setEditingRunner] = useState<Runner | null>(null);
  const [generatedToken, setGeneratedToken] = useState<string | null>(null);

  useEffect(() => {
    fetchRunners();
  }, [fetchRunners]);

  const handleGenerateToken = async () => {
    try {
      const token = await createToken();
      setGeneratedToken(token);
    } catch (err) {
      console.error("Failed to generate token:", err);
      toast.error(getLocalizedErrorMessage(err, i18n, i18n("common.error")));
    }
  };

  return (
    <div className="space-y-6">
      {error && (
        <div className="bg-destructive/10 border border-destructive text-destructive px-4 py-3 rounded-lg flex items-center justify-between">
          <span>{error}</span>
          <button onClick={clearError} className="text-sm underline">
            {t("settings.members.dismiss")}
          </button>
        </div>
      )}

      <RunnersPanel
        runners={runners}
        loading={loading}
        onEdit={setEditingRunner}
        onDelete={deleteRunner}
        onGenerateToken={handleGenerateToken}
        t={t}
      />

      {editingRunner && (
        <EditRunnerDialog
          runner={editingRunner}
          onClose={() => setEditingRunner(null)}
          onSave={async (id, data) => {
            await updateRunner(id, data);
            setEditingRunner(null);
          }}
          t={t}
        />
      )}

      {generatedToken && (
        <TokenDialog
          token={generatedToken}
          onClose={() => setGeneratedToken(null)}
          onCopy={() => navigator.clipboard.writeText(generatedToken)}
          t={t}
        />
      )}
    </div>
  );
}

interface RunnersPanelProps {
  runners: Runner[];
  loading: boolean;
  onEdit: (runner: Runner) => void;
  onDelete: (id: number) => Promise<void>;
  onGenerateToken: () => void;
  t: TranslationFn;
}

function RunnersPanel({
  runners,
  loading,
  onEdit,
  onDelete,
  onGenerateToken,
  t,
}: RunnersPanelProps) {
  // Confirm dialog for delete
  const { dialogProps, confirm } = useConfirmDialog();

  const handleDeleteWithConfirm = useCallback(async (id: number) => {
    const confirmed = await confirm({
      title: t("settings.runnersSection.deleteDialog.title"),
      description: t("settings.runnersSection.deleteDialog.description"),
      variant: "destructive",
      confirmText: t("settings.runnersSection.deleteDialog.delete"),
      cancelText: t("settings.runnersSection.deleteDialog.cancel"),
    });
    if (confirmed) {
      try {
        await onDelete(id);
      } catch (err) {
        console.error("Failed to delete runner:", err);
      }
    }
  }, [confirm, onDelete, t]);

  const formatLastSeen = (dateString?: string) => {
    if (!dateString) return "Never";
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffSec = Math.floor(diffMs / 1000);

    if (diffSec < 60) return t("settings.runnersSection.justNow");
    if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m ago`;
    if (diffSec < 86400) return `${Math.floor(diffSec / 3600)}h ago`;
    return date.toLocaleDateString();
  };

  return (
    <div className="border border-border rounded-lg p-6">
      <div className="mb-4 flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold">{t("settings.runnersSection.title")}</h2>
          <p className="text-sm text-muted-foreground">
            {t("settings.runnersSection.description")}
          </p>
        </div>
        <Button variant="outline" onClick={onGenerateToken}>
          {t("settings.runnersSection.generateToken")}
        </Button>
      </div>

      {loading ? (
        <div className="text-center py-4 text-muted-foreground">{t("settings.runnersSection.loading")}</div>
      ) : runners.length === 0 ? (
        <div className="text-center py-8 text-muted-foreground">
          {t("settings.runnersSection.noRunners")}
        </div>
      ) : (
        <div className="space-y-3">
          {runners.map((runner) => (
            <RunnerCard
              key={runner.id}
              runner={runner}
              onEdit={onEdit}
              onDelete={() => handleDeleteWithConfirm(runner.id)}
              formatLastSeen={formatLastSeen}
              t={t}
            />
          ))}
        </div>
      )}

      {/* Confirm Delete Dialog */}
      <ConfirmDialog {...dialogProps} />
    </div>
  );
}
