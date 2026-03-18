import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, act } from "@testing-library/react";
import { MutableRefObject } from "react";
import { AutopilotStartButton } from "../AutopilotStartButton";

// Mock stores
let mockPods: Array<{ pod_key: string; name?: string }> = [];

vi.mock("@/stores/pod", () => ({
  usePodStore: (selector?: (s: Record<string, unknown>) => unknown) => {
    const state = {
      pods: mockPods,
    };
    return selector ? selector(state) : state;
  },
}));

vi.mock("@/lib/pod-utils", () => ({
  getPodDisplayName: (pod: { name?: string; pod_key: string }) => pod.name || pod.pod_key,
}));

vi.mock("@/components/autopilot", () => ({
  CreateAutopilotControllerModal: (props: { open: boolean; onClose: () => void; podKey: string; podTitle?: string }) => {
    return props.open ? (
      <div data-testid="modal">
        <span data-testid="modal-pod-key">{props.podKey}</span>
        <span data-testid="modal-pod-title">{props.podTitle}</span>
        <button data-testid="modal-close" onClick={props.onClose}>Close</button>
      </div>
    ) : null;
  },
}));

describe("AutopilotStartButton", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockPods = [{ pod_key: "pod-1", name: "My Pod" }];
  });

  it("does not render modal when trigger has not been called", () => {
    const triggerRef: MutableRefObject<(() => void) | null> = { current: null };
    render(<AutopilotStartButton podKey="pod-1" triggerRef={triggerRef} />);

    expect(screen.queryByTestId("modal")).not.toBeInTheDocument();
  });

  it("exposes trigger function via triggerRef", () => {
    const triggerRef: MutableRefObject<(() => void) | null> = { current: null };
    render(<AutopilotStartButton podKey="pod-1" triggerRef={triggerRef} />);

    expect(triggerRef.current).toBeInstanceOf(Function);
  });

  it("opens modal when trigger is called", () => {
    const triggerRef: MutableRefObject<(() => void) | null> = { current: null };
    render(<AutopilotStartButton podKey="pod-1" triggerRef={triggerRef} />);

    act(() => {
      triggerRef.current!();
    });

    expect(screen.getByTestId("modal")).toBeInTheDocument();
  });

  it("passes correct podKey to modal", () => {
    const triggerRef: MutableRefObject<(() => void) | null> = { current: null };
    render(<AutopilotStartButton podKey="pod-1" triggerRef={triggerRef} />);

    act(() => {
      triggerRef.current!();
    });

    expect(screen.getByTestId("modal-pod-key")).toHaveTextContent("pod-1");
  });

  it("passes pod title derived from store", () => {
    const triggerRef: MutableRefObject<(() => void) | null> = { current: null };
    render(<AutopilotStartButton podKey="pod-1" triggerRef={triggerRef} />);

    act(() => {
      triggerRef.current!();
    });

    expect(screen.getByTestId("modal-pod-title")).toHaveTextContent("My Pod");
  });

  it("falls back to truncated podKey when pod not found in store", () => {
    mockPods = [];
    const triggerRef: MutableRefObject<(() => void) | null> = { current: null };
    render(<AutopilotStartButton podKey="abcdef12-3456" triggerRef={triggerRef} />);

    act(() => {
      triggerRef.current!();
    });

    expect(screen.getByTestId("modal-pod-title")).toHaveTextContent("abcdef12");
  });

  it("closes modal when onClose is called", () => {
    const triggerRef: MutableRefObject<(() => void) | null> = { current: null };
    render(<AutopilotStartButton podKey="pod-1" triggerRef={triggerRef} />);

    // Open
    act(() => {
      triggerRef.current!();
    });
    expect(screen.getByTestId("modal")).toBeInTheDocument();

    // Close
    act(() => {
      screen.getByTestId("modal-close").click();
    });
    expect(screen.queryByTestId("modal")).not.toBeInTheDocument();
  });

  it("cleans up triggerRef on unmount", () => {
    const triggerRef: MutableRefObject<(() => void) | null> = { current: null };
    const { unmount } = render(<AutopilotStartButton podKey="pod-1" triggerRef={triggerRef} />);

    expect(triggerRef.current).toBeInstanceOf(Function);

    unmount();

    expect(triggerRef.current).toBeNull();
  });
});
