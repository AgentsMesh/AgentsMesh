import { useState, useEffect, useMemo } from "react";
import {
  RunnerData,
  AgentData,
  RepositoryData,
} from "@/lib/api";
import { getRunnerService, getAgentService } from "@/lib/wasm-core";
import { useRepositories, useRepositoryStore } from "@/stores/repository";

export interface PodCreationData {
  runners: RunnerData[];
  agents: AgentData[];
  repositories: RepositoryData[];
  loading: boolean;
  error: string | null;
  // Runner selection state
  selectedRunner: RunnerData | null;
  setSelectedRunnerId: (id: number | null) => void;
  // Agents filtered by selected runner's available agents
  availableAgents: AgentData[];
}

/**
 * Hook to load data required for pod creation (runners, agents, repositories)
 * Agents are filtered based on the selected runner's available agents
 * Only loads when enabled is true (e.g., when modal is open)
 */
export function usePodCreationData(enabled: boolean): PodCreationData {
  const [runners, setRunners] = useState<RunnerData[]>([]);
  const [agents, setAgents] = useState<AgentData[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedRunnerId, setSelectedRunnerId] = useState<number | null>(null);

  // Repos come from the shared store; trigger a fetch when the consumer enables.
  const repositories = useRepositories();
  const fetchRepositories = useRepositoryStore((s) => s.fetchRepositories);
  useEffect(() => {
    if (enabled) fetchRepositories();
  }, [enabled, fetchRepositories]);

  // Load runners and agents (repos handled by store)
  useEffect(() => {
    if (!enabled) return;

    let cancelled = false;

    const loadData = async () => {
      setLoading(true);
      setError(null);
      try {
        const [runnersRes, agentsRes] = await Promise.allSettled([
          getRunnerService().fetch_runners(null).then((j: string) => JSON.parse(j)),
          getAgentService().list_agents().then((j: string) => JSON.parse(j)),
        ]);

        if (cancelled) return;

        if (runnersRes.status === "fulfilled") {
          // Only online runners
          const allRunners: RunnerData[] = runnersRes.value.runners || [];
          const onlineRunners = allRunners.filter((r: RunnerData) => r.status === "online");
          setRunners(onlineRunners);
        }
        if (agentsRes.status === "fulfilled") {
          const res = agentsRes.value;
          const agentList = [...(res.builtin_agents || []), ...(res.custom_agents || []), ...(res.agents || [])];
          setAgents(agentList);
        }
      } catch (err) {
        if (cancelled) return;
        const message = err instanceof Error ? err.message : "Failed to load data";
        setError(message);
        console.error("Failed to load pod creation data:", err);
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    loadData();

    return () => {
      cancelled = true;
    };
  }, [enabled]);

  // Reset selected runner when modal closes
  useEffect(() => {
    if (!enabled) {
      setSelectedRunnerId(null);
    }
  }, [enabled]);

  // Get selected runner object
  const selectedRunner = useMemo(() => {
    if (!selectedRunnerId) return null;
    return runners.find(r => r.id === selectedRunnerId) || null;
  }, [runners, selectedRunnerId]);

  // Filter agents based on selected runner's available agents
  // When no runner is manually selected: union of all online runners' available agents
  // When runner is manually selected: filter by that runner's available agents
  const availableAgents = useMemo((): AgentData[] => {
    if (selectedRunner?.available_agents?.length) {
      return agents.filter(agent => selectedRunner.available_agents!.includes(agent.slug));
    }

    // No runner selected: show union of all online runners' available agents
    const allSlugs = new Set(runners.flatMap(r => r.available_agents || []));
    if (allSlugs.size === 0) return [];
    return agents.filter(agent => allSlugs.has(agent.slug));
  }, [selectedRunner, runners, agents]);

  return {
    runners,
    agents,
    repositories,
    loading,
    error,
    selectedRunner,
    setSelectedRunnerId,
    availableAgents,
  };
}
