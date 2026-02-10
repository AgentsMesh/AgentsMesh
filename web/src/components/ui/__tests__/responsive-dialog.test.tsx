import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import {
  ResponsiveDialog,
  ResponsiveDialogContent,
  ResponsiveDialogHeader,
  ResponsiveDialogTitle,
  ResponsiveDialogBody,
  ResponsiveDialogFooter,
} from "../responsive-dialog";

// Mock useBreakpoint
const mockUseBreakpoint = vi.fn();
vi.mock("@/components/layout/useBreakpoint", () => ({
  useBreakpoint: (...args: unknown[]) => mockUseBreakpoint(...args),
}));

// Mock vaul - render Drawer components as simple divs preserving className
vi.mock("vaul", () => ({
  Drawer: {
    Root: ({
      children,
      open,
    }: {
      children: React.ReactNode;
      open: boolean;
    }) => (open ? <div data-testid="drawer-root">{children}</div> : null),
    Portal: ({ children }: { children: React.ReactNode }) => (
      <div data-testid="drawer-portal">{children}</div>
    ),
    Overlay: ({ className }: { className?: string }) => (
      <div data-testid="drawer-overlay" className={className} />
    ),
    Content: ({
      children,
      className,
    }: {
      children: React.ReactNode;
      className?: string;
    }) => (
      <div data-testid="drawer-content" className={className}>
        {children}
      </div>
    ),
    Title: ({
      children,
      className,
    }: {
      children: React.ReactNode;
      className?: string;
    }) => (
      <span data-testid="drawer-title" className={className}>
        {children}
      </span>
    ),
  },
}));

function setMobile() {
  mockUseBreakpoint.mockReturnValue({
    breakpoint: "mobile",
    isMobile: true,
    isTablet: false,
    isDesktop: false,
    width: 375,
  });
}

function setDesktop() {
  mockUseBreakpoint.mockReturnValue({
    breakpoint: "desktop",
    isMobile: false,
    isTablet: false,
    isDesktop: true,
    width: 1280,
  });
}

describe("ResponsiveDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setDesktop();
  });

  describe("Desktop mode", () => {
    it("renders children in a portal overlay", () => {
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

  describe("Mobile mode", () => {
    beforeEach(() => {
      setMobile();
    });

    it("renders using vaul Drawer", () => {
      render(
        <ResponsiveDialog open={true} onOpenChange={vi.fn()}>
          <ResponsiveDialogContent title="Test Dialog">
            <div>Mobile Content</div>
          </ResponsiveDialogContent>
        </ResponsiveDialog>
      );

      expect(screen.getByTestId("drawer-root")).toBeInTheDocument();
      expect(screen.getByTestId("drawer-content")).toBeInTheDocument();
      expect(screen.getByText("Mobile Content")).toBeInTheDocument();
    });

    it("does not render when closed", () => {
      render(
        <ResponsiveDialog open={false} onOpenChange={vi.fn()}>
          <ResponsiveDialogContent title="Test Dialog">
            <div>Hidden Mobile Content</div>
          </ResponsiveDialogContent>
        </ResponsiveDialog>
      );

      expect(
        screen.queryByText("Hidden Mobile Content")
      ).not.toBeInTheDocument();
    });
  });
});

describe("ResponsiveDialogContent", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("Mobile mode", () => {
    beforeEach(() => {
      setMobile();
    });

    it("uses dvh units for max height instead of vh", () => {
      render(
        <ResponsiveDialog open={true} onOpenChange={vi.fn()}>
          <ResponsiveDialogContent title="Test">
            <div>Content</div>
          </ResponsiveDialogContent>
        </ResponsiveDialog>
      );

      const drawerContent = screen.getByTestId("drawer-content");
      expect(drawerContent.className).toContain("max-h-[85dvh]");
      expect(drawerContent.className).not.toContain("max-h-[90vh]");
    });

    it("uses overflow-hidden on wrapper div instead of overflow-y-auto to prevent nested scrolling", () => {
      render(
        <ResponsiveDialog open={true} onOpenChange={vi.fn()}>
          <ResponsiveDialogContent title="Test">
            <div data-testid="child">Content</div>
          </ResponsiveDialogContent>
        </ResponsiveDialog>
      );

      // The wrapper div is the parent of the children inside drawer-content
      const child = screen.getByTestId("child");
      const wrapperDiv = child.parentElement!;
      expect(wrapperDiv.className).toContain("flex-1");
      expect(wrapperDiv.className).toContain("flex-col");
      expect(wrapperDiv.className).toContain("min-h-0");
      expect(wrapperDiv.className).toContain("overflow-hidden");
      expect(wrapperDiv.className).not.toContain("overflow-y-auto");
    });

    it("renders the drag handle", () => {
      const { container } = render(
        <ResponsiveDialog open={true} onOpenChange={vi.fn()}>
          <ResponsiveDialogContent title="Test">
            <div>Content</div>
          </ResponsiveDialogContent>
        </ResponsiveDialog>
      );

      const handle = container.querySelector(".rounded-full.bg-muted");
      expect(handle).toBeInTheDocument();
    });

    it("renders accessible title as sr-only", () => {
      render(
        <ResponsiveDialog open={true} onOpenChange={vi.fn()}>
          <ResponsiveDialogContent title="Accessible Title">
            <div>Content</div>
          </ResponsiveDialogContent>
        </ResponsiveDialog>
      );

      const title = screen.getByTestId("drawer-title");
      expect(title).toHaveTextContent("Accessible Title");
      expect(title.className).toContain("sr-only");
    });
  });

  describe("Desktop mode", () => {
    beforeEach(() => {
      setDesktop();
    });

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
  });
});

describe("ResponsiveDialogBody", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("has overflow-y-auto as the single scroll container", () => {
    setDesktop();
    render(
      <ResponsiveDialogBody>
        <div>Body Content</div>
      </ResponsiveDialogBody>
    );

    const body = screen.getByText("Body Content").parentElement!;
    expect(body.className).toContain("overflow-y-auto");
  });

  it("has min-h-0 for proper flex shrinking", () => {
    setDesktop();
    render(
      <ResponsiveDialogBody>
        <div>Body Content</div>
      </ResponsiveDialogBody>
    );

    const body = screen.getByText("Body Content").parentElement!;
    expect(body.className).toContain("min-h-0");
  });

  it("has overscroll-contain on mobile to prevent scroll chaining", () => {
    setMobile();
    render(
      <ResponsiveDialogBody>
        <div>Body Content</div>
      </ResponsiveDialogBody>
    );

    const body = screen.getByText("Body Content").parentElement!;
    expect(body.className).toContain("overscroll-contain");
  });

  it("does not have overscroll-contain on desktop", () => {
    setDesktop();
    render(
      <ResponsiveDialogBody>
        <div>Body Content</div>
      </ResponsiveDialogBody>
    );

    const body = screen.getByText("Body Content").parentElement!;
    expect(body.className).not.toContain("overscroll-contain");
  });

  it("uses px-4 padding on mobile", () => {
    setMobile();
    render(
      <ResponsiveDialogBody>
        <div>Body Content</div>
      </ResponsiveDialogBody>
    );

    const body = screen.getByText("Body Content").parentElement!;
    expect(body.className).toContain("px-4");
  });

  it("uses px-6 padding on desktop", () => {
    setDesktop();
    render(
      <ResponsiveDialogBody>
        <div>Body Content</div>
      </ResponsiveDialogBody>
    );

    const body = screen.getByText("Body Content").parentElement!;
    expect(body.className).toContain("px-6");
  });
});

describe("ResponsiveDialogHeader", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("has flex-shrink-0 to prevent collapsing", () => {
    setDesktop();
    render(
      <ResponsiveDialogHeader>
        <div>Header</div>
      </ResponsiveDialogHeader>
    );

    const header = screen.getByText("Header").parentElement!;
    expect(header.className).toContain("flex-shrink-0");
  });

  it("uses px-4 on mobile", () => {
    setMobile();
    render(
      <ResponsiveDialogHeader>
        <div>Header</div>
      </ResponsiveDialogHeader>
    );

    const header = screen.getByText("Header").parentElement!;
    expect(header.className).toContain("px-4");
  });

  it("uses px-6 on desktop", () => {
    setDesktop();
    render(
      <ResponsiveDialogHeader>
        <div>Header</div>
      </ResponsiveDialogHeader>
    );

    const header = screen.getByText("Header").parentElement!;
    expect(header.className).toContain("px-6");
  });
});

describe("ResponsiveDialogFooter", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("uses flex-col-reverse on mobile for proper button order", () => {
    setMobile();
    render(
      <ResponsiveDialogFooter>
        <button>Cancel</button>
        <button>Submit</button>
      </ResponsiveDialogFooter>
    );

    const footer = screen.getByText("Cancel").parentElement!;
    expect(footer.className).toContain("flex-col-reverse");
    expect(footer.className).toContain("px-4");
  });

  it("uses justify-end on desktop", () => {
    setDesktop();
    render(
      <ResponsiveDialogFooter>
        <button>Cancel</button>
        <button>Submit</button>
      </ResponsiveDialogFooter>
    );

    const footer = screen.getByText("Cancel").parentElement!;
    expect(footer.className).toContain("justify-end");
    expect(footer.className).toContain("px-6");
  });

  it("has flex-shrink-0 to prevent collapsing", () => {
    setDesktop();
    render(
      <ResponsiveDialogFooter>
        <button>Actions</button>
      </ResponsiveDialogFooter>
    );

    const footer = screen.getByText("Actions").parentElement!;
    expect(footer.className).toContain("flex-shrink-0");
  });
});
