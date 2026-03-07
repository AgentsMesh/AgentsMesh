import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import {
  ResponsiveDialog,
  ResponsiveDialogContent,
  ResponsiveDialogHeader,
  ResponsiveDialogBody,
  ResponsiveDialogFooter,
} from "../responsive-dialog";

describe("ResponsiveDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders children in a portal overlay when open", () => {
    render(
      <ResponsiveDialog open={true} onOpenChange={vi.fn()}>
        <ResponsiveDialogContent>
          <div>Dialog Content</div>
        </ResponsiveDialogContent>
      </ResponsiveDialog>
    );

    expect(screen.getByText("Dialog Content")).toBeInTheDocument();
  });

  it("does not render when closed", () => {
    render(
      <ResponsiveDialog open={false} onOpenChange={vi.fn()}>
        <ResponsiveDialogContent>
          <div>Hidden Content</div>
        </ResponsiveDialogContent>
      </ResponsiveDialog>
    );

    expect(screen.queryByText("Hidden Content")).not.toBeInTheDocument();
  });
});

describe("ResponsiveDialogContent", () => {
  it("applies max-h-[90vh] and overflow-hidden", () => {
    render(
      <ResponsiveDialog open={true} onOpenChange={vi.fn()}>
        <ResponsiveDialogContent>
          <div>Content</div>
        </ResponsiveDialogContent>
      </ResponsiveDialog>
    );

    const content = screen.getByText("Content").parentElement!;
    expect(content.className).toContain("max-h-[90vh]");
    expect(content.className).toContain("overflow-hidden");
  });

  it("applies border and rounded-lg for floating style", () => {
    render(
      <ResponsiveDialog open={true} onOpenChange={vi.fn()}>
        <ResponsiveDialogContent>
          <div>Content</div>
        </ResponsiveDialogContent>
      </ResponsiveDialog>
    );

    const content = screen.getByText("Content").parentElement!;
    expect(content.className).toContain("border");
    expect(content.className).toContain("rounded-lg");
    expect(content.className).toContain("shadow-lg");
  });

  it("merges custom className", () => {
    render(
      <ResponsiveDialog open={true} onOpenChange={vi.fn()}>
        <ResponsiveDialogContent className="max-w-md">
          <div>Content</div>
        </ResponsiveDialogContent>
      </ResponsiveDialog>
    );

    const content = screen.getByText("Content").parentElement!;
    expect(content.className).toContain("max-w-md");
  });
});

describe("ResponsiveDialogHeader", () => {
  it("has flex-shrink-0 to prevent collapsing", () => {
    render(
      <ResponsiveDialogHeader>
        <div>Header</div>
      </ResponsiveDialogHeader>
    );

    const header = screen.getByText("Header").parentElement!;
    expect(header.className).toContain("flex-shrink-0");
  });

  it("uses responsive padding (px-4 md:px-6)", () => {
    render(
      <ResponsiveDialogHeader>
        <div>Header</div>
      </ResponsiveDialogHeader>
    );

    const header = screen.getByText("Header").parentElement!;
    expect(header.className).toContain("px-4");
    expect(header.className).toContain("md:px-6");
  });

  it("renders close button when onClose is provided", () => {
    const onClose = vi.fn();
    render(
      <ResponsiveDialogHeader onClose={onClose}>
        <div>Header</div>
      </ResponsiveDialogHeader>
    );

    expect(screen.getByRole("button", { name: "Close" })).toBeInTheDocument();
  });

  it("does not render close button when onClose is not provided", () => {
    render(
      <ResponsiveDialogHeader>
        <div>Header</div>
      </ResponsiveDialogHeader>
    );

    expect(screen.queryByRole("button", { name: "Close" })).not.toBeInTheDocument();
  });
});

describe("ResponsiveDialogBody", () => {
  it("has overflow-y-auto as the scroll container", () => {
    render(
      <ResponsiveDialogBody>
        <div>Body Content</div>
      </ResponsiveDialogBody>
    );

    const body = screen.getByText("Body Content").parentElement!;
    expect(body.className).toContain("overflow-y-auto");
  });

  it("has min-h-0 for proper flex shrinking", () => {
    render(
      <ResponsiveDialogBody>
        <div>Body Content</div>
      </ResponsiveDialogBody>
    );

    const body = screen.getByText("Body Content").parentElement!;
    expect(body.className).toContain("min-h-0");
  });

  it("has overscroll-contain to prevent scroll chaining", () => {
    render(
      <ResponsiveDialogBody>
        <div>Body Content</div>
      </ResponsiveDialogBody>
    );

    const body = screen.getByText("Body Content").parentElement!;
    expect(body.className).toContain("overscroll-contain");
  });

  it("uses responsive padding (px-4 md:px-6)", () => {
    render(
      <ResponsiveDialogBody>
        <div>Body Content</div>
      </ResponsiveDialogBody>
    );

    const body = screen.getByText("Body Content").parentElement!;
    expect(body.className).toContain("px-4");
    expect(body.className).toContain("md:px-6");
  });
});

describe("ResponsiveDialogFooter", () => {
  it("uses responsive layout classes", () => {
    render(
      <ResponsiveDialogFooter>
        <button>Cancel</button>
        <button>Submit</button>
      </ResponsiveDialogFooter>
    );

    const footer = screen.getByText("Cancel").parentElement!;
    expect(footer.className).toContain("flex-col-reverse");
    expect(footer.className).toContain("md:flex-row");
    expect(footer.className).toContain("md:justify-end");
  });

  it("uses responsive padding (px-4 md:px-6)", () => {
    render(
      <ResponsiveDialogFooter>
        <button>Cancel</button>
      </ResponsiveDialogFooter>
    );

    const footer = screen.getByText("Cancel").parentElement!;
    expect(footer.className).toContain("px-4");
    expect(footer.className).toContain("md:px-6");
  });

  it("has flex-shrink-0 to prevent collapsing", () => {
    render(
      <ResponsiveDialogFooter>
        <button>Actions</button>
      </ResponsiveDialogFooter>
    );

    const footer = screen.getByText("Actions").parentElement!;
    expect(footer.className).toContain("flex-shrink-0");
  });
});
