import { createDocsMetadata } from "@/lib/docs-metadata";

export const metadata = createDocsMetadata("/docs/runners/mcp-tools");

export default function Layout({ children }: { children: React.ReactNode }) {
  return children;
}
