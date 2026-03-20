import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "About",
  description:
    "Learn about AgentsMesh — the agent fleet command center empowering teams with agent orchestration, swarm coordination, and harnessed engineering productivity at scale.",
  alternates: {
    canonical: "https://agentsmesh.ai/about",
  },
  openGraph: {
    title: "About | AgentsMesh",
    description:
      "Learn about AgentsMesh — the agent fleet command center empowering teams with agent orchestration, swarm coordination, and harnessed engineering productivity at scale.",
    url: "https://agentsmesh.ai/about",
  },
};

export default function AboutLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return children;
}
