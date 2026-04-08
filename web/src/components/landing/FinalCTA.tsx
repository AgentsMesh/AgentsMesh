"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { useTranslations } from "next-intl";

export function FinalCTA() {
  const t = useTranslations();

  return (
    <section className="py-24 relative overflow-hidden">
      {/* Background effects - enhanced */}
      <div className="absolute inset-0 bg-gradient-to-t from-primary/10 via-primary/3 to-transparent" />
      <div className="absolute bottom-0 left-1/2 -translate-x-1/2 w-[800px] h-[400px] bg-primary/25 blur-[120px] rounded-full animate-glow-pulse" />
      <div className="absolute top-1/4 left-1/4 w-[300px] h-[300px] bg-primary/10 blur-[100px] rounded-full animate-glow-pulse" style={{ animationDelay: '1s' }} />
      <div className="absolute top-1/3 right-1/4 w-[250px] h-[250px] bg-primary/8 blur-[80px] rounded-full animate-glow-pulse" style={{ animationDelay: '2s' }} />
      {/* Grid overlay */}
      <div
        className="absolute inset-0 opacity-[0.03]"
        style={{
          backgroundImage: `linear-gradient(var(--primary) 1px, transparent 1px),
                           linear-gradient(90deg, var(--primary) 1px, transparent 1px)`,
          backgroundSize: '60px 60px'
        }}
      />

      <div className="container mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        <div className="text-center max-w-3xl mx-auto">
          <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold mb-6">
            {t("landing.finalCta.title1")}
            <br />
            <span className="bg-gradient-to-r from-primary to-primary/60 bg-clip-text text-transparent">{t("landing.finalCta.title2")}</span>
          </h2>

          <p className="text-lg text-muted-foreground mb-8">
            {t("landing.finalCta.description")}
          </p>

          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            <Link href="/register">
              <Button
                size="lg"
                className="relative w-full sm:w-auto bg-primary text-primary-foreground hover:bg-primary/90 text-base px-8 shadow-lg shadow-primary/25 hover:shadow-primary/50 hover:shadow-xl transition-all hover:-translate-y-0.5 overflow-hidden group"
              >
                <span className="relative z-10">{t("landing.finalCta.getStartedFree")}</span>
                <div className="absolute inset-0 bg-gradient-to-r from-primary via-primary/80 to-primary opacity-0 group-hover:opacity-100 transition-opacity" />
              </Button>
            </Link>
            <Link href="/login">
              <Button size="lg" variant="outline" className="w-full sm:w-auto text-base px-8">
                {t("landing.finalCta.signInConsole")}
              </Button>
            </Link>
          </div>

          <p className="mt-4 text-sm text-muted-foreground/60">
            {t("landing.finalCta.enterpriseNote")}{" "}
            <Link href="/demo" className="underline hover:text-primary transition-colors">
              {t("landing.finalCta.contactUs")}
            </Link>
          </p>

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
          </div>
        </div>
      </div>
    </section>
  );
}
