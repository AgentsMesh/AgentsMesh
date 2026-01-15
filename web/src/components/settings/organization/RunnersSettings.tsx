"use client";

import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useRunnerStore, Runner, getRunnerStatusInfo } from "@/stores/runner";
import type { TranslationFn } from "./GeneralSettings";

interface RunnersSettingsProps {
  t: TranslationFn;
}

export function RunnersSettings({ t }: RunnersSettingsProps) {
  const {
    runners,
    loading,
    error,
    fetchRunners,
    updateRunner,
    deleteRunner,
    regenerateAuthToken,
    clearError,
  } = useRunnerStore();

  const [editingRunner, setEditingRunner] = useState<Runner | null>(null);

  useEffect(() => {
    fetchRunners();
  }, [fetchRunners]);

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
        onRegenerateToken={regenerateAuthToken}
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
    </div>
  );
}

function RunnersPanel({
  runners,
  loading,
  onEdit,
  onDelete,
  onRegenerateToken,
  t,
}: {
  runners: Runner[];
  loading: boolean;
  onEdit: (runner: Runner) => void;
  onDelete: (id: number) => Promise<void>;
  onRegenerateToken: (id: number) => Promise<string>;
  t: TranslationFn;
}) {
  const [confirmDelete, setConfirmDelete] = useState<number | null>(null);
  const [regeneratedToken, setRegeneratedToken] = useState<{ id: number; token: string } | null>(null);

  const handleDelete = async (id: number) => {
    try {
      await onDelete(id);
      setConfirmDelete(null);
    } catch (err) {
      console.error("Failed to delete runner:", err);
    }
  };

  const handleRegenerateToken = async (id: number) => {
    try {
      const token = await onRegenerateToken(id);
      setRegeneratedToken({ id, token });
    } catch (err) {
      console.error("Failed to regenerate token:", err);
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
  };

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
      <div className="mb-4">
        <h2 className="text-lg font-semibold">{t("settings.runnersSection.title")}</h2>
        <p className="text-sm text-muted-foreground">
          {t("settings.runnersSection.description")}
        </p>
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
              onDelete={() => setConfirmDelete(runner.id)}
              onRegenerateToken={() => handleRegenerateToken(runner.id)}
              formatLastSeen={formatLastSeen}
              t={t}
            />
          ))}
        </div>
      )}

      {/* Confirm Delete Dialog */}
      {confirmDelete !== null && (
        <ConfirmDeleteDialog
          onConfirm={() => handleDelete(confirmDelete)}
          onCancel={() => setConfirmDelete(null)}
          t={t}
        />
      )}

      {/* Regenerated Token Dialog */}
      {regeneratedToken && (
        <TokenDialog
          token={regeneratedToken.token}
          onClose={() => setRegeneratedToken(null)}
          onCopy={() => copyToClipboard(regeneratedToken.token)}
          t={t}
        />
      )}
    </div>
  );
}

function RunnerCard({
  runner,
  onEdit,
  onDelete,
  onRegenerateToken,
  formatLastSeen,
  t,
}: {
  runner: Runner;
  onEdit: (runner: Runner) => void;
  onDelete: () => void;
  onRegenerateToken: () => void;
  formatLastSeen: (dateString?: string) => string;
  t: TranslationFn;
}) {
  const statusInfo = getRunnerStatusInfo(runner.status as "online" | "offline" | "maintenance" | "busy");

  return (
    <div
      className={`p-4 border rounded-lg ${
        runner.is_enabled ? "border-border" : "border-border bg-muted/50"
      }`}
    >
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <span className="font-medium">{runner.node_id}</span>
            <span
              className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium ${statusInfo?.color}`}
            >
              <span className={`w-1.5 h-1.5 rounded-full ${statusInfo?.dotColor}`} />
              {statusInfo?.label}
            </span>
            {!runner.is_enabled && (
              <span className="text-xs bg-yellow-100 text-yellow-800 px-2 py-0.5 rounded">
                {t("settings.runnersSection.disabled")}
              </span>
            )}
          </div>
          {runner.description && (
            <p className="text-sm text-muted-foreground mt-1">
              {runner.description}
            </p>
          )}
          <div className="flex items-center gap-4 text-sm text-muted-foreground mt-2">
            <span>
              {t("settings.runnersSection.pods")} {runner.current_pods} / {runner.max_concurrent_pods}
            </span>
            {runner.runner_version && <span>v{runner.runner_version}</span>}
            <span>{t("settings.runnersSection.lastSeen")} {formatLastSeen(runner.last_heartbeat)}</span>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => onEdit(runner)}>
            {t("settings.runnersSection.edit")}
          </Button>
          <Button variant="outline" size="sm" onClick={onRegenerateToken}>
            {t("settings.runnersSection.regenerateToken")}
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="text-destructive hover:text-destructive"
            onClick={onDelete}
          >
            {t("settings.runnersSection.delete")}
          </Button>
        </div>
      </div>
    </div>
  );
}

function ConfirmDeleteDialog({
  onConfirm,
  onCancel,
  t,
}: {
  onConfirm: () => void;
  onCancel: () => void;
  t: TranslationFn;
}) {
  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-background border border-border rounded-lg p-6 w-full max-w-sm">
        <h3 className="text-lg font-semibold mb-2">{t("settings.runnersSection.deleteDialog.title")}</h3>
        <p className="text-muted-foreground mb-4">
          {t("settings.runnersSection.deleteDialog.description")}
        </p>
        <div className="flex gap-3">
          <Button variant="outline" className="flex-1" onClick={onCancel}>
            {t("settings.runnersSection.deleteDialog.cancel")}
          </Button>
          <Button variant="destructive" className="flex-1" onClick={onConfirm}>
            {t("settings.runnersSection.deleteDialog.delete")}
          </Button>
        </div>
      </div>
    </div>
  );
}

function TokenDialog({
  token,
  onClose,
  onCopy,
  t,
}: {
  token: string;
  onClose: () => void;
  onCopy: () => void;
  t: TranslationFn;
}) {
  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-background border border-border rounded-lg p-6 w-full max-w-md">
        <h3 className="text-lg font-semibold mb-4">{t("settings.runnersSection.tokenDialog.title")}</h3>
        <p className="text-sm text-muted-foreground mb-4">
          {t("settings.runnersSection.tokenDialog.description")}
        </p>
        <div className="bg-muted p-3 rounded-lg mb-4 flex items-center justify-between">
          <code className="text-sm break-all">{token}</code>
          <Button variant="ghost" size="sm" onClick={onCopy}>
            {t("settings.runnersSection.tokenDialog.copy")}
          </Button>
        </div>
        <Button className="w-full" onClick={onClose}>
          {t("settings.runnersSection.tokenDialog.done")}
        </Button>
      </div>
    </div>
  );
}

function EditRunnerDialog({
  runner,
  onClose,
  onSave,
  t,
}: {
  runner: Runner;
  onClose: () => void;
  onSave: (id: number, data: { description?: string; max_concurrent_pods?: number; is_enabled?: boolean }) => Promise<void>;
  t: TranslationFn;
}) {
  const [description, setDescription] = useState(runner.description || "");
  const [maxPods, setMaxPods] = useState(runner.max_concurrent_pods.toString());
  const [isEnabled, setIsEnabled] = useState(runner.is_enabled);
  const [saving, setSaving] = useState(false);

  const handleSave = async () => {
    setSaving(true);
    try {
      await onSave(runner.id, {
        description: description || undefined,
        max_concurrent_pods: parseInt(maxPods, 10),
        is_enabled: isEnabled,
      });
    } catch (err) {
      console.error("Failed to save runner:", err);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-background border border-border rounded-lg p-6 w-full max-w-md">
        <h3 className="text-lg font-semibold mb-4">{t("settings.runnersSection.editDialog.title")}</h3>
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-2">{t("settings.runnersSection.editDialog.nodeIdLabel")}</label>
            <Input value={runner.node_id} disabled />
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">{t("settings.runnersSection.editDialog.descriptionLabel")}</label>
            <Input
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder={t("settings.runnersSection.editDialog.descriptionPlaceholder")}
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">
              {t("settings.runnersSection.editDialog.maxPodsLabel")}
            </label>
            <Input
              type="number"
              value={maxPods}
              onChange={(e) => setMaxPods(e.target.value)}
              min="1"
            />
          </div>
          <div className="flex items-center justify-between">
            <label className="text-sm font-medium">{t("settings.runnersSection.editDialog.enabledLabel")}</label>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                className="sr-only peer"
                checked={isEnabled}
                onChange={(e) => setIsEnabled(e.target.checked)}
              />
              <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"></div>
            </label>
          </div>
        </div>
        <div className="flex gap-3 mt-6">
          <Button variant="outline" className="flex-1" onClick={onClose}>
            {t("settings.runnersSection.editDialog.cancel")}
          </Button>
          <Button className="flex-1" onClick={handleSave} disabled={saving}>
            {saving ? t("settings.runnersSection.editDialog.saving") : t("settings.runnersSection.editDialog.saveChanges")}
          </Button>
        </div>
      </div>
    </div>
  );
}
