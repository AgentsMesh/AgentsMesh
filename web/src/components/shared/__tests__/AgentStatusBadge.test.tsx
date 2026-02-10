import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { AgentStatusBadge } from "../AgentStatusBadge";

describe("AgentStatusBadge", () => {
  describe("renders null when podStatus is not running", () => {
    it.each(["initializing", "completed", "terminated", "error"])(
      "returns null for podStatus=%s",
      (podStatus) => {
        const { container } = render(
          <AgentStatusBadge
            podStatus={podStatus}
            agentStatus="executing"
          />
        );
        expect(container.firstChild).toBeNull();
      }
    );
  });

  describe("dot variant", () => {
    it("renders dot variant correctly with pulse for executing", () => {
      const { container } = render(
        <AgentStatusBadge
          podStatus="running"
          agentStatus="executing"
          variant="dot"
        />
      );
      const dot = container.querySelector("span");
      expect(dot).not.toBeNull();
      expect(dot!.className).toContain("w-2");
      expect(dot!.className).toContain("h-2");
      expect(dot!.className).toContain("rounded-full");
      expect(dot!.className).toContain("animate-pulse");
      expect(dot!.getAttribute("title")).toBe("Executing");
    });

    it("does not have animate-pulse for idle status", () => {
      const { container } = render(
        <AgentStatusBadge
          podStatus="running"
          agentStatus="idle"
          variant="dot"
        />
      );
      const dot = container.querySelector("span");
      expect(dot).not.toBeNull();
      expect(dot!.className).not.toContain("animate-pulse");
      expect(dot!.getAttribute("title")).toBe("Idle");
    });
  });

  describe("badge variant (default)", () => {
    it("renders badge variant correctly for executing", () => {
      const { container } = render(
        <AgentStatusBadge
          podStatus="running"
          agentStatus="executing"
        />
      );
      expect(screen.getByText("Executing")).toBeInTheDocument();
      const badge = container.querySelector("span");
      expect(badge).not.toBeNull();
      expect(badge!.className).toContain("rounded-full");
    });
  });

  describe("inline variant", () => {
    it("renders inline variant correctly for waiting", () => {
      render(
        <AgentStatusBadge
          podStatus="running"
          agentStatus="waiting"
          variant="inline"
        />
      );
      expect(screen.getByText("Waiting for Input")).toBeInTheDocument();
    });
  });

  describe("status labels", () => {
    it("renders idle status correctly", () => {
      render(
        <AgentStatusBadge
          podStatus="running"
          agentStatus="idle"
        />
      );
      expect(screen.getByText("Idle")).toBeInTheDocument();
    });

    it("falls back to idle for unknown status", () => {
      render(
        <AgentStatusBadge
          podStatus="running"
          agentStatus="unknown_value"
        />
      );
      expect(screen.getByText("Idle")).toBeInTheDocument();
    });
  });

  describe("status colors", () => {
    it("renders executing status with green colors", () => {
      const { container } = render(
        <AgentStatusBadge
          podStatus="running"
          agentStatus="executing"
        />
      );
      const badge = container.querySelector("span");
      expect(badge).not.toBeNull();
      expect(badge!.className).toContain("text-green-600");
      expect(badge!.className).toContain("bg-green-500/10");
    });

    it("renders waiting status with amber colors", () => {
      const { container } = render(
        <AgentStatusBadge
          podStatus="running"
          agentStatus="waiting"
        />
      );
      const badge = container.querySelector("span");
      expect(badge).not.toBeNull();
      expect(badge!.className).toContain("text-amber-600");
      expect(badge!.className).toContain("bg-amber-500/10");
    });
  });
});
