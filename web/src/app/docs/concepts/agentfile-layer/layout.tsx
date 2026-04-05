import { createDocsMetadata } from "@/lib/docs-metadata";

export const metadata = createDocsMetadata("/docs/concepts/agentfile-layer");

export default function Layout({ children }: { children: React.ReactNode }) {
  return children;
}
