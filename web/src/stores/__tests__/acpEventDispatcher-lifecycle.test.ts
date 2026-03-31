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
      type: "contentChunk", sessionId: sid, text: "Create a hello world app", role: "user",
    });
    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "sessionState", sessionId: sid, state: "processing",
    });

    expect(getSession().state).toBe("processing");

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "thinkingUpdate", sessionId: sid, text: "I'll create a simple ",
    });
    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "thinkingUpdate", sessionId: sid, text: "Node.js hello world application.",
    });

    expect(getSession().thinkings).toHaveLength(1);
    expect(getSession().thinkings[0].text).toContain("Node.js hello world");

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "contentChunk", sessionId: sid, text: "I'll create a hello world app for you.", role: "assistant",
    });

    expect(getSession().thinkings[0].complete).toBe(true);

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "toolCallUpdate", sessionId: sid,
      toolCallId: "tc-write", toolName: "write_file", status: "running", argumentsJson: '{"path":"main.ts"}',
    });
    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "toolCallUpdate", sessionId: sid,
      toolCallId: "tc-write", toolName: "write_file", status: "completed",
      argumentsJson: '{"path":"main.ts","content":"console.log(\'hello\')"}',
    });

    const tc = getSession().toolCalls["tc-write"];
    expect(tc.status).toBe("completed");
    expect(tc.success).toBeUndefined();

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "toolCallResult", sessionId: sid,
      toolCallId: "tc-write", success: true, resultText: "File written", errorMessage: "",
    });

    expect(getSession().toolCalls["tc-write"].success).toBe(true);

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "contentChunk", sessionId: sid,
      text: "\n\nDone! I've created main.ts with a hello world program.", role: "assistant",
    });

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "sessionState", sessionId: sid, state: "idle",
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
      type: "sessionState", sessionId: sid, state: "processing",
    });

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "permissionRequest", sessionId: sid,
      requestId: "perm-1", toolName: "bash",
      argumentsJson: '{"cmd":"npm install"}', description: "Execute: npm install",
    });
    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "sessionState", sessionId: sid, state: "waiting_permission",
    });

    let s = getSession();
    expect(s.state).toBe("waiting_permission");
    expect(s.pendingPermissions).toHaveLength(1);

    useAcpSessionStore.getState().removePermissionRequest(POD, "perm-1");

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "sessionState", sessionId: sid, state: "processing",
    });
    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "toolCallUpdate", sessionId: sid,
      toolCallId: "tc-npm", toolName: "bash", status: "running",
      argumentsJson: '{"cmd":"npm install"}',
    });

    s = getSession();
    expect(s.state).toBe("processing");
    expect(s.pendingPermissions).toHaveLength(0);
    expect(s.toolCalls["tc-npm"]).toBeDefined();
  });

  it("simulates plan update during execution", () => {
    const sid = "session-plan";

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "planUpdate", sessionId: sid,
      steps: [
        { title: "Analyze codebase", status: "in_progress" },
        { title: "Write tests", status: "pending" },
        { title: "Run tests", status: "pending" },
      ],
    });

    expect(getSession().plan).toHaveLength(3);
    expect(getSession().plan[0].status).toBe("in_progress");

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "planUpdate", sessionId: sid,
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
      type: "thinkingUpdate", sessionId: sid, text: "Round 1 thinking...",
    });
    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "contentChunk", sessionId: sid, text: "Response 1", role: "assistant",
    });

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "thinkingUpdate", sessionId: sid, text: "Round 2 thinking...",
    });
    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "toolCallUpdate", sessionId: sid,
      toolCallId: "tc-x", toolName: "bash", status: "running", argumentsJson: "{}",
    });

    const th = getSession().thinkings;
    expect(th).toHaveLength(2);
    expect(th[0].complete).toBe(true);
    expect(th[1].complete).toBe(true);
  });

  it("simulates failed tool call", () => {
    const sid = "session-fail";

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "toolCallUpdate", sessionId: sid,
      toolCallId: "tc-fail", toolName: "bash", status: "running",
      argumentsJson: '{"cmd":"invalid-command"}',
    });
    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "toolCallUpdate", sessionId: sid,
      toolCallId: "tc-fail", toolName: "bash", status: "completed",
      argumentsJson: '{"cmd":"invalid-command"}',
    });
    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "toolCallResult", sessionId: sid,
      toolCallId: "tc-fail", success: false, resultText: "",
      errorMessage: "command not found: invalid-command",
    });

    const tc = getSession().toolCalls["tc-fail"];
    expect(tc.success).toBe(false);
    expect(tc.errorMessage).toBe("command not found: invalid-command");
  });

  it("simulates slash command (/compact)", () => {
    const sid = "session-slash";

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "contentChunk", sessionId: sid, text: "/compact", role: "user",
    });
    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "sessionState", sessionId: sid, state: "processing",
    });

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "contentChunk", sessionId: sid,
      text: "Context compacted. Conversation reduced from 50k to 10k tokens.", role: "assistant",
    });
    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "sessionState", sessionId: sid, state: "idle",
    });

    const s = getSession();
    expect(s.messages[0].text).toBe("/compact");
    expect(s.messages[0].role).toBe("user");
    expect(s.messages[1].role).toBe("assistant");
  });

  it("simulates reconnect with snapshot restoring tool calls", () => {
    const sid = "session-reconn";

    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "contentChunk", sessionId: sid, text: "Working on it", role: "assistant",
    });
    dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
      type: "toolCallUpdate", sessionId: sid,
      toolCallId: "tc-1", toolName: "read_file", status: "completed",
      argumentsJson: '{"path":"main.ts"}',
    });

    dispatchAcpRelayEvent(POD, MsgType.AcpSnapshot, {
      sessionId: sid,
      state: "processing",
      messages: [
        { text: "Fix the bug", role: "user" },
        { text: "Working on it", role: "assistant" },
      ],
      toolCalls: [
        {
          toolCallId: "tc-1", toolName: "read_file", status: "completed",
          argumentsJson: '{"path":"main.ts"}', success: true, resultText: "file content",
        },
        {
          toolCallId: "tc-2", toolName: "write_file", status: "running",
          argumentsJson: '{"path":"main.ts"}',
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
