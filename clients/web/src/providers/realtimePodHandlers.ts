import { usePodStore } from "@/stores/pod";
import { useMeshStore } from "@/stores/mesh";
import { getPodState } from "@/lib/wasm-core";
import { useWorkspaceStore } from "@/stores/workspace";
import {
  type RealtimeEvent,
  decodeEventData,
  PodStatusChangedEventDataSchema,
  PodInitProgressEventDataSchema,
} from "@/lib/realtime";

// A created/restarting pod must enter the filtered sidebar; the server filter is
// authoritative, so refetch the active filter (silent = no spinner). Status-only
// changes never refetch — they patch in place.
const refreshSidebar = () => {
  const s = usePodStore.getState();
  s.fetchSidebarPods?.(s.currentSidebarFilter, { silent: true });
};
const bumpPods = () => usePodStore.setState((s) => ({ _tick: s._tick + 1 }));
const bumpMesh = () => useMeshStore.setState((s) => ({ _tick: s._tick + 1 }));
const refreshMeshTopology = () => useMeshStore.getState().fetchTopology?.();

export function handlePodEvent(event: RealtimeEvent) {
  switch (event.type) {
    case "pod:created":
    case "pod:restarting": {
      refreshSidebar();
      refreshMeshTopology();
      break;
    }
    case "pod:status_changed": {
      const data = decodeEventData(PodStatusChangedEventDataSchema, event.data);
      // not-cached pod needs the full entity from the server (dispatch only patches cached).
      if (!getPodState().get_pod_json(data.podKey)) {
        usePodStore.getState().fetchPod?.(data.podKey);
      } else {
        bumpPods();
      }
      // failed/error arrive here, not as pod:terminated (backend cmd/server/eventbus_pod.go).
      if (data.status === "terminated" || data.status === "failed" || data.status === "error") {
        useWorkspaceStore.getState().removePaneByPodKey(data.podKey);
        refreshMeshTopology();
      } else {
        bumpMesh();
      }
      break;
    }
    case "pod:agent_status_changed": {
      bumpPods();
      bumpMesh();
      break;
    }
    case "pod:terminated": {
      const data = decodeEventData(PodStatusChangedEventDataSchema, event.data);
      useWorkspaceStore.getState().removePaneByPodKey(data.podKey);
      bumpPods();
      refreshMeshTopology();
      break;
    }
    case "pod:title_changed":
    case "pod:alias_changed":
    case "pod:perpetual_changed": {
      bumpPods();
      break;
    }
    case "pod:init_progress": {
      // Init progress lives in a transient side-map (creating-pod UI), not the
      // pod list — keep the explicit apply path.
      const data = decodeEventData(PodInitProgressEventDataSchema, event.data);
      usePodStore.getState().updatePodInitProgress(data.podKey, data.phase, data.progress, data.message);
      break;
    }
  }
}
