import { createDocsMetadata } from "@/lib/docs-metadata";

export const metadata = createDocsMetadata("/docs/features/loops");

export default function Layout({ children }: { children: React.ReactNode }) {
  return children;
}
