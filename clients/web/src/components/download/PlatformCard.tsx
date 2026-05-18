"use client";

import type { DesktopAsset, Platform } from "@/lib/download/asset-types";
import { useTranslations } from "next-intl";
import { PlatformIcon } from "./PlatformIcon";
import { AssetDownloadButton } from "./AssetDownloadButton";

interface Props {
  platform: Platform;
  assets: DesktopAsset[];
  title: string;
  requirements: string;
}

export function PlatformCard({ platform, assets, title, requirements }: Props) {
  const t = useTranslations();

  return (
    <div className="relative group flex flex-col rounded-2xl border border-white/10 bg-[var(--azure-bg-card)]/60 backdrop-blur p-7 transition-all hover:border-[var(--azure-cyan)]/40 hover:azure-glow-cyan">
      <div className="flex items-center gap-3 mb-2 text-[var(--azure-cyan)]">
        <PlatformIcon platform={platform} />
        <h3 className="font-headline text-2xl font-bold tracking-tight">{title}</h3>
      </div>
      <p className="text-xs uppercase tracking-[0.18em] text-[var(--azure-text-muted)]/70 mb-6">{requirements}</p>

      {assets.length === 0 ? (
        <div className="flex-1 flex flex-col items-center justify-center py-10 text-center">
          <div className="text-sm text-[var(--azure-text-muted)] mb-1">{t("landing.download.platforms.unavailable")}</div>
          <div className="text-xs text-[var(--azure-text-muted)]/60">{t("landing.download.platforms.checkBack")}</div>
        </div>
      ) : (
        <div className="flex flex-col gap-3 flex-1">
          {assets.map((asset, i) => (
            <AssetDownloadButton key={asset.url} asset={asset} primary={i === 0} />
          ))}
        </div>
      )}
    </div>
  );
}
