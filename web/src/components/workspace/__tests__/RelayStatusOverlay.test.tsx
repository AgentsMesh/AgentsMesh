import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { RelayStatusOverlay } from "../RelayStatusOverlay";

// Mock i18n client
vi.mock("@/lib/i18n/client", () => ({
  useTranslations: () => (key: string) => {
    const translations: Record<string, string> = {
      "relayStatus.connected": "Relay Connected",
      "relayStatus.connecting": "Connecting to Relay...",
      "relayStatus.disconnected": "Relay Disconnected",
      "relayStatus.error": "Relay Connection Error",
      "relayStatus.runnerDisconnected": "Runner Disconnected",
    };
    return translations[key] || key;
  },
}));

describe("RelayStatusOverlay", () => {
  describe("connected state", () => {
    it("shows connected text with Wifi icon", () => {
      render(
        <RelayStatusOverlay connectionStatus="connected" isRunnerDisconnected={false} />
      );
      expect(screen.getByText("Relay Connected")).toBeInTheDocument();
    });

    it("applies green styling when connected", () => {
      const { container } = render(
        <RelayStatusOverlay connectionStatus="connected" isRunnerDisconnected={false} />
      );
      const badge = container.querySelector(".bg-green-500\\/15");
      expect(badge).toBeInTheDocument();
    });
  });

  describe("connecting state", () => {
    it("shows connecting text", () => {
      render(
        <RelayStatusOverlay connectionStatus="connecting" isRunnerDisconnected={false} />
      );
      expect(screen.getByText("Connecting to Relay...")).toBeInTheDocument();
    });

    it("applies yellow styling when connecting", () => {
      const { container } = render(
        <RelayStatusOverlay connectionStatus="connecting" isRunnerDisconnected={false} />
      );
      const badge = container.querySelector(".bg-yellow-500\\/15");
      expect(badge).toBeInTheDocument();
    });

    it("shows spinning loader icon", () => {
      const { container } = render(
        <RelayStatusOverlay connectionStatus="connecting" isRunnerDisconnected={false} />
      );
      const spinner = container.querySelector(".animate-spin");
      expect(spinner).toBeInTheDocument();
    });
  });

  describe("disconnected state", () => {
    it("shows disconnected text", () => {
      render(
        <RelayStatusOverlay connectionStatus="disconnected" isRunnerDisconnected={false} />
      );
      expect(screen.getByText("Relay Disconnected")).toBeInTheDocument();
    });

    it("applies red styling when disconnected", () => {
      const { container } = render(
        <RelayStatusOverlay connectionStatus="disconnected" isRunnerDisconnected={false} />
      );
      const badge = container.querySelector(".bg-red-500\\/15");
      expect(badge).toBeInTheDocument();
    });
  });

  describe("error state", () => {
    it("shows error text", () => {
      render(
        <RelayStatusOverlay connectionStatus="error" isRunnerDisconnected={false} />
      );
      expect(screen.getByText("Relay Connection Error")).toBeInTheDocument();
    });

    it("applies red styling when error", () => {
      const { container } = render(
        <RelayStatusOverlay connectionStatus="error" isRunnerDisconnected={false} />
      );
      const badge = container.querySelector(".bg-red-500\\/15");
      expect(badge).toBeInTheDocument();
    });
  });

  describe("runner disconnected state", () => {
    it("shows runner disconnected text when runner is disconnected", () => {
      render(
        <RelayStatusOverlay connectionStatus="connected" isRunnerDisconnected={true} />
      );
      expect(screen.getByText("Runner Disconnected")).toBeInTheDocument();
    });

    it("runner disconnect takes priority over connected status", () => {
      render(
        <RelayStatusOverlay connectionStatus="connected" isRunnerDisconnected={true} />
      );
      expect(screen.queryByText("Relay Connected")).not.toBeInTheDocument();
      expect(screen.getByText("Runner Disconnected")).toBeInTheDocument();
    });

    it("applies red styling when runner is disconnected", () => {
      const { container } = render(
        <RelayStatusOverlay connectionStatus="connected" isRunnerDisconnected={true} />
      );
      const badge = container.querySelector(".bg-red-500\\/15");
      expect(badge).toBeInTheDocument();
    });
  });

  describe("overlay positioning", () => {
    it("renders as absolute positioned overlay", () => {
      const { container } = render(
        <RelayStatusOverlay connectionStatus="connected" isRunnerDisconnected={false} />
      );
      const overlay = container.firstChild as HTMLElement;
      expect(overlay).toHaveClass("absolute", "top-0", "left-0", "right-0", "z-10");
    });

    it("is not interactive (pointer-events-none)", () => {
      const { container } = render(
        <RelayStatusOverlay connectionStatus="connected" isRunnerDisconnected={false} />
      );
      const overlay = container.firstChild as HTMLElement;
      expect(overlay).toHaveClass("pointer-events-none");
    });
  });

  describe("className prop", () => {
    it("applies custom className", () => {
      const { container } = render(
        <RelayStatusOverlay
          connectionStatus="connected"
          isRunnerDisconnected={false}
          className="custom-class"
        />
      );
      const overlay = container.firstChild as HTMLElement;
      expect(overlay).toHaveClass("custom-class");
    });
  });
});
