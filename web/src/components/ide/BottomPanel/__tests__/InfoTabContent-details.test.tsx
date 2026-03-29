import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { InfoTabContent } from "../InfoTabContent";
import type { PodData } from "@/lib/api/pod";

// Mock pod store
let mockPods: PodData[] = [];

vi.mock("@/stores/pod", () => ({
  usePodStore: (selector?: (s: Record<string, unknown>) => unknown) => {
    const state = { pods: mockPods };
    return selector ? selector(state) : state;
  },
}));

const mockT = (key: string, params?: Record<string, string | number>) => {
  if (params) {
    return Object.entries(params).reduce(
      (str, [k, v]) => str.replace(`{${k}}`, String(v)),
      key
    );
  }
  return key;
};

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

describe("InfoTabContent - optional fields and related pods", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockPods = [];
  });

  describe("optional fields", () => {
    it("should display agent when available", () => {
      const pod = createMockPod({
        agent: { name: "Claude Code", slug: "claude-code" },
      });
      render(
        <InfoTabContent
          selectedPodKey={pod.pod_key}
          pod={pod}
          orgSlug="test-org"
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
          orgSlug="test-org"
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
          orgSlug="test-org"
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
          slug: "org/my-repo",
          provider_type: "github",
        },
      });
      render(
        <InfoTabContent
          selectedPodKey={pod.pod_key}
          pod={pod}
          orgSlug="test-org"
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
          orgSlug="test-org"
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
          orgSlug="test-org"
          t={mockT}
        />
      );
      expect(
        screen.getByText("/tmp/worktrees/feature-branch")
      ).toBeInTheDocument();
    });

    it("should display ticket when available", () => {
      const pod = createMockPod({
        ticket: { id: 1, slug: "PROJ-42", title: "Fix login bug" },
      });
      render(
        <InfoTabContent
          selectedPodKey={pod.pod_key}
          pod={pod}
          orgSlug="test-org"
          t={mockT}
        />
      );
      expect(
        screen.getByText("PROJ-42 - Fix login bug")
      ).toBeInTheDocument();
    });

    it("should render repository as a clickable link", () => {
      const pod = createMockPod({
        repository: {
          id: 5,
          name: "my-repo",
          slug: "org/my-repo",
          provider_type: "github",
        },
      });
      render(
        <InfoTabContent
          selectedPodKey={pod.pod_key}
          pod={pod}
          orgSlug="test-org"
          t={mockT}
        />
      );
      const repoLink = screen.getByText("org/my-repo");
      expect(repoLink.closest("a")).toHaveAttribute(
        "href",
        "/test-org/repositories/5"
      );
    });

    it("should render ticket as a clickable link", () => {
      const pod = createMockPod({
        ticket: { id: 1, slug: "PROJ-42", title: "Fix login bug" },
      });
      render(
        <InfoTabContent
          selectedPodKey={pod.pod_key}
          pod={pod}
          orgSlug="test-org"
          t={mockT}
        />
      );
      const ticketLink = screen.getByText("PROJ-42 - Fix login bug");
      expect(ticketLink.closest("a")).toHaveAttribute(
        "href",
        "/test-org/tickets/PROJ-42"
      );
    });

    it("should display created by when available", () => {
      const pod = createMockPod({
        created_by: { id: 1, username: "john", name: "John Doe" },
      });
      render(
        <InfoTabContent
          selectedPodKey={pod.pod_key}
          pod={pod}
          orgSlug="test-org"
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
          orgSlug="test-org"
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
          orgSlug="test-org"
          t={mockT}
        />
      );
      expect(
        screen.getByText("ide.bottomPanel.infoTab.startedAt:")
      ).toBeInTheDocument();
    });

    it("should not display optional fields when absent", () => {
      const pod = createMockPod({
        agent: undefined,
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
          orgSlug="test-org"
          t={mockT}
        />
      );
      expect(
        screen.queryByText("ide.bottomPanel.infoTab.agent:")
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

  describe("related pods", () => {
    it("should display related pods sharing the same ticket", () => {
      const ticket = { id: 10, slug: "PROJ-10", title: "Shared task" };
      const pod = createMockPod({
        pod_key: "pod-main",
        ticket,
      });
      const relatedPod1 = createMockPod({
        id: 2,
        pod_key: "pod-related-1",
        status: "running",
        ticket,
        agent: { name: "Aider", slug: "aider" },
      });
      const relatedPod2 = createMockPod({
        id: 3,
        pod_key: "pod-related-2",
        status: "completed",
        ticket,
        agent: { name: "Claude Code", slug: "claude-code" },
      });

      mockPods = [pod, relatedPod1, relatedPod2];

      render(
        <InfoTabContent
          selectedPodKey={pod.pod_key}
          pod={pod}
          orgSlug="test-org"
          t={mockT}
        />
      );

      expect(
        screen.getByText("ide.bottomPanel.infoTab.relatedPods".replace("{count}", "2"))
      ).toBeInTheDocument();
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
          orgSlug="test-org"
          t={mockT}
        />
      );

      expect(
        screen.queryByText(/ide\.bottomPanel\.infoTab\.relatedPods/)
      ).not.toBeInTheDocument();
    });

    it("should not display related pods section when no other pods share the ticket", () => {
      const pod = createMockPod({
        ticket: { id: 10, slug: "PROJ-10", title: "Solo task" },
      });
      mockPods = [pod];

      render(
        <InfoTabContent
          selectedPodKey={pod.pod_key}
          pod={pod}
          orgSlug="test-org"
          t={mockT}
        />
      );

      expect(
        screen.queryByText(/ide\.bottomPanel\.infoTab\.relatedPods/)
      ).not.toBeInTheDocument();
    });

    it("should not include the current pod in related pods list", () => {
      const ticket = { id: 10, slug: "PROJ-10", title: "Task" };
      const pod = createMockPod({
        pod_key: "pod-current",
        ticket,
        agent: { name: "CurrentAgent", slug: "current" },
      });
      const otherPod = createMockPod({
        id: 2,
        pod_key: "pod-other",
        ticket,
        agent: { name: "OtherAgent", slug: "other" },
      });

      mockPods = [pod, otherPod];

      render(
        <InfoTabContent
          selectedPodKey={pod.pod_key}
          pod={pod}
          orgSlug="test-org"
          t={mockT}
        />
      );

      expect(
        screen.getByText("ide.bottomPanel.infoTab.relatedPods".replace("{count}", "1"))
      ).toBeInTheDocument();
      expect(screen.getByText("OtherAgent")).toBeInTheDocument();
    });
  });
});
