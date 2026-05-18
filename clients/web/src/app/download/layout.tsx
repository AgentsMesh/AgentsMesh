import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Download AgentsMesh Desktop & Runner",
  description:
    "Download AgentsMesh — the AI agent workforce platform. Native desktop apps for macOS, Windows, and Linux, plus the self-hosted Runner CLI.",
  alternates: {
    canonical: "https://agentsmesh.ai/download",
  },
  openGraph: {
    title: "Download AgentsMesh",
    description:
      "Native desktop apps for macOS, Windows, and Linux. Plus the self-hosted Runner that keeps your agents on your infrastructure.",
    url: "https://agentsmesh.ai/download",
  },
};

export default function DownloadLayout({ children }: { children: React.ReactNode }) {
  return children;
}
