import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { TerminalGrid } from "../TerminalGrid";
import { useWorkspaceStore } from "@/stores/workspace";
import type { GridLayout } from "@/stores/workspace";

// Mock TerminalPane component
vi.mock("../TerminalPane", () => ({
  TerminalPane: ({
    paneId,
    podKey,
    isActive,
    onClose,
    onPopout,
  }: {
    paneId: string;
    podKey: string;
    isActive: boolean;
    onClose?: () => void;
    onPopout?: () => void;
  }) => (
    <div
      data-testid={`terminal-pane-${paneId}`}
      data-pod-key={podKey}
      data-active={isActive}
    >
      <span>Terminal: {podKey}</span>
      {onClose && (
        <button data-testid={`close-${paneId}`} onClick={onClose}>
          Close
        </button>
      )}
      {onPopout && (
        <button data-testid={`popout-${paneId}`} onClick={onPopout}>
          Popout
        </button>
      )}
    </div>
  ),
}));

// Mock react-resizable-panels
vi.mock("react-resizable-panels", () => ({
  Group: ({
    children,
    className,
    orientation,
  }: {
    children: React.ReactNode;
    className?: string;
    orientation?: string;
  }) => (
    <div data-testid="panel-group" data-orientation={orientation} className={className}>
      {children}
    </div>
  ),
  Panel: ({
    children,
    defaultSize,
    minSize,
  }: {
    children: React.ReactNode;
    defaultSize?: number;
    minSize?: number;
  }) => (
    <div data-testid="panel" data-default-size={defaultSize} data-min-size={minSize}>
      {children}
    </div>
  ),
  Separator: ({
    children,
    className,
  }: {
    children?: React.ReactNode;
    className?: string;
  }) => (
    <div data-testid="separator" className={className}>
      {children}
    </div>
  ),
}));

describe("TerminalGrid", () => {
  beforeEach(() => {
    // Reset store to initial state
    useWorkspaceStore.setState({
      panes: [],
      activePane: null,
      gridLayout: { type: "1x1", rows: 1, cols: 1 },
      mobileActiveIndex: 0,
      terminalFontSize: 14,
      _hasHydrated: false,
    });
  });

  describe("Empty State", () => {
    it("should render empty state when no panes exist", () => {
      render(<TerminalGrid />);

      expect(screen.getByText("No terminals open")).toBeInTheDocument();
      expect(screen.getByText("Open a pod to start a terminal session")).toBeInTheDocument();
    });

    it("should render 'Open Terminal' button when onAddNew is provided", () => {
      const onAddNew = vi.fn();
      render(<TerminalGrid onAddNew={onAddNew} />);

      const button = screen.getByRole("button", { name: /open terminal/i });
      expect(button).toBeInTheDocument();

      fireEvent.click(button);
      expect(onAddNew).toHaveBeenCalledTimes(1);
    });

    it("should not render 'Open Terminal' button when onAddNew is not provided", () => {
      render(<TerminalGrid />);

      expect(screen.queryByRole("button", { name: /open terminal/i })).not.toBeInTheDocument();
    });

    it("should apply custom className to empty state", () => {
      const { container } = render(<TerminalGrid className="custom-class" />);

      const emptyState = container.firstChild as HTMLElement;
      expect(emptyState).toHaveClass("custom-class");
    });
  });

  describe("1x1 Layout", () => {
    beforeEach(() => {
      useWorkspaceStore.setState({
        panes: [
          { id: "pane-1", podKey: "pod-1" },
        ],
        activePane: "pane-1",
        gridLayout: { type: "1x1", rows: 1, cols: 1 },
      });
    });

    it("should render single pane in 1x1 layout", () => {
      render(<TerminalGrid />);

      expect(screen.getByTestId("terminal-pane-pane-1")).toBeInTheDocument();
      expect(screen.getByText("Terminal: pod-1")).toBeInTheDocument();
    });

    it("should not render PanelGroup in 1x1 layout", () => {
      render(<TerminalGrid />);

      expect(screen.queryByTestId("panel-group")).not.toBeInTheDocument();
    });

    it("should apply custom className to 1x1 layout", () => {
      const { container } = render(<TerminalGrid className="custom-class" />);

      const wrapper = container.firstChild as HTMLElement;
      expect(wrapper).toHaveClass("custom-class");
    });
  });

  describe("1x2 Layout (Two Columns)", () => {
    beforeEach(() => {
      useWorkspaceStore.setState({
        panes: [
          { id: "pane-1", podKey: "pod-1" },
          { id: "pane-2", podKey: "pod-2" },
        ],
        activePane: "pane-1",
        gridLayout: { type: "1x2", rows: 1, cols: 2 },
      });
    });

    it("should render two panes in 1x2 layout", () => {
      render(<TerminalGrid />);

      expect(screen.getByTestId("terminal-pane-pane-1")).toBeInTheDocument();
      expect(screen.getByTestId("terminal-pane-pane-2")).toBeInTheDocument();
    });

    it("should render PanelGroup with horizontal orientation", () => {
      render(<TerminalGrid />);

      const panelGroup = screen.getByTestId("panel-group");
      expect(panelGroup).toHaveAttribute("data-orientation", "horizontal");
    });

    it("should render resize handle (separator)", () => {
      render(<TerminalGrid />);

      expect(screen.getByTestId("separator")).toBeInTheDocument();
    });

    it("should render panels with correct default and min sizes", () => {
      render(<TerminalGrid />);

      const panels = screen.getAllByTestId("panel");
      expect(panels).toHaveLength(2);
      panels.forEach((panel) => {
        expect(panel).toHaveAttribute("data-default-size", "50");
        expect(panel).toHaveAttribute("data-min-size", "20");
      });
    });
  });

  describe("2x1 Layout (Two Rows)", () => {
    beforeEach(() => {
      useWorkspaceStore.setState({
        panes: [
          { id: "pane-1", podKey: "pod-1" },
          { id: "pane-2", podKey: "pod-2" },
        ],
        activePane: "pane-1",
        gridLayout: { type: "2x1", rows: 2, cols: 1 },
      });
    });

    it("should render two panes in 2x1 layout", () => {
      render(<TerminalGrid />);

      expect(screen.getByTestId("terminal-pane-pane-1")).toBeInTheDocument();
      expect(screen.getByTestId("terminal-pane-pane-2")).toBeInTheDocument();
    });

    it("should render PanelGroup with vertical orientation", () => {
      render(<TerminalGrid />);

      const panelGroup = screen.getByTestId("panel-group");
      expect(panelGroup).toHaveAttribute("data-orientation", "vertical");
    });
  });

  describe("2x2 Layout (Four Grid)", () => {
    beforeEach(() => {
      useWorkspaceStore.setState({
        panes: [
          { id: "pane-1", podKey: "pod-1" },
          { id: "pane-2", podKey: "pod-2" },
          { id: "pane-3", podKey: "pod-3" },
          { id: "pane-4", podKey: "pod-4" },
        ],
        activePane: "pane-1",
        gridLayout: { type: "2x2", rows: 2, cols: 2 },
      });
    });

    it("should render four panes in 2x2 layout", () => {
      render(<TerminalGrid />);

      expect(screen.getByTestId("terminal-pane-pane-1")).toBeInTheDocument();
      expect(screen.getByTestId("terminal-pane-pane-2")).toBeInTheDocument();
      expect(screen.getByTestId("terminal-pane-pane-3")).toBeInTheDocument();
      expect(screen.getByTestId("terminal-pane-pane-4")).toBeInTheDocument();
    });

    it("should render nested PanelGroups for 2x2 layout", () => {
      render(<TerminalGrid />);

      // Should have 3 panel groups: 1 vertical outer + 2 horizontal inner
      const panelGroups = screen.getAllByTestId("panel-group");
      expect(panelGroups.length).toBeGreaterThanOrEqual(3);
    });

    it("should render multiple separators for 2x2 layout", () => {
      render(<TerminalGrid />);

      // Should have 3 separators: 1 vertical + 2 horizontal
      const separators = screen.getAllByTestId("separator");
      expect(separators.length).toBeGreaterThanOrEqual(3);
    });
  });

  describe("Fallback Layout", () => {
    it("should render fallback layout for unknown layout type", () => {
      useWorkspaceStore.setState({
        panes: [
          { id: "pane-1", podKey: "pod-1" },
        ],
        activePane: "pane-1",
        gridLayout: { type: "custom" as GridLayout["type"], rows: 3, cols: 3 },
      });

      render(<TerminalGrid />);

      // Should fallback to rendering first pane
      expect(screen.getByTestId("terminal-pane-pane-1")).toBeInTheDocument();
      expect(screen.queryByTestId("panel-group")).not.toBeInTheDocument();
    });
  });

  describe("Empty Pane Slots", () => {
    it("should render empty slot when pane count is less than grid capacity", () => {
      useWorkspaceStore.setState({
        panes: [
          { id: "pane-1", podKey: "pod-1" },
        ],
        activePane: "pane-1",
        gridLayout: { type: "1x2", rows: 1, cols: 2 },
      });

      const onAddNew = vi.fn();
      render(<TerminalGrid onAddNew={onAddNew} />);

      // Should have "Add Terminal" button in empty slot
      const addButtons = screen.getAllByRole("button", { name: /add terminal/i });
      expect(addButtons.length).toBeGreaterThanOrEqual(1);
    });

    it("should call onAddNew when clicking Add Terminal in empty slot", () => {
      useWorkspaceStore.setState({
        panes: [
          { id: "pane-1", podKey: "pod-1" },
        ],
        activePane: "pane-1",
        gridLayout: { type: "1x2", rows: 1, cols: 2 },
      });

      const onAddNew = vi.fn();
      render(<TerminalGrid onAddNew={onAddNew} />);

      const addButtons = screen.getAllByRole("button", { name: /add terminal/i });
      fireEvent.click(addButtons[0]);

      expect(onAddNew).toHaveBeenCalledTimes(1);
    });

    it("should not render Add Terminal button in empty slot when onAddNew is not provided", () => {
      useWorkspaceStore.setState({
        panes: [
          { id: "pane-1", podKey: "pod-1" },
        ],
        activePane: "pane-1",
        gridLayout: { type: "1x2", rows: 1, cols: 2 },
      });

      render(<TerminalGrid />);

      // Empty slot should exist but without button
      expect(screen.queryByRole("button", { name: /add terminal/i })).not.toBeInTheDocument();
    });
  });

  describe("Pane Interactions", () => {
    beforeEach(() => {
      useWorkspaceStore.setState({
        panes: [
          { id: "pane-1", podKey: "pod-1" },
          { id: "pane-2", podKey: "pod-2" },
        ],
        activePane: "pane-1",
        gridLayout: { type: "1x2", rows: 1, cols: 2 },
      });
    });

    it("should call removePane when close button is clicked", () => {
      render(<TerminalGrid />);

      const closeButton = screen.getByTestId("close-pane-1");
      fireEvent.click(closeButton);

      // Check that pane was removed from store
      const state = useWorkspaceStore.getState();
      expect(state.panes.find((p) => p.id === "pane-1")).toBeUndefined();
    });

    it("should call onPopout when popout button is clicked", () => {
      const onPopout = vi.fn();
      render(<TerminalGrid onPopout={onPopout} />);

      const popoutButton = screen.getByTestId("popout-pane-1");
      fireEvent.click(popoutButton);

      expect(onPopout).toHaveBeenCalledWith("pane-1");
    });

    it("should not render popout button when onPopout is not provided", () => {
      render(<TerminalGrid />);

      expect(screen.queryByTestId("popout-pane-1")).not.toBeInTheDocument();
    });

    it("should pass correct isActive prop to TerminalPane", () => {
      render(<TerminalGrid />);

      const pane1 = screen.getByTestId("terminal-pane-pane-1");
      const pane2 = screen.getByTestId("terminal-pane-pane-2");

      expect(pane1).toHaveAttribute("data-active", "true");
      expect(pane2).toHaveAttribute("data-active", "false");
    });
  });

  describe("Visible Panes Calculation", () => {
    it("should show panes around active pane when there are more panes than grid capacity", () => {
      useWorkspaceStore.setState({
        panes: [
          { id: "pane-1", podKey: "pod-1" },
          { id: "pane-2", podKey: "pod-2" },
          { id: "pane-3", podKey: "pod-3" },
          { id: "pane-4", podKey: "pod-4" },
          { id: "pane-5", podKey: "pod-5" },
        ],
        activePane: "pane-3",
        gridLayout: { type: "1x2", rows: 1, cols: 2 },
      });

      render(<TerminalGrid />);

      // With active pane at index 2 and maxVisible=2, should show panes around it
      // startIndex = max(0, 2 - floor(2/2)) = max(0, 1) = 1
      // So should show pane-2 and pane-3
      expect(screen.getByTestId("terminal-pane-pane-2")).toBeInTheDocument();
      expect(screen.getByTestId("terminal-pane-pane-3")).toBeInTheDocument();
    });

    it("should show first panes when no active pane", () => {
      useWorkspaceStore.setState({
        panes: [
          { id: "pane-1", podKey: "pod-1" },
          { id: "pane-2", podKey: "pod-2" },
          { id: "pane-3", podKey: "pod-3" },
        ],
        activePane: null,
        gridLayout: { type: "1x2", rows: 1, cols: 2 },
      });

      render(<TerminalGrid />);

      expect(screen.getByTestId("terminal-pane-pane-1")).toBeInTheDocument();
      expect(screen.getByTestId("terminal-pane-pane-2")).toBeInTheDocument();
      expect(screen.queryByTestId("terminal-pane-pane-3")).not.toBeInTheDocument();
    });

    it("should show first panes when active pane is not found", () => {
      useWorkspaceStore.setState({
        panes: [
          { id: "pane-1", podKey: "pod-1" },
          { id: "pane-2", podKey: "pod-2" },
        ],
        activePane: "non-existent-pane",
        gridLayout: { type: "1x2", rows: 1, cols: 2 },
      });

      render(<TerminalGrid />);

      expect(screen.getByTestId("terminal-pane-pane-1")).toBeInTheDocument();
      expect(screen.getByTestId("terminal-pane-pane-2")).toBeInTheDocument();
    });
  });

  describe("ResizeHandle Component", () => {
    it("should render horizontal resize handle with correct classes", () => {
      useWorkspaceStore.setState({
        panes: [
          { id: "pane-1", podKey: "pod-1" },
          { id: "pane-2", podKey: "pod-2" },
        ],
        activePane: "pane-1",
        gridLayout: { type: "1x2", rows: 1, cols: 2 },
      });

      render(<TerminalGrid />);

      const separator = screen.getByTestId("separator");
      expect(separator).toHaveClass("cursor-col-resize");
    });

    it("should render vertical resize handle with correct classes", () => {
      useWorkspaceStore.setState({
        panes: [
          { id: "pane-1", podKey: "pod-1" },
          { id: "pane-2", podKey: "pod-2" },
        ],
        activePane: "pane-1",
        gridLayout: { type: "2x1", rows: 2, cols: 1 },
      });

      render(<TerminalGrid />);

      const separator = screen.getByTestId("separator");
      expect(separator).toHaveClass("cursor-row-resize");
    });
  });
});
