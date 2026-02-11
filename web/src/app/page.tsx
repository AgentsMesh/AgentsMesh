"use client";

import { useEffect, useMemo } from "react";

import { useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/auth";
import {
  Navbar,
  HeroSection,
  DemoVideo,
  AgentLogos,
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

  // Determine if we should redirect based on auth state
  const shouldRedirect = useMemo(() => {
    if (!_hasHydrated) return false;

    // Check if user navigated from within the site (internal navigation)
    // If referrer is from the same origin, user intentionally visited landing page
    if (typeof window !== "undefined") {
      const referrer = document.referrer;
      const isInternalNavigation = referrer && new URL(referrer).origin === window.location.origin;
      // Only redirect if user is authenticated with an org and came from external source
      return !!(token && currentOrg && !isInternalNavigation);
    }
    return false;
  }, [_hasHydrated, token, currentOrg]);

  // Handle redirect in effect
  useEffect(() => {
    if (shouldRedirect && currentOrg) {
      router.replace(`/${currentOrg.slug}`);
    }
  }, [shouldRedirect, currentOrg, router]);

  // Show loading state while hydrating or redirecting
  if (!_hasHydrated || shouldRedirect) {
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
        <DemoVideo />
        <AgentLogos />
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
