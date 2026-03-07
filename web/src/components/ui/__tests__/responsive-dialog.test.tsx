import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import {
  ResponsiveDialog,
  ResponsiveDialogContent,
  ResponsiveDialogHeader,
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
  it("is a single scrollable container with padding", () => {
    render(
      <ResponsiveDialog open={true} onOpenChange={vi.fn()}>
        <ResponsiveDialogContent>
          <div>Content</div>
        </ResponsiveDialogContent>
      </ResponsiveDialog>
    );

    const content = screen.getByText("Content").parentElement!;
    expect(content.className).toContain("overflow-y-auto");
    expect(content.className).toContain("max-h-[90vh]");
    expect(content.className).toContain("p-4");
    expect(content.className).toContain("rounded-lg");
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
  it("renders close button when onClose is provided", () => {
    render(
      <ResponsiveDialogHeader onClose={vi.fn()}>
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
});
