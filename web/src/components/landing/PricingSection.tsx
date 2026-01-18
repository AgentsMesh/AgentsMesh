"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { useTranslations } from "@/lib/i18n/client";

export function PricingSection() {
  const t = useTranslations();

  const plans = [
    {
      name: t("landing.pricing.free.name"),
      price: t("landing.pricing.free.price"),
      period: t("landing.pricing.free.period"),
      description: t("landing.pricing.free.description"),
      features: [
        t("landing.pricing.free.features.0"),
        t("landing.pricing.free.features.1"),
        t("landing.pricing.free.features.2"),
        t("landing.pricing.free.features.3"),
        t("landing.pricing.free.features.4"),
      ],
      cta: t("landing.pricing.free.cta"),
      href: "/register",
      highlighted: false,
    },
    {
      name: t("landing.pricing.pro.name"),
      price: t("landing.pricing.pro.price"),
      period: t("landing.pricing.pro.period"),
      description: t("landing.pricing.pro.description"),
      features: [
        t("landing.pricing.pro.features.0"),
        t("landing.pricing.pro.features.1"),
        t("landing.pricing.pro.features.2"),
        t("landing.pricing.pro.features.3"),
        t("landing.pricing.pro.features.4"),
        t("landing.pricing.pro.features.5"),
      ],
      cta: t("landing.pricing.pro.cta"),
      href: "/register?plan=pro",
      highlighted: true,
    },
    {
      name: t("landing.pricing.enterprise.name"),
      price: t("landing.pricing.enterprise.price"),
      period: t("landing.pricing.enterprise.period"),
      description: t("landing.pricing.enterprise.description"),
      features: [
        t("landing.pricing.enterprise.features.0"),
        t("landing.pricing.enterprise.features.1"),
        t("landing.pricing.enterprise.features.2"),
        t("landing.pricing.enterprise.features.3"),
        t("landing.pricing.enterprise.features.4"),
        t("landing.pricing.enterprise.features.5"),
        t("landing.pricing.enterprise.features.6"),
      ],
      cta: t("landing.pricing.enterprise.cta"),
      href: "/register?plan=enterprise",
      highlighted: false,
    },
    {
      name: t("landing.pricing.onpremise.name"),
      price: t("landing.pricing.onpremise.price"),
      period: t("landing.pricing.onpremise.period"),
      description: t("landing.pricing.onpremise.description"),
      features: [
        t("landing.pricing.onpremise.features.0"),
        t("landing.pricing.onpremise.features.1"),
        t("landing.pricing.onpremise.features.2"),
        t("landing.pricing.onpremise.features.3"),
        t("landing.pricing.onpremise.features.4"),
        t("landing.pricing.onpremise.features.5"),
        t("landing.pricing.onpremise.features.6"),
      ],
      cta: t("landing.pricing.onpremise.cta"),
      href: "/contact",
      highlighted: false,
    },
  ];

  return (
    <section className="py-24" id="pricing">
      <div className="container mx-auto px-4 sm:px-6 lg:px-8">
        {/* Section header */}
        <div className="text-center mb-16">
          <h2 className="text-3xl sm:text-4xl font-bold mb-4">
            {t("landing.pricing.title")} <span className="text-primary">{t("landing.pricing.titleHighlight")}</span>
          </h2>
          <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
            {t("landing.pricing.description")}
          </p>
        </div>

        {/* Pricing cards */}
        <div className="grid md:grid-cols-2 xl:grid-cols-4 gap-6 max-w-7xl mx-auto">
          {plans.map((plan, index) => (
            <div
              key={index}
              className={`relative rounded-2xl border ${
                plan.highlighted
                  ? "border-primary bg-primary/5"
                  : "border-border bg-secondary/10"
              } p-8 flex flex-col`}
            >
              {plan.highlighted && (
                <div className="absolute -top-4 left-1/2 -translate-x-1/2 px-4 py-1 bg-primary text-primary-foreground text-sm font-medium rounded-full">
                  {t("landing.pricing.mostPopular")}
                </div>
              )}

              <div className="mb-6">
                <h3 className="text-xl font-semibold mb-2">{plan.name}</h3>
                <div className="flex items-baseline gap-1">
                  <span className="text-4xl font-bold">{plan.price}</span>
                  <span className="text-muted-foreground">/{plan.period}</span>
                </div>
                <p className="text-sm text-muted-foreground mt-2">{plan.description}</p>
              </div>

              <ul className="space-y-3 mb-8 flex-grow">
                {plan.features.map((feature, i) => (
                  <li key={i} className="flex items-center gap-3 text-sm">
                    <svg
                      className={`w-5 h-5 flex-shrink-0 ${
                        plan.highlighted ? "text-primary" : "text-green-500 dark:text-green-400"
                      }`}
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

              <Link href={plan.href}>
                <Button
                  className={`w-full ${
                    plan.highlighted
                      ? "bg-primary text-primary-foreground hover:bg-primary/90"
                      : ""
                  }`}
                  variant={plan.highlighted ? "default" : "outline"}
                >
                  {plan.cta}
                </Button>
              </Link>
            </div>
          ))}
        </div>

        {/* FAQ or additional info */}
        <div className="mt-16 text-center">
          <p className="text-muted-foreground">
            {t("landing.pricing.footer")}
          </p>
          <p className="text-muted-foreground mt-2">
            {t("landing.pricing.customPlan")}{" "}
            <a href="mailto:bd@agentsmesh.ai" className="text-primary hover:underline">
              {t("landing.pricing.contactUs")}
            </a>
          </p>
        </div>
      </div>
    </section>
  );
}
