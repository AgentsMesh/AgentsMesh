import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, waitFor } from "@/test/test-utils";
import RepositoriesRedirect from "../page";

const mockReplace = vi.fn();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace: mockReplace }),
  useParams: () => ({ org: "my-org" }),
}));

describe("RepositoriesRedirect", () => {
  beforeEach(() => mockReplace.mockReset());

  it("redirects legacy /repositories to /infra?tab=repositories", async () => {
    render(<RepositoriesRedirect />);
    await waitFor(() => {
      expect(mockReplace).toHaveBeenCalledWith("/my-org/infra?tab=repositories");
    });
  });
});
