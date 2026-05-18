import { Download as DownloadIcon } from "lucide-react";
import { type DesktopAsset, DESKTOP_KIND_LABEL, archLabel } from "@/lib/download/asset-types";
import { formatBytes } from "@/lib/download/platform-detect";

interface Props {
  asset: DesktopAsset;
  primary?: boolean;
  customLabel?: string;
}

export function AssetDownloadButton({ asset, primary, customLabel }: Props) {
  const archText = asset.arch === "universal" ? "x64 / ARM64" : archLabel(asset.arch);
  const subLabel = customLabel ?? `${archText} · ${DESKTOP_KIND_LABEL[asset.kind]}`;

  const baseCls = primary
    ? "azure-gradient-bg azure-cta-glow font-semibold"
    : "border border-white/10 hover:border-[var(--azure-cyan)]/40 bg-white/5 hover:bg-white/10 text-foreground";

  return (
    <a
      href={asset.url}
      className={`group flex items-center justify-between gap-4 px-5 py-3.5 rounded-xl transition-all ${baseCls}`}
    >
      <div className="flex items-center gap-3 min-w-0">
        <DownloadIcon className="w-5 h-5 flex-shrink-0" />
        <div className="min-w-0">
          <div className={`text-sm ${primary ? "font-semibold" : "font-medium"} truncate`}>
            {subLabel}
          </div>
          <div className={`text-[10px] uppercase tracking-wider ${primary ? "opacity-80" : "text-[var(--azure-text-muted)]/70"}`}>
            {formatBytes(asset.size)}
          </div>
        </div>
      </div>
      <span className={`text-xs font-headline tracking-[0.2em] uppercase opacity-70 group-hover:opacity-100 transition-opacity`}>
        ↓
      </span>
    </a>
  );
}
