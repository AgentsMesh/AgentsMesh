"use client";

import { useMemo, useSyncExternalStore } from "react";
import { useTranslations } from "next-intl";
import { Download as DownloadIcon, ExternalLink as ExternalIcon } from "lucide-react";
import type { DesktopAsset, ReleaseSummary } from "@/lib/download/asset-types";
import { platformLabel } from "@/lib/download/asset-types";
import {
  type DetectedPlatform,
  getDetectedPlatform,
} from "@/lib/download/platform-detect";
import { pickPrimaryDesktop } from "@/lib/download/asset-picker";
import { PlatformIcon } from "./PlatformIcon";

interface Props {
  release: ReleaseSummary;
}

const subscribe = () => () => {};
const getServerSnapshot = () => null;
const DATE_FMT: Intl.DateTimeFormatOptions = { year: "numeric", month: "short", day: "numeric" };

function useDetected(): DetectedPlatform | null {
  return useSyncExternalStore(subscribe, getDetectedPlatform, getServerSnapshot);
}

function formatReleaseDate(iso: string): string {
  return iso ? new Date(iso).toLocaleDateString(undefined, DATE_FMT) : "";
}

export function DownloadHero({ release }: Props) {
  const t = useTranslations();
  const detected = useDetected();

  const primary: DesktopAsset | null = useMemo(
    () => pickPrimaryDesktop(release.desktop, detected),
    [release.desktop, detected],
  );
  const releaseDate = formatReleaseDate(release.publishedAt);

  return (
    <section className="relative pt-36 pb-16 px-4 overflow-hidden">
      <div className="absolute inset-0 azure-mesh-bg pointer-events-none" />
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[900px] h-[900px] bg-[var(--azure-cyan)]/10 blur-[180px] rounded-full pointer-events-none" />

      <div className="container mx-auto max-w-4xl text-center relative z-10">
        <div className="inline-flex items-center gap-2 px-4 py-1.5 mb-6 rounded-full border border-[var(--azure-cyan)]/30 bg-[var(--azure-cyan)]/5 text-[var(--azure-cyan)]">
          <span className="w-1.5 h-1.5 rounded-full bg-[var(--azure-cyan)] animate-pulse" />
          <span className="text-[10px] font-headline uppercase tracking-[0.25em] font-semibold">
            {t("landing.download.hero.badge")} v{release.version}
          </span>
        </div>

        <h1 className="font-headline text-5xl md:text-6xl font-bold mb-6 leading-[1.05] tracking-tight">
          {t("landing.download.hero.title")}{" "}
          <span className="azure-gradient-text">{t("landing.download.hero.titleHighlight")}</span>
        </h1>
        <p className="text-lg md:text-xl text-[var(--azure-text-muted)] max-w-2xl mx-auto mb-10 font-light">
          {t("landing.download.hero.subtitle")}
        </p>

        {primary && (
          <div className="flex flex-col items-center gap-3 mb-6">
            <a
              href={primary.url}
              className="inline-flex items-center gap-3 px-8 py-4 rounded-full azure-gradient-bg azure-cta-glow font-headline font-bold text-base tracking-wide"
            >
              <DownloadIcon className="w-5 h-5" />
              <span>
                {t("landing.download.hero.downloadFor", { platform: platformLabel(primary.platform) })}
              </span>
              <PlatformIcon platform={primary.platform} className="w-5 h-5" />
            </a>
            <p className="text-xs text-[var(--azure-text-muted)]/70 uppercase tracking-[0.18em]">
              {t("landing.download.hero.releasedOn", { date: releaseDate })} · {primary.name}
            </p>
          </div>
        )}

        <a
          href={release.htmlUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-2 text-sm text-[var(--azure-text-muted)] hover:text-[var(--azure-cyan)] transition-colors"
        >
          {t("landing.download.hero.viewReleaseNotes")}
          <ExternalIcon className="w-4 h-4" />
        </a>
      </div>
    </section>
  );
}
