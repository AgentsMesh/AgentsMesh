import { createDocsMetadata } from "@/lib/docs-metadata";

export const metadata = createDocsMetadata("/docs/features/tickets");

export default function Layout({ children }: { children: React.ReactNode }) {
  return children;
}
