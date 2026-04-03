import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Request a Demo",
  description: "See how AgentsMesh can multiply your team's output with AI agent teams. Request a personalized demo today.",
  alternates: {
    canonical: "https://agentsmesh.ai/demo",
  },
  openGraph: {
    title: "Request a Demo | AgentsMesh",
    description: "See how AgentsMesh can multiply your team's output with AI agent teams.",
    url: "https://agentsmesh.ai/demo",
  },
};

export default function DemoLayout({ children }: { children: React.ReactNode }) {
  return children;
}
