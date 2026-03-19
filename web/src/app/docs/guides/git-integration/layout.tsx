import { createDocsMetadata } from "@/lib/docs-metadata";

export const metadata = createDocsMetadata("/docs/guides/git-integration");

export default function Layout({ children }: { children: React.ReactNode }) {
  return children;
}
