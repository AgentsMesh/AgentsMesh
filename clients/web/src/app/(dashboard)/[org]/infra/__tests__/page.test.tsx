import { describe, it, expect, vi, beforeEach } from "vitest";
import { fireEvent, render, screen } from "@/test/test-utils";
import InfraPage from "../page";

const mockReplace = vi.fn();
let mockSearch = "tab=runners";

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn(), replace: mockReplace }),
  useParams: () => ({ org: "rcx" }),
  useSearchParams: () => new URLSearchParams(mockSearch),
}));

vi.mock("@/components/infra/InfraRepositoryDetail", () => ({
  InfraRepositoryDetail: () => <div data-testid="repo-detail" />,
}));

vi.mock("@/components/infra/InfraRunnerDetail", () => ({
  InfraRunnerDetail: () => <div data-testid="runner-detail" />,
}));

vi.mock("@/components/ide/modals/AddRunnerModal", () => ({
  AddRunnerModal: ({ open, onClose }: { open: boolean; onClose: () => void }) =>
    open ? (
      <div data-testid="add-runner-modal">
        <button onClick={onClose}>close-add-runner</button>
      </div>
    ) : null,
}));

vi.mock("@/components/ide/modals/ImportRepositoryModal", () => ({
  ImportRepositoryModal: ({ open, onClose }: { open: boolean; onClose: () => void }) =>
    open ? (
      <div data-testid="import-repo-modal">
        <button onClick={onClose}>close-import-repo</button>
      </div>
    ) : null,
}));

vi.mock("@/stores/auth", () => ({
  useCurrentOrg: () => ({ slug: "rcx", id: 1 }),
}));

vi.mock("@/stores/runner", () => ({
  useRunners: () => [],
  useRunnerStore: (selector: (s: { loading: boolean; fetched: boolean; fetchRunners: () => void }) => unknown) =>
    selector({ loading: false, fetched: true, fetchRunners: vi.fn() }),
}));

vi.mock("@/stores/repository", () => ({
  useRepositories: () => [],
  useRepositoryStore: (
    selector: (s: {
      isLoading: boolean;
      fetched: boolean;
      error: string | null;
      fetchRepositories: () => Promise<void>;
    }) => unknown,
  ) => selector({ isLoading: false, fetched: true, error: null, fetchRepositories: vi.fn() }),
}));

describe("InfraPage — Runner empty state", () => {
  beforeEach(() => {
    mockSearch = "tab=runners";
    mockReplace.mockReset();
  });

  it("opens the Add Runner modal when the empty-state button is clicked", () => {
    render(<InfraPage />);

    expect(screen.queryByTestId("add-runner-modal")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Add Runner" }));
    expect(screen.getByTestId("add-runner-modal")).toBeInTheDocument();
  });

  it("closes the Add Runner modal via onClose", () => {
    render(<InfraPage />);

    fireEvent.click(screen.getByRole("button", { name: "Add Runner" }));
    expect(screen.getByTestId("add-runner-modal")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "close-add-runner" }));
    expect(screen.queryByTestId("add-runner-modal")).not.toBeInTheDocument();
  });
});

describe("InfraPage — Repository empty state", () => {
  beforeEach(() => {
    mockSearch = "tab=repositories";
    mockReplace.mockReset();
  });

  it("opens the Import Repository modal when the empty-state button is clicked", async () => {
    render(<InfraPage />);

    const importBtn = await screen.findByRole("button", { name: /^Import/ });
    expect(screen.queryByTestId("import-repo-modal")).not.toBeInTheDocument();
    fireEvent.click(importBtn);
    expect(screen.getByTestId("import-repo-modal")).toBeInTheDocument();
  });

  it("closes the Import Repository modal via onClose", async () => {
    render(<InfraPage />);

    const importBtn = await screen.findByRole("button", { name: /^Import/ });
    fireEvent.click(importBtn);
    expect(screen.getByTestId("import-repo-modal")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "close-import-repo" }));
    expect(screen.queryByTestId("import-repo-modal")).not.toBeInTheDocument();
  });
});
