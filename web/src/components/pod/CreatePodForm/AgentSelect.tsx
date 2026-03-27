"use client";

import type { AgentData } from "@/lib/api";

interface AgentSelectProps {
  agents: AgentData[];
  selectedAgentSlug: string | null;
  onSelect: (agentSlug: string | null) => void;
  error?: string;
  t: (key: string) => string;
}

/**
 * Agent selection dropdown component
 */
export function AgentSelect({
  agents,
  selectedAgentSlug,
  onSelect,
  error,
  t,
}: AgentSelectProps) {
  if (agents.length === 0) {
    return (
      <div>
        <label className="block text-sm font-medium mb-2">
          {t("ide.createPod.selectAgent")}
        </label>
        <p className="text-sm text-muted-foreground py-2">
          {t("ide.createPod.noAgentsForRunner")}
        </p>
      </div>
    );
  }

  return (
    <div>
      <label
        htmlFor="agent-select"
        className="block text-sm font-medium mb-2"
      >
        {t("ide.createPod.selectAgent")}
      </label>
      <select
        id="agent-select"
        className={`w-full px-3 py-2 border rounded-md bg-background ${
          error ? "border-destructive" : "border-border"
        }`}
        value={selectedAgentSlug || ""}
        onChange={(e) =>
          onSelect(e.target.value || null)
        }
        aria-required="true"
        aria-invalid={!!error}
        aria-describedby={error ? "agent-error" : undefined}
      >
        <option value="">{t("ide.createPod.selectAgentPlaceholder")}</option>
        {agents.map((agent) => (
          <option key={agent.slug} value={agent.slug}>
            {agent.name}
          </option>
        ))}
      </select>
      {error && (
        <p id="agent-error" className="text-xs text-destructive mt-1">
          {error}
        </p>
      )}
    </div>
  );
}
