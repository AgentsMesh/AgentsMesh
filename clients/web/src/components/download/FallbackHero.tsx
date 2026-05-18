"use client";

import { useTranslations } from "next-intl";
import { ExternalLink as ExternalIcon } from "lucide-react";
import { RELEASES_PAGE_URL } from "@/lib/download/github-release";

export function FallbackHero() {
  const t = useTranslations();

  return (
    <section className="relative pt-36 pb-32 px-4 overflow-hidden">
      <div className="absolute inset-0 azure-mesh-bg pointer-events-none" />

      <div className="container mx-auto max-w-3xl text-center relative z-10">
        <h1 className="font-headline text-4xl md:text-5xl font-bold mb-6 tracking-tight">
          {t("landing.download.fallback.title")}
        </h1>
        <p className="text-[var(--azure-text-muted)] mb-8">{t("landing.download.fallback.description")}</p>
        <a
          href={RELEASES_PAGE_URL}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-2 px-7 py-3.5 rounded-full azure-gradient-bg font-headline font-bold tracking-wide azure-cta-glow"
        >
          {t("landing.download.fallback.cta")}
          <ExternalIcon className="w-4 h-4" />
        </a>
      </div>
    </section>
  );
}
