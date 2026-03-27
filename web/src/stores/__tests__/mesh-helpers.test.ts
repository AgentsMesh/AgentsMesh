import { describe, it, expect } from "vitest";
import {
  getPodStatusInfo,
  getAgentStatusInfo,
  getBindingStatusInfo,
} from "../mesh";

describe("Helper Functions", () => {
  describe("getPodStatusInfo", () => {
    it("should return correct info for initializing status", () => {
      const info = getPodStatusInfo("initializing");
      expect(info.label).toBe("Initializing");
      expect(info.color).toBe("text-blue-600 dark:text-blue-400");
      expect(info.bgColor).toBe("bg-blue-100 dark:bg-blue-900/30");
    });

    it("should return correct info for running status", () => {
      const info = getPodStatusInfo("running");
      expect(info.label).toBe("Running");
      expect(info.color).toBe("text-green-600 dark:text-green-400");
    });

    it("should return correct info for paused status", () => {
      const info = getPodStatusInfo("paused");
      expect(info.label).toBe("Paused");
      expect(info.color).toBe("text-yellow-600 dark:text-yellow-400");
    });

    it("should return correct info for terminated status", () => {
      const info = getPodStatusInfo("terminated");
      expect(info.label).toBe("Terminated");
      expect(info.color).toBe("text-gray-600 dark:text-gray-400");
    });

    it("should return correct info for failed status", () => {
      const info = getPodStatusInfo("failed");
      expect(info.label).toBe("Failed");
      expect(info.color).toBe("text-red-600 dark:text-red-400");
    });

    it("should return terminated info for unknown status", () => {
      const info = getPodStatusInfo("unknown");
      expect(info).toEqual(getPodStatusInfo("terminated"));
    });
  });

  describe("getAgentStatusInfo", () => {
    it("should return correct info for executing status", () => {
      const info = getAgentStatusInfo("executing");
      expect(info.label).toBe("Executing");
      expect(info.color).toBe("text-green-600 dark:text-green-400");
      expect(info.dotColor).toBe("bg-green-500");
      expect(info.bgColor).toBe("bg-green-500/10");
      expect(info.icon).toBeDefined();
    });

    it("should return correct info for waiting status", () => {
      const info = getAgentStatusInfo("waiting");
      expect(info.label).toBe("Waiting for Input");
      expect(info.color).toBe("text-amber-600 dark:text-amber-400");
      expect(info.dotColor).toBe("bg-amber-500");
      expect(info.bgColor).toBe("bg-amber-500/10");
      expect(info.icon).toBeDefined();
    });

    it("should return correct info for idle status", () => {
      const info = getAgentStatusInfo("idle");
      expect(info.label).toBe("Idle");
      expect(info.color).toBe("text-gray-500 dark:text-gray-400");
      expect(info.dotColor).toBe("bg-gray-400");
      expect(info.bgColor).toBe("bg-gray-400/10");
      expect(info.icon).toBeDefined();
    });

    it("should return idle info as fallback for unknown status", () => {
      const info = getAgentStatusInfo("unknown-status");
      const idleInfo = getAgentStatusInfo("idle");
      expect(info).toEqual(idleInfo);
    });
  });

  describe("getBindingStatusInfo", () => {
    it("should return correct info for active status", () => {
      const info = getBindingStatusInfo("active");
      expect(info.label).toBe("Active");
      expect(info.color).toBe("stroke-green-500");
    });

    it("should return correct info for pending status", () => {
      const info = getBindingStatusInfo("pending");
      expect(info.label).toBe("Pending");
      expect(info.color).toBe("stroke-yellow-500");
    });

    it("should return correct info for revoked status", () => {
      const info = getBindingStatusInfo("revoked");
      expect(info.label).toBe("Revoked");
      expect(info.color).toBe("stroke-red-500");
    });

    it("should return correct info for expired status", () => {
      const info = getBindingStatusInfo("expired");
      expect(info.label).toBe("Expired");
      expect(info.color).toBe("stroke-gray-500");
    });

    it("should return active info for unknown status", () => {
      const info = getBindingStatusInfo("unknown");
      expect(info).toEqual(getBindingStatusInfo("active"));
    });
  });
});
