import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Dialog, DialogContent } from "../dialog";

describe("Dialog", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders children in a portal overlay when open", () => {
    render(
      <Dialog open={true} onOpenChange={vi.fn()}>
        <DialogContent>
          <div>Dialog Content</div>
        </DialogContent>
      </Dialog>
    );

    expect(screen.getByText("Dialog Content")).toBeInTheDocument();
  });

  it("does not render when closed", () => {
    render(
      <Dialog open={false} onOpenChange={vi.fn()}>
        <DialogContent>
          <div>Hidden Content</div>
        </DialogContent>
      </Dialog>
    );

    expect(screen.queryByText("Hidden Content")).not.toBeInTheDocument();
  });

  it("marks overlay with data-dialog-overlay attribute", () => {
    render(
      <Dialog open={true} onOpenChange={vi.fn()}>
        <DialogContent>
          <div>Content</div>
        </DialogContent>
      </Dialog>
    );

    const overlay = document.querySelector("[data-dialog-overlay]");
    expect(overlay).toBeInTheDocument();
    expect(overlay).toHaveClass("fixed", "inset-0", "z-50");
  });

  it("removes data-dialog-overlay from DOM when closed", () => {
    const { rerender } = render(
      <Dialog open={true} onOpenChange={vi.fn()}>
        <DialogContent>
          <div>Content</div>
        </DialogContent>
      </Dialog>
    );

    expect(document.querySelector("[data-dialog-overlay]")).toBeInTheDocument();

    rerender(
      <Dialog open={false} onOpenChange={vi.fn()}>
        <DialogContent>
          <div>Content</div>
        </DialogContent>
      </Dialog>
    );

    expect(document.querySelector("[data-dialog-overlay]")).not.toBeInTheDocument();
  });

  it("calls onOpenChange(false) on overlay click", () => {
    const onOpenChange = vi.fn();
    render(
      <Dialog open={true} onOpenChange={onOpenChange}>
        <DialogContent>
          <div>Content</div>
        </DialogContent>
      </Dialog>
    );

    const overlay = document.querySelector("[data-dialog-overlay]")!;
    fireEvent.click(overlay);
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("does not close on content click (stopPropagation)", () => {
    const onOpenChange = vi.fn();
    render(
      <Dialog open={true} onOpenChange={onOpenChange}>
        <DialogContent>
          <div>Content</div>
        </DialogContent>
      </Dialog>
    );

    fireEvent.click(screen.getByText("Content"));
    expect(onOpenChange).not.toHaveBeenCalled();
  });

  it("calls onOpenChange(false) on Escape key", () => {
    const onOpenChange = vi.fn();
    render(
      <Dialog open={true} onOpenChange={onOpenChange}>
        <DialogContent>
          <div>Content</div>
        </DialogContent>
      </Dialog>
    );

    fireEvent.keyDown(document, { key: "Escape" });
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});

describe("DialogContent", () => {
  it("renders title and description when provided", () => {
    render(
      <Dialog open={true} onOpenChange={vi.fn()}>
        <DialogContent title="My Title" description="My Description">
          <div>Body</div>
        </DialogContent>
      </Dialog>
    );

    expect(screen.getByText("My Title")).toBeInTheDocument();
    expect(screen.getByText("My Description")).toBeInTheDocument();
  });

  it("merges custom className", () => {
    render(
      <Dialog open={true} onOpenChange={vi.fn()}>
        <DialogContent className="max-w-sm">
          <div>Content</div>
        </DialogContent>
      </Dialog>
    );

    const content = screen.getByText("Content").closest(".bg-background")!;
    expect(content.className).toContain("max-w-sm");
  });
});
