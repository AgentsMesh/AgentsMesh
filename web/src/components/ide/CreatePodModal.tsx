"use client";

import React, { useState, useEffect } from "react";
import { podApi, agentApi, runnerApi, repositoryApi, RepositoryData } from "@/lib/api/client";
import { Button } from "@/components/ui/button";

interface AgentType {
  id: number;
  slug: string;
  name: string;
  description?: string;
}

interface Runner {
  id: number;
  node_id: string;
  status: string;
  current_pods: number;
  max_concurrent_pods?: number;
}

interface Pod {
  id: number;
  pod_key: string;
  status: string;
  agent_status: string;
  created_at: string;
}

interface CreatePodModalProps {
  open: boolean;
  onClose: () => void;
  onCreated: (pod?: Pod) => void;
}

export function CreatePodModal({ open, onClose, onCreated }: CreatePodModalProps) {
  const [agentTypes, setAgentTypes] = useState<AgentType[]>([]);
  const [runners, setRunners] = useState<Runner[]>([]);
  const [repositories, setRepositories] = useState<RepositoryData[]>([]);

  const [selectedAgent, setSelectedAgent] = useState<number | null>(null);
  const [selectedRunner, setSelectedRunner] = useState<number | null>(null);
  const [selectedRepository, setSelectedRepository] = useState<number | null>(null);
  const [selectedBranch, setSelectedBranch] = useState<string>("");
  const [prompt, setPrompt] = useState("");

  const [loading, setLoading] = useState(false);
  const [loadingData, setLoadingData] = useState(true);

  // Load data when modal opens
  useEffect(() => {
    if (!open) return;

    const loadData = async () => {
      setLoadingData(true);
      try {
        const [runnersRes, agentsRes, reposRes] = await Promise.allSettled([
          runnerApi.list(),
          agentApi.listTypes(),
          repositoryApi.list(),
        ]);

        if (runnersRes.status === "fulfilled") {
          // Only online runners
          setRunners((runnersRes.value.runners || []).filter(r => r.status === "online"));
        }
        if (agentsRes.status === "fulfilled") {
          setAgentTypes(agentsRes.value.agent_types || []);
        }
        if (reposRes.status === "fulfilled") {
          setRepositories(reposRes.value.repositories || []);
        }
      } catch (error) {
        console.error("Failed to load data:", error);
      } finally {
        setLoadingData(false);
      }
    };

    loadData();
  }, [open]);

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

  // Reset form when modal closes
  useEffect(() => {
    if (!open) {
      setSelectedAgent(null);
      setSelectedRunner(null);
      setSelectedRepository(null);
      setSelectedBranch("");
      setPrompt("");
    }
  }, [open]);

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

  if (!open) return null;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-background border border-border rounded-lg w-full max-w-md p-4 md:p-6 max-h-[90vh] overflow-y-auto">
        <h2 className="text-lg md:text-xl font-semibold mb-4">Create New Pod</h2>

        {loadingData ? (
          <div className="flex items-center justify-center py-8">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
          </div>
        ) : (
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
              {runners.length === 0 && (
                <p className="text-xs text-muted-foreground mt-1">
                  No online runners available
                </p>
              )}
            </div>

            <div>
              <label className="block text-sm font-medium mb-2">Repository (optional)</label>
              <select
                className="w-full px-3 py-2 border border-border rounded-md bg-background"
                value={selectedRepository || ""}
                onChange={(e) => setSelectedRepository(e.target.value ? Number(e.target.value) : null)}
              >
                <option value="">Select a repository...</option>
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
        )}

        <div className="flex flex-col-reverse sm:flex-row justify-end gap-3 mt-6">
          <Button variant="outline" onClick={onClose} className="w-full sm:w-auto">
            Cancel
          </Button>
          <Button
            onClick={handleCreate}
            disabled={!selectedAgent || !selectedRunner || loading || loadingData}
            className="w-full sm:w-auto"
          >
            {loading ? "Creating..." : "Create Pod"}
          </Button>
        </div>
      </div>
    </div>
  );
}

export default CreatePodModal;
