import { usePodStore } from "@/stores/pod";
import { useWorkspaceStore } from "@/stores/workspace";
import { useMeshStore } from "@/stores/mesh";
import type { DebounceRef } from "./realtimeEventHandlers";
import type {
  RealtimeEvent, PodStatusChangedData, PodCreatedData,
  PodTitleChangedData, PodAliasChangedData, PodInitProgressData,
} from "@/lib/realtime";

function debouncedSidebarRefresh(ref?: DebounceRef) {
  if (!ref) {
    const { currentSidebarFilter, fetchSidebarPods } = usePodStore.getState();
    fetchSidebarPods?.(currentSidebarFilter);
    return;
  }
  if (ref.current) clearTimeout(ref.current);
  ref.current = setTimeout(() => {
    ref.current = null;
    const { currentSidebarFilter, fetchSidebarPods } = usePodStore.getState();
    fetchSidebarPods?.(currentSidebarFilter);
  }, 500);
}

export function handlePodEvent(event: RealtimeEvent, sidebarDebounceRef?: DebounceRef) {
  switch (event.type) {
    case "pod:created": {
      const data = event.data as PodCreatedData;
      usePodStore.getState().fetchPod?.(data.pod_key);
      debouncedSidebarRefresh(sidebarDebounceRef);
      useMeshStore.getState().fetchTopology?.();
      break;
    }
    case "pod:status_changed": {
      const data = event.data as PodStatusChangedData;
      const podState = usePodStore.getState();
      const existingPod = podState.pods.find(p => p.pod_key === data.pod_key);
      if (!existingPod) {
        podState.fetchPod?.(data.pod_key);
      } else if (podState.updatePodStatus) {
        podState.updatePodStatus(data.pod_key, data.status as "running" | "initializing" | "failed" | "paused" | "terminated" | "error", data.agent_status, data.error_code, data.error_message);
      }
      if (data.status === "terminated" || data.status === "failed" || data.status === "error") {
        useWorkspaceStore.getState().removePaneByPodKey(data.pod_key);
      }
      debouncedSidebarRefresh(sidebarDebounceRef);
      useMeshStore.getState().fetchTopology?.();
      break;
    }
    case "pod:agent_status_changed": {
      const data = event.data as PodStatusChangedData;
      if (data.agent_status) usePodStore.getState().updateAgentStatus(data.pod_key, data.agent_status);
      break;
    }
    case "pod:terminated": {
      const data = event.data as PodStatusChangedData;
      usePodStore.getState().updatePodStatus?.(data.pod_key, "terminated");
      useWorkspaceStore.getState().removePaneByPodKey(data.pod_key);
      debouncedSidebarRefresh(sidebarDebounceRef);
      useMeshStore.getState().fetchTopology?.();
      break;
    }
    case "pod:title_changed": {
      const data = event.data as PodTitleChangedData;
      usePodStore.getState().updatePodTitle(data.pod_key, data.title);
      break;
    }
    case "pod:alias_changed": {
      const data = event.data as PodAliasChangedData;
      usePodStore.getState().updatePodAliasFromEvent(data.pod_key, data.alias);
      break;
    }
    case "pod:init_progress": {
      const data = event.data as PodInitProgressData;
      usePodStore.getState().updatePodInitProgress(data.pod_key, data.phase, data.progress, data.message);
      break;
    }
    case "pod:restarting": {
      const data = event.data as { pod_key: string };
      usePodStore.getState().fetchPod?.(data.pod_key);
      debouncedSidebarRefresh(sidebarDebounceRef);
      break;
    }
  }
}
