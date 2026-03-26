import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { TerminalLoadingState, TerminalErrorState } from "../TerminalStateViews";

describe("TerminalLoadingState", () => {
  const defaultProps = {
    podStatus: "initializing",
    onClose: vi.fn(),
  };

  describe("loading state (non-completed)", () => {
    it("shows spinner and waiting message", () => {
      render(<TerminalLoadingState {...defaultProps} />);

      expect(screen.getByText("Waiting for Pod to be ready...")).toBeInTheDocument();
      expect(screen.queryByText("Pod completed")).not.toBeInTheDocument();
    });

    it("shows status text with yellow styling", () => {
      render(<TerminalLoadingState {...defaultProps} podStatus="initializing" />);

      const statusText = screen.getByText("initializing");
      expect(statusText).toBeInTheDocument();
      expect(statusText).toHaveClass("text-yellow-500");
    });

    it("does not show close button for initializing status", () => {
      render(<TerminalLoadingState {...defaultProps} podStatus="initializing" />);

      expect(screen.queryByText("Close Terminal")).not.toBeInTheDocument();
    });

    it("does not show close button for running status", () => {
      render(<TerminalLoadingState {...defaultProps} podStatus="running" />);

      expect(screen.queryByText("Close Terminal")).not.toBeInTheDocument();
    });

    it("shows init progress when provided", () => {
      const initProgress = { progress: 50, phase: "Cloning", message: "Cloning repository..." };
      render(<TerminalLoadingState {...defaultProps} initProgress={initProgress} />);

      expect(screen.getByText("Cloning repository...")).toBeInTheDocument();
      expect(screen.getByText("Cloning - 50%")).toBeInTheDocument();
    });
  });

  describe("unknown status", () => {
    it("shows close button", () => {
      render(<TerminalLoadingState {...defaultProps} podStatus="unknown" />);

      expect(screen.getByText("Close Terminal")).toBeInTheDocument();
    });

    it("calls onClose when close button is clicked", () => {
      const onClose = vi.fn();
      render(<TerminalLoadingState {...defaultProps} podStatus="unknown" onClose={onClose} />);

      fireEvent.click(screen.getByText("Close Terminal"));
      expect(onClose).toHaveBeenCalledTimes(1);
    });
  });

  describe("completed status", () => {
    it("shows 'Pod completed' text instead of waiting message", () => {
      render(<TerminalLoadingState {...defaultProps} podStatus="completed" />);

      expect(screen.getByText("Pod completed")).toBeInTheDocument();
      expect(screen.queryByText("Waiting for Pod to be ready...")).not.toBeInTheDocument();
    });

    it("shows status text with green styling", () => {
      render(<TerminalLoadingState {...defaultProps} podStatus="completed" />);

      const statusText = screen.getByText("completed");
      expect(statusText).toBeInTheDocument();
      expect(statusText).toHaveClass("text-green-500");
    });

    it("shows close button", () => {
      render(<TerminalLoadingState {...defaultProps} podStatus="completed" />);

      expect(screen.getByText("Close Terminal")).toBeInTheDocument();
    });

    it("calls onClose when close button is clicked", () => {
      const onClose = vi.fn();
      render(<TerminalLoadingState {...defaultProps} podStatus="completed" onClose={onClose} />);

      fireEvent.click(screen.getByText("Close Terminal"));
      expect(onClose).toHaveBeenCalledTimes(1);
    });

    it("does not show close button when onClose is not provided", () => {
      render(
        <TerminalLoadingState
          podStatus="completed"
        />
      );

      expect(screen.queryByText("Close Terminal")).not.toBeInTheDocument();
    });
  });
});

describe("TerminalErrorState", () => {
  it("shows error message", () => {
    render(<TerminalErrorState error="Pod failed" />);

    expect(screen.getByText("Pod failed")).toBeInTheDocument();
    expect(
      screen.getByText("The pod cannot be connected. Please check the pod status or create a new one.")
    ).toBeInTheDocument();
  });

  it("shows close button when onClose is provided", () => {
    render(<TerminalErrorState error="Pod failed" onClose={vi.fn()} />);

    expect(screen.getByText("Close Terminal")).toBeInTheDocument();
  });

  it("calls onClose when close button is clicked", () => {
    const onClose = vi.fn();
    render(<TerminalErrorState error="Pod failed" onClose={onClose} />);

    fireEvent.click(screen.getByText("Close Terminal"));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("does not show close button when onClose is not provided", () => {
    render(<TerminalErrorState error="Pod terminated" />);

    expect(screen.queryByText("Close Terminal")).not.toBeInTheDocument();
  });
});
