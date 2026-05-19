"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { CenteredSpinner } from "@/components/ui/spinner";
import { useTranslations } from "next-intl";
import type { PublicPricingResponse, Currency } from "@/lib/api/billing-types";
import { fetchPublicPricing } from "@/lib/public-api";

type BillingCycle = "monthly" | "yearly";

const FALLBACK_PRICING: PublicPricingResponse = {
  deployment_type: "global",
  currency: "USD" as Currency,
  plans: [
    { name: "based", display_name: "Based", price_monthly: 9.9, price_yearly: 99, max_users: 1, max_runners: 1, max_repositories: 5, max_concurrent_pods: 5 },
    { name: "pro", display_name: "Pro", price_monthly: 19.99, price_yearly: 199.9, max_users: 5, max_runners: 10, max_repositories: 10, max_concurrent_pods: 10 },
    { name: "enterprise", display_name: "Team", price_monthly: 99.99, price_yearly: 999.9, max_users: 50, max_runners: 100, max_repositories: -1, max_concurrent_pods: 50 },
  ],
};

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

  useEffect(() => {
    let settled = false;
    const finish = (data: PublicPricingResponse) => {
      if (settled) return;
      settled = true;
      setPricing(data);
      setLoading(false);
    };
    const timer = setTimeout(() => finish(FALLBACK_PRICING), 2500);
    fetchPublicPricing()
      .then((data) => {
        finish(data?.plans?.length ? data : FALLBACK_PRICING);
      })
      .catch(() => finish(FALLBACK_PRICING));
    return () => clearTimeout(timer);
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
        featureCount: 5,
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

  const plans = buildPlans();

  return (
    <section className="py-20 sm:py-32 relative" id="pricing">
      <div className="container mx-auto px-4 sm:px-6 lg:px-8">
        <div className="text-center mb-12 sm:mb-16">
          <h2 className="font-headline text-3xl sm:text-4xl md:text-5xl font-bold mb-5 sm:mb-6 leading-tight">
            {t("landing.pricing.title")}{" "}
            <span className="azure-gradient-text">{t("landing.pricing.titleHighlight")}</span>
          </h2>
          <p className="text-[var(--azure-text-muted)] max-w-2xl mx-auto text-base sm:text-lg font-light">
            {t("landing.pricing.description")}
          </p>
        </div>

        <div className="flex items-center justify-center mb-12 sm:mb-16">
          <div className="inline-flex items-center rounded-full azure-glass p-1 border border-white/10">
            <button
              onClick={() => setBillingCycle("monthly")}
              className={`px-4 sm:px-5 py-2 rounded-full text-xs sm:text-sm font-headline font-bold uppercase tracking-wider transition-all ${
                billingCycle === "monthly"
                  ? "azure-gradient-bg"
                  : "text-[var(--azure-text-muted)] hover:text-foreground"
              }`}
            >
              {t("landing.pricing.monthly")}
            </button>
            <button
              onClick={() => setBillingCycle("yearly")}
              className={`px-4 sm:px-5 py-2 rounded-full text-xs sm:text-sm font-headline font-bold uppercase tracking-wider transition-all ${
                billingCycle === "yearly"
                  ? "azure-gradient-bg"
                  : "text-[var(--azure-text-muted)] hover:text-foreground"
              }`}
            >
              {t("landing.pricing.yearly")}
              <span className="ml-1.5 sm:ml-2 text-[9px] sm:text-[10px] opacity-80">{t("landing.pricing.yearlyDiscount")}</span>
            </button>
          </div>
        </div>

        {loading && <CenteredSpinner className="py-20" />}

        {!loading && (
          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6 max-w-6xl mx-auto items-stretch">
            {plans.map((plan) => (
              <div
                key={plan.key}
                className={`relative rounded-3xl p-6 sm:p-8 flex flex-col transition-all ${
                  plan.highlighted
                    ? "azure-glass border-2 border-[var(--azure-cyan)] azure-glow-cyan-lg lg:scale-105 z-10"
                    : "bg-[var(--azure-bg-low)] border border-[var(--azure-outline-variant)]/30"
                }`}
              >
                {plan.highlighted && (
                  <div className="absolute -top-3 left-1/2 -translate-x-1/2 azure-gradient-bg px-4 py-1 rounded-full font-headline text-[10px] font-black uppercase tracking-[0.2em] whitespace-nowrap">
                    {t("landing.pricing.mostPopular")}
                  </div>
                )}
                {plan.badge && !plan.highlighted && (
                  <div className="absolute -top-3 left-1/2 -translate-x-1/2 bg-[var(--azure-mint)] text-[var(--azure-on-cyan)] px-4 py-1 rounded-full font-headline text-[10px] font-black uppercase tracking-[0.2em] whitespace-nowrap">
                    {plan.badge}
                  </div>
                )}

                <span className={`font-headline text-xs font-bold tracking-[0.3em] uppercase mb-6 ${plan.highlighted ? "text-[var(--azure-cyan)]" : "text-[var(--azure-text-muted)]"}`}>
                  {plan.name}
                </span>
                <div className="mb-3">
                  <span className="font-headline text-5xl font-bold">{plan.price}</span>
                  <span className="text-[var(--azure-text-muted)] text-sm ml-1">/{plan.period}</span>
                </div>
                <p className="text-sm text-[var(--azure-text-muted)] mb-8">{plan.description}</p>

                <ul className="space-y-3.5 mb-10 flex-grow">
                  {plan.features.map((feature) => (
                    <li key={feature} className="flex items-start gap-3 text-sm">
                      <span
                        className="mt-1.5 w-1.5 h-1.5 rounded-full flex-shrink-0"
                        style={{ backgroundColor: plan.highlighted ? "var(--azure-mint)" : "var(--azure-cyan-soft)" }}
                      />
                      <span className={plan.highlighted ? "text-foreground" : "text-[var(--azure-text-muted)]"}>{feature}</span>
                    </li>
                  ))}
                </ul>

                <Link href={plan.href}>
                  <Button
                    className={`w-full rounded-full font-headline font-black uppercase tracking-[0.18em] text-xs h-12 ${
                      plan.highlighted
                        ? "azure-gradient-bg azure-cta-glow border-0"
                        : "bg-transparent border border-[var(--azure-outline-variant)] hover:border-[var(--azure-cyan)]/60 hover:bg-[var(--azure-bg-bright)] text-foreground"
                    }`}
                  >
                    {plan.cta}
                  </Button>
                </Link>
              </div>
            ))}
          </div>
        )}

        <div className="mt-16 text-center">
          <p className="text-[var(--azure-text-muted)] text-sm">{t("landing.pricing.footer")}</p>
        </div>
      </div>
    </section>
  );
}
