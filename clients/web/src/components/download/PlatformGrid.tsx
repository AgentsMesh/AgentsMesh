"use client";

import type { DesktopAsset, DesktopKind, Platform } from "@/lib/download/asset-types";
import { useTranslations } from "next-intl";
import { PlatformCard } from "./PlatformCard";

interface Props {
  desktop: DesktopAsset[];
}

const PRIMARY_KIND: Record<Platform, DesktopKind> = {
  macos: "dmg",
  windows: "exe",
  linux: "appimage",
};

const includeAsset = (a: DesktopAsset): boolean =>
  a.platform !== "macos" || a.kind === "dmg" || a.kind === "zip";

function partition(desktop: DesktopAsset[]): Record<Platform, DesktopAsset[]> {
  const buckets: Record<Platform, DesktopAsset[]> = { macos: [], windows: [], linux: [] };
  for (const a of desktop) {
    if (includeAsset(a)) buckets[a.platform].push(a);
  }
  for (const p of Object.keys(buckets) as Platform[]) {
    const primary = PRIMARY_KIND[p];
    buckets[p].sort((a, b) => Number(a.kind !== primary) - Number(b.kind !== primary));
  }
  return buckets;
}

export function PlatformGrid({ desktop }: Props) {
  const t = useTranslations();
  const byPlatform = partition(desktop);

  return (
    <section className="py-24 px-4 relative">
      <div className="container mx-auto max-w-6xl">
        <div className="text-center mb-14">
          <h2 className="font-headline text-3xl md:text-4xl font-bold mb-4">
            {t("landing.download.platforms.title")}
          </h2>
          <p className="text-[var(--azure-text-muted)] max-w-2xl mx-auto">
            {t("landing.download.platforms.description")}
          </p>
        </div>

        <div className="grid md:grid-cols-3 gap-6">
          {(Object.keys(byPlatform) as Platform[]).map((p) => (
            <PlatformCard
              key={p}
              platform={p}
              assets={byPlatform[p]}
              title={t(`landing.download.platforms.${p}.title`)}
              requirements={t(`landing.download.platforms.${p}.requirements`)}
            />
          ))}
        </div>
      </div>
    </section>
  );
}
