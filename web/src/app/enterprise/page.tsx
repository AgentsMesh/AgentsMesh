"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { PageHeader, PageFooter } from "@/components/common";
import { EnterpriseFeatures, SelfHostedCTA } from "@/components/landing";
import { useTranslations } from "next-intl";

export default function EnterprisePage() {
  const t = useTranslations();

  const onPremiseFeatures = Array.from({ length: 7 }, (_, i) =>
    t(`landing.pricing.onpremise.features.${i}`)
  );

  return (
    <div className="min-h-screen bg-background">
      <PageHeader />

      {/* Hero */}
      <section className="pt-20 pb-8 px-4">
        <div className="container mx-auto max-w-4xl text-center">
          <h1 className="text-4xl md:text-5xl font-bold mb-6">
            {t("enterprise.hero.title")}{" "}
            <span className="text-primary">{t("enterprise.hero.titleHighlight")}</span>
          </h1>
          <p className="text-xl text-muted-foreground max-w-2xl mx-auto">
            {t("enterprise.hero.subtitle")}
          </p>
        </div>
      </section>

      {/* Enterprise Features (reused from landing) */}
      <EnterpriseFeatures />

      {/* OnPremise Pricing */}
      <section className="py-24 bg-muted/30">
        <div className="container mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-16">
            <h2 className="text-3xl sm:text-4xl font-bold mb-4">
              {t("enterprise.onpremise.title")}{" "}
              <span className="text-primary">{t("enterprise.onpremise.titleHighlight")}</span>
            </h2>
            <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
              {t("enterprise.onpremise.description")}
            </p>
          </div>

          <div className="max-w-lg mx-auto">
            <div className="relative rounded-2xl border-2 border-primary/30 bg-primary/5 p-8 flex flex-col">
              <div className="mb-6">
                <h3 className="text-xl font-semibold mb-2">{t("landing.pricing.onpremise.name")}</h3>
                <div className="flex items-baseline gap-1">
                  <span className="text-4xl font-bold">{t("landing.pricing.onpremise.price")}</span>
                  <span className="text-muted-foreground">/ {t("landing.pricing.onpremise.period")}</span>
                </div>
                <p className="text-sm text-muted-foreground mt-2">
                  {t("landing.pricing.onpremise.description")}
                </p>
              </div>

              <ul className="space-y-3 mb-8 flex-grow">
                {onPremiseFeatures.map((feature, i) => (
                  <li key={i} className="flex items-center gap-3 text-sm">
                    <svg
                      className="w-5 h-5 flex-shrink-0 text-primary"
                      fill="none"
                      viewBox="0 0 24 24"
                      stroke="currentColor"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M5 13l4 4L19 7"
                      />
                    </svg>
                    {feature}
                  </li>
                ))}
              </ul>

              <Link href="/demo">
                <Button className="w-full bg-primary text-primary-foreground hover:bg-primary/90">
                  {t("landing.pricing.onpremise.cta")}
                </Button>
              </Link>
            </div>
          </div>
        </div>
      </section>

      {/* Self-Hosted CTA */}
      <SelfHostedCTA />

      <PageFooter />
    </div>
  );
}
