import { describe, it, expect, beforeEach } from "vitest";
import { useAcpSessionStore } from "../acpSession";

const POD = "pod-1";
const SID = "sess-1";

function getSession() {
  return useAcpSessionStore.getState().sessions[POD];
}

describe("acpSession store", () => {
  beforeEach(() => {
    useAcpSessionStore.setState({ sessions: {} });
  });

  describe("addContentChunk aggregation", () => {
    it("aggregates consecutive same-role chunks into one message", () => {
      const store = useAcpSessionStore.getState();
      store.addContentChunk(POD, SID, "Hello", "assistant");
      store.addContentChunk(POD, SID, " world", "assistant");
      store.addContentChunk(POD, SID, "!", "assistant");

      const msgs = getSession().messages;
      expect(msgs).toHaveLength(1);
      expect(msgs[0].text).toBe("Hello world!");
      expect(msgs[0].role).toBe("assistant");
    });

    it("creates new message when role changes", () => {
      const store = useAcpSessionStore.getState();
      store.addContentChunk(POD, SID, "Hello", "assistant");
      store.addContentChunk(POD, SID, "Hi", "user");

      const msgs = getSession().messages;
      expect(msgs).toHaveLength(2);
      expect(msgs[0].text).toBe("Hello");
      expect(msgs[0].role).toBe("assistant");
      expect(msgs[1].text).toBe("Hi");
      expect(msgs[1].role).toBe("user");
      expect(msgs[1].complete).toBe(true); // user messages are auto-complete
    });

    it("auto-marks user messages as complete", () => {
      const store = useAcpSessionStore.getState();
      store.addContentChunk(POD, SID, "first question", "user");
      store.addContentChunk(POD, SID, "second question", "user");

      const msgs = getSession().messages;
      // Each user message is a separate card (not aggregated)
      expect(msgs).toHaveLength(2);
      expect(msgs[0].text).toBe("first question");
      expect(msgs[0].complete).toBe(true);
      expect(msgs[1].text).toBe("second question");
      expect(msgs[1].complete).toBe(true);
    });

    it("deduplicates consecutive identical user messages", () => {
      const store = useAcpSessionStore.getState();
      store.addContentChunk(POD, SID, "你好", "user");
      store.addContentChunk(POD, SID, "你好", "user"); // relay echo — should be deduped

      const msgs = getSession().messages;
      expect(msgs).toHaveLength(1);
      expect(msgs[0].text).toBe("你好");
    });

    it("allows same user text after an assistant message in between", () => {
      const store = useAcpSessionStore.getState();
      store.addContentChunk(POD, SID, "你好", "user");
      store.addContentChunk(POD, SID, "Hi!", "assistant");
      store.markLastMessageComplete(POD);
      store.addContentChunk(POD, SID, "你好", "user"); // intentional repeat

      const msgs = getSession().messages;
      expect(msgs).toHaveLength(3);
      expect(msgs[0].text).toBe("你好");
      expect(msgs[2].text).toBe("你好");
    });

    it("creates new message after markLastMessageComplete", () => {
      const store = useAcpSessionStore.getState();
      store.addContentChunk(POD, SID, "First", "assistant");
      store.markLastMessageComplete(POD);
      store.addContentChunk(POD, SID, "Second", "assistant");

      const msgs = getSession().messages;
      expect(msgs).toHaveLength(2);
      expect(msgs[0].text).toBe("First");
      expect(msgs[0].complete).toBe(true);
      expect(msgs[1].text).toBe("Second");
      expect(msgs[1].complete).toBeUndefined();
    });

    it("respects 500-message limit", () => {
      const store = useAcpSessionStore.getState();
      // Create 500 complete messages, then add one more
      for (let i = 0; i < 500; i++) {
        store.addContentChunk(POD, SID, `msg-${i}`, "assistant");
        store.markLastMessageComplete(POD);
      }
      store.addContentChunk(POD, SID, "overflow", "assistant");

      const msgs = getSession().messages;
      expect(msgs).toHaveLength(500);
      expect(msgs[msgs.length - 1].text).toBe("overflow");
    });
  });

  describe("markLastMessageComplete", () => {
    it("marks the last message as complete", () => {
      const store = useAcpSessionStore.getState();
      store.addContentChunk(POD, SID, "Hello", "assistant");
      store.markLastMessageComplete(POD);

      const msgs = getSession().messages;
      expect(msgs[0].complete).toBe(true);
    });

    it("is a no-op for non-existent session", () => {
      const store = useAcpSessionStore.getState();
      store.markLastMessageComplete("nonexistent");
      // Should not throw
      expect(useAcpSessionStore.getState().sessions["nonexistent"]).toBeUndefined();
    });

    it("is a no-op for empty messages", () => {
      const store = useAcpSessionStore.getState();
      store.updateSessionState(POD, SID, "idle"); // creates session
      store.markLastMessageComplete(POD);
      expect(getSession().messages).toHaveLength(0);
    });
  });

  describe("updateSessionState", () => {
    it("updates session state", () => {
      const store = useAcpSessionStore.getState();
      store.updateSessionState(POD, SID, "processing");
      expect(getSession().state).toBe("processing");
    });
  });

  describe("clearSession", () => {
    it("removes session data", () => {
      const store = useAcpSessionStore.getState();
      store.addContentChunk(POD, SID, "data", "assistant");
      store.clearSession(POD);
      expect(getSession()).toBeUndefined();
    });
  });

  describe("updateToolCall timestamp", () => {
    it("assigns timestamp on first call", () => {
      const store = useAcpSessionStore.getState();
      store.updateToolCall(POD, SID, {
        toolCallId: "tc-1",
        toolName: "read_file",
        status: "running",
        argumentsJson: "{}",
        timestamp: 0, // will be overwritten since no existing entry
      });

      const tc = getSession().toolCalls["tc-1"];
      expect(tc.timestamp).toBeGreaterThan(0);
    });

    it("preserves original timestamp on subsequent updates", () => {
      const store = useAcpSessionStore.getState();
      store.updateToolCall(POD, SID, {
        toolCallId: "tc-2",
        toolName: "write_file",
        status: "running",
        argumentsJson: "{}",
        timestamp: 0,
      });

      const firstTimestamp = getSession().toolCalls["tc-2"].timestamp;

      // Simulate a later update (e.g. status=completed)
      store.updateToolCall(POD, SID, {
        toolCallId: "tc-2",
        toolName: "write_file",
        status: "completed",
        argumentsJson: "{}",
        timestamp: 0,
      });

      expect(getSession().toolCalls["tc-2"].timestamp).toBe(firstTimestamp);
      expect(getSession().toolCalls["tc-2"].status).toBe("completed");
    });
  });

  describe("setToolCallResult", () => {
    it("sets success and result fields", () => {
      const store = useAcpSessionStore.getState();
      store.updateToolCall(POD, SID, {
        toolCallId: "tc-r1",
        toolName: "bash",
        status: "running",
        argumentsJson: "{}",
        timestamp: 0,
      });
      store.setToolCallResult(POD, SID, "tc-r1", true, "ok", "");

      const tc = getSession().toolCalls["tc-r1"];
      expect(tc.success).toBe(true);
      expect(tc.resultText).toBe("ok");
      expect(tc.status).toBe("completed");
    });

    it("is a no-op for unknown tool call id", () => {
      const store = useAcpSessionStore.getState();
      store.updateSessionState(POD, SID, "idle"); // create session
      store.setToolCallResult(POD, SID, "nonexistent", false, "", "err");
      expect(getSession().toolCalls["nonexistent"]).toBeUndefined();
    });
  });

  describe("addThinking aggregation", () => {
    it("accumulates consecutive thinking chunks into one entry", () => {
      const store = useAcpSessionStore.getState();
      store.addThinking(POD, SID, "Let me ");
      store.addThinking(POD, SID, "think about ");
      store.addThinking(POD, SID, "this.");

      const th = getSession().thinkings;
      expect(th).toHaveLength(1);
      expect(th[0].text).toBe("Let me think about this.");
      expect(th[0].complete).toBeUndefined();
    });

    it("creates new entry after thinking is sealed by content chunk", () => {
      const store = useAcpSessionStore.getState();
      store.addThinking(POD, SID, "First round");
      store.addContentChunk(POD, SID, "Response", "assistant"); // seals thinking
      store.addThinking(POD, SID, "Second round");

      const th = getSession().thinkings;
      expect(th).toHaveLength(2);
      expect(th[0].text).toBe("First round");
      expect(th[0].complete).toBe(true);
      expect(th[1].text).toBe("Second round");
      expect(th[1].complete).toBeUndefined();
    });

    it("respects 100-thinking limit", () => {
      const store = useAcpSessionStore.getState();
      for (let i = 0; i < 100; i++) {
        store.addThinking(POD, SID, `thought-${i}`);
        store.addContentChunk(POD, SID, `resp-${i}`, "assistant"); // seal each
        store.markLastMessageComplete(POD);
      }
      store.addThinking(POD, SID, "overflow");

      const th = getSession().thinkings;
      expect(th).toHaveLength(100);
      expect(th[th.length - 1].text).toBe("overflow");
    });
  });

  describe("sealLastThinking side-effects", () => {
    it("seals thinking on updateToolCall", () => {
      const store = useAcpSessionStore.getState();
      store.addThinking(POD, SID, "thinking...");
      store.updateToolCall(POD, SID, {
        toolCallId: "tc-seal", toolName: "bash", status: "running",
        argumentsJson: "{}", timestamp: 0,
      });

      expect(getSession().thinkings[0].complete).toBe(true);
    });

    it("seals thinking on updatePlan", () => {
      const store = useAcpSessionStore.getState();
      store.addThinking(POD, SID, "thinking...");
      store.updatePlan(POD, SID, [{ title: "step", status: "pending" }]);

      expect(getSession().thinkings[0].complete).toBe(true);
    });

    it("seals thinking on addPermissionRequest", () => {
      const store = useAcpSessionStore.getState();
      store.addThinking(POD, SID, "thinking...");
      store.addPermissionRequest(POD, {
        requestId: "r1", toolName: "bash", argumentsJson: "{}", description: "run cmd",
      });

      expect(getSession().thinkings[0].complete).toBe(true);
    });

    it("seals thinking on updateSessionState", () => {
      const store = useAcpSessionStore.getState();
      store.addThinking(POD, SID, "thinking...");
      store.updateSessionState(POD, SID, "idle");

      expect(getSession().thinkings[0].complete).toBe(true);
    });

    it("seals thinking on setToolCallResult", () => {
      const store = useAcpSessionStore.getState();
      store.updateToolCall(POD, SID, {
        toolCallId: "tc-tr", toolName: "bash", status: "running",
        argumentsJson: "{}", timestamp: 0,
      });
      store.addThinking(POD, SID, "thinking...");
      store.setToolCallResult(POD, SID, "tc-tr", true, "ok", "");

      expect(getSession().thinkings[0].complete).toBe(true);
    });
  });

  describe("trimToolCalls", () => {
    it("evicts oldest completed tool calls when exceeding 500 limit", () => {
      const store = useAcpSessionStore.getState();
      // Create 500 completed tool calls
      for (let i = 0; i < 500; i++) {
        store.updateToolCall(POD, SID, {
          toolCallId: `tc-${i}`, toolName: "bash", status: "completed",
          argumentsJson: "{}", timestamp: 0,
        });
      }
      // Add one more
      store.updateToolCall(POD, SID, {
        toolCallId: "tc-new", toolName: "bash", status: "running",
        argumentsJson: "{}", timestamp: 0,
      });

      const tcs = getSession().toolCalls;
      expect(Object.keys(tcs).length).toBeLessThanOrEqual(500);
      expect(tcs["tc-new"]).toBeDefined();
    });
  });
});
