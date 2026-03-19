import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Changelog",
  description:
    "Stay up to date with the latest AgentsMesh features, improvements, and bug fixes.",
  alternates: {
    canonical: "https://agentsmesh.ai/changelog",
  },
  openGraph: {
    title: "Changelog | AgentsMesh",
    description:
      "Stay up to date with the latest AgentsMesh features, improvements, and bug fixes.",
    url: "https://agentsmesh.ai/changelog",
  },
};

export default function ChangelogLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return children;
}
