import { createDocsMetadata } from "@/lib/docs-metadata";

export const metadata = createDocsMetadata("/docs/features/workspace");

export default function Layout({ children }: { children: React.ReactNode }) {
  return children;
}
