import type { PodData } from "@/lib/api";

export type Pod = PodData;

export const SIDEBAR_STATUS_MAP: Record<string, string> = {
  mine: "running,initializing",
  org: "running,initializing",
  completed: "terminated,failed,paused,completed,error",
};
export const SIDEBAR_PAGE_SIZE = 20;

export interface PodInitProgress {
  phase: string;
  progress: number;
  message: string;
}

export interface PodState {
  _tick: number;
  loading: boolean;
  error: string | null;
  initProgress: Record<string, PodInitProgress>;
  podTotal: number;
  podHasMore: boolean;
  loadingMore: boolean;
  currentSidebarFilter: string;
  sidebarLoadedCount: number;

  fetchPods: (filters?: { status?: string; runnerId?: number }) => Promise<void>;
  fetchPod: (podKey: string) => Promise<void>;
  fetchSidebarPods: (statusFilter: string, opts?: { silent?: boolean }) => Promise<void>;
  loadMorePods: () => Promise<void>;
  terminatePod: (podKey: string) => Promise<void>;
  setCurrentPod: (pod: Pod | null) => void;
  updatePodStatus: (podKey: string, status: Pod["status"], agentStatus?: string, errorCode?: string, errorMessage?: string) => void;
  updateAgentStatus: (podKey: string, agentStatus: string) => void;
  updatePodTitle: (podKey: string, title: string) => void;
  updatePodAlias: (podKey: string, alias: string | null) => Promise<void>;
  updatePodAliasFromEvent: (podKey: string, alias: string | null) => void;
  updatePodPerpetual: (podKey: string, perpetual: boolean) => Promise<void>;
  updatePodPerpetualFromEvent: (podKey: string, perpetual: boolean) => void;
  upsertPod: (pod: Pod) => void;
  updatePodInitProgress: (podKey: string, phase: string, progress: number, message: string) => void;
  clearInitProgress: (podKey: string) => void;
  clearError: () => void;
}
