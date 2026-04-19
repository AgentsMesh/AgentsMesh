"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { FormField } from "@/components/ui/form-field";
import { Share2 } from "lucide-react";
import { ShareDialog } from "@/components/shared/ShareDialog";
import type { RunnerData } from "@/lib/api/runnerTypes";
import { getRunnerService } from "@/lib/wasm-core";
import { getLocalizedErrorMessage } from "@/lib/api/errors";
import { getShortPodKey } from "@/lib/pod-display-name";

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
  const [visibility, setVisibility] = useState<string>(runner.visibility || "organization");
  const [tags, setTags] = useState<string[]>(runner.tags || []);
  const [tagInput, setTagInput] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [shareOpen, setShareOpen] = useState(false);

  const handleUpdate = async () => {
    setLoading(true);
    setError(null);
    try {
      await getRunnerService().update_runner(BigInt(runner.id), JSON.stringify({
        description: description || undefined,
        max_concurrent_pods: maxPods,
        visibility,
        tags,
      }));
      onUpdated();
    } catch (err) {
      console.error("Failed to update runner:", err);
      setError(getLocalizedErrorMessage(err, t, t("runners.configModal.saveFailed") || "Failed to save"));
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
          <FormField label={t("runners.configModal.nodeIdLabel")}>
            <code className="block w-full p-3 bg-muted rounded text-sm">
              {runner.node_id}
            </code>
          </FormField>

          <FormField
            label={t("runners.configModal.descriptionLabel")}
            htmlFor="runner-description"
          >
            <Input
              id="runner-description"
              placeholder={t("runners.configModal.descriptionPlaceholder")}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </FormField>

          <FormField
            label={t("runners.configModal.maxPodsLabel")}
            htmlFor="runner-max-pods"
          >
            <Input
              id="runner-max-pods"
              type="number"
              value={maxPods}
              onChange={(e) => setMaxPods(parseInt(e.target.value) || 1)}
              min={1}
              max={100}
            />
          </FormField>

          <FormField
            label={t("runners.configModal.visibilityLabel")}
            htmlFor="runner-visibility"
          >
            <select
              id="runner-visibility"
              className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
              value={visibility}
              onChange={(e) => setVisibility(e.target.value)}
            >
              <option value="organization">{t("runners.configModal.visibilityOrganization")}</option>
              <option value="private">{t("runners.configModal.visibilityPrivate")}</option>
            </select>
          </FormField>

          {visibility === "private" && (
            <Button variant="outline" size="sm" className="w-full" onClick={() => setShareOpen(true)}>
              <Share2 className="w-3 h-3 mr-1" /> {t("share.share")}
            </Button>
          )}

          <FormField
            label={t("runners.configModal.tagsLabel") || "Tags"}
            htmlFor="runner-tags"
          >
            <div className="flex flex-wrap gap-1.5 mb-2">
              {tags.map((tag) => (
                <span
                  key={tag}
                  className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"
                >
                  {tag}
                  <button
                    type="button"
                    className="hover:text-destructive"
                    onClick={() => setTags(tags.filter((t) => t !== tag))}
                  >
                    &times;
                  </button>
                </span>
              ))}
            </div>
            <Input
              id="runner-tags"
              placeholder={t("runners.configModal.tagsPlaceholder") || "Add a tag and press Enter..."}
              value={tagInput}
              onChange={(e) => setTagInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  const val = tagInput.trim().toLowerCase();
                  if (val && !tags.includes(val)) {
                    setTags([...tags, val]);
                  }
                  setTagInput("");
                }
              }}
            />
          </FormField>

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
                    <code>{getShortPodKey(pod.pod_key)}</code>
                    <span className="text-muted-foreground">{pod.status}</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {error && (
            <div className="text-sm text-destructive bg-destructive/10 border border-destructive/20 rounded-md p-3">
              {error}
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
      <ShareDialog
        open={shareOpen}
        onOpenChange={setShareOpen}
        resourceType="runner"
        resourceId={String(runner.id)}
      />
    </div>
  );
}
