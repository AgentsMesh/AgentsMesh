import { createDocsMetadata } from "@/lib/docs-metadata";

export const metadata = createDocsMetadata("/docs/guides/multi-agent-workflows");

export default function Layout({ children }: { children: React.ReactNode }) {
  return children;
}
