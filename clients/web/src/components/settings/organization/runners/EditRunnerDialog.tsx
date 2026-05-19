"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Runner } from "@/stores/runner";
import { getLocalizedErrorMessage } from "@/lib/api/errors";
import type { TranslationFn } from "../GeneralSettings";

interface EditRunnerDialogProps {
  runner: Runner;
  onClose: () => void;
  onSave: (id: number, data: { description?: string; max_concurrent_pods?: number; is_enabled?: boolean }) => Promise<void>;
  t: TranslationFn;
}

export function EditRunnerDialog({
  runner,
  onClose,
  onSave,
  t,
}: EditRunnerDialogProps) {
  const [description, setDescription] = useState(runner.description || "");
  const [maxPods, setMaxPods] = useState(runner.max_concurrent_pods.toString());
  const [isEnabled, setIsEnabled] = useState(runner.is_enabled);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSave = async () => {
    setSaving(true);
    setError(null);
    try {
      await onSave(runner.id, {
        description: description || undefined,
        max_concurrent_pods: parseInt(maxPods, 10),
        is_enabled: isEnabled,
      });
    } catch (err) {
      console.error("Failed to save runner:", err);
      setError(getLocalizedErrorMessage(err, t, t("settings.runnersSection.editDialog.saveFailed") || "Failed to save"));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-background border border-border rounded-lg p-6 w-full max-w-md">
        <h3 className="text-lg font-semibold mb-4">
          {t("settings.runnersSection.editDialog.title")}
        </h3>
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-2">
              {t("settings.runnersSection.editDialog.nodeIdLabel")}
            </label>
            <Input value={runner.node_id} disabled />
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">
              {t("settings.runnersSection.editDialog.descriptionLabel")}
            </label>
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
            <label className="text-sm font-medium">
              {t("settings.runnersSection.editDialog.enabledLabel")}
            </label>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                className="sr-only peer"
                checked={isEnabled}
                onChange={(e) => setIsEnabled(e.target.checked)}
              />
              <div className="w-11 h-6 bg-muted peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-transparent after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-background after:border-border after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"></div>
            </label>
          </div>
        </div>
        {error && (
          <div className="text-sm text-destructive bg-destructive/10 border border-destructive/20 rounded-md p-3 mt-4">
            {error}
          </div>
        )}
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

export default EditRunnerDialog;
