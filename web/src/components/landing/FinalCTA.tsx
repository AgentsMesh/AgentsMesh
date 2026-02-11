"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { useTranslations } from "next-intl";

export function FinalCTA() {
  const t = useTranslations();

  return (
    <section className="py-24 relative overflow-hidden">
      {/* Background effects */}
      <div className="absolute inset-0 bg-gradient-to-t from-primary/10 via-transparent to-transparent" />
      <div className="absolute bottom-0 left-1/2 -translate-x-1/2 w-[800px] h-[400px] bg-primary/20 blur-[120px] rounded-full" />

      <div className="container mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        <div className="text-center max-w-3xl mx-auto">
          <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold mb-6">
            {t("landing.finalCta.title1")}
            <br />
            <span className="text-primary">{t("landing.finalCta.title2")}</span>
          </h2>

          <p className="text-lg text-muted-foreground mb-8">
            {t("landing.finalCta.description")}
          </p>

          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            <Link href="/register">
              <Button
                size="lg"
                className="w-full sm:w-auto bg-primary text-primary-foreground hover:bg-primary/90 text-base px-8"
              >
                {t("landing.finalCta.getStartedFree")}
              </Button>
            </Link>
            <a href="mailto:bd@agentsmesh.ai">
              <Button size="lg" variant="outline" className="w-full sm:w-auto text-base px-8">
                {t("landing.finalCta.scheduleDemo")}
              </Button>
            </a>
          </div>

          {/* Quick stats */}
          <div className="mt-12 flex flex-wrap justify-center gap-8 text-sm text-muted-foreground">
            <div className="flex items-center gap-2">
              <svg className="w-4 h-4 text-green-500 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
              {t("landing.finalCta.freeTier")}
            </div>
            <div className="flex items-center gap-2">
              <svg className="w-4 h-4 text-green-500 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
              {t("landing.finalCta.noCreditCard")}
            </div>
            <div className="flex items-center gap-2">
              <svg className="w-4 h-4 text-green-500 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
              {t("landing.finalCta.selfHostedOption")}
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
