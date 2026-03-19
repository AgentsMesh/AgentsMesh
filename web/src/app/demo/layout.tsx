import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Request a Demo",
  description: "See how AgentsMesh can accelerate your team's development workflow with AI agent orchestration. Request a personalized demo today.",
  alternates: {
    canonical: "https://agentsmesh.ai/demo",
  },
  openGraph: {
    title: "Request a Demo | AgentsMesh",
    description: "See how AgentsMesh can accelerate your team's development workflow with AI agent orchestration.",
    url: "https://agentsmesh.ai/demo",
  },
};

export default function DemoLayout({ children }: { children: React.ReactNode }) {
  return children;
}
