import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "About",
  description:
    "Learn about AgentsMesh — our mission to empower development teams with AI agent orchestration at scale.",
  alternates: {
    canonical: "https://agentsmesh.ai/about",
  },
  openGraph: {
    title: "About | AgentsMesh",
    description:
      "Learn about AgentsMesh — our mission to empower development teams with AI agent orchestration at scale.",
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
