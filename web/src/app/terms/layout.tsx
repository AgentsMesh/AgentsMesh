import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Terms of Service",
  description:
    "AgentsMesh terms of service — the agreement governing your use of our platform.",
  alternates: {
    canonical: "https://agentsmesh.ai/terms",
  },
  openGraph: {
    title: "Terms of Service | AgentsMesh",
    description:
      "AgentsMesh terms of service — the agreement governing your use of our platform.",
    url: "https://agentsmesh.ai/terms",
  },
};

export default function TermsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return children;
}
