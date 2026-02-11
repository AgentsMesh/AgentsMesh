"use client";

import { useTranslations } from "next-intl";

export function WhyTerminalBased() {
  const t = useTranslations();

  const benefits = [
    {
      icon: (
        <svg className="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M13 10V3L4 14h7v7l9-11h-7z" />
        </svg>
      ),
      title: t("landing.whyTerminal.benefits.autonomous.title"),
      description: t("landing.whyTerminal.benefits.autonomous.description"),
    },
    {
      icon: (
        <svg className="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
        </svg>
      ),
      title: t("landing.whyTerminal.benefits.fullCapabilities.title"),
      description: t("landing.whyTerminal.benefits.fullCapabilities.description"),
    },
    {
      icon: (
        <svg className="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
        </svg>
      ),
      title: t("landing.whyTerminal.benefits.dataControl.title"),
      description: t("landing.whyTerminal.benefits.dataControl.description"),
    },
  ];

  const comparisonData = [
    { feature: t("landing.whyTerminal.comparison.autonomy"), ide: t("landing.whyTerminal.comparison.ide.autonomy"), terminal: t("landing.whyTerminal.comparison.terminal.autonomy") },
    { feature: t("landing.whyTerminal.comparison.capabilities"), ide: t("landing.whyTerminal.comparison.ide.capabilities"), terminal: t("landing.whyTerminal.comparison.terminal.capabilities") },
    { feature: t("landing.whyTerminal.comparison.environment"), ide: t("landing.whyTerminal.comparison.ide.environment"), terminal: t("landing.whyTerminal.comparison.terminal.environment") },
    { feature: t("landing.whyTerminal.comparison.multiAgent"), ide: t("landing.whyTerminal.comparison.ide.multiAgent"), terminal: t("landing.whyTerminal.comparison.terminal.multiAgent") },
    { feature: t("landing.whyTerminal.comparison.selfHosted"), ide: t("landing.whyTerminal.comparison.ide.selfHosted"), terminal: t("landing.whyTerminal.comparison.terminal.selfHosted") },
  ];

  return (
    <section className="py-24 relative" id="why-terminal">
      {/* Background */}
      <div className="absolute inset-0 bg-gradient-to-b from-transparent via-primary/5 to-transparent" />

      <div className="container mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        {/* Section header */}
        <div className="text-center mb-16">
          <h2 className="text-3xl sm:text-4xl font-bold mb-4">
            {t("landing.whyTerminal.title")} <span className="text-primary">{t("landing.whyTerminal.titleHighlight")}</span>{t("landing.whyTerminal.titleEnd")}
          </h2>
          <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
            {t("landing.whyTerminal.description")}
          </p>
        </div>

        {/* Benefits cards */}
        <div className="grid md:grid-cols-3 gap-6 mb-16">
          {benefits.map((benefit, index) => (
            <div
              key={index}
              className="p-6 bg-secondary/20 rounded-xl border border-border hover:border-primary/50 transition-all duration-300 hover:-translate-y-1"
            >
              <div className="w-14 h-14 rounded-lg bg-primary/10 flex items-center justify-center text-primary mb-4">
                {benefit.icon}
              </div>
              <h3 className="text-xl font-semibold mb-2">{benefit.title}</h3>
              <p className="text-muted-foreground">{benefit.description}</p>
            </div>
          ))}
        </div>

        {/* Comparison table */}
        <div className="max-w-4xl mx-auto">
          <div className="bg-card rounded-xl border border-border overflow-hidden">
            {/* Table header */}
            <div className="grid grid-cols-3 bg-muted border-b border-border">
              <div className="p-4 font-semibold text-sm"></div>
              <div className="p-4 font-semibold text-sm text-center text-muted-foreground">
                {t("landing.whyTerminal.comparison.idePlugins")}
                <div className="text-xs font-normal mt-1">{t("landing.whyTerminal.comparison.idePluginsSubtitle")}</div>
              </div>
              <div className="p-4 font-semibold text-sm text-center text-primary">
                {t("landing.whyTerminal.comparison.terminalBased")}
                <div className="text-xs font-normal mt-1 text-primary/70">{t("landing.whyTerminal.comparison.terminalBasedSubtitle")}</div>
              </div>
            </div>

            {/* Table rows */}
            {comparisonData.map((row, index) => (
              <div
                key={index}
                className={`grid grid-cols-3 border-b border-border last:border-b-0 ${
                  index % 2 === 0 ? "bg-transparent" : "bg-secondary/10"
                }`}
              >
                <div className="p-4 font-medium text-sm">{row.feature}</div>
                <div className="p-4 text-sm text-center text-muted-foreground">
                  <span className="inline-flex items-center gap-1">
                    <svg className="w-4 h-4 text-red-500 dark:text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                    </svg>
                    {row.ide}
                  </span>
                </div>
                <div className="p-4 text-sm text-center">
                  <span className="inline-flex items-center gap-1 text-green-500 dark:text-green-400">
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    {row.terminal}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}
