import { create } from "zustand";

// ACP message types
interface AcpContentChunk {
  text: string;
  role: string;
  timestamp: number;
  complete?: boolean; // marked true when result/idle arrives
}

interface AcpToolCall {
  toolCallId: string;
  toolName: string;
  status: string;
  argumentsJson: string;
  resultText?: string;
  errorMessage?: string;
  success?: boolean;
  timestamp: number;
}

interface AcpPlanStep {
  title: string;
  status: string;
}

interface AcpThinking {
  text: string;
  timestamp: number;
  complete?: boolean; // marked true when next non-thinking event arrives
}

interface AcpPermissionRequest {
  requestId: string;
  toolName: string;
  argumentsJson: string;
  description: string;
}

interface AcpLog {
  level: string;
  message: string;
  timestamp: number;
}

interface AcpSessionState {
  messages: AcpContentChunk[];
  toolCalls: Record<string, AcpToolCall>;
  plan: AcpPlanStep[];
  thinkings: AcpThinking[];
  logs: AcpLog[];
  state: string; // idle, processing, waiting_permission
  pendingPermissions: AcpPermissionRequest[];
}

interface AcpSessionStore {
  sessions: Record<string, AcpSessionState>; // keyed by pod_key

  // Mutations (called from RealtimeProvider)
  addContentChunk: (podKey: string, sessionId: string, text: string, role: string) => void;
  markLastMessageComplete: (podKey: string) => void;
  updateToolCall: (podKey: string, sessionId: string, toolCall: AcpToolCall) => void;
  setToolCallResult: (podKey: string, sessionId: string, toolCallId: string, success: boolean, resultText: string, errorMessage: string) => void;
  updatePlan: (podKey: string, sessionId: string, steps: AcpPlanStep[]) => void;
  addThinking: (podKey: string, sessionId: string, text: string) => void;
  addPermissionRequest: (podKey: string, req: AcpPermissionRequest) => void;
  removePermissionRequest: (podKey: string, requestId: string) => void;
  updateSessionState: (podKey: string, sessionId: string, state: string) => void;
  addLog: (podKey: string, level: string, message: string) => void;
  clearSession: (podKey: string) => void;
}

// Re-export types for component use
export type { AcpToolCall, AcpPlanStep, AcpPermissionRequest, AcpSessionState, AcpThinking, AcpLog };

function getOrCreateSession(sessions: Record<string, AcpSessionState>, podKey: string): AcpSessionState {
  return sessions[podKey] || {
    messages: [],
    toolCalls: {},
    plan: [],
    thinkings: [],
    logs: [],
    state: "idle",
    pendingPermissions: [],
  };
}

/** Mark the last thinking entry as complete (called when a non-thinking event arrives). */
function sealLastThinking(thinkings: AcpThinking[]): AcpThinking[] {
  if (thinkings.length === 0) return thinkings;
  const last = thinkings[thinkings.length - 1];
  if (last.complete) return thinkings;
  const copy = [...thinkings];
  copy[copy.length - 1] = { ...last, complete: true };
  return copy;
}

/** Max tool calls to keep per session. Evicts oldest completed entries when exceeded. */
const MAX_TOOL_CALLS = 500;

/** Trim oldest completed tool calls when exceeding the limit. */
function trimToolCalls(toolCalls: Record<string, AcpToolCall>): Record<string, AcpToolCall> {
  const entries = Object.entries(toolCalls);
  if (entries.length <= MAX_TOOL_CALLS) return toolCalls;
  // Sort by timestamp, evict oldest completed entries
  entries.sort((a, b) => a[1].timestamp - b[1].timestamp);
  const toRemove = entries.length - MAX_TOOL_CALLS;
  let removed = 0;
  const keep = new Set(entries.map(([k]) => k));
  for (const [key, tc] of entries) {
    if (removed >= toRemove) break;
    if (tc.status === "completed") {
      keep.delete(key);
      removed++;
    }
  }
  if (removed === 0) return toolCalls; // all running, can't evict
  const result: Record<string, AcpToolCall> = {};
  for (const key of keep) {
    result[key] = toolCalls[key];
  }
  return result;
}

export const useAcpSessionStore = create<AcpSessionStore>((set) => ({
  sessions: {},

  addContentChunk: (podKey, _sessionId, text, role) =>
    set((state) => {
      const session = getOrCreateSession(state.sessions, podKey);
      const messages = [...session.messages];
      const lastMsg = messages[messages.length - 1];
      const thinkings = sealLastThinking(session.thinkings);

      // User messages are always complete (sent as a whole, not streamed).
      const isUserRole = role === "user";

      // Deduplicate user messages: the relay echoes user prompts back as content_chunk
      // events, and AcpSnapshot replays may also include them. Skip if the last
      // message is a complete user message with identical text.
      if (isUserRole && lastMsg && lastMsg.role === "user" && lastMsg.complete && lastMsg.text === text) {
        return state;
      }

      if (lastMsg && lastMsg.role === role && !lastMsg.complete && !isUserRole) {
        // Accumulate into the last message of the same role (assistant streaming)
        messages[messages.length - 1] = {
          ...lastMsg,
          text: lastMsg.text + text,
          timestamp: Date.now(),
        };
      } else {
        // New role, previous complete, or user message — create new message
        messages.push({ text, role, timestamp: Date.now(), ...(isUserRole ? { complete: true } : {}) });
      }

      return {
        sessions: {
          ...state.sessions,
          [podKey]: { ...session, messages: messages.slice(-500), thinkings },
        },
      };
    }),

  markLastMessageComplete: (podKey) =>
    set((state) => {
      const session = state.sessions[podKey];
      if (!session || session.messages.length === 0) return state;
      const messages = [...session.messages];
      messages[messages.length - 1] = { ...messages[messages.length - 1], complete: true };
      return {
        sessions: { ...state.sessions, [podKey]: { ...session, messages } },
      };
    }),

  updateToolCall: (podKey, _sessionId, toolCall) =>
    set((state) => {
      const session = getOrCreateSession(state.sessions, podKey);
      const existing = session.toolCalls[toolCall.toolCallId];
      // Preserve original timestamp if already recorded; otherwise stamp now.
      const timestamped = { ...toolCall, timestamp: existing?.timestamp ?? Date.now() };
      const thinkings = sealLastThinking(session.thinkings);
      const toolCalls = trimToolCalls({ ...session.toolCalls, [toolCall.toolCallId]: timestamped });
      return {
        sessions: {
          ...state.sessions,
          [podKey]: { ...session, thinkings, toolCalls },
        },
      };
    }),

  setToolCallResult: (podKey, _sessionId, toolCallId, success, resultText, errorMessage) =>
    set((state) => {
      const session = getOrCreateSession(state.sessions, podKey);
      const existing = session.toolCalls[toolCallId];
      if (!existing) return state;
      const thinkings = sealLastThinking(session.thinkings);
      return {
        sessions: {
          ...state.sessions,
          [podKey]: {
            ...session,
            thinkings,
            toolCalls: {
              ...session.toolCalls,
              [toolCallId]: { ...existing, success, resultText: resultText, errorMessage: errorMessage, status: "completed" },
            },
          },
        },
      };
    }),

  updatePlan: (podKey, _sessionId, steps) =>
    set((state) => {
      const session = getOrCreateSession(state.sessions, podKey);
      const thinkings = sealLastThinking(session.thinkings);
      return {
        sessions: { ...state.sessions, [podKey]: { ...session, plan: steps, thinkings } },
      };
    }),

  addThinking: (podKey, _sessionId, text) =>
    set((state) => {
      const session = getOrCreateSession(state.sessions, podKey);
      const thinkings = [...session.thinkings];
      const last = thinkings[thinkings.length - 1];

      if (last && !last.complete) {
        // Accumulate into the current thinking round
        thinkings[thinkings.length - 1] = {
          ...last,
          text: last.text + text,
          timestamp: Date.now(),
        };
      } else {
        // New thinking round
        thinkings.push({ text, timestamp: Date.now() });
      }

      return {
        sessions: {
          ...state.sessions,
          [podKey]: {
            ...session,
            thinkings: thinkings.slice(-100),
          },
        },
      };
    }),

  addPermissionRequest: (podKey, req) =>
    set((state) => {
      const session = getOrCreateSession(state.sessions, podKey);
      const thinkings = sealLastThinking(session.thinkings);
      return {
        sessions: {
          ...state.sessions,
          [podKey]: {
            ...session,
            thinkings,
            pendingPermissions: [...session.pendingPermissions, req],
          },
        },
      };
    }),

  removePermissionRequest: (podKey, requestId) =>
    set((state) => {
      const session = getOrCreateSession(state.sessions, podKey);
      return {
        sessions: {
          ...state.sessions,
          [podKey]: {
            ...session,
            pendingPermissions: session.pendingPermissions.filter((p) => p.requestId !== requestId),
          },
        },
      };
    }),

  updateSessionState: (podKey, _sessionId, newState) =>
    set((state) => {
      const session = getOrCreateSession(state.sessions, podKey);
      const thinkings = sealLastThinking(session.thinkings);
      return {
        sessions: { ...state.sessions, [podKey]: { ...session, state: newState, thinkings } },
      };
    }),

  addLog: (podKey, level, message) =>
    set((state) => {
      const session = getOrCreateSession(state.sessions, podKey);
      const logs = [...session.logs, { level, message, timestamp: Date.now() }];
      // Cap at 50 log entries
      if (logs.length > 50) logs.splice(0, logs.length - 50);
      return {
        sessions: { ...state.sessions, [podKey]: { ...session, logs } },
      };
    }),

  clearSession: (podKey) =>
    set((state) => {
      const { [podKey]: _, ...rest } = state.sessions;
      void _;
      return { sessions: rest };
    }),
}));
