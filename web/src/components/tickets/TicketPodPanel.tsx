"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { ticketApi, runnerApi } from "@/lib/api/client";
import { getPodStatusInfo, getAgentStatusInfo } from "@/stores/devmesh";
import { useWorkspaceStore } from "@/stores/workspace";
import { useAuthStore } from "@/stores/auth";
import { Play, ExternalLink, Terminal } from "lucide-react";

interface TicketPod {
  pod_key: string;
  status: string;
  agent_status: string;
  model?: string;
  started_at?: string;
  runner_id: number;
  created_by_id: number;
}

interface Runner {
  id: number;
  node_id: string;
  status: string;
  current_pods: number;
  max_concurrent_pods?: number;
}

interface TicketPodPanelProps {
  ticketIdentifier: string;
  ticketTitle: string;
  onPodCreated?: () => void;
}

export default function TicketPodPanel({
  ticketIdentifier,
  ticketTitle,
  onPodCreated,
}: TicketPodPanelProps) {
  const [pods, setPods] = useState<TicketPod[]>([]);
  const [runners, setRunners] = useState<Runner[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Create form state
  const [selectedRunner, setSelectedRunner] = useState<number | null>(null);
  const [initialPrompt, setInitialPrompt] = useState("");
  const [model, setModel] = useState("claude-sonnet-4-20250514");
  const [permissionMode, setPermissionMode] = useState("default");

  const fetchPods = useCallback(async () => {
    try {
      const response = await ticketApi.getPods(ticketIdentifier);
      setPods(response.pods || []);
    } catch (err: any) {
      console.error("Failed to fetch pods:", err);
    }
  }, [ticketIdentifier]);

  const fetchRunners = useCallback(async () => {
    try {
      const response = await runnerApi.list();
      setRunners(response.runners?.filter((r) => r.status === "online") || []);
    } catch (err: any) {
      console.error("Failed to fetch runners:", err);
    }
  }, []);

  useEffect(() => {
    const loadData = async () => {
      setLoading(true);
      await Promise.all([fetchPods(), fetchRunners()]);
      setLoading(false);
    };
    loadData();
  }, [fetchPods, fetchRunners]);

  // Poll for pod updates
  useEffect(() => {
    const interval = setInterval(fetchPods, 5000);
    return () => clearInterval(interval);
  }, [fetchPods]);

  const handleCreatePod = async () => {
    if (!selectedRunner) {
      setError("Please select a runner");
      return;
    }

    setCreating(true);
    setError(null);

    try {
      await ticketApi.createPod(ticketIdentifier, {
        runner_id: selectedRunner,
        initial_prompt: initialPrompt || `Work on ticket: ${ticketTitle}`,
        model,
        permission_mode: permissionMode,
      });

      // Reset form
      setShowCreateForm(false);
      setSelectedRunner(null);
      setInitialPrompt("");
      setModel("claude-sonnet-4-20250514");
      setPermissionMode("default");

      // Refresh pods
      await fetchPods();
      onPodCreated?.();
    } catch (err: any) {
      setError(err.message || "Failed to create pod");
    } finally {
      setCreating(false);
    }
  };

  const activePods = pods.filter(
    (s) => s.status === "running" || s.status === "initializing"
  );
  const inactivePods = pods.filter(
    (s) => s.status !== "running" && s.status !== "initializing"
  );

  if (loading) {
    return (
      <div className="p-4 border border-border rounded-lg">
        <div className="flex items-center justify-center py-8">
          <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-primary"></div>
        </div>
      </div>
    );
  }

  return (
    <div className="border border-border rounded-lg">
      {/* Header */}
      <div className="px-4 py-3 border-b border-border flex items-center justify-between">
        <div className="flex items-center gap-2">
          <svg className="w-5 h-5 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
          </svg>
          <h3 className="font-medium">AgentPods</h3>
          {activePods.length > 0 && (
            <span className="px-2 py-0.5 text-xs rounded-full bg-green-100 text-green-700">
              {activePods.length} active
            </span>
          )}
        </div>
        <Button
          size="sm"
          onClick={() => setShowCreateForm(!showCreateForm)}
          disabled={runners.length === 0}
        >
          <svg className="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          New Pod
        </Button>
      </div>

      {/* Create Form */}
      {showCreateForm && (
        <div className="p-4 border-b border-border bg-muted/30">
          <h4 className="text-sm font-medium mb-3">Create New Pod</h4>
          <div className="space-y-3">
            {/* Runner Selection */}
            <div>
              <label className="block text-xs text-muted-foreground mb-1">Runner</label>
              <select
                className="w-full px-3 py-2 text-sm border border-border rounded-md bg-background"
                value={selectedRunner || ""}
                onChange={(e) => setSelectedRunner(Number(e.target.value) || null)}
              >
                <option value="">Select a runner...</option>
                {runners.map((runner) => (
                  <option key={runner.id} value={runner.id}>
                    {runner.node_id} ({runner.current_pods}{runner.max_concurrent_pods ? `/${runner.max_concurrent_pods}` : ""})
                  </option>
                ))}
              </select>
            </div>

            {/* Model Selection */}
            <div>
              <label className="block text-xs text-muted-foreground mb-1">Model</label>
              <select
                className="w-full px-3 py-2 text-sm border border-border rounded-md bg-background"
                value={model}
                onChange={(e) => setModel(e.target.value)}
              >
                <option value="claude-sonnet-4-20250514">Claude Sonnet 4</option>
                <option value="claude-opus-4-20250514">Claude Opus 4</option>
                <option value="claude-3-5-sonnet-20241022">Claude 3.5 Sonnet</option>
              </select>
            </div>

            {/* Permission Mode */}
            <div>
              <label className="block text-xs text-muted-foreground mb-1">Permission Mode</label>
              <select
                className="w-full px-3 py-2 text-sm border border-border rounded-md bg-background"
                value={permissionMode}
                onChange={(e) => setPermissionMode(e.target.value)}
              >
                <option value="default">Default</option>
                <option value="plan">Plan Mode</option>
                <option value="dangerously-skip-permissions">Auto-approve (Dangerous)</option>
              </select>
            </div>

            {/* Initial Prompt */}
            <div>
              <label className="block text-xs text-muted-foreground mb-1">
                Initial Prompt (optional)
              </label>
              <textarea
                className="w-full px-3 py-2 text-sm border border-border rounded-md bg-background resize-none"
                rows={3}
                placeholder={`Work on ticket: ${ticketTitle}`}
                value={initialPrompt}
                onChange={(e) => setInitialPrompt(e.target.value)}
              />
            </div>

            {/* Error */}
            {error && (
              <div className="text-sm text-destructive">{error}</div>
            )}

            {/* Actions */}
            <div className="flex justify-end gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setShowCreateForm(false)}
              >
                Cancel
              </Button>
              <Button
                size="sm"
                onClick={handleCreatePod}
                disabled={!selectedRunner || creating}
              >
                {creating ? "Creating..." : "Create Pod"}
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* Pods List */}
      <div className="divide-y divide-border">
        {/* Active Pods */}
        {activePods.map((pod) => (
          <PodItem key={pod.pod_key} pod={pod} ticketIdentifier={ticketIdentifier} />
        ))}

        {/* Inactive Pods (collapsed by default if there are active ones) */}
        {inactivePods.length > 0 && (
          <details className="group">
            <summary className="px-4 py-2 text-sm text-muted-foreground cursor-pointer hover:bg-muted/50">
              {inactivePods.length} previous pod{inactivePods.length !== 1 ? "s" : ""}
            </summary>
            <div className="divide-y divide-border border-t border-border">
              {inactivePods.map((pod) => (
                <PodItem key={pod.pod_key} pod={pod} ticketIdentifier={ticketIdentifier} />
              ))}
            </div>
          </details>
        )}

        {/* Empty State */}
        {pods.length === 0 && (
          <div className="px-4 py-8 text-center text-muted-foreground">
            <svg className="w-10 h-10 mx-auto mb-2 text-muted-foreground/50" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            <p className="text-sm">No pods for this ticket yet</p>
            {runners.length === 0 && (
              <p className="text-xs mt-1 text-yellow-600">
                No online runners available
              </p>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

interface PodItemProps {
  pod: TicketPod;
  ticketIdentifier: string;
}

function PodItem({ pod, ticketIdentifier }: PodItemProps) {
  const router = useRouter();
  const { currentOrg } = useAuthStore();
  const { addPane } = useWorkspaceStore();
  const statusInfo = getPodStatusInfo(pod.status);
  const agentInfo = getAgentStatusInfo(pod.agent_status);
  const isActive = pod.status === "running" || pod.status === "initializing";

  const handleConnect = () => {
    // Add to workspace and navigate
    addPane(pod.pod_key, `${ticketIdentifier} Pod`);
    router.push(`/${currentOrg?.slug}/workspace`);
  };

  const handleOpenInNewTab = () => {
    // Open pod detail in new tab
    window.open(`/${currentOrg?.slug}/workspace?pod=${pod.pod_key}`, "_blank");
  };

  return (
    <div className={`px-4 py-3 ${isActive ? "bg-green-50/50 dark:bg-green-900/10" : ""}`}>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          {/* Status Indicator */}
          <div
            className={`w-2 h-2 rounded-full ${
              pod.status === "running"
                ? "bg-green-500 animate-pulse"
                : pod.status === "initializing"
                ? "bg-yellow-500 animate-pulse"
                : pod.status === "failed"
                ? "bg-red-500"
                : "bg-gray-400"
            }`}
          />

          {/* Pod Info */}
          <div>
            <code className="text-xs font-mono text-muted-foreground">
              {pod.pod_key.substring(0, 12)}...
            </code>
            <div className="flex items-center gap-2 mt-0.5">
              <span
                className={`px-1.5 py-0.5 text-xs rounded ${statusInfo.bgColor} ${statusInfo.color}`}
              >
                {statusInfo.label}
              </span>
              {isActive && (
                <span className={`text-xs flex items-center gap-1 ${agentInfo.color}`}>
                  <span>{agentInfo.icon}</span>
                  {agentInfo.label}
                </span>
              )}
            </div>
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-2">
          {pod.model && (
            <span className="text-xs text-muted-foreground hidden sm:inline">{pod.model}</span>
          )}
          {isActive && (
            <>
              <Button size="sm" variant="outline" onClick={handleConnect}>
                <Terminal className="w-3.5 h-3.5 mr-1" />
                Connect
              </Button>
              <Button
                size="sm"
                variant="ghost"
                className="hidden sm:flex"
                onClick={handleOpenInNewTab}
                title="Open in new tab"
              >
                <ExternalLink className="w-3.5 h-3.5" />
              </Button>
            </>
          )}
        </div>
      </div>

      {/* Started At */}
      {pod.started_at && (
        <div className="mt-1 text-xs text-muted-foreground ml-5">
          Started: {new Date(pod.started_at).toLocaleString()}
        </div>
      )}
    </div>
  );
}
