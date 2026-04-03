"use client";

import { useTranslations } from "next-intl";

/**
 * ParadigmShift - "Before / After" comparison section
 * Replaces WhyTerminalBased with a user-centric narrative
 */
export function ParadigmShift() {
  const t = useTranslations();

  const beforeItems = [
    t("landing.paradigmShift.before.items.0"),
    t("landing.paradigmShift.before.items.1"),
    t("landing.paradigmShift.before.items.2"),
    t("landing.paradigmShift.before.items.3"),
  ];

  const afterItems = [
    t("landing.paradigmShift.after.items.0"),
    t("landing.paradigmShift.after.items.1"),
    t("landing.paradigmShift.after.items.2"),
    t("landing.paradigmShift.after.items.3"),
  ];

  return (
    <section className="py-24 relative" id="paradigm-shift">
      <div className="absolute inset-0 bg-gradient-to-b from-transparent via-primary/5 to-transparent" />

      <div className="container mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        {/* Section header */}
        <div className="text-center mb-16">
          <h2 className="text-3xl sm:text-4xl font-bold mb-4">
            {t("landing.paradigmShift.title")}
          </h2>
          <p className="text-xl sm:text-2xl font-semibold bg-gradient-to-r from-primary to-primary/60 bg-clip-text text-transparent">
            {t("landing.paradigmShift.titleHighlight")}
          </p>
        </div>

        {/* Before / After comparison */}
        <div className="max-w-4xl mx-auto grid md:grid-cols-2 gap-6 mb-12">
          {/* Before column */}
          <div className="p-6 bg-secondary/20 rounded-xl border border-border/50 relative overflow-hidden">
            <div className="absolute top-0 left-0 right-0 h-1 bg-gradient-to-r from-red-500/50 to-red-500/20" />
            <h3 className="text-sm font-bold uppercase tracking-wider text-muted-foreground mb-6">
              {t("landing.paradigmShift.before.label")}
            </h3>
            <ul className="space-y-4">
              {beforeItems.map((item, i) => (
                <li key={i} className="flex items-start gap-3">
                  <div className="mt-1 w-5 h-5 rounded-full bg-red-500/10 flex items-center justify-center flex-shrink-0">
                    <svg className="w-3 h-3 text-red-500 dark:text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </div>
                  <span className="text-muted-foreground">{item}</span>
                </li>
              ))}
            </ul>
          </div>

          {/* After column */}
          <div className="p-6 bg-primary/5 rounded-xl border border-primary/20 relative overflow-hidden">
            <div className="absolute top-0 left-0 right-0 h-1 bg-gradient-to-r from-primary to-primary/50" />
            <h3 className="text-sm font-bold uppercase tracking-wider text-primary mb-6">
              {t("landing.paradigmShift.after.label")}
            </h3>
            <ul className="space-y-4">
              {afterItems.map((item, i) => (
                <li key={i} className="flex items-start gap-3">
                  <div className="mt-1 w-5 h-5 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0">
                    <svg className="w-3 h-3 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
                    </svg>
                  </div>
                  <span className="text-foreground">{item}</span>
                </li>
              ))}
            </ul>
          </div>
        </div>

        {/* Punchline */}
        <div className="text-center">
          <p className="text-lg sm:text-xl font-medium text-muted-foreground italic">
            {t("landing.paradigmShift.punchline")}
          </p>
        </div>
      </div>
    </section>
  );
}
