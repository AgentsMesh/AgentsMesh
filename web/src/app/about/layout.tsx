import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "About",
  description:
    "Software is built by teams. Now teams are built differently. AgentsMesh helps organizations scale beyond headcount with AI agent teams.",
  alternates: {
    canonical: "https://agentsmesh.ai/about",
  },
  openGraph: {
    title: "About | AgentsMesh",
    description:
      "Software is built by teams. Now teams are built differently. AgentsMesh helps organizations scale beyond headcount with AI agent teams.",
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
