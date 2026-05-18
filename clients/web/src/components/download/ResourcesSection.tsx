"use client";

import Link from "next/link";
import { useTranslations } from "next-intl";
import { ExternalLink as ExternalIcon } from "lucide-react";
import { RELEASES_PAGE_URL } from "@/lib/download/github-release";

interface Props {
  checksumsUrl?: string;
}

export function ResourcesSection({ checksumsUrl }: Props) {
  const t = useTranslations();

  return (
    <section className="py-24 px-4">
      <div className="container mx-auto max-w-4xl">
        <div className="grid md:grid-cols-2 gap-6">
          <ResourceCard
            title={t("landing.download.resources.older.title")}
            description={t("landing.download.resources.older.description")}
            href={RELEASES_PAGE_URL}
            external
            cta={t("landing.download.resources.older.cta")}
          />
          {checksumsUrl && (
            <ResourceCard
              title={t("landing.download.resources.verify.title")}
              description={t("landing.download.resources.verify.description")}
              href={checksumsUrl}
              external
              cta={t("landing.download.resources.verify.cta")}
            />
          )}
          <ResourceCard
            title={t("landing.download.resources.runner.title")}
            description={t("landing.download.resources.runner.description")}
            href="/docs/runners/setup"
            cta={t("landing.download.resources.runner.cta")}
          />
          <ResourceCard
            title={t("landing.download.resources.changelog.title")}
            description={t("landing.download.resources.changelog.description")}
            href="/changelog"
            cta={t("landing.download.resources.changelog.cta")}
          />
        </div>
      </div>
    </section>
  );
}

interface ResourceCardProps {
  title: string;
  description: string;
  href: string;
  cta: string;
  external?: boolean;
}

function ResourceCard({ title, description, href, cta, external }: ResourceCardProps) {
  const className = "group block rounded-xl border border-white/10 bg-[var(--azure-bg-card)]/40 hover:bg-[var(--azure-bg-card)]/70 hover:border-[var(--azure-cyan)]/40 p-6 transition-all";
  const content = (
    <>
      <h3 className="font-headline text-lg font-bold mb-2 group-hover:text-[var(--azure-cyan)] transition-colors">
        {title}
      </h3>
      <p className="text-sm text-[var(--azure-text-muted)] mb-4 leading-relaxed">{description}</p>
      <span className="inline-flex items-center gap-1.5 text-xs font-headline uppercase tracking-[0.18em] text-[var(--azure-cyan)]">
        {cta}
        {external ? <ExternalIcon className="w-3.5 h-3.5" /> : <span>→</span>}
      </span>
    </>
  );

  if (external) {
    return (
      <a href={href} target="_blank" rel="noopener noreferrer" className={className}>
        {content}
      </a>
    );
  }
  return <Link href={href} className={className}>{content}</Link>;
}
