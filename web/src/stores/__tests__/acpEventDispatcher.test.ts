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

});
