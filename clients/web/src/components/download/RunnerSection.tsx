"use client";

import { type RunnerAsset, RUNNER_KIND_EXT, platformLabel, archLabel } from "@/lib/download/asset-types";
import { useTranslations } from "next-intl";
import { Terminal as TerminalIcon, Download as DownloadIcon } from "lucide-react";
import { formatBytes } from "@/lib/download/platform-detect";
import { INSTALL_COMMANDS } from "@/lib/download/install-commands";
import { CopyButton } from "./CopyButton";

interface Props {
  runner: RunnerAsset[];
}

export function RunnerSection({ runner }: Props) {
  const t = useTranslations();

  return (
    <section className="py-24 px-4 relative bg-[var(--azure-bg-deeper)]">
      <div className="container mx-auto max-w-6xl">
        <div className="text-center mb-14">
          <div className="inline-flex items-center gap-2 mb-4 text-[var(--azure-cyan)]">
            <TerminalIcon className="w-5 h-5" />
            <span className="text-[10px] font-headline uppercase tracking-[0.25em] font-semibold">
              {t("landing.download.runner.badge")}
            </span>
          </div>
          <h2 className="font-headline text-3xl md:text-4xl font-bold mb-4">
            {t("landing.download.runner.title")}
          </h2>
          <p className="text-[var(--azure-text-muted)] max-w-2xl mx-auto">
            {t("landing.download.runner.description")}
          </p>
        </div>

        <div className="grid md:grid-cols-2 gap-6 mb-10">
          <InstallSnippet label={t("landing.download.runner.macLinux")} command={INSTALL_COMMANDS.unix} />
          <InstallSnippet label={t("landing.download.runner.windows")} command={INSTALL_COMMANDS.windows} />
        </div>

        <div className="mb-4 text-center">
          <h3 className="font-headline text-sm uppercase tracking-[0.25em] text-[var(--azure-text-muted)]/80">
            {t("landing.download.runner.manualTitle")}
          </h3>
        </div>

        <div className="grid sm:grid-cols-2 md:grid-cols-3 gap-3">
          {runner.map((asset) => (
            <a
              key={asset.url}
              href={asset.url}
              className="group flex items-center justify-between gap-3 px-4 py-3 rounded-lg border border-white/10 bg-[var(--azure-bg-card)]/50 hover:border-[var(--azure-cyan)]/40 hover:bg-white/5 transition-all"
            >
              <div className="min-w-0">
                <div className="text-sm font-medium truncate">
                  {platformLabel(asset.platform)} · {archLabel(asset.arch)}
                </div>
                <div className="text-[10px] uppercase tracking-wider text-[var(--azure-text-muted)]/70">
                  {RUNNER_KIND_EXT[asset.kind]} · {formatBytes(asset.size)}
                </div>
              </div>
              <DownloadIcon className="w-4 h-4 text-[var(--azure-text-muted)] group-hover:text-[var(--azure-cyan)] transition-colors" />
            </a>
          ))}
        </div>
      </div>
    </section>
  );
}

function InstallSnippet({ label, command }: { label: string; command: string }) {
  return (
    <div className="relative rounded-xl border border-white/10 bg-[var(--azure-bg-low)]/80 p-5 pt-12">
      <div className="absolute top-3 left-4 text-[10px] font-headline uppercase tracking-[0.18em] text-[var(--azure-text-muted)]/70">
        {label}
      </div>
      <CopyButton text={command} />
      <pre className="font-mono text-sm text-[var(--azure-cyan-soft)] overflow-x-auto">
        <code>{command}</code>
      </pre>
    </div>
  );
}
