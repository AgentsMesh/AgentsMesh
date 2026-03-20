import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Enterprise",
  description: "Self-hosted AI agent orchestration and agent swarm management for enterprises. Harness engineering productivity with full data control, air-gapped deployment, SSO, audit logs, and dedicated support.",
  alternates: {
    canonical: "https://agentsmesh.ai/enterprise",
  },
  openGraph: {
    title: "Enterprise | AgentsMesh",
    description: "Self-hosted agent orchestration and agent swarm management for enterprises. Harness engineering productivity with full data control, air-gapped deployment, and dedicated support.",
    url: "https://agentsmesh.ai/enterprise",
  },
};

export default function EnterpriseLayout({ children }: { children: React.ReactNode }) {
  return children;
}
