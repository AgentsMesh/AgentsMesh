import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Privacy Policy",
  description:
    "AgentsMesh privacy policy — how we collect, use, and protect your data.",
  alternates: {
    canonical: "https://agentsmesh.ai/privacy",
  },
  openGraph: {
    title: "Privacy Policy | AgentsMesh",
    description:
      "AgentsMesh privacy policy — how we collect, use, and protect your data.",
    url: "https://agentsmesh.ai/privacy",
  },
};

export default function PrivacyLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return children;
}
