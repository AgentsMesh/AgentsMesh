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
        <InfoTabContent selectedPodKey={null} pod={null} orgSlug="test-org" t={mockT} />
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
          orgSlug="test-org"
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
          orgSlug="test-org"
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
          orgSlug="test-org"
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
          orgSlug="test-org"
          t={mockT}
        />
      );
      expect(
        screen.getByText("ide.bottomPanel.infoTab.createdAt:")
      ).toBeInTheDocument();
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
          orgSlug="test-org"
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
          orgSlug="test-org"
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
          orgSlug="test-org"
          t={mockT}
        />
      );
      expect(
        screen.queryByText("ide.bottomPanel.infoTab.error:")
      ).not.toBeInTheDocument();
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
            orgSlug="test-org"
            t={mockT}
          />
        );
        expect(screen.getByText(expectedLabel)).toBeInTheDocument();
      }
    );
  });
});
