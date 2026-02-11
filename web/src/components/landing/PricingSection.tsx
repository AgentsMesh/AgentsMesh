"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { CenteredSpinner } from "@/components/ui/spinner";
import { useTranslations } from "next-intl";
import { publicBillingApi, type PublicPricingResponse, type Currency } from "@/lib/api/billing";

type BillingCycle = "monthly" | "yearly";

// Plan type for pricing cards
interface PricingPlan {
  key: string;
  name: string;
  price: string;
  period: string;
  description: string;
  badge?: string;
  features: string[];
  cta: string;
  href: string;
  highlighted: boolean;
}

const CURRENCY_SYMBOLS: Record<Currency, string> = {
  USD: "$",
  CNY: "¥",
};

export function PricingSection() {
  const t = useTranslations();
  const [billingCycle, setBillingCycle] = useState<BillingCycle>("monthly");
  const [pricing, setPricing] = useState<PublicPricingResponse | null>(null);
  const [loading, setLoading] = useState(true);

  // Fetch pricing data from API (Single Source of Truth)
  useEffect(() => {
    publicBillingApi.getPricing()
      .then(setPricing)
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  const formatPrice = (priceMonthly: number, priceYearly: number) => {
    if (!pricing) return "";
    const price = billingCycle === "monthly" ? priceMonthly : priceYearly;
    return `${CURRENCY_SYMBOLS[pricing.currency]}${price}`;
  };

  const getPeriodLabel = (planName: string) => {
    if (planName === "based") {
      return billingCycle === "monthly" ? t("landing.pricing.month") : t("landing.pricing.year");
    }
    return billingCycle === "monthly"
      ? t("landing.pricing.seatMonth")
      : t("landing.pricing.seatYear");
  };

  // Build plans array from API data
  const buildPlans = (): PricingPlan[] => {
    if (!pricing) return [];

    const planConfigs: Record<string, {
      nameKey: string;
      descKey: string;
      featuresKey: string;
      ctaKey: string;
      href: string;
      highlighted: boolean;
      badge?: string;
      featureCount: number;
    }> = {
      based: {
        nameKey: "landing.pricing.based.name",
        descKey: "landing.pricing.based.description",
        featuresKey: "landing.pricing.based.features",
        ctaKey: "landing.pricing.based.cta",
        href: "/register",
        highlighted: false,
        badge: t("landing.pricing.based.badge"),
        featureCount: 5,
      },
      pro: {
        nameKey: "landing.pricing.pro.name",
        descKey: "landing.pricing.pro.description",
        featuresKey: "landing.pricing.pro.features",
        ctaKey: "landing.pricing.pro.cta",
        href: "/register?plan=pro",
        highlighted: true,
        featureCount: 6,
      },
      enterprise: {
        nameKey: "landing.pricing.enterprise.name",
        descKey: "landing.pricing.enterprise.description",
        featuresKey: "landing.pricing.enterprise.features",
        ctaKey: "landing.pricing.enterprise.cta",
        href: "/register?plan=enterprise",
        highlighted: false,
        featureCount: 7,
      },
    };

    return pricing.plans
      .filter(plan => planConfigs[plan.name])
      .map(plan => {
        const config = planConfigs[plan.name];
        return {
          key: plan.name,
          name: t(config.nameKey),
          price: formatPrice(plan.price_monthly, plan.price_yearly),
          period: getPeriodLabel(plan.name),
          description: t(config.descKey),
          badge: config.badge,
          features: Array.from({ length: config.featureCount }, (_, i) =>
            t(`${config.featuresKey}.${i}`)
          ),
          cta: t(config.ctaKey),
          href: config.href,
          highlighted: config.highlighted,
        };
      });
  };

  // OnPremise plan (static, not from API)
  const onPremisePlan: PricingPlan = {
    key: "onpremise",
    name: t("landing.pricing.onpremise.name"),
    price: t("landing.pricing.onpremise.price"),
    period: t("landing.pricing.onpremise.period"),
    description: t("landing.pricing.onpremise.description"),
    features: Array.from({ length: 7 }, (_, i) =>
      t(`landing.pricing.onpremise.features.${i}`)
    ),
    cta: t("landing.pricing.onpremise.cta"),
    href: "/contact",
    highlighted: false,
  };

  const plans = [...buildPlans(), onPremisePlan];

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

        {/* Billing cycle toggle */}
        <div className="flex items-center justify-center mb-12">
          <div className="inline-flex items-center rounded-lg bg-secondary/50 p-1">
            <button
              onClick={() => setBillingCycle("monthly")}
              className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                billingCycle === "monthly"
                  ? "bg-background text-foreground shadow-sm"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              {t("landing.pricing.monthly")}
            </button>
            <button
              onClick={() => setBillingCycle("yearly")}
              className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                billingCycle === "yearly"
                  ? "bg-background text-foreground shadow-sm"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              {t("landing.pricing.yearly")}
              <span className="ml-1 text-xs text-green-600 dark:text-green-400">
                {t("landing.pricing.yearlyDiscount")}
              </span>
            </button>
          </div>
        </div>

        {/* Loading state */}
        {loading && (
          <CenteredSpinner className="py-20" />
        )}

        {/* Pricing cards */}
        {!loading && (
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

                {plan.badge && (
                  <div className="absolute -top-4 left-1/2 -translate-x-1/2 px-4 py-1 bg-green-500 text-white text-sm font-medium rounded-full">
                    {plan.badge}
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
        )}

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
