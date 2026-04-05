import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { useEffect } from "react";
import { MobileSidebar } from "../MobileSidebar";

// Track onOpenChange passed to Drawer.Root via a ref-like container
const captured: { onOpenChange: ((open: boolean) => void) | null } = { onOpenChange: null };

// Mock vaul Drawer
vi.mock("vaul", () => {
  const DrawerRoot = ({ children, onOpenChange, open }: {
    children: React.ReactNode;
    onOpenChange: (open: boolean) => void;
    open: boolean;
    direction?: string;
  }) => {
    // Capture via useEffect to satisfy React compiler lint rules
    useEffect(() => {
      captured.onOpenChange = onOpenChange;
    });
    return open ? <div data-testid="drawer-root">{children}</div> : null;
  };

  return {
    Drawer: {
      Root: DrawerRoot,
      Portal: ({ children }: { children: React.ReactNode }) => <>{children}</>,
      Overlay: ({ className }: { className?: string }) => (
        <div data-testid="drawer-overlay" className={className} />
      ),
      Content: ({ children, className }: { children: React.ReactNode; className?: string }) => (
        <div data-testid="drawer-content" className={className}>{children}</div>
      ),
      Title: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
    },
  };
});

// Mock radix visually hidden
vi.mock("@radix-ui/react-visually-hidden", () => ({
  Root: ({ children }: { children: React.ReactNode }) => <span>{children}</span>,
}));

// Mock sidebar content components
vi.mock("@/components/ide/sidebar/WorkspaceSidebarContent", () => ({
  WorkspaceSidebarContent: () => <div data-testid="workspace-content">Workspace</div>,
}));
vi.mock("@/components/ide/sidebar/TicketsSidebarContent", () => ({
  TicketsSidebarContent: () => <div data-testid="tickets-content">Tickets</div>,
}));
vi.mock("@/components/ide/sidebar/MeshSidebarContent", () => ({
  MeshSidebarContent: () => <div data-testid="mesh-content">Mesh</div>,
}));
vi.mock("@/components/ide/sidebar/RepositoriesSidebarContent", () => ({
  RepositoriesSidebarContent: () => <div data-testid="repos-content">Repos</div>,
}));
vi.mock("@/components/ide/sidebar/RunnersSidebarContent", () => ({
  RunnersSidebarContent: () => <div data-testid="runners-content">Runners</div>,
}));
vi.mock("@/components/ide/sidebar/SettingsSidebarContent", () => ({
  SettingsSidebarContent: () => <div data-testid="settings-content">Settings</div>,
}));

// Mock modals
vi.mock("@/components/ide/CreatePodModal", () => ({
  CreatePodModal: () => null,
}));
vi.mock("@/components/ide/modals/AddRunnerModal", () => ({
  AddRunnerModal: () => null,
}));
vi.mock("@/components/ide/modals/ImportRepositoryModal", () => ({
  ImportRepositoryModal: () => null,
}));

// Mock stores
const mockSetMobileSidebarOpen = vi.fn();
vi.mock("@/stores/ide", () => ({
  useIDEStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector({
      activeActivity: "tickets",
      mobileSidebarOpen: true,
      setMobileSidebarOpen: mockSetMobileSidebarOpen,
    }),
}));
vi.mock("@/stores/auth", () => ({
  useAuthStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector({
      currentOrg: { name: "TestOrg", slug: "test-org" },
    }),
}));
vi.mock("@/stores/workspace", () => ({
  useWorkspaceStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector({ addPane: vi.fn() }),
}));
vi.mock("@/stores/pod", () => ({
  usePodStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector({ fetchPods: vi.fn() }),
}));

// Mock sonner and pod-utils
vi.mock("sonner", () => ({ toast: { info: vi.fn() } }));
vi.mock("@/lib/pod-utils", () => ({
  getPodDisplayName: (pod: { pod_key: string }) => pod.pod_key,
}));

describe("MobileSidebar", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    captured.onOpenChange = null;
  });

  afterEach(() => {
    // Clean up any dialog overlays added during tests
    document.querySelectorAll("[data-dialog-overlay]").forEach((el) => el.remove());
  });

  it("renders sidebar content based on active activity", () => {
    render(<MobileSidebar />);
    expect(screen.getByTestId("tickets-content")).toBeInTheDocument();
  });

  it("allows drawer to close when no dialog overlay is present", () => {
    render(<MobileSidebar />);

    // Simulate vaul trying to close the drawer
    captured.onOpenChange?.(false);

    expect(mockSetMobileSidebarOpen).toHaveBeenCalledWith(false);
  });

  it("prevents drawer from closing when a dialog overlay is present", () => {
    render(<MobileSidebar />);

    // Simulate a nested dialog being open (portal to document.body)
    const dialogOverlay = document.createElement("div");
    dialogOverlay.setAttribute("data-dialog-overlay", "");
    document.body.appendChild(dialogOverlay);

    // Simulate vaul trying to close the drawer
    captured.onOpenChange?.(false);

    // Drawer should NOT close
    expect(mockSetMobileSidebarOpen).not.toHaveBeenCalled();
  });

  it("allows drawer to open even when a dialog overlay is present", () => {
    render(<MobileSidebar />);

    // Simulate a nested dialog being open
    const dialogOverlay = document.createElement("div");
    dialogOverlay.setAttribute("data-dialog-overlay", "");
    document.body.appendChild(dialogOverlay);

    // Simulate vaul opening the drawer
    captured.onOpenChange?.(true);

    expect(mockSetMobileSidebarOpen).toHaveBeenCalledWith(true);
  });

  it("allows drawer to close after dialog overlay is removed", () => {
    render(<MobileSidebar />);

    // Add and then remove dialog overlay
    const dialogOverlay = document.createElement("div");
    dialogOverlay.setAttribute("data-dialog-overlay", "");
    document.body.appendChild(dialogOverlay);

    // Can't close while dialog is open
    captured.onOpenChange?.(false);
    expect(mockSetMobileSidebarOpen).not.toHaveBeenCalled();

    // Remove the dialog
    dialogOverlay.remove();

    // Now drawer can close
    captured.onOpenChange?.(false);
    expect(mockSetMobileSidebarOpen).toHaveBeenCalledWith(false);
  });

  it("renders close button in header", () => {
    render(<MobileSidebar />);
    // The X button in the header directly calls setMobileSidebarOpen(false)
    const closeButtons = screen.getAllByRole("button");
    expect(closeButtons.length).toBeGreaterThan(0);
  });

  it("displays activity title in header", () => {
    render(<MobileSidebar />);
    const header = screen.getByText("Tickets", { selector: ".font-semibold" });
    expect(header).toBeInTheDocument();
  });

  it("displays organization initial", () => {
    render(<MobileSidebar />);
    expect(screen.getByText("T")).toBeInTheDocument();
  });
});
