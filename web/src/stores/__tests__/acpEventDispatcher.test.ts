import { describe, it, expect, beforeEach } from "vitest";
import { useAcpSessionStore } from "@/stores/acpSession";
import { dispatchAcpRelayEvent } from "@/stores/acpEventDispatcher";
import { MsgType } from "@/stores/relayProtocol";

const POD = "pod-e2e";

function getSession() {
  return useAcpSessionStore.getState().sessions[POD];
}

describe("acpEventDispatcher", () => {
  beforeEach(() => {
    useAcpSessionStore.setState({ sessions: {} });
  });

  describe("AcpEvent routing", () => {
    it("routes content_chunk to addContentChunk", () => {
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "content_chunk",
        session_id: "s1",
        text: "Hello",
        role: "assistant",
      });

      const s = getSession();
      expect(s.messages).toHaveLength(1);
      expect(s.messages[0]).toMatchObject({ text: "Hello", role: "assistant" });
    });

    it("routes tool_call_update to updateToolCall", () => {
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "tool_call_update",
        session_id: "s1",
        tool_call_id: "tc1",
        tool_name: "read_file",
        status: "running",
        arguments_json: '{"path":"src/main.ts"}',
      });

      const tc = getSession().toolCalls["tc1"];
      expect(tc).toBeDefined();
      expect(tc.tool_name).toBe("read_file");
      expect(tc.status).toBe("running");
    });

    it("routes tool_call_result to setToolCallResult", () => {
      // First create the tool call
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "tool_call_update",
        session_id: "s1",
        tool_call_id: "tc2",
        tool_name: "bash",
        status: "completed",
        arguments_json: '{"cmd":"ls"}',
      });

      // Then deliver result
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "tool_call_result",
        session_id: "s1",
        tool_call_id: "tc2",
        success: true,
        result_text: "file1.ts\nfile2.ts",
        error_message: "",
      });

      const tc = getSession().toolCalls["tc2"];
      expect(tc.success).toBe(true);
      expect(tc.result_text).toBe("file1.ts\nfile2.ts");
    });

    it("routes plan_update to updatePlan", () => {
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "plan_update",
        session_id: "s1",
        steps: [
          { title: "Read files", status: "completed" },
          { title: "Write code", status: "in_progress" },
          { title: "Run tests", status: "pending" },
        ],
      });

      const plan = getSession().plan;
      expect(plan).toHaveLength(3);
      expect(plan[0]).toMatchObject({ title: "Read files", status: "completed" });
    });

    it("routes thinking_update to addThinking", () => {
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "thinking_update",
        session_id: "s1",
        text: "Let me analyze this...",
      });

      const th = getSession().thinkings;
      expect(th).toHaveLength(1);
      expect(th[0].text).toBe("Let me analyze this...");
    });

    it("routes permission_request to addPermissionRequest", () => {
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "permission_request",
        session_id: "s1",
        request_id: "perm1",
        tool_name: "bash",
        arguments_json: '{"cmd":"rm -rf /tmp/test"}',
        description: "Execute shell command",
      });

      const perms = getSession().pendingPermissions;
      expect(perms).toHaveLength(1);
      expect(perms[0].request_id).toBe("perm1");
      expect(perms[0].tool_name).toBe("bash");
    });

    it("routes session_state and marks messages complete on idle", () => {
      // Add an incomplete assistant message
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "content_chunk", session_id: "s1", text: "Done", role: "assistant",
      });

      // Transition to idle
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "session_state", session_id: "s1", state: "idle",
      });

      const s = getSession();
      expect(s.state).toBe("idle");
      expect(s.messages[0].complete).toBe(true);
    });

    it("handles log events without crashing", () => {
      // Should not throw or add to store
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "log",
        session_id: "s1",
        level: "error",
        message: "Something went wrong",
      });

      // No session created for log-only events
      expect(getSession()).toBeUndefined();
    });

    it("handles unknown event types gracefully", () => {
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "unknown_future_event",
        session_id: "s1",
      });

      // Should not crash
      expect(getSession()).toBeUndefined();
    });

    it("handles malformed payload without crashing", () => {
      // null payload
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, null);
      // number payload
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, 42);
      // Should not throw
    });
  });

  describe("AcpSnapshot replay", () => {
    it("replays full snapshot with messages, plan, tool_calls, and permissions", () => {
      // Pre-fill some data that should be cleared
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "content_chunk", session_id: "s1", text: "old msg", role: "user",
      });

      // Send snapshot
      dispatchAcpRelayEvent(POD, MsgType.AcpSnapshot, {
        session_id: "s2",
        state: "idle",
        messages: [
          { text: "hello", role: "user" },
          { text: "Hi! How can I help?", role: "assistant" },
        ],
        plan: [
          { title: "Step 1", status: "completed" },
        ],
        tool_calls: [
          {
            tool_call_id: "tc-snap",
            tool_name: "read_file",
            status: "completed",
            arguments_json: '{"path":"main.ts"}',
            success: true,
            result_text: "file content",
          },
        ],
        pending_permissions: [
          {
            request_id: "perm-snap",
            tool_name: "bash",
            arguments_json: "{}",
            description: "run command",
          },
        ],
      });

      const s = getSession();
      expect(s.state).toBe("idle");
      expect(s.messages).toHaveLength(2);
      expect(s.messages[0].text).toBe("hello");
      expect(s.messages[1].text).toBe("Hi! How can I help?");
      expect(s.plan).toHaveLength(1);
      expect(s.plan[0].title).toBe("Step 1");
      expect(s.toolCalls["tc-snap"]).toBeDefined();
      expect(s.toolCalls["tc-snap"].success).toBe(true);
      expect(s.pendingPermissions).toHaveLength(1);
    });

    it("clears previous session before replaying", () => {
      // Fill session with data
      const store = useAcpSessionStore.getState();
      store.addContentChunk(POD, "s1", "old message", "user");
      store.addThinking(POD, "s1", "old thinking");

      // Snapshot with empty data
      dispatchAcpRelayEvent(POD, MsgType.AcpSnapshot, {
        session_id: "s2",
        state: "idle",
      });

      const s = getSession();
      expect(s.messages).toHaveLength(0);
      expect(s.thinkings).toHaveLength(0);
    });
  });

  describe("full session lifecycle", () => {
    it("simulates complete Claude Code interaction", () => {
      const sid = "session-1";

      // 1. User sends prompt
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "content_chunk", session_id: sid, text: "Create a hello world app", role: "user",
      });
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "session_state", session_id: sid, state: "processing",
      });

      expect(getSession().state).toBe("processing");

      // 2. Agent starts thinking
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "thinking_update", session_id: sid, text: "I'll create a simple ",
      });
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "thinking_update", session_id: sid, text: "Node.js hello world application.",
      });

      expect(getSession().thinkings).toHaveLength(1); // aggregated
      expect(getSession().thinkings[0].text).toContain("Node.js hello world");

      // 3. Agent responds with text
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "content_chunk", session_id: sid, text: "I'll create a hello world app for you.", role: "assistant",
      });

      // Thinking should be sealed
      expect(getSession().thinkings[0].complete).toBe(true);

      // 4. Agent makes tool calls
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
      expect(tc.success).toBeUndefined(); // no result yet

      // 5. Tool result arrives
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "tool_call_result", session_id: sid,
        tool_call_id: "tc-write", success: true, result_text: "File written", error_message: "",
      });

      expect(getSession().toolCalls["tc-write"].success).toBe(true);

      // 6. Agent sends more text
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "content_chunk", session_id: sid,
        text: "\n\nDone! I've created main.ts with a hello world program.", role: "assistant",
      });

      // 7. Session returns to idle
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "session_state", session_id: sid, state: "idle",
      });

      const final = getSession();
      expect(final.state).toBe("idle");
      expect(final.messages).toHaveLength(2); // 1 user + 1 assistant (aggregated)
      expect(final.messages[0].role).toBe("user");
      expect(final.messages[1].role).toBe("assistant");
      expect(final.messages[1].complete).toBe(true);
      expect(Object.keys(final.toolCalls)).toHaveLength(1);
      expect(final.thinkings).toHaveLength(1);
    });

    it("simulates permission request and approval flow", () => {
      const sid = "session-perm";

      // Agent is processing
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "session_state", session_id: sid, state: "processing",
      });

      // Agent requests permission
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

      // User approves (simulated by removing permission + sending command)
      useAcpSessionStore.getState().removePermissionRequest(POD, "perm-1");

      // Agent resumes processing
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

      // Initial plan
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

      // Plan progresses
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

      // Round 1
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "thinking_update", session_id: sid, text: "Round 1 thinking...",
      });
      // Content seals round 1
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "content_chunk", session_id: sid, text: "Response 1", role: "assistant",
      });

      // Round 2
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "thinking_update", session_id: sid, text: "Round 2 thinking...",
      });
      // Tool call seals round 2
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

      // User sends slash command
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "content_chunk", session_id: sid, text: "/compact", role: "user",
      });
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "session_state", session_id: sid, state: "processing",
      });

      // Claude processes it and responds
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

      // Normal flow: some events arrive
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "content_chunk", session_id: sid, text: "Working on it", role: "assistant",
      });
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "tool_call_update", session_id: sid,
        tool_call_id: "tc-1", tool_name: "read_file", status: "completed",
        arguments_json: '{"path":"main.ts"}',
      });

      // Reconnect → snapshot arrives (clears + replays)
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
});
