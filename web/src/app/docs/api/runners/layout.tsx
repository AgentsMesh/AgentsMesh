import { createDocsMetadata } from "@/lib/docs-metadata";

export const metadata = createDocsMetadata("/docs/api/runners");

export default function Layout({ children }: { children: React.ReactNode }) {
  return children;
}
