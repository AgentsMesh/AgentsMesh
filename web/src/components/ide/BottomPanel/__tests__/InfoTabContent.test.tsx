import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { InfoTabContent } from "../InfoTabContent";
import type { PodData } from "@/lib/api/pod";

// Mock pod store
let mockPods: PodData[] = [];

vi.mock("@/stores/pod", () => ({
  usePodStore: () => ({
    pods: mockPods,
  }),
}));

// Mock t function - returns the key for easy assertion
const mockT = (key: string, params?: Record<string, string | number>) => {
  if (params) {
    return Object.entries(params).reduce(
      (str, [k, v]) => str.replace(`{${k}}`, String(v)),
      key
    );
  }
  return key;
};

// Factory to create mock PodData
function createMockPod(overrides: Partial<PodData> = {}): PodData {
  return {
    id: 1,
    pod_key: "pod-abc12345-def6-7890",
    status: "running",
    agent_status: "executing",
    created_at: "2026-01-15T10:00:00Z",
    ...overrides,
  };
}

describe("InfoTabContent", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockPods = [];
  });

  describe("empty states", () => {
    it("should show select pod message when no pod is selected", () => {
      render(
        <InfoTabContent selectedPodKey={null} pod={null} t={mockT} />
      );
      expect(
        screen.getByText("ide.bottomPanel.selectPodFirst")
      ).toBeInTheDocument();
    });

    it("should show not found message when pod key is set but pod data is null", () => {
      render(
        <InfoTabContent
          selectedPodKey="pod-123"
          pod={null}
          t={mockT}
        />
      );
      expect(
        screen.getByText("ide.bottomPanel.infoTab.notFound")
      ).toBeInTheDocument();
    });
  });

  describe("basic pod info", () => {
    it("should display pod key", () => {
      const pod = createMockPod();
      render(
        <InfoTabContent
          selectedPodKey={pod.pod_key}
          pod={pod}
          t={mockT}
        />
      );
      expect(screen.getByText(pod.pod_key)).toBeInTheDocument();
    });

    it("should display pod status badge", () => {
      const pod = createMockPod({ status: "running" });
      render(
        <InfoTabContent
          selectedPodKey={pod.pod_key}
          pod={pod}
          t={mockT}
        />
      );
      expect(screen.getByText("Running")).toBeInTheDocument();
    });

    it("should display created at time", () => {
      const pod = createMockPod({
        created_at: "2026-01-15T10:00:00Z",
      });
      render(
        <InfoTabContent
          selectedPodKey={pod.pod_key}
          pod={pod}
          t={mockT}
        />
      );
      // The label should be present
      expect(
        screen.getByText("ide.bottomPanel.infoTab.createdAt:")
      ).toBeInTheDocument();
    });
  });

  describe("optional fields", () => {
    it("should display agent type when available", () => {
      const pod = createMockPod({
        agent_type: { id: 1, name: "Claude Code", slug: "claude-code" },
      });
      render(
        <InfoTabContent
          selectedPodKey={pod.pod_key}
          pod={pod}
          t={mockT}
        />
      );
      expect(screen.getByText("Claude Code")).toBeInTheDocument();
    });

    it("should display agent status when available", () => {
      const pod = createMockPod({ agent_status: "executing", status: "running" });
      render(
        <InfoTabContent
          selectedPodKey={pod.pod_key}
          pod={pod}
          t={mockT}
        />
      );
      expect(screen.getByText("Executing")).toBeInTheDocument();
    });

    it("should display runner info when available", () => {
      const pod = createMockPod({
        runner: { id: 1, node_id: "runner-node-abc", status: "online" },
      });
      render(
        <InfoTabContent
          selectedPodKey={pod.pod_key}
          pod={pod}
          t={mockT}
        />
      );
      expect(screen.getByText("runner-node-abc")).toBeInTheDocument();
    });

    it("should display repository when available", () => {
      const pod = createMockPod({
        repository: {
          id: 1,
          name: "my-repo",
          full_path: "org/my-repo",
          provider_type: "github",
        },
      });
      render(
        <InfoTabContent
          selectedPodKey={pod.pod_key}
          pod={pod}
          t={mockT}
        />
      );
      expect(screen.getByText("org/my-repo")).toBeInTheDocument();
    });

    it("should display branch when available", () => {
      const pod = createMockPod({ branch_name: "feature/new-ui" });
      render(
        <InfoTabContent
          selectedPodKey={pod.pod_key}
          pod={pod}
          t={mockT}
        />
      );
      expect(screen.getByText("feature/new-ui")).toBeInTheDocument();
    });

    it("should display worktree (sandbox_path) when available", () => {
      const pod = createMockPod({
        sandbox_path: "/tmp/worktrees/feature-branch",
      });
      render(
        <InfoTabContent
          selectedPodKey={pod.pod_key}
          pod={pod}
          t={mockT}
        />
      );
      expect(
        screen.getByText("/tmp/worktrees/feature-branch")
      ).toBeInTheDocument();
    });

    it("should display ticket when available", () => {
      const pod = createMockPod({
        ticket: { id: 1, identifier: "PROJ-42", title: "Fix login bug" },
      });
      render(
        <InfoTabContent
          selectedPodKey={pod.pod_key}
          pod={pod}
          t={mockT}
        />
      );
      expect(
        screen.getByText("PROJ-42 - Fix login bug")
      ).toBeInTheDocument();
    });

    it("should display created by when available", () => {
      const pod = createMockPod({
        created_by: { id: 1, username: "john", name: "John Doe" },
      });
      render(
        <InfoTabContent
          selectedPodKey={pod.pod_key}
          pod={pod}
          t={mockT}
        />
      );
      expect(screen.getByText("John Doe")).toBeInTheDocument();
    });

    it("should fall back to username when name is not available", () => {
      const pod = createMockPod({
        created_by: { id: 1, username: "john" },
      });
      render(
        <InfoTabContent
          selectedPodKey={pod.pod_key}
          pod={pod}
          t={mockT}
        />
      );
      expect(screen.getByText("john")).toBeInTheDocument();
    });

    it("should display started at when available", () => {
      const pod = createMockPod({
        started_at: "2026-01-15T10:05:00Z",
      });
      render(
        <InfoTabContent
          selectedPodKey={pod.pod_key}
          pod={pod}
          t={mockT}
        />
      );
      expect(
        screen.getByText("ide.bottomPanel.infoTab.startedAt:")
      ).toBeInTheDocument();
    });

    it("should not display optional fields when absent", () => {
      const pod = createMockPod({
        agent_type: undefined,
        runner: undefined,
        repository: undefined,
        branch_name: undefined,
        sandbox_path: undefined,
        ticket: undefined,
        created_by: undefined,
        started_at: undefined,
      });
      render(
        <InfoTabContent
          selectedPodKey={pod.pod_key}
          pod={pod}
          t={mockT}
        />
      );
      expect(
        screen.queryByText("ide.bottomPanel.infoTab.agentType:")
      ).not.toBeInTheDocument();
      expect(
        screen.queryByText("ide.bottomPanel.infoTab.runner:")
      ).not.toBeInTheDocument();
      expect(
        screen.queryByText("ide.bottomPanel.infoTab.repository:")
      ).not.toBeInTheDocument();
      expect(
        screen.queryByText("ide.bottomPanel.infoTab.branch:")
      ).not.toBeInTheDocument();
      expect(
        screen.queryByText("ide.bottomPanel.infoTab.worktree:")
      ).not.toBeInTheDocument();
      expect(
        screen.queryByText("ide.bottomPanel.infoTab.ticket:")
      ).not.toBeInTheDocument();
      expect(
        screen.queryByText("ide.bottomPanel.infoTab.createdBy:")
      ).not.toBeInTheDocument();
      expect(
        screen.queryByText("ide.bottomPanel.infoTab.startedAt:")
      ).not.toBeInTheDocument();
    });
  });

  describe("error display", () => {
    it("should display error message when present", () => {
      const pod = createMockPod({
        error_message: "Connection timeout",
        error_code: "TIMEOUT",
      });
      render(
        <InfoTabContent
          selectedPodKey={pod.pod_key}
          pod={pod}
          t={mockT}
        />
      );
      expect(
        screen.getByText("[TIMEOUT] Connection timeout")
      ).toBeInTheDocument();
    });

    it("should display error message without code when code is absent", () => {
      const pod = createMockPod({
        error_message: "Unknown error occurred",
      });
      render(
        <InfoTabContent
          selectedPodKey={pod.pod_key}
          pod={pod}
          t={mockT}
        />
      );
      expect(
        screen.getByText("Unknown error occurred")
      ).toBeInTheDocument();
    });

    it("should not display error section when no error", () => {
      const pod = createMockPod();
      render(
        <InfoTabContent
          selectedPodKey={pod.pod_key}
          pod={pod}
          t={mockT}
        />
      );
      expect(
        screen.queryByText("ide.bottomPanel.infoTab.error:")
      ).not.toBeInTheDocument();
    });
  });

  describe("related pods", () => {
    it("should display related pods sharing the same ticket", () => {
      const ticket = { id: 10, identifier: "PROJ-10", title: "Shared task" };
      const pod = createMockPod({
        pod_key: "pod-main",
        ticket,
      });
      const relatedPod1 = createMockPod({
        id: 2,
        pod_key: "pod-related-1",
        status: "running",
        ticket,
        agent_type: { id: 2, name: "Aider", slug: "aider" },
      });
      const relatedPod2 = createMockPod({
        id: 3,
        pod_key: "pod-related-2",
        status: "completed",
        ticket,
        agent_type: { id: 1, name: "Claude Code", slug: "claude-code" },
      });

      mockPods = [pod, relatedPod1, relatedPod2];

      render(
        <InfoTabContent
          selectedPodKey={pod.pod_key}
          pod={pod}
          t={mockT}
        />
      );

      // Should show related pods header with count
      expect(
        screen.getByText("ide.bottomPanel.infoTab.relatedPods".replace("{count}", "2"))
      ).toBeInTheDocument();

      // Should show agent types for related pods
      expect(screen.getByText("Aider")).toBeInTheDocument();
      expect(screen.getByText("Claude Code")).toBeInTheDocument();
    });

    it("should not display related pods section when no ticket", () => {
      const pod = createMockPod({ ticket: undefined });
      mockPods = [pod];

      render(
        <InfoTabContent
          selectedPodKey={pod.pod_key}
          pod={pod}
          t={mockT}
        />
      );

      expect(
        screen.queryByText(/ide\.bottomPanel\.infoTab\.relatedPods/)
      ).not.toBeInTheDocument();
    });

    it("should not display related pods section when no other pods share the ticket", () => {
      const pod = createMockPod({
        ticket: { id: 10, identifier: "PROJ-10", title: "Solo task" },
      });
      mockPods = [pod];

      render(
        <InfoTabContent
          selectedPodKey={pod.pod_key}
          pod={pod}
          t={mockT}
        />
      );

      expect(
        screen.queryByText(/ide\.bottomPanel\.infoTab\.relatedPods/)
      ).not.toBeInTheDocument();
    });

    it("should not include the current pod in related pods list", () => {
      const ticket = { id: 10, identifier: "PROJ-10", title: "Task" };
      const pod = createMockPod({
        pod_key: "pod-current",
        ticket,
        agent_type: { id: 1, name: "CurrentAgent", slug: "current" },
      });
      const otherPod = createMockPod({
        id: 2,
        pod_key: "pod-other",
        ticket,
        agent_type: { id: 2, name: "OtherAgent", slug: "other" },
      });

      mockPods = [pod, otherPod];

      render(
        <InfoTabContent
          selectedPodKey={pod.pod_key}
          pod={pod}
          t={mockT}
        />
      );

      // Only 1 related pod (not including self)
      expect(
        screen.getByText("ide.bottomPanel.infoTab.relatedPods".replace("{count}", "1"))
      ).toBeInTheDocument();
      expect(screen.getByText("OtherAgent")).toBeInTheDocument();
    });
  });

  describe("pod status variants", () => {
    it.each([
      ["initializing", "Initializing"],
      ["running", "Running"],
      ["paused", "Paused"],
      ["terminated", "Terminated"],
      ["failed", "Failed"],
    ] as const)(
      "should display correct status badge for %s",
      (status, expectedLabel) => {
        const pod = createMockPod({ status });
        render(
          <InfoTabContent
            selectedPodKey={pod.pod_key}
            pod={pod}
            t={mockT}
          />
        );
        expect(screen.getByText(expectedLabel)).toBeInTheDocument();
      }
    );
  });
});
