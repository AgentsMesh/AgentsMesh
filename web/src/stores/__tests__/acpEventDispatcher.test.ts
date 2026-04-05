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
    it("routes contentChunk to addContentChunk", () => {
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "contentChunk",
        sessionId: "s1",
        text: "Hello",
        role: "assistant",
      });

      const s = getSession();
      expect(s.messages).toHaveLength(1);
      expect(s.messages[0]).toMatchObject({ text: "Hello", role: "assistant" });
    });

    it("routes toolCallUpdate to updateToolCall", () => {
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "toolCallUpdate",
        sessionId: "s1",
        toolCallId: "tc1",
        toolName: "read_file",
        status: "running",
        argumentsJson: '{"path":"src/main.ts"}',
      });

      const tc = getSession().toolCalls["tc1"];
      expect(tc).toBeDefined();
      expect(tc.toolName).toBe("read_file");
      expect(tc.status).toBe("running");
    });

    it("routes toolCallResult to setToolCallResult", () => {
      // First create the tool call
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "toolCallUpdate",
        sessionId: "s1",
        toolCallId: "tc2",
        toolName: "bash",
        status: "completed",
        argumentsJson: '{"cmd":"ls"}',
      });

      // Then deliver result
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "toolCallResult",
        sessionId: "s1",
        toolCallId: "tc2",
        success: true,
        resultText: "file1.ts\nfile2.ts",
        errorMessage: "",
      });

      const tc = getSession().toolCalls["tc2"];
      expect(tc.success).toBe(true);
      expect(tc.resultText).toBe("file1.ts\nfile2.ts");
    });

    it("routes planUpdate to updatePlan", () => {
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "planUpdate",
        sessionId: "s1",
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

    it("routes thinkingUpdate to addThinking", () => {
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "thinkingUpdate",
        sessionId: "s1",
        text: "Let me analyze this...",
      });

      const th = getSession().thinkings;
      expect(th).toHaveLength(1);
      expect(th[0].text).toBe("Let me analyze this...");
    });

    it("routes permissionRequest to addPermissionRequest", () => {
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "permissionRequest",
        sessionId: "s1",
        requestId: "perm1",
        toolName: "bash",
        argumentsJson: '{"cmd":"rm -rf /tmp/test"}',
        description: "Execute shell command",
      });

      const perms = getSession().pendingPermissions;
      expect(perms).toHaveLength(1);
      expect(perms[0].requestId).toBe("perm1");
      expect(perms[0].toolName).toBe("bash");
    });

    it("routes sessionState and marks messages complete on idle", () => {
      // Add an incomplete assistant message
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "contentChunk", sessionId: "s1", text: "Done", role: "assistant",
      });

      // Transition to idle
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "sessionState", sessionId: "s1", state: "idle",
      });

      const s = getSession();
      expect(s.state).toBe("idle");
      expect(s.messages[0].complete).toBe(true);
    });

    it("handles log events and stores error/warn in session", () => {
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "log",
        sessionId: "s1",
        level: "error",
        message: "Something went wrong",
      });

      // Error-level logs are now stored in session
      const session = getSession();
      expect(session).toBeDefined();
      expect(session!.logs).toHaveLength(1);
      expect(session!.logs[0].level).toBe("error");
      expect(session!.logs[0].message).toBe("Something went wrong");
    });

    it("handles unknown event types gracefully", () => {
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "unknown_future_event",
        sessionId: "s1",
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
    it("replays full snapshot with messages, plan, toolCalls, and permissions", () => {
      // Pre-fill some data that should be cleared
      dispatchAcpRelayEvent(POD, MsgType.AcpEvent, {
        type: "contentChunk", sessionId: "s1", text: "old msg", role: "user",
      });

      // Send snapshot
      dispatchAcpRelayEvent(POD, MsgType.AcpSnapshot, {
        sessionId: "s2",
        state: "idle",
        messages: [
          { text: "hello", role: "user" },
          { text: "Hi! How can I help?", role: "assistant" },
        ],
        plan: [
          { title: "Step 1", status: "completed" },
        ],
        toolCalls: [
          {
            toolCallId: "tc-snap",
            toolName: "read_file",
            status: "completed",
            argumentsJson: '{"path":"main.ts"}',
            success: true,
            resultText: "file content",
          },
        ],
        pendingPermissions: [
          {
            requestId: "perm-snap",
            toolName: "bash",
            argumentsJson: "{}",
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
        sessionId: "s2",
        state: "idle",
      });

      const s = getSession();
      expect(s.messages).toHaveLength(0);
      expect(s.thinkings).toHaveLength(0);
    });
  });

});
