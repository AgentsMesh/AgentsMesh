"use client";

import Link from "next/link";
import { useTranslations } from "next-intl";

/**
 * Shared footer for public/static pages (about, careers, changelog, privacy, terms, blog).
 * Single source of truth — avoids duplicating footer markup across every page.
 */
export function PageFooter() {
  const t = useTranslations();

  return (
    <footer className="border-t border-border mt-16">
      <div className="container mx-auto px-4 py-8">
        <div className="flex flex-col md:flex-row justify-between items-center gap-4">
          <p className="text-sm text-muted-foreground">
            &copy; {new Date().getFullYear()} AgentsMesh.{" "}
            {t("common.allRightsReserved")}
          </p>
          <div className="flex gap-6">
            <Link
              href="/privacy"
              className="text-sm text-muted-foreground hover:text-foreground"
            >
              {t("landing.footer.legal.privacy")}
            </Link>
            <Link
              href="/terms"
              className="text-sm text-muted-foreground hover:text-foreground"
            >
              {t("landing.footer.legal.terms")}
            </Link>
            <Link
              href="/docs"
              className="text-sm text-muted-foreground hover:text-foreground"
            >
              {t("landing.footer.resources.documentation")}
            </Link>
          </div>
        </div>
      </div>
    </footer>
  );
}
