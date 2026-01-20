"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/auth";
import {
  Navbar,
  HeroSection,
  AgentLogos,
  WhyTerminalBased,
  CoreFeatures,
  HowItWorks,
  EnterpriseFeatures,
  PricingSection,
  SelfHostedCTA,
  FinalCTA,
  Footer,
} from "@/components/landing";

export default function Home() {
  const router = useRouter();
  const { token, currentOrg, _hasHydrated } = useAuthStore();
  const [shouldShowLanding, setShouldShowLanding] = useState(false);

  useEffect(() => {
    // Wait for hydration to complete before checking auth state
    if (!_hasHydrated) return;

    // Check if user navigated from within the site (internal navigation)
    // If referrer is from the same origin, user intentionally visited landing page
    const referrer = document.referrer;
    const isInternalNavigation = referrer && new URL(referrer).origin === window.location.origin;

    // Only redirect if:
    // 1. User is authenticated with an org
    // 2. User came from external source (not internal navigation)
    if (token && currentOrg && !isInternalNavigation) {
      router.replace(`/${currentOrg.slug}`);
      return;
    }

    // Show landing page
    setShouldShowLanding(true);
  }, [_hasHydrated, token, currentOrg, router]);

  // Show loading state while checking auth
  if (!shouldShowLanding) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background">
      <Navbar />
      <main>
        <HeroSection />
        <AgentLogos />
        <WhyTerminalBased />
        <CoreFeatures />
        <HowItWorks />
        <EnterpriseFeatures />
        <PricingSection />
        <SelfHostedCTA />
        <FinalCTA />
      </main>
      <Footer />
    </div>
  );
}
