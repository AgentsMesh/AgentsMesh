import { create } from "zustand";
import { useMemo } from "react";
import { getLoopalManager } from "@/lib/wasm-core";

export interface LoopalBgTask {
  id: string;
  description: string;
  status: string;
  exit_code: number | null;
  output: string;
  created_at_unix_ms: number;
}

export interface LoopalCronJob {
  id: string;
  cron_expr: string;
  prompt: string;
  recurring: boolean;
  next_fire_unix_ms: number | null;
  durable: boolean;
}

export interface LoopalTask {
  id: string;
  subject: string;
  status: string;
  blocked_by: string[];
}

export interface LoopalAgentNode {
  name: string;
  agent_id: string;
  parent: string | null;
  model: string | null;
}

export interface LoopalMcpServer {
  name: string;
  status: string;
  tool_count: number;
}

export interface LoopalGoal {
  goal_id: string;
  objective: string;
  status: string;
}

export interface LoopalSession {
  bg_tasks: LoopalBgTask[];
  crons: LoopalCronJob[];
  tasks: LoopalTask[];
  topology: LoopalAgentNode[];
  mcp: LoopalMcpServer[];
  thread_goal: LoopalGoal | null;
  mode: string | null;
  thinking: string | null;
  model: string | null;
}

const EMPTY: LoopalSession = {
  bg_tasks: [],
  crons: [],
  tasks: [],
  topology: [],
  mcp: [],
  thread_goal: null,
  mode: null,
  thinking: null,
  model: null,
};

interface LoopalConsoleStore {
  _tick: number;
  cache: Record<string, LoopalSession>;
  dispatchEvent: (podKey: string, eventType: string, data: Record<string, unknown>) => void;
  dispatchSnapshot: (podKey: string, snapshot: Record<string, unknown>) => void;
  clearSession: (podKey: string) => void;
}

const mgr = () => getLoopalManager();

function arr<T>(v: unknown): T[] {
  return Array.isArray(v) ? (v as T[]) : [];
}

function normalize(raw: unknown): LoopalSession | null {
  if (!raw || typeof raw !== "object") return null;
  const o = raw as Record<string, unknown>;
  if (!Array.isArray(o.bg_tasks)) return null;
  return {
    bg_tasks: arr<LoopalBgTask>(o.bg_tasks),
    crons: arr<LoopalCronJob>(o.crons),
    tasks: arr<LoopalTask>(o.tasks),
    topology: arr<LoopalAgentNode>(o.topology),
    mcp: arr<LoopalMcpServer>(o.mcp),
    thread_goal: (o.thread_goal as LoopalGoal | null) ?? null,
    mode: (o.mode as string | null) ?? null,
    thinking: (o.thinking as string | null) ?? null,
    model: (o.model as string | null) ?? null,
  };
}

function readSession(podKey: string): LoopalSession | null {
  try {
    const raw = mgr().get_session_json(podKey);
    if (!raw) return null;
    return normalize(typeof raw === "string" ? JSON.parse(raw) : raw);
  } catch (err) {
    console.error("[loopalConsole] wasm read failed", { podKey, err });
    return null;
  }
}

function refreshCache(podKey: string): void {
  const session = readSession(podKey);
  useLoopalConsoleStore.setState((s) => {
    const next = { ...s.cache };
    if (session) next[podKey] = session;
    else delete next[podKey];
    return { _tick: s._tick + 1, cache: next };
  });
}

export const useLoopalConsoleStore = create<LoopalConsoleStore>(() => ({
  _tick: 0,
  cache: {},

  dispatchEvent: (podKey, eventType, data) => {
    mgr().dispatch_event(podKey, eventType, JSON.stringify(data));
    refreshCache(podKey);
  },

  dispatchSnapshot: (podKey, snapshot) => {
    mgr().dispatch_snapshot(podKey, JSON.stringify(snapshot));
    refreshCache(podKey);
  },

  clearSession: (podKey) => {
    mgr().clear_session(podKey);
    useLoopalConsoleStore.setState((s) => {
      if (!(podKey in s.cache)) return { _tick: s._tick + 1 };
      const next = { ...s.cache };
      delete next[podKey];
      return { _tick: s._tick + 1, cache: next };
    });
  },
}));

export function useLoopalSession(podKey: string | null | undefined): LoopalSession {
  const tick = useLoopalConsoleStore((s) => s._tick);
  return useMemo(() => {
    if (!podKey) return EMPTY;
    return useLoopalConsoleStore.getState().cache[podKey] ?? EMPTY;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tick, podKey]);
}
