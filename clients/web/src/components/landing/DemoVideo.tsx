"use client";

import { useTranslations } from "next-intl";

export function DemoVideo() {
  const t = useTranslations();

  return (
    <section className="py-24 relative overflow-hidden">
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[600px] bg-[var(--azure-cyan)]/5 blur-[120px] rounded-full pointer-events-none" />

      <div className="container mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        <div className="text-center max-w-3xl mx-auto mb-12">
          <h2 className="font-headline text-4xl md:text-5xl font-bold mb-4">
            {t("landing.demoVideo.title")}{" "}
            <span className="azure-gradient-text">{t("landing.demoVideo.titleHighlight")}</span>
          </h2>
          <p className="text-[var(--azure-text-muted)] font-light text-lg">
            {t("landing.demoVideo.description")}
          </p>
        </div>

        <div className="max-w-4xl mx-auto relative">
          <div className="azure-glass rounded-3xl border border-white/10 azure-glow-cyan-lg p-3 sm:p-4">
            <div
              className="relative rounded-2xl overflow-hidden bg-[var(--azure-bg-deeper)]"
              style={{ aspectRatio: "16/9" }}
            >
              <iframe
                src="https://www.youtube-nocookie.com/embed/FZrUO0tim0U?rel=0&modestbranding=1"
                title={t("landing.demoVideo.iframeTitle")}
                allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
                allowFullScreen
                loading="lazy"
                className="absolute inset-0 w-full h-full"
              />
            </div>
          </div>

          <div className="flex flex-wrap justify-center gap-3 mt-8">
            {[
              t("landing.demoVideo.highlight1"),
              t("landing.demoVideo.highlight2"),
              t("landing.demoVideo.highlight3"),
            ].map((label) => (
              <span
                key={label}
                className="inline-flex items-center gap-2 px-4 py-2 rounded-full text-sm font-medium azure-glass border border-white/10 text-[var(--azure-text-muted)] hover:border-[var(--azure-cyan)]/30 hover:text-[var(--azure-cyan)] transition-all"
              >
                <span className="w-1.5 h-1.5 rounded-full bg-[var(--azure-mint)]" />
                {label}
              </span>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}
