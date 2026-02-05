"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { runnerApi, type RunnerData } from "@/lib/api";

interface RunnerConfigModalProps {
  t: (key: string, params?: Record<string, string | number>) => string;
  runner: RunnerData;
  onClose: () => void;
  onUpdated: () => void;
}

/**
 * RunnerConfigModal - Modal for configuring runner settings
 */
export function RunnerConfigModal({ t, runner, onClose, onUpdated }: RunnerConfigModalProps) {
  const [description, setDescription] = useState(runner.description || "");
  const [maxPods, setMaxPods] = useState(runner.max_concurrent_pods);
  const [loading, setLoading] = useState(false);

  const handleUpdate = async () => {
    setLoading(true);
    try {
      await runnerApi.update(runner.id, {
        description: description || undefined,
        max_concurrent_pods: maxPods,
      });
      onUpdated();
    } catch (error) {
      console.error("Failed to update runner:", error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-background border border-border rounded-lg w-full max-w-md p-4 md:p-6">
        <h2 className="text-lg md:text-xl font-semibold mb-4">
          {t("runners.configModal.title")}
        </h2>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-2">{t("runners.configModal.nodeIdLabel")}</label>
            <code className="block w-full p-3 bg-muted rounded text-sm">
              {runner.node_id}
            </code>
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">{t("runners.configModal.descriptionLabel")}</label>
            <input
              type="text"
              className="w-full px-3 py-2 border border-border rounded-md bg-background"
              placeholder={t("runners.configModal.descriptionPlaceholder")}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">
              {t("runners.configModal.maxPodsLabel")}
            </label>
            <input
              type="number"
              className="w-full px-3 py-2 border border-border rounded-md bg-background"
              value={maxPods}
              onChange={(e) => setMaxPods(parseInt(e.target.value) || 1)}
              min={1}
              max={100}
            />
          </div>

          {runner.active_pods && runner.active_pods.length > 0 && (
            <div>
              <label className="block text-sm font-medium mb-2">
                {t("runners.configModal.activePodsLabel", { count: runner.active_pods.length })}
              </label>
              <div className="space-y-2 max-h-32 overflow-y-auto">
                {runner.active_pods.map((pod) => (
                  <div
                    key={pod.pod_key}
                    className="flex items-center justify-between p-2 bg-muted rounded text-sm"
                  >
                    <code>{pod.pod_key.substring(0, 12)}...</code>
                    <span className="text-muted-foreground">{pod.status}</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          <div className="flex flex-col-reverse sm:flex-row justify-end gap-3 mt-6">
            <Button variant="outline" onClick={onClose}>
              {t("runners.configModal.cancel")}
            </Button>
            <Button onClick={handleUpdate} disabled={loading}>
              {loading ? t("runners.configModal.saving") : t("runners.configModal.save")}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
