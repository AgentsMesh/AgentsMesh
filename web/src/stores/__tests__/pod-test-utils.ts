import { vi } from "vitest";
import { usePodStore, Pod } from "../pod";

export const mockPod: Pod = {
  id: 1,
  pod_key: "pod-abc-123",
  status: "running",
  agent_status: "executing",
  created_at: "2024-01-01T00:00:00Z",
  runner: {
    id: 1,
    node_id: "runner-1",
    status: "online",
  },
};

export const mockPod2: Pod = {
  id: 2,
  pod_key: "pod-def-456",
  status: "running",
  agent_status: "waiting",
  created_at: "2024-01-02T00:00:00Z",
  runner: {
    id: 1,
    node_id: "runner-1",
    status: "online",
  },
};

export function resetPodStore() {
  vi.clearAllMocks();
  usePodStore.setState({
    pods: [],
    currentPod: null,
    loading: false,
    error: null,
    initProgress: {},
    podTimestamps: {},
    podTotal: 0,
    podHasMore: false,
    loadingMore: false,
    currentSidebarFilter: "mine",
  });
}
