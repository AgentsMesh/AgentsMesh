import { describe, it, expect, vi, beforeEach } from "vitest";
import { fireEvent, render, screen } from "@/test/test-utils";
import InfraPage from "../page";

const mockPush = vi.fn();
const mockReplace = vi.fn();
let mockSearch = "tab=runners";

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: mockPush, replace: mockReplace }),
  useParams: () => ({ org: "rcx" }),
  useSearchParams: () => new URLSearchParams(mockSearch),
}));

vi.mock("@/components/infra/InfraRepositoryDetail", () => ({
  InfraRepositoryDetail: () => <div data-testid="repo-detail" />,
}));

vi.mock("@/components/infra/InfraRunnerDetail", () => ({
  InfraRunnerDetail: () => <div data-testid="runner-detail" />,
}));

describe("InfraPage runner empty state", () => {
  beforeEach(() => {
    mockSearch = "tab=runners";
    mockPush.mockReset();
    mockReplace.mockReset();
  });

  it("sets the add query when the empty-state Add Runner button is clicked", () => {
    render(<InfraPage />);

    fireEvent.click(screen.getByRole("button", { name: "Add Runner" }));

    expect(mockPush).toHaveBeenCalledWith("/rcx/infra?tab=runners&add=1");
  });

  it("opens the Add Runner modal from the add query", () => {
    mockSearch = "tab=runners&add=1";

    render(<InfraPage />);

    expect(
      screen.getByText("Generate a registration token to connect a new Runner"),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Generate Token" })).toBeInTheDocument();
  });

  it("removes the add query when the Add Runner modal closes", () => {
    mockSearch = "tab=runners&add=1";

    render(<InfraPage />);
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));

    expect(mockReplace).toHaveBeenCalledWith("/rcx/infra?tab=runners");
  });
});
