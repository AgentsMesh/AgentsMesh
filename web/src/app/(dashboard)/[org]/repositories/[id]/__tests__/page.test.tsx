import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, waitFor } from "@/test/test-utils";
import RepositoryDetailRedirect from "../page";

const mockReplace = vi.fn();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace: mockReplace }),
  useParams: () => ({ org: "my-org", id: "42" }),
}));

describe("RepositoryDetailRedirect", () => {
  beforeEach(() => mockReplace.mockReset());

  it("redirects /repositories/[id] to /infra?tab=repositories&id=<id>", async () => {
    render(<RepositoryDetailRedirect />);
    await waitFor(() => {
      expect(mockReplace).toHaveBeenCalledWith("/my-org/infra?tab=repositories&id=42");
    });
  });
});
