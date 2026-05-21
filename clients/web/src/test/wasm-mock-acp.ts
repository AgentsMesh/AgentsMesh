/**
 * Stateful ACP session manager mock.
 * Implements the same aggregation/sealing logic as the Rust WASM AcpManager.
 */

interface AcpMsg { text: string; role: string; timestamp: number; complete?: boolean }
interface AcpTC { id: string; name: string; status: string; args: unknown; result_text?: string; error_message?: string; success?: boolean; timestamp: number }
interface AcpThink { text: string; timestamp: number; complete?: boolean }
interface AcpPerm { id: string; tool_name: string; args: unknown; description: string }
interface AcpLog { level: string; message: string; timestamp: number }
interface AcpConfig { permission_mode: string; model: string }
interface AcpSession {
  messages: AcpMsg[]; tool_calls: Record<string, AcpTC>; plan: unknown[];
  thinkings: AcpThink[]; logs: AcpLog[]; state: string; pending_permissions: AcpPerm[];
  configuration: AcpConfig;
}

const MAX_MESSAGES = 500;
const MAX_THINKINGS = 100;
const MAX_TOOL_CALLS = 500;

export function createAcpManager() {
  const sessions = new Map<string, AcpSession>();

  function getOrCreate(key: string): AcpSession {
    let s = sessions.get(key);
    if (!s) {
      s = { messages: [], tool_calls: {}, plan: [], thinkings: [], logs: [], state: 'idle', pending_permissions: [], configuration: { permission_mode: '', model: '' } };
      sessions.set(key, s);
    }
    return s;
  }

  function sealThinking(key: string) {
    const s = sessions.get(key);
    if (!s || s.thinkings.length === 0) return;
    const last = s.thinkings[s.thinkings.length - 1];
    if (last && !last.complete) last.complete = true;
  }

  function trimToolCalls(s: AcpSession) {
    const ids = Object.keys(s.tool_calls);
    if (ids.length <= MAX_TOOL_CALLS) return;
    const completed = ids.filter((id) => s.tool_calls[id].status === 'completed')
      .sort((a, b) => (s.tool_calls[a].timestamp || 0) - (s.tool_calls[b].timestamp || 0));
    const toRemove = ids.length - MAX_TOOL_CALLS;
    for (let i = 0; i < Math.min(toRemove, completed.length); i++) {
      delete s.tool_calls[completed[i]];
    }
  }

  return {
    add_content_chunk: (podKey: string, text: string, role: string) => {
      const s = getOrCreate(podKey);
      sealThinking(podKey);
      if (role === 'user') {
        const last = s.messages[s.messages.length - 1];
        if (last && last.role === 'user' && last.text === text) return;
        s.messages.push({ text, role, timestamp: Date.now(), complete: true });
      } else {
        const last = s.messages[s.messages.length - 1];
        if (last && last.role === role && !last.complete) {
          last.text += text;
        } else {
          s.messages.push({ text, role, timestamp: Date.now() });
        }
      }
      while (s.messages.length > MAX_MESSAGES) s.messages.shift();
    },

    mark_last_message_complete: (podKey: string) => {
      const s = sessions.get(podKey);
      if (!s || s.messages.length === 0) return;
      s.messages[s.messages.length - 1].complete = true;
    },

    update_tool_call: (podKey: string, json: string) => {
      const s = getOrCreate(podKey);
      sealThinking(podKey);
      const tc = JSON.parse(json) as AcpTC;
      const existing = s.tool_calls[tc.id];
      if (existing) {
        s.tool_calls[tc.id] = { ...existing, ...tc, timestamp: existing.timestamp };
      } else {
        s.tool_calls[tc.id] = { ...tc, timestamp: tc.timestamp || Date.now() };
      }
      trimToolCalls(s);
    },

    set_tool_call_result: (podKey: string, id: string, success: boolean, resultText?: string, errorMessage?: string) => {
      const s = sessions.get(podKey);
      if (!s) return;
      sealThinking(podKey);
      const tc = s.tool_calls[id];
      if (!tc) return;
      tc.success = success;
      tc.result_text = resultText || undefined;
      tc.error_message = errorMessage || undefined;
      tc.status = 'completed';
    },

    update_plan: (podKey: string, json: string) => {
      const s = getOrCreate(podKey);
      sealThinking(podKey);
      s.plan = JSON.parse(json);
    },

    add_thinking: (podKey: string, text: string) => {
      const s = getOrCreate(podKey);
      const last = s.thinkings[s.thinkings.length - 1];
      if (last && !last.complete) {
        last.text += text;
      } else {
        s.thinkings.push({ text, timestamp: Date.now() });
      }
      while (s.thinkings.length > MAX_THINKINGS) s.thinkings.shift();
    },

    add_permission_request: (podKey: string, json: string) => {
      const s = getOrCreate(podKey);
      sealThinking(podKey);
      s.pending_permissions.push(JSON.parse(json));
    },

    remove_permission_request: (podKey: string, id: string) => {
      const s = sessions.get(podKey);
      if (!s) return;
      s.pending_permissions = s.pending_permissions.filter((p) => p.id !== id);
    },

    update_session_state: (podKey: string, newState: string) => {
      const s = getOrCreate(podKey);
      sealThinking(podKey);
      s.state = newState;
    },

    add_log: (podKey: string, level: string, message: string) => {
      const s = getOrCreate(podKey);
      s.logs.push({ level, message, timestamp: Date.now() });
    },

    update_configuration: (podKey: string, json: string) => {
      const s = getOrCreate(podKey);
      const cfg = JSON.parse(json) as Partial<AcpConfig>;
      if (cfg.permission_mode) s.configuration.permission_mode = cfg.permission_mode;
      if (cfg.model) s.configuration.model = cfg.model;
    },

    clear_session: (podKey: string) => { sessions.delete(podKey); },

    get_session_json: (podKey: string) => {
      const s = sessions.get(podKey);
      if (!s) return null;
      return JSON.stringify(s);
    },

    _seed: (podKey: string, partial: Partial<AcpSession>) => {
      const merged: AcpSession = {
        messages: partial.messages ?? [],
        tool_calls: partial.tool_calls ?? {},
        plan: partial.plan ?? [],
        thinkings: partial.thinkings ?? [],
        logs: partial.logs ?? [],
        state: partial.state ?? 'idle',
        pending_permissions: partial.pending_permissions ?? [],
        configuration: partial.configuration ?? { permission_mode: '', model: '' },
      };
      sessions.set(podKey, merged);
    },

    _reset: () => { sessions.clear(); },
  };
}
