"use client";

import type { Runner } from "@/stores/runner";

interface RunnerSelectProps {
  runners: Runner[];
  selectedRunnerId: number | null;
  onSelect: (runnerId: number | null) => void;
  error?: string;
  t: (key: string) => string;
}

/**
 * Runner selection dropdown component
 */
export function RunnerSelect({
  runners,
  selectedRunnerId,
  onSelect,
  error,
  t,
}: RunnerSelectProps) {
  return (
    <div>
      <label
        htmlFor="runner-select"
        className="block text-sm font-medium mb-2"
      >
        {t("ide.createPod.selectRunner")}
      </label>
      <select
        id="runner-select"
        className={`w-full px-3 py-2 border rounded-md bg-background ${
          error ? "border-destructive" : "border-border"
        }`}
        value={selectedRunnerId || ""}
        onChange={(e) =>
          onSelect(e.target.value ? Number(e.target.value) : null)
        }
        aria-required="true"
        aria-invalid={!!error}
        aria-describedby={
          error ? "runner-error" : runners.length === 0 ? "runner-help" : undefined
        }
      >
        <option value="">{t("ide.createPod.selectRunnerPlaceholder")}</option>
        {runners.map((runner) => (
          <option key={runner.id} value={runner.id}>
            {runner.node_id} ({runner.current_pods}/{runner.max_concurrent_pods})
          </option>
        ))}
      </select>
      {error && (
        <p id="runner-error" className="text-xs text-destructive mt-1">
          {error}
        </p>
      )}
      {!error && runners.length === 0 && (
        <p id="runner-help" className="text-xs text-muted-foreground mt-1">
          {t("ide.createPod.noRunnersAvailable")}
        </p>
      )}
    </div>
  );
}
