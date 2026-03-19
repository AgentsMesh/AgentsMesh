import Link from "next/link";
import type { Metadata } from "next";
import { PageHeader, PageFooter } from "@/components/common";
import { getTranslations } from "next-intl/server";

export const metadata: Metadata = {
  title: "Page Not Found",
};

export default async function NotFound() {
  const t = await getTranslations();

  return (
    <div className="min-h-screen bg-background flex flex-col">
      <PageHeader />

      <main className="flex-1 flex items-center justify-center px-4">
        <div className="text-center">
          <h1 className="text-8xl font-bold text-primary mb-4">404</h1>
          <p className="text-xl text-muted-foreground mb-8">
            {t("common.pageNotFound")}
          </p>
          <Link
            href="/"
            className="inline-flex items-center gap-2 px-6 py-3 rounded-lg bg-primary text-primary-foreground font-medium hover:bg-primary/90 transition-colors"
          >
            {t("common.backToHome")}
          </Link>
        </div>
      </main>

      <PageFooter />
    </div>
  );
}
