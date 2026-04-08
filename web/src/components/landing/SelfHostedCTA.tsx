"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { useTranslations } from "next-intl";

export function SelfHostedCTA() {
  const t = useTranslations();

  return (
    <section className="py-24">
      <div className="container mx-auto px-4 sm:px-6 lg:px-8">
        <div className="relative max-w-4xl mx-auto">
          {/* Glow effect */}
          <div className="absolute inset-0 bg-primary/10 blur-3xl rounded-full opacity-30" />

          {/* Content box */}
          <div className="relative bg-card rounded-2xl border-2 border-primary/30 p-8 sm:p-12">
            <div className="flex flex-col lg:flex-row items-center gap-8">
              {/* Icon */}
              <div className="flex-shrink-0">
                <div className="w-20 h-20 rounded-2xl bg-primary/10 flex items-center justify-center">
                  <svg
                    className="w-10 h-10 text-primary"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={1.5}
                      d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"
                    />
                  </svg>
                </div>
              </div>

              {/* Text */}
              <div className="flex-grow text-center lg:text-left">
                <h3 className="text-2xl sm:text-3xl font-bold mb-4">
                  {t("landing.selfHosted.title")} <span className="text-primary">{t("landing.selfHosted.titleHighlight")}</span> {t("landing.selfHosted.titleEnd")}
                </h3>
                <p className="text-muted-foreground mb-6">
                  {t("landing.selfHosted.description")}
                </p>

                {/* Features */}
                <div className="flex flex-wrap justify-center lg:justify-start gap-4 mb-6">
                  <div className="flex items-center gap-2 text-sm">
                    <svg className="w-4 h-4 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    {t("landing.selfHosted.dockerCompose")}
                  </div>
                  <div className="flex items-center gap-2 text-sm">
                    <svg className="w-4 h-4 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    {t("landing.selfHosted.kubernetesHelm")}
                  </div>
                  <div className="flex items-center gap-2 text-sm">
                    <svg className="w-4 h-4 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    {t("landing.selfHosted.airGappedSupport")}
                  </div>
                </div>

                {/* CTAs */}
                <div className="flex flex-col sm:flex-row gap-4 justify-center lg:justify-start">
                  <Link href="/demo">
                    <Button className="w-full sm:w-auto bg-primary text-primary-foreground hover:bg-primary/90">
                      {t("landing.selfHosted.contactSales")}
                    </Button>
                  </Link>
                </div>
                <p className="mt-4 text-sm text-muted-foreground/70">
                  {t("landing.selfHosted.saasNote")}{" "}
                  <Link href="/register" className="text-primary underline hover:text-primary/80 transition-colors font-medium">
                    {t("landing.selfHosted.saasRegister")}
                  </Link>
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
