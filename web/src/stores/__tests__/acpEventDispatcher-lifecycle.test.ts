import { describe, it, expect, beforeEach } from "vitest";
import { useAcpSessionStore } from "@/stores/acpSession";
import { dispatchAcpRelayEvent } from "@/stores/acpEventDispatcher";
import { MsgType } from "@/stores/relayProtocol";

const POD = "pod-e2e";

function getSession() {
  return useAcpSessionStore.getState().sessions[POD];
}

describe("acpEventDispatcher - full session lifecycle", () => {
  beforeEach(() => {
    useAcpSessionStore.setState({ sessions: {} });
  });

  it("simulates complete Claude Code interaction", () => {
    const sid = "session-1";

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "content_chunk", session_id: sid, text: "Create a hello world app", role: "user",
    });
    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "session_state", session_id: sid, state: "processing",
    });

    expect(getSession().state).toBe("processing");

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "thinking_update", session_id: sid, text: "I'll create a simple ",
    });
    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "thinking_update", session_id: sid, text: "Node.js hello world application.",
    });

    expect(getSession().thinkings).toHaveLength(1);
    expect(getSession().thinkings[0].text).toContain("Node.js hello world");

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "content_chunk", session_id: sid, text: "I'll create a hello world app for you.", role: "assistant",
    });

    expect(getSession().thinkings[0].complete).toBe(true);

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "tool_call_update", session_id: sid,
      tool_call_id: "tc-write", tool_name: "write_file", status: "running", arguments_json: '{"path":"main.ts"}',
    });
    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "tool_call_update", session_id: sid,
      tool_call_id: "tc-write", tool_name: "write_file", status: "completed",
      arguments_json: '{"path":"main.ts","content":"console.log(\'hello\')"}',
    });

    const tc = getSession().toolCalls["tc-write"];
    expect(tc.status).toBe("completed");
    expect(tc.success).toBeUndefined();

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "tool_call_result", session_id: sid,
      tool_call_id: "tc-write", success: true, result_text: "File written", error_message: "",
    });

    expect(getSession().toolCalls["tc-write"].success).toBe(true);

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "content_chunk", session_id: sid,
      text: "\n\nDone! I've created main.ts with a hello world program.", role: "assistant",
    });

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "session_state", session_id: sid, state: "idle",
    });

    const final = getSession();
    expect(final.state).toBe("idle");
    expect(final.messages).toHaveLength(2);
    expect(final.messages[0].role).toBe("user");
    expect(final.messages[1].role).toBe("assistant");
    expect(final.messages[1].complete).toBe(true);
    expect(Object.keys(final.toolCalls)).toHaveLength(1);
    expect(final.thinkings).toHaveLength(1);
  });

  it("simulates permission request and approval flow", () => {
    const sid = "session-perm";

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "session_state", session_id: sid, state: "processing",
    });

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "permission_request", session_id: sid,
      request_id: "perm-1", tool_name: "bash",
      arguments_json: '{"cmd":"npm install"}', description: "Execute: npm install",
    });
    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "session_state", session_id: sid, state: "waiting_permission",
    });

    let s = getSession();
    expect(s.state).toBe("waiting_permission");
    expect(s.pendingPermissions).toHaveLength(1);

    useAcpSessionStore.getState().removePermissionRequest(POD, "perm-1");

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "session_state", session_id: sid, state: "processing",
    });
    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "tool_call_update", session_id: sid,
      tool_call_id: "tc-npm", tool_name: "bash", status: "running",
      arguments_json: '{"cmd":"npm install"}',
    });

    s = getSession();
    expect(s.state).toBe("processing");
    expect(s.pendingPermissions).toHaveLength(0);
    expect(s.toolCalls["tc-npm"]).toBeDefined();
  });

  it("simulates plan update during execution", () => {
    const sid = "session-plan";

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "plan_update", session_id: sid,
      steps: [
        { title: "Analyze codebase", status: "in_progress" },
        { title: "Write tests", status: "pending" },
        { title: "Run tests", status: "pending" },
      ],
    });

    expect(getSession().plan).toHaveLength(3);
    expect(getSession().plan[0].status).toBe("in_progress");

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "plan_update", session_id: sid,
      steps: [
        { title: "Analyze codebase", status: "completed" },
        { title: "Write tests", status: "in_progress" },
        { title: "Run tests", status: "pending" },
      ],
    });

    expect(getSession().plan[0].status).toBe("completed");
    expect(getSession().plan[1].status).toBe("in_progress");
  });

  it("simulates multiple thinking rounds", () => {
    const sid = "session-think";

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "thinking_update", session_id: sid, text: "Round 1 thinking...",
    });
    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "content_chunk", session_id: sid, text: "Response 1", role: "assistant",
    });

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "thinking_update", session_id: sid, text: "Round 2 thinking...",
    });
    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "tool_call_update", session_id: sid,
      tool_call_id: "tc-x", tool_name: "bash", status: "running", arguments_json: "{}",
    });

    const th = getSession().thinkings;
    expect(th).toHaveLength(2);
    expect(th[0].complete).toBe(true);
    expect(th[1].complete).toBe(true);
  });

  it("simulates failed tool call", () => {
    const sid = "session-fail";

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "tool_call_update", session_id: sid,
      tool_call_id: "tc-fail", tool_name: "bash", status: "running",
      arguments_json: '{"cmd":"invalid-command"}',
    });
    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "tool_call_update", session_id: sid,
      tool_call_id: "tc-fail", tool_name: "bash", status: "completed",
      arguments_json: '{"cmd":"invalid-command"}',
    });
    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "tool_call_result", session_id: sid,
      tool_call_id: "tc-fail", success: false, result_text: "",
      error_message: "command not found: invalid-command",
    });

    const tc = getSession().toolCalls["tc-fail"];
    expect(tc.success).toBe(false);
    expect(tc.error_message).toBe("command not found: invalid-command");
  });

  it("simulates slash command (/compact)", () => {
    const sid = "session-slash";

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "content_chunk", session_id: sid, text: "/compact", role: "user",
    });
    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "session_state", session_id: sid, state: "processing",
    });

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "content_chunk", session_id: sid,
      text: "Context compacted. Conversation reduced from 50k to 10k tokens.", role: "assistant",
    });
    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "session_state", session_id: sid, state: "idle",
    });

    const s = getSession();
    expect(s.messages[0].text).toBe("/compact");
    expect(s.messages[0].role).toBe("user");
    expect(s.messages[1].role).toBe("assistant");
  });

  it("simulates reconnect with snapshot restoring tool calls", () => {
    const sid = "session-reconn";

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "content_chunk", session_id: sid, text: "Working on it", role: "assistant",
    });
    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "tool_call_update", session_id: sid,
      tool_call_id: "tc-1", tool_name: "read_file", status: "completed",
      arguments_json: '{"path":"main.ts"}',
    });

    dispatchAcpRelayEvent(POD, MsgType.AcpSnapshot, {
      session_id: sid,
      state: "processing",
      messages: [
        { text: "Fix the bug", role: "user" },
        { text: "Working on it", role: "assistant" },
      ],
      tool_calls: [
        {
          tool_call_id: "tc-1", tool_name: "read_file", status: "completed",
          arguments_json: '{"path":"main.ts"}', success: true, result_text: "file content",
        },
        {
          tool_call_id: "tc-2", tool_name: "write_file", status: "running",
          arguments_json: '{"path":"main.ts"}',
        },
      ],
      plan: [
        { title: "Read code", status: "completed" },
        { title: "Fix bug", status: "in_progress" },
      ],
    });

    const s = getSession();
    expect(s.state).toBe("processing");
    expect(s.messages).toHaveLength(2);
    expect(Object.keys(s.toolCalls)).toHaveLength(2);
    expect(s.toolCalls["tc-1"].success).toBe(true);
    expect(s.toolCalls["tc-2"].status).toBe("running");
    expect(s.plan).toHaveLength(2);
  });
});
