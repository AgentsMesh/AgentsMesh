import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@/test/test-utils";

// Mock useBreakpoint
const mockUseBreakpoint = vi.fn();
vi.mock("@/components/layout/useBreakpoint", () => ({
  useBreakpoint: (...args: unknown[]) => mockUseBreakpoint(...args),
}));

// Mock vaul
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
      <div>{children}</div>
    ),
    Overlay: ({ className }: { className?: string }) => (
      <div className={className} />
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
    }) => <span className={className}>{children}</span>,
  },
}));

// Mock ticketApi
const mockCreate = vi.fn();
vi.mock("@/lib/api", () => ({
  ticketApi: {
    create: (...args: unknown[]) => mockCreate(...args),
  },
}));

// Mock RepositorySelect
vi.mock("@/components/common/RepositorySelect", () => ({
  RepositorySelect: ({ placeholder }: { placeholder?: string }) => (
    <div data-testid="repository-select">{placeholder}</div>
  ),
}));

// Mock BlockEditor (lazy loaded)
vi.mock("@/components/ui/block-editor", () => ({
  default: ({
    editable,
  }: {
    onChange?: (v: string) => void;
    editable?: boolean;
  }) => (
    <div data-testid="block-editor" data-editable={editable}>
      Mock Editor
    </div>
  ),
}));

import { TicketCreateDialog } from "../TicketCreateDialog";

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

describe("TicketCreateDialog", () => {
  const defaultProps = {
    open: true,
    onOpenChange: vi.fn(),
    onCreated: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
    setDesktop();
    mockCreate.mockResolvedValue({ id: 1, identifier: "TICKET-1" });
  });

  describe("Rendering", () => {
    it("renders all form fields when open", async () => {
      render(<TicketCreateDialog {...defaultProps} />);

      expect(
        screen.getByPlaceholderText("Enter ticket title")
      ).toBeInTheDocument();
      expect(screen.getByTestId("repository-select")).toBeInTheDocument();

      // BlockEditor is lazy-loaded, so wait for Suspense to resolve
      await waitFor(() => {
        expect(screen.getByTestId("block-editor")).toBeInTheDocument();
      });
    });

    it("does not render when closed", () => {
      render(<TicketCreateDialog {...defaultProps} open={false} />);

      expect(
        screen.queryByPlaceholderText("Enter ticket title")
      ).not.toBeInTheDocument();
    });

    it("renders dialog title", () => {
      render(<TicketCreateDialog {...defaultProps} />);

      // "Create Ticket" appears as both the heading and submit button
      expect(
        screen.getByRole("heading", { name: "Create Ticket" })
      ).toBeInTheDocument();
    });

    it("renders sub-ticket title when parentTicketId is provided", () => {
      render(<TicketCreateDialog {...defaultProps} parentTicketId={42} />);

      expect(screen.getByText("Create Sub-ticket")).toBeInTheDocument();
    });
  });

  describe("Form validation", () => {
    it("shows error when submitting without title", async () => {
      render(<TicketCreateDialog {...defaultProps} />);

      const submitButton = screen.getByRole("button", {
        name: "Create Ticket",
      });
      fireEvent.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText("Title is required")).toBeInTheDocument();
      });
      expect(mockCreate).not.toHaveBeenCalled();
    });

    it("clears error when user starts typing", async () => {
      render(<TicketCreateDialog {...defaultProps} />);

      // Trigger validation error
      const submitButton = screen.getByRole("button", {
        name: "Create Ticket",
      });
      fireEvent.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText("Title is required")).toBeInTheDocument();
      });

      // Start typing
      const titleInput = screen.getByPlaceholderText("Enter ticket title");
      fireEvent.change(titleInput, { target: { value: "A" } });

      expect(screen.queryByText("Title is required")).not.toBeInTheDocument();
    });
  });

  describe("Form submission", () => {
    it("submits form with valid data", async () => {
      render(<TicketCreateDialog {...defaultProps} />);

      const titleInput = screen.getByPlaceholderText("Enter ticket title");
      fireEvent.change(titleInput, { target: { value: "Test Ticket" } });

      const submitButton = screen.getByRole("button", {
        name: "Create Ticket",
      });
      fireEvent.click(submitButton);

      await waitFor(() => {
        expect(mockCreate).toHaveBeenCalledWith(
          expect.objectContaining({
            title: "Test Ticket",
            type: "task",
            priority: "medium",
          })
        );
      });
    });

    it("calls onCreated callback after successful creation", async () => {
      const onCreated = vi.fn();
      render(<TicketCreateDialog {...defaultProps} onCreated={onCreated} />);

      const titleInput = screen.getByPlaceholderText("Enter ticket title");
      fireEvent.change(titleInput, { target: { value: "Test Ticket" } });

      const submitButton = screen.getByRole("button", {
        name: "Create Ticket",
      });
      fireEvent.click(submitButton);

      await waitFor(() => {
        expect(onCreated).toHaveBeenCalledWith(1, "TICKET-1");
      });
    });

    it("includes parentId in API call for sub-tickets", async () => {
      render(<TicketCreateDialog {...defaultProps} parentTicketId={42} />);

      const titleInput = screen.getByPlaceholderText("Enter ticket title");
      fireEvent.change(titleInput, { target: { value: "Sub-task" } });

      const submitButton = screen.getByRole("button", {
        name: "Create Ticket",
      });
      fireEvent.click(submitButton);

      await waitFor(() => {
        expect(mockCreate).toHaveBeenCalledWith(
          expect.objectContaining({
            title: "Sub-task",
            parentId: 42,
          })
        );
      });
    });

    it("shows error message on API failure", async () => {
      mockCreate.mockRejectedValue(new Error("Network error"));

      render(<TicketCreateDialog {...defaultProps} />);

      const titleInput = screen.getByPlaceholderText("Enter ticket title");
      fireEvent.change(titleInput, { target: { value: "Test Ticket" } });

      const submitButton = screen.getByRole("button", {
        name: "Create Ticket",
      });
      fireEvent.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText("Network error")).toBeInTheDocument();
      });
    });

    it("closes dialog after successful creation", async () => {
      const onOpenChange = vi.fn();
      render(
        <TicketCreateDialog {...defaultProps} onOpenChange={onOpenChange} />
      );

      const titleInput = screen.getByPlaceholderText("Enter ticket title");
      fireEvent.change(titleInput, { target: { value: "Test Ticket" } });

      const submitButton = screen.getByRole("button", {
        name: "Create Ticket",
      });
      fireEvent.click(submitButton);

      await waitFor(() => {
        expect(onOpenChange).toHaveBeenCalledWith(false);
      });
    });
  });

  describe("Mobile responsive", () => {
    beforeEach(() => {
      setMobile();
    });

    it("renders in mobile drawer mode", () => {
      render(<TicketCreateDialog {...defaultProps} />);

      expect(screen.getByTestId("drawer-root")).toBeInTheDocument();
      expect(screen.getByTestId("drawer-content")).toBeInTheDocument();
    });

    it("uses compact min-height for block editor on mobile", () => {
      render(<TicketCreateDialog {...defaultProps} />);

      const editorWrapper = screen.getByTestId("block-editor").parentElement!;
      expect(editorWrapper.className).toContain("min-h-[100px]");
      expect(editorWrapper.className).not.toContain("min-h-[150px]");
    });

    it("uses dvh-based max height on the drawer", () => {
      render(<TicketCreateDialog {...defaultProps} />);

      const drawerContent = screen.getByTestId("drawer-content");
      expect(drawerContent.className).toContain("max-h-[85dvh]");
      expect(drawerContent.className).not.toContain("max-h-[90vh]");
    });

    it("title input is visible and accessible in mobile mode", () => {
      render(<TicketCreateDialog {...defaultProps} />);

      const titleInput = screen.getByPlaceholderText("Enter ticket title");
      expect(titleInput).toBeInTheDocument();
      expect(titleInput).toBeVisible();
    });

    it("uses full-width buttons on mobile", () => {
      render(<TicketCreateDialog {...defaultProps} />);

      const cancelButton = screen.getByRole("button", { name: "Cancel" });
      const submitButton = screen.getByRole("button", {
        name: "Create Ticket",
      });

      expect(cancelButton.className).toContain("w-full");
      expect(submitButton.className).toContain("w-full");
    });
  });
});
