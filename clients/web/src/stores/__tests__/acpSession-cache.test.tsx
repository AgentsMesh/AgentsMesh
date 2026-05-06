import { describe, it, expect, beforeEach, vi } from "vitest";
import { render } from "@testing-library/react";
import { getAcpManager } from "@/lib/wasm-core";
import {
  useAcpSessionStore,
  readAcpSession,
  useAcpSession,
  useAcpSessionField,
  __resetAcpSessionsForTests,
} from "../acpSession";

const POD_A = "pod-cache-a";
const POD_B = "pod-cache-b";

describe("acpSession cache invariants (regression: WASM recursive borrow)", () => {
  beforeEach(() => {
    __resetAcpSessionsForTests();
    vi.restoreAllMocks();
  });

  describe("cache writes are immediately readable", () => {
    it("populates cache after a mutator runs", () => {
      const store = useAcpSessionStore.getState();
      store.addContentChunk(POD_A, "sid", "hello", "assistant");

      const cache = useAcpSessionStore.getState().cache;
      expect(cache[POD_A]).toBeDefined();
      expect(cache[POD_A].messages[0].text).toBe("hello");
    });

    it("readAcpSession serves from cache without touching wasm", () => {
      const store = useAcpSessionStore.getState();
      store.addContentChunk(POD_A, "sid", "x", "assistant");

      const mgr = getAcpManager();
      const spy = vi.spyOn(mgr, "get_session_json");

      const session = readAcpSession(POD_A);
      expect(session?.messages[0].text).toBe("x");
      expect(spy).not.toHaveBeenCalled();
    });

    it("returns null for a pod that has never been mutated", () => {
      expect(readAcpSession("never-touched")).toBeNull();
    });
  });

  describe("per-podKey isolation", () => {
    it("mutating one pod does not touch another pod's cache", () => {
      const store = useAcpSessionStore.getState();
      store.addContentChunk(POD_A, "sa", "a-msg", "assistant");
      store.addContentChunk(POD_B, "sb", "b-msg", "assistant");

      expect(readAcpSession(POD_A)?.messages[0].text).toBe("a-msg");
      expect(readAcpSession(POD_B)?.messages[0].text).toBe("b-msg");
    });

    it("clearSession only drops the targeted pod", () => {
      const store = useAcpSessionStore.getState();
      store.addContentChunk(POD_A, "sa", "a", "assistant");
      store.addContentChunk(POD_B, "sb", "b", "assistant");

      store.clearSession(POD_A);
      expect(readAcpSession(POD_A)).toBeNull();
      expect(readAcpSession(POD_B)?.messages[0].text).toBe("b");
    });
  });

  describe("render-time isolation — the actual fix for `recursive use of an object`", () => {
    it("useAcpSession hook does not call wasm during render", () => {
      const store = useAcpSessionStore.getState();
      store.addContentChunk(POD_A, "sid", "rendered", "assistant");

      const mgr = getAcpManager();
      const spy = vi.spyOn(mgr, "get_session_json");

      function Probe() {
        const s = useAcpSession(POD_A);
        return <div>{s?.messages[0]?.text ?? ""}</div>;
      }
      render(<Probe />);
      expect(spy).not.toHaveBeenCalled();
    });

    it("useAcpSessionField hook does not call wasm during render", () => {
      const store = useAcpSessionStore.getState();
      store.addContentChunk(POD_A, "sid", "x", "assistant");

      const mgr = getAcpManager();
      const spy = vi.spyOn(mgr, "get_session_json");

      function Probe() {
        const state = useAcpSessionField(POD_A, (s) => s.state);
        return <div>{state}</div>;
      }
      render(<Probe />);
      expect(spy).not.toHaveBeenCalled();
    });

    it("six concurrent hooks (the production AgentPanel pattern) → zero wasm reads at render", () => {
      const store = useAcpSessionStore.getState();
      store.addContentChunk(POD_A, "sid", "x", "assistant");

      const mgr = getAcpManager();
      const spy = vi.spyOn(mgr, "get_session_json");

      function Probe() {
        const s = useAcpSession(POD_A);
        const msgs = useAcpSessionField(POD_A, (s) => s.messages);
        const plan = useAcpSessionField(POD_A, (s) => s.plan);
        const tools = useAcpSessionField(POD_A, (s) => s.toolCalls);
        const state = useAcpSessionField(POD_A, (s) => s.state);
        const perms = useAcpSessionField(POD_A, (s) => s.pendingPermissions);
        return (
          <div>
            {s?.state}-{msgs.length}-{plan.length}-{Object.keys(tools).length}-{state}-{perms.length}
          </div>
        );
      }
      render(<Probe />);
      // Pre-fix: this rendered 6 synchronous wasm `get_session_json` calls
      // per render pass, which is exactly what made the wasm-bindgen borrow
      // checker race with the `&mut self` writes triggered by relay events.
      expect(spy).not.toHaveBeenCalled();
    });
  });

  describe("wasm error containment", () => {
    it("swallows a get_session_json failure during refresh; cache stays absent and writers don't throw", () => {
      const mgr = getAcpManager();
      vi.spyOn(mgr, "get_session_json").mockImplementationOnce(() => {
        throw new Error("recursive use of an object detected which would lead to unsafe aliasing in rust");
      });
      const consoleErr = vi.spyOn(console, "error").mockImplementation(() => {});

      const store = useAcpSessionStore.getState();
      expect(() =>
        store.addContentChunk(POD_A, "sid", "x", "assistant"),
      ).not.toThrow();
      expect(readAcpSession(POD_A)).toBeNull();
      expect(consoleErr).toHaveBeenCalled();
    });
  });
});
