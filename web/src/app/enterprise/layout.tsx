import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Enterprise",
  description: "Your agents. Your infrastructure. Your rules. Enterprise-grade security and governance for organizations managing agent workforces at scale.",
  alternates: {
    canonical: "https://agentsmesh.ai/enterprise",
  },
  openGraph: {
    title: "Enterprise | AgentsMesh",
    description: "Your agents. Your infrastructure. Your rules. Enterprise-grade security and governance for organizations managing agent workforces at scale.",
    url: "https://agentsmesh.ai/enterprise",
  },
};

export default function EnterpriseLayout({ children }: { children: React.ReactNode }) {
  return children;
}
