import type { Metadata } from "next";
import DocsShell from "@/components/docs/DocsShell";

export const metadata: Metadata = {
  title: {
    template: "%s | AgentsMesh Docs",
    default: "Documentation",
  },
  description:
    "AgentsMesh documentation — orchestrate AI coding agents at scale.",
};

export default function DocsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <DocsShell>{children}</DocsShell>;
}
