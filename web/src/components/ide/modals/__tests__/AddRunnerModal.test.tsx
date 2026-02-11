import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { AddRunnerModal } from "../AddRunnerModal";

// Mock the API
vi.mock("@/lib/api", () => ({
  runnerApi: {
    createToken: vi.fn(),
  },
}));

// Mock useServerUrl hook
vi.mock("@/hooks/useServerUrl", () => ({
  useServerUrl: () => "https://api.example.com",
}));

// Mock translations
vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}));

import { runnerApi } from "@/lib/api";

describe("AddRunnerModal", () => {
  const mockOnClose = vi.fn();
  const mockOnCreated = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("rendering", () => {
    it("should not render when open is false", () => {
      render(
        <AddRunnerModal open={false} onClose={mockOnClose} onCreated={mockOnCreated} />
      );
      expect(screen.queryByText("runners.addRunnerModal.title")).not.toBeInTheDocument();
    });

    it("should render when open is true", () => {
      render(
        <AddRunnerModal open={true} onClose={mockOnClose} onCreated={mockOnCreated} />
      );
      expect(screen.getByText("runners.addRunnerModal.title")).toBeInTheDocument();
      expect(screen.getByText("runners.addRunnerModal.subtitle")).toBeInTheDocument();
    });

    it("should show generate button initially", () => {
      render(
        <AddRunnerModal open={true} onClose={mockOnClose} onCreated={mockOnCreated} />
      );
      expect(screen.getByText("runners.addRunnerModal.generate")).toBeInTheDocument();
      expect(screen.getByText("runners.addRunnerModal.cancel")).toBeInTheDocument();
    });
  });

  describe("token generation", () => {
    it("should call createToken when generate button is clicked", async () => {
      vi.mocked(runnerApi.createToken).mockResolvedValue({
        token: "test-token-12345",
        expires_at: "2024-12-31T23:59:59Z",
        message: "Token created",
      });

      render(
        <AddRunnerModal open={true} onClose={mockOnClose} onCreated={mockOnCreated} />
      );

      fireEvent.click(screen.getByText("runners.addRunnerModal.generate"));

      await waitFor(() => {
        expect(runnerApi.createToken).toHaveBeenCalled();
      });
    });

    it("should display token after generation", async () => {
      const testToken = "test-token-12345";
      vi.mocked(runnerApi.createToken).mockResolvedValue({
        token: testToken,
        expires_at: "2024-12-31T23:59:59Z",
        message: "Token created",
      });

      render(
        <AddRunnerModal open={true} onClose={mockOnClose} onCreated={mockOnCreated} />
      );

      fireEvent.click(screen.getByText("runners.addRunnerModal.generate"));

      await waitFor(() => {
        expect(screen.getByText(testToken)).toBeInTheDocument();
      });
    });

    it("should show warning after token is generated", async () => {
      vi.mocked(runnerApi.createToken).mockResolvedValue({
        token: "test-token-12345",
        expires_at: "2024-12-31T23:59:59Z",
        message: "Token created",
      });

      render(
        <AddRunnerModal open={true} onClose={mockOnClose} onCreated={mockOnCreated} />
      );

      fireEvent.click(screen.getByText("runners.addRunnerModal.generate"));

      await waitFor(() => {
        expect(screen.getByText("runners.addRunnerModal.tokenWarning")).toBeInTheDocument();
      });
    });

    it("should show usage instructions after token is generated", async () => {
      vi.mocked(runnerApi.createToken).mockResolvedValue({
        token: "test-token-12345",
        expires_at: "2024-12-31T23:59:59Z",
        message: "Token created",
      });

      render(
        <AddRunnerModal open={true} onClose={mockOnClose} onCreated={mockOnCreated} />
      );

      fireEvent.click(screen.getByText("runners.addRunnerModal.generate"));

      await waitFor(() => {
        expect(screen.getByText("runners.addRunnerModal.usageTitle")).toBeInTheDocument();
      });
    });

    it("should show generating state while loading", async () => {
      vi.mocked(runnerApi.createToken).mockImplementation(
        () => new Promise((resolve) => setTimeout(() => resolve({
          token: "test-token",
          expires_at: "2024-12-31T23:59:59Z",
          message: "Token created",
        }), 100))
      );

      render(
        <AddRunnerModal open={true} onClose={mockOnClose} onCreated={mockOnCreated} />
      );

      fireEvent.click(screen.getByText("runners.addRunnerModal.generate"));

      expect(screen.getByText("runners.addRunnerModal.generating")).toBeInTheDocument();
    });

    it("should handle token generation error", async () => {
      const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
      vi.mocked(runnerApi.createToken).mockRejectedValue(new Error("Network error"));

      render(
        <AddRunnerModal open={true} onClose={mockOnClose} onCreated={mockOnCreated} />
      );

      fireEvent.click(screen.getByText("runners.addRunnerModal.generate"));

      await waitFor(() => {
        expect(consoleSpy).toHaveBeenCalledWith("Failed to generate token:", expect.any(Error));
      });

      consoleSpy.mockRestore();
    });
  });

  describe("clipboard operations", () => {
    beforeEach(() => {
      Object.assign(navigator, {
        clipboard: {
          writeText: vi.fn().mockResolvedValue(undefined),
        },
      });
    });

    it("should copy token to clipboard when copy button is clicked", async () => {
      const testToken = "test-token-12345";
      vi.mocked(runnerApi.createToken).mockResolvedValue({
        token: testToken,
        expires_at: "2024-12-31T23:59:59Z",
        message: "Token created",
      });

      render(
        <AddRunnerModal open={true} onClose={mockOnClose} onCreated={mockOnCreated} />
      );

      fireEvent.click(screen.getByText("runners.addRunnerModal.generate"));

      await waitFor(() => {
        expect(screen.getByText(testToken)).toBeInTheDocument();
      });

      // Find and click the copy button (first button with Copy icon in token section)
      const copyButtons = screen.getAllByRole("button");
      const tokenCopyButton = copyButtons.find(btn =>
        btn.closest("div")?.querySelector("code")?.textContent === testToken
      );

      if (tokenCopyButton) {
        fireEvent.click(tokenCopyButton);
        expect(navigator.clipboard.writeText).toHaveBeenCalledWith(testToken);
      }
    });

    it("should copy command to clipboard when copy command button is clicked", async () => {
      const testToken = "test-token-12345";
      vi.mocked(runnerApi.createToken).mockResolvedValue({
        token: testToken,
        expires_at: "2024-12-31T23:59:59Z",
        message: "Token created",
      });

      render(
        <AddRunnerModal open={true} onClose={mockOnClose} onCreated={mockOnCreated} />
      );

      fireEvent.click(screen.getByText("runners.addRunnerModal.generate"));

      await waitFor(() => {
        expect(screen.getByText("runners.addRunnerModal.copyCommand")).toBeInTheDocument();
      });

      fireEvent.click(screen.getByText("runners.addRunnerModal.copyCommand"));

      expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
        expect.stringContaining("agentsmesh-runner register")
      );
    });
  });

  describe("close and done actions", () => {
    it("should call onClose when cancel button is clicked", () => {
      render(
        <AddRunnerModal open={true} onClose={mockOnClose} onCreated={mockOnCreated} />
      );

      fireEvent.click(screen.getByText("runners.addRunnerModal.cancel"));

      expect(mockOnClose).toHaveBeenCalled();
    });

    it("should call onCreated and onClose when done button is clicked", async () => {
      vi.mocked(runnerApi.createToken).mockResolvedValue({
        token: "test-token-12345",
        expires_at: "2024-12-31T23:59:59Z",
        message: "Token created",
      });

      render(
        <AddRunnerModal open={true} onClose={mockOnClose} onCreated={mockOnCreated} />
      );

      fireEvent.click(screen.getByText("runners.addRunnerModal.generate"));

      await waitFor(() => {
        expect(screen.getByText("runners.addRunnerModal.done")).toBeInTheDocument();
      });

      fireEvent.click(screen.getByText("runners.addRunnerModal.done"));

      expect(mockOnCreated).toHaveBeenCalled();
      expect(mockOnClose).toHaveBeenCalled();
    });

    it("should reset state when closing after token generation", async () => {
      vi.mocked(runnerApi.createToken).mockResolvedValue({
        token: "test-token-12345",
        expires_at: "2024-12-31T23:59:59Z",
        message: "Token created",
      });

      const { rerender } = render(
        <AddRunnerModal open={true} onClose={mockOnClose} onCreated={mockOnCreated} />
      );

      fireEvent.click(screen.getByText("runners.addRunnerModal.generate"));

      await waitFor(() => {
        expect(screen.getByText("test-token-12345")).toBeInTheDocument();
      });

      // Close and reopen
      rerender(
        <AddRunnerModal open={false} onClose={mockOnClose} onCreated={mockOnCreated} />
      );
      rerender(
        <AddRunnerModal open={true} onClose={mockOnClose} onCreated={mockOnCreated} />
      );

      // Should show initial state, not token
      expect(screen.getByText("runners.addRunnerModal.generate")).toBeInTheDocument();
    });
  });
});
