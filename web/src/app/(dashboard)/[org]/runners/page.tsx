"use client";

import { useState, useEffect } from "react";
import { runnerApi, type RunnerData, type RegistrationToken } from "@/lib/api";
import { Button } from "@/components/ui/button";
import {
  Server,
  Plus,
  Copy,
  Trash2,
  RefreshCw,
  Settings2,
  Power,
  PowerOff,
  AlertCircle,
  CheckCircle,
  Clock,
  Cpu,
  HardDrive,
  Activity,
} from "lucide-react";
import { cn } from "@/lib/utils";

export default function RunnersPage() {
  const [runners, setRunners] = useState<RunnerData[]>([]);
  const [tokens, setTokens] = useState<RegistrationToken[]>([]);
  const [loading, setLoading] = useState(true);
  const [showTokenModal, setShowTokenModal] = useState(false);
  const [selectedRunner, setSelectedRunner] = useState<RunnerData | null>(null);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      const [runnersRes, tokensRes] = await Promise.all([
        runnerApi.list(),
        runnerApi.listTokens(),
      ]);
      setRunners(runnersRes.runners || []);
      setTokens(tokensRes.tokens || []);
    } catch (error) {
      console.error("Failed to load data:", error);
    } finally {
      setLoading(false);
    }
  };

  const getStatusIcon = (status: RunnerData["status"]) => {
    switch (status) {
      case "online":
        return <CheckCircle className="w-4 h-4 text-green-500" />;
      case "offline":
        return <PowerOff className="w-4 h-4 text-gray-500" />;
      case "busy":
        return <Activity className="w-4 h-4 text-yellow-500" />;
      case "maintenance":
        return <AlertCircle className="w-4 h-4 text-orange-500" />;
      default:
        return <Clock className="w-4 h-4 text-gray-400" />;
    }
  };

  const getStatusColor = (status: RunnerData["status"]) => {
    switch (status) {
      case "online":
        return "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400";
      case "offline":
        return "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400";
      case "busy":
        return "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400";
      case "maintenance":
        return "bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400";
      default:
        return "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400";
    }
  };

  const handleToggleEnabled = async (runner: RunnerData) => {
    try {
      await runnerApi.update(runner.id, { is_enabled: !runner.is_enabled });
      loadData();
    } catch (error) {
      console.error("Failed to update runner:", error);
    }
  };

  const handleDeleteRunner = async (runner: RunnerData) => {
    if (!confirm(`Are you sure you want to delete runner "${runner.node_id}"?`)) {
      return;
    }
    try {
      await runnerApi.delete(runner.id);
      loadData();
    } catch (error) {
      console.error("Failed to delete runner:", error);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>
    );
  }

  const onlineCount = runners.filter((r) => r.status === "online").length;
  const totalPods = runners.reduce((sum, r) => sum + r.current_pods, 0);
  const totalCapacity = runners.reduce((sum, r) => sum + r.max_concurrent_pods, 0);

  return (
    <div className="p-4 md:p-6 space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-xl md:text-2xl font-bold text-foreground">Runners</h1>
          <p className="text-sm text-muted-foreground">
            Manage your compute resources
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={loadData}>
            <RefreshCw className="w-4 h-4 mr-2" />
            Refresh
          </Button>
          <Button onClick={() => setShowTokenModal(true)}>
            <Plus className="w-4 h-4 mr-2" />
            Add Runner
          </Button>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 md:gap-4">
        <StatCard
          title="Total Runners"
          value={runners.length}
          icon={<Server className="w-5 h-5" />}
        />
        <StatCard
          title="Online"
          value={onlineCount}
          icon={<Power className="w-5 h-5" />}
          variant="success"
        />
        <StatCard
          title="Active Pods"
          value={totalPods}
          icon={<Cpu className="w-5 h-5" />}
        />
        <StatCard
          title="Total Capacity"
          value={totalCapacity}
          icon={<HardDrive className="w-5 h-5" />}
        />
      </div>

      {/* Runners List */}
      <div className="space-y-4">
        <h2 className="text-lg font-semibold">Active Runners</h2>

        {/* Mobile: Card view */}
        <div className="block md:hidden space-y-3">
          {runners.map((runner) => (
            <div
              key={runner.id}
              className="p-4 border border-border rounded-lg bg-card"
            >
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-2">
                  {getStatusIcon(runner.status)}
                  <span className="font-medium truncate">{runner.node_id}</span>
                </div>
                <span
                  className={cn(
                    "px-2 py-1 text-xs rounded-full",
                    getStatusColor(runner.status)
                  )}
                >
                  {runner.status}
                </span>
              </div>

              <div className="space-y-2 text-sm text-muted-foreground mb-3">
                <div className="flex justify-between">
                  <span>Pods:</span>
                  <span>
                    {runner.current_pods} / {runner.max_concurrent_pods}
                  </span>
                </div>
                {runner.host_info && (
                  <>
                    <div className="flex justify-between">
                      <span>OS:</span>
                      <span>{runner.host_info.os || "-"}</span>
                    </div>
                    <div className="flex justify-between">
                      <span>CPU:</span>
                      <span>{runner.host_info.cpu_cores || "-"} cores</span>
                    </div>
                  </>
                )}
                <div className="flex justify-between">
                  <span>Version:</span>
                  <span>{runner.runner_version || "-"}</span>
                </div>
              </div>

              <div className="flex gap-2">
                <Button
                  size="sm"
                  variant="outline"
                  className="flex-1"
                  onClick={() => setSelectedRunner(runner)}
                >
                  <Settings2 className="w-4 h-4 mr-1" />
                  Configure
                </Button>
                <Button
                  size="sm"
                  variant={runner.is_enabled ? "outline" : "default"}
                  onClick={() => handleToggleEnabled(runner)}
                >
                  {runner.is_enabled ? (
                    <PowerOff className="w-4 h-4" />
                  ) : (
                    <Power className="w-4 h-4" />
                  )}
                </Button>
                <Button
                  size="sm"
                  variant="destructive"
                  onClick={() => handleDeleteRunner(runner)}
                >
                  <Trash2 className="w-4 h-4" />
                </Button>
              </div>
            </div>
          ))}
          {runners.length === 0 && (
            <div className="text-center py-8 text-muted-foreground border border-dashed border-border rounded-lg">
              <Server className="w-12 h-12 mx-auto mb-3 opacity-50" />
              <p>No runners registered.</p>
              <p className="text-sm mt-1">Add a runner to start deploying pods.</p>
            </div>
          )}
        </div>

        {/* Desktop: Table view */}
        <div className="hidden md:block border border-border rounded-lg overflow-hidden">
          <table className="w-full">
            <thead className="bg-muted">
              <tr>
                <th className="px-4 py-3 text-left text-sm font-medium">Runner</th>
                <th className="px-4 py-3 text-left text-sm font-medium">Status</th>
                <th className="px-4 py-3 text-left text-sm font-medium">Pods</th>
                <th className="px-4 py-3 text-left text-sm font-medium">Host Info</th>
                <th className="px-4 py-3 text-left text-sm font-medium">Version</th>
                <th className="px-4 py-3 text-right text-sm font-medium">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {runners.map((runner) => (
                <tr key={runner.id} className="hover:bg-muted/50">
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      {getStatusIcon(runner.status)}
                      <code className="text-sm bg-muted px-2 py-1 rounded">
                        {runner.node_id}
                      </code>
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <span
                      className={cn(
                        "px-2 py-1 text-xs rounded-full",
                        getStatusColor(runner.status)
                      )}
                    >
                      {runner.status}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">
                    {runner.current_pods} / {runner.max_concurrent_pods}
                  </td>
                  <td className="px-4 py-3 text-muted-foreground text-sm">
                    {runner.host_info ? (
                      <span>
                        {runner.host_info.os} · {runner.host_info.cpu_cores} cores
                      </span>
                    ) : (
                      "-"
                    )}
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">
                    {runner.runner_version || "-"}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <Button
                      size="sm"
                      variant="outline"
                      className="mr-2"
                      onClick={() => setSelectedRunner(runner)}
                    >
                      Configure
                    </Button>
                    <Button
                      size="sm"
                      variant={runner.is_enabled ? "outline" : "default"}
                      className="mr-2"
                      onClick={() => handleToggleEnabled(runner)}
                    >
                      {runner.is_enabled ? "Disable" : "Enable"}
                    </Button>
                    <Button
                      size="sm"
                      variant="destructive"
                      onClick={() => handleDeleteRunner(runner)}
                    >
                      Delete
                    </Button>
                  </td>
                </tr>
              ))}
              {runners.length === 0 && (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-muted-foreground">
                    No runners registered. Add a runner to start deploying pods.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Registration Tokens Section */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">Registration Tokens</h2>
          <Button size="sm" variant="outline" onClick={() => setShowTokenModal(true)}>
            <Plus className="w-4 h-4 mr-2" />
            New Token
          </Button>
        </div>

        <div className="border border-border rounded-lg overflow-hidden">
          <table className="w-full">
            <thead className="bg-muted">
              <tr>
                <th className="px-4 py-3 text-left text-sm font-medium">Description</th>
                <th className="px-4 py-3 text-left text-sm font-medium">Usage</th>
                <th className="px-4 py-3 text-left text-sm font-medium">Status</th>
                <th className="px-4 py-3 text-left text-sm font-medium">Created</th>
                <th className="px-4 py-3 text-right text-sm font-medium">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {tokens.map((token) => (
                <tr key={token.id} className="hover:bg-muted/50">
                  <td className="px-4 py-3">{token.description || "No description"}</td>
                  <td className="px-4 py-3 text-muted-foreground">
                    {token.used_count} / {token.max_uses || "∞"}
                  </td>
                  <td className="px-4 py-3">
                    <span
                      className={cn(
                        "px-2 py-1 text-xs rounded-full",
                        token.is_active
                          ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"
                          : "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400"
                      )}
                    >
                      {token.is_active ? "Active" : "Revoked"}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">
                    {new Date(token.created_at).toLocaleDateString()}
                  </td>
                  <td className="px-4 py-3 text-right">
                    {token.is_active && (
                      <Button
                        size="sm"
                        variant="destructive"
                        onClick={async () => {
                          await runnerApi.revokeToken(token.id);
                          loadData();
                        }}
                      >
                        Revoke
                      </Button>
                    )}
                  </td>
                </tr>
              ))}
              {tokens.length === 0 && (
                <tr>
                  <td colSpan={5} className="px-4 py-8 text-center text-muted-foreground">
                    No registration tokens. Create one to add runners.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Token Modal */}
      {showTokenModal && (
        <CreateTokenModal
          onClose={() => setShowTokenModal(false)}
          onCreated={() => {
            setShowTokenModal(false);
            loadData();
          }}
        />
      )}

      {/* Runner Config Modal */}
      {selectedRunner && (
        <RunnerConfigModal
          runner={selectedRunner}
          onClose={() => setSelectedRunner(null)}
          onUpdated={() => {
            setSelectedRunner(null);
            loadData();
          }}
        />
      )}
    </div>
  );
}

function StatCard({
  title,
  value,
  icon,
  variant,
}: {
  title: string;
  value: number;
  icon: React.ReactNode;
  variant?: "success" | "warning" | "error";
}) {
  return (
    <div className="p-3 md:p-4 border border-border rounded-lg bg-card">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-xs md:text-sm text-muted-foreground">{title}</p>
          <p className="text-xl md:text-2xl font-bold">{value}</p>
        </div>
        <div
          className={cn(
            "w-8 h-8 md:w-10 md:h-10 rounded-lg flex items-center justify-center",
            variant === "success"
              ? "bg-green-500/10 text-green-500"
              : variant === "warning"
                ? "bg-yellow-500/10 text-yellow-500"
                : variant === "error"
                  ? "bg-red-500/10 text-red-500"
                  : "bg-primary/10 text-primary"
          )}
        >
          {icon}
        </div>
      </div>
    </div>
  );
}

function CreateTokenModal({
  onClose,
  onCreated,
}: {
  onClose: () => void;
  onCreated: () => void;
}) {
  const [description, setDescription] = useState("");
  const [maxUses, setMaxUses] = useState<number | undefined>(undefined);
  const [loading, setLoading] = useState(false);
  const [generatedToken, setGeneratedToken] = useState<string | null>(null);

  const handleCreate = async () => {
    setLoading(true);
    try {
      const res = await runnerApi.createToken(description || undefined, maxUses);
      setGeneratedToken(res.token);
    } catch (error) {
      console.error("Failed to create token:", error);
    } finally {
      setLoading(false);
    }
  };

  const copyToken = () => {
    if (generatedToken) {
      navigator.clipboard.writeText(generatedToken);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-background border border-border rounded-lg w-full max-w-md p-4 md:p-6">
        <h2 className="text-lg md:text-xl font-semibold mb-4">
          {generatedToken ? "Token Created" : "Create Registration Token"}
        </h2>

        {generatedToken ? (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Copy this token and use it to register a new runner. This token will only be shown once.
            </p>
            <div className="flex gap-2">
              <code className="flex-1 p-3 bg-muted rounded text-sm break-all">
                {generatedToken}
              </code>
              <Button variant="outline" size="sm" onClick={copyToken}>
                <Copy className="w-4 h-4" />
              </Button>
            </div>
            <div className="flex justify-end">
              <Button onClick={onCreated}>Done</Button>
            </div>
          </div>
        ) : (
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium mb-2">
                Description (optional)
              </label>
              <input
                type="text"
                className="w-full px-3 py-2 border border-border rounded-md bg-background"
                placeholder="e.g., Production runner"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-2">
                Max Uses (optional)
              </label>
              <input
                type="number"
                className="w-full px-3 py-2 border border-border rounded-md bg-background"
                placeholder="Unlimited"
                value={maxUses || ""}
                onChange={(e) =>
                  setMaxUses(e.target.value ? parseInt(e.target.value) : undefined)
                }
                min={1}
              />
            </div>

            <div className="flex flex-col-reverse sm:flex-row justify-end gap-3 mt-6">
              <Button variant="outline" onClick={onClose}>
                Cancel
              </Button>
              <Button onClick={handleCreate} disabled={loading}>
                {loading ? "Creating..." : "Create Token"}
              </Button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

function RunnerConfigModal({
  runner,
  onClose,
  onUpdated,
}: {
  runner: RunnerData;
  onClose: () => void;
  onUpdated: () => void;
}) {
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
          Configure Runner
        </h2>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-2">Node ID</label>
            <code className="block w-full p-3 bg-muted rounded text-sm">
              {runner.node_id}
            </code>
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">Description</label>
            <input
              type="text"
              className="w-full px-3 py-2 border border-border rounded-md bg-background"
              placeholder="Add a description..."
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">
              Max Concurrent Pods
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
                Active Pods ({runner.active_pods.length})
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
              Cancel
            </Button>
            <Button onClick={handleUpdate} disabled={loading}>
              {loading ? "Saving..." : "Save Changes"}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
