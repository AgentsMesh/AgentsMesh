"use client";

import { useState, useEffect } from "react";
import { toast } from "sonner";
import { podApi, runnerApi, agentApi, repositoryApi, RepositoryData } from "@/lib/api/client";
import { useWorkspaceStore } from "@/stores/workspace";
import { WorkspaceManager } from "@/components/workspace";
import { Button } from "@/components/ui/button";
import { Terminal, Plus } from "lucide-react";

interface Pod {
  id: number;
  pod_key: string;
  status: string;
  agent_status: string;
  created_at: string;
  runner?: {
    node_id: string;
  };
}

interface Runner {
  id: number;
  node_id: string;
  status: string;
  current_pods: number;
  max_concurrent_pods?: number;
}

interface AgentType {
  id: number;
  slug: string;
  name: string;
  description?: string;
}

export default function WorkspacePage() {
  const { panes, addPane } = useWorkspaceStore();
  const [agentTypes, setAgentTypes] = useState<AgentType[]>([]);
  const [runners, setRunners] = useState<Runner[]>([]);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      // Load runners and agent types in parallel, but handle failures gracefully
      const [runnersRes, agentsRes] = await Promise.allSettled([
        runnerApi.list(),
        agentApi.listTypes(),
      ]);

      if (runnersRes.status === "fulfilled") {
        setRunners(runnersRes.value.runners || []);
      } else {
        console.error("Failed to load runners:", runnersRes.reason);
      }

      if (agentsRes.status === "fulfilled") {
        setAgentTypes(agentsRes.value.agent_types || []);
      } else {
        console.error("Failed to load agent types:", agentsRes.reason);
      }
    } catch (error) {
      console.error("Failed to load data:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleOpenPod = (podKey: string, title?: string) => {
    addPane(podKey, title || `Pod ${podKey.substring(0, 8)}`);
  };

  // Empty state when no terminals are open
  if (panes.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-full p-8">
        <Terminal className="w-16 h-16 mb-4 text-muted-foreground/30" />
        <h2 className="text-xl font-semibold mb-2">No terminals open</h2>
        <p className="text-muted-foreground text-center mb-6 max-w-md">
          Click on a pod in the sidebar to open its terminal, or create a new pod to get started.
        </p>
        <Button onClick={() => setShowCreateModal(true)}>
          <Plus className="w-4 h-4 mr-2" />
          Create New Pod
        </Button>

        {/* Create Modal */}
        {showCreateModal && (
          <CreatePodModal
            agentTypes={agentTypes}
            runners={runners.filter((r) => r.status === "online")}
            onClose={() => setShowCreateModal(false)}
            onCreated={(pod) => {
              setShowCreateModal(false);
              loadData();
              if (pod?.pod_key) {
                toast.info("Pod created! Waiting for it to start...", {
                  description: `Pod: ${pod.pod_key.substring(0, 8)}`,
                });
                handleOpenPod(pod.pod_key);
              }
            }}
          />
        )}
      </div>
    );
  }

  // Terminal workspace
  return (
    <div className="flex flex-col h-full">
      <WorkspaceManager className="flex-1" />

      {/* Create Modal */}
      {showCreateModal && (
        <CreatePodModal
          agentTypes={agentTypes}
          runners={runners.filter((r) => r.status === "online")}
          onClose={() => setShowCreateModal(false)}
          onCreated={(pod) => {
            setShowCreateModal(false);
            loadData();
            if (pod?.pod_key) {
              toast.info("Pod created! Waiting for it to start...", {
                description: `Pod: ${pod.pod_key.substring(0, 8)}`,
              });
              handleOpenPod(pod.pod_key);
            }
          }}
        />
      )}
    </div>
  );
}

function CreatePodModal({
  agentTypes,
  runners,
  onClose,
  onCreated,
}: {
  agentTypes: AgentType[];
  runners: Runner[];
  onClose: () => void;
  onCreated: (pod?: Pod) => void;
}) {
  const [selectedAgent, setSelectedAgent] = useState<number | null>(null);
  const [selectedRunner, setSelectedRunner] = useState<number | null>(null);
  const [prompt, setPrompt] = useState("");
  const [loading, setLoading] = useState(false);

  // Repository and Branch state
  const [repositories, setRepositories] = useState<RepositoryData[]>([]);
  const [selectedRepository, setSelectedRepository] = useState<number | null>(null);
  const [selectedBranch, setSelectedBranch] = useState<string>("");
  const [loadingRepos, setLoadingRepos] = useState(true);

  // Load repositories on mount
  useEffect(() => {
    const loadRepositories = async () => {
      try {
        const res = await repositoryApi.list();
        setRepositories(res.repositories || []);
      } catch (error) {
        console.error("Failed to load repositories:", error);
      } finally {
        setLoadingRepos(false);
      }
    };
    loadRepositories();
  }, []);

  // Auto-select default branch when repository is selected
  useEffect(() => {
    if (!selectedRepository) {
      setSelectedBranch("");
      return;
    }

    const repo = repositories.find((r) => r.id === selectedRepository);
    if (repo?.default_branch) {
      setSelectedBranch(repo.default_branch);
    }
  }, [selectedRepository, repositories]);

  const handleCreate = async () => {
    if (!selectedAgent || !selectedRunner) return;

    setLoading(true);
    try {
      const response = await podApi.create({
        agent_type_id: selectedAgent,
        runner_id: selectedRunner,
        repository_id: selectedRepository || undefined,
        branch_name: selectedBranch || undefined,
        initial_prompt: prompt,
      });
      onCreated(response.pod);
    } catch (error) {
      console.error("Failed to create pod:", error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-background border border-border rounded-lg w-full max-w-md p-4 md:p-6 max-h-[90vh] overflow-y-auto">
        <h2 className="text-lg md:text-xl font-semibold mb-4">Create New Pod</h2>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-2">Agent Type</label>
            <select
              className="w-full px-3 py-2 border border-border rounded-md bg-background"
              value={selectedAgent || ""}
              onChange={(e) => setSelectedAgent(Number(e.target.value))}
            >
              <option value="">Select an agent...</option>
              {agentTypes.map((agent) => (
                <option key={agent.id} value={agent.id}>
                  {agent.name}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium mb-2">Runner</label>
            <select
              className="w-full px-3 py-2 border border-border rounded-md bg-background"
              value={selectedRunner || ""}
              onChange={(e) => setSelectedRunner(Number(e.target.value))}
            >
              <option value="">Select a runner...</option>
              {runners.map((runner) => (
                <option key={runner.id} value={runner.id}>
                  {runner.node_id} ({runner.current_pods}/{runner.max_concurrent_pods})
                </option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium mb-2">Repository (optional)</label>
            <select
              className="w-full px-3 py-2 border border-border rounded-md bg-background"
              value={selectedRepository || ""}
              onChange={(e) => setSelectedRepository(e.target.value ? Number(e.target.value) : null)}
              disabled={loadingRepos}
            >
              <option value="">
                {loadingRepos ? "Loading..." : "Select a repository..."}
              </option>
              {repositories.map((repo) => (
                <option key={repo.id} value={repo.id}>
                  {repo.full_path}
                </option>
              ))}
            </select>
          </div>

          {selectedRepository && (
            <div>
              <label className="block text-sm font-medium mb-2">Branch</label>
              <input
                type="text"
                className="w-full px-3 py-2 border border-border rounded-md bg-background"
                placeholder="Enter branch name (e.g., main)"
                value={selectedBranch}
                onChange={(e) => setSelectedBranch(e.target.value)}
              />
            </div>
          )}

          <div>
            <label className="block text-sm font-medium mb-2">Initial Prompt (optional)</label>
            <textarea
              className="w-full px-3 py-2 border border-border rounded-md bg-background resize-none"
              rows={3}
              placeholder="Enter an initial prompt for the agent..."
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
            />
          </div>
        </div>

        <div className="flex flex-col-reverse sm:flex-row justify-end gap-3 mt-6">
          <Button variant="outline" onClick={onClose} className="w-full sm:w-auto">
            Cancel
          </Button>
          <Button
            onClick={handleCreate}
            disabled={!selectedAgent || !selectedRunner || loading}
            className="w-full sm:w-auto"
          >
            {loading ? "Creating..." : "Create Pod"}
          </Button>
        </div>
      </div>
    </div>
  );
}
