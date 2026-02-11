"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useTranslations } from "next-intl";
import { getPrevNext } from "@/lib/docs-navigation";

export function DocNavigation() {
  const pathname = usePathname();
  const t = useTranslations();
  const { prev, next } = getPrevNext(pathname);

  if (!prev && !next) return null;

  return (
    <nav className="mt-16 pt-8 border-t border-border flex justify-between items-center">
      {prev ? (
        <Link
          href={prev.href}
          className="group flex flex-col items-start gap-1 text-sm hover:text-primary transition-colors"
        >
          <span className="text-muted-foreground text-xs">
            ← {t("docs.pagination.previous")}
          </span>
          <span className="font-medium group-hover:underline">
            {t(prev.titleKey)}
          </span>
        </Link>
      ) : (
        <div />
      )}
      {next ? (
        <Link
          href={next.href}
          className="group flex flex-col items-end gap-1 text-sm hover:text-primary transition-colors"
        >
          <span className="text-muted-foreground text-xs">
            {t("docs.pagination.next")} →
          </span>
          <span className="font-medium group-hover:underline">
            {t(next.titleKey)}
          </span>
        </Link>
      ) : (
        <div />
      )}
    </nav>
  );
}
