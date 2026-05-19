"use client";

import { useEffect, useMemo } from "react";

import { useRouter } from "next/navigation";
import {
  Navbar,
  HeroSection,
  AgentLogos,
  ParadigmShift,
  CoreFeatures,
  HowItWorks,
  PricingSection,
  Footer,
} from "@/components/landing";
import { getDefaultRoute } from "@/lib/default-route";
import { useLightSession } from "@/hooks/useLightSession";

export default function Home() {
  const router = useRouter();
  const { session, hydrated } = useLightSession();

  const shouldRedirect = useMemo(() => {
    if (!hydrated) return false;
    if (!session?.isAuthenticated || !session.currentOrgSlug) return false;

    // Check if user navigated from within the site (internal navigation)
    // If referrer is from the same origin, user intentionally visited landing page
    if (typeof window !== "undefined") {
      const referrer = document.referrer;
      const isInternalNavigation = referrer && new URL(referrer).origin === window.location.origin;
      return !isInternalNavigation;
    }
    return false;
  }, [hydrated, session]);

  useEffect(() => {
    if (shouldRedirect && session?.currentOrgSlug) {
      router.replace(getDefaultRoute(session.currentOrgSlug));
    }
  }, [shouldRedirect, session, router]);

  if (!hydrated || shouldRedirect) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  const jsonLd = {
    "@context": "https://schema.org",
    "@type": "SoftwareApplication",
    name: "AgentsMesh",
    applicationCategory: "DeveloperApplication",
    operatingSystem: "Web, Linux, macOS, Windows",
    description: "AgentsMesh is the agent workforce platform for ambitious teams. Give every team member an AI agent squad — assign tasks, track progress, and let them collaborate autonomously.",
    alternateName: ["AgentMesh", "Agents Mesh"],
    url: "https://agentsmesh.ai",
    offers: {
      "@type": "Offer",
      price: "0",
      priceCurrency: "USD",
      description: "Free tier available",
    },
    keywords: "agentsmesh, agent workforce platform, AI agent team, agent collaboration, multi-agent orchestration, team productivity, AI coding agents, agent management",
    publisher: {
      "@type": "Organization",
      name: "AgentsMesh",
      url: "https://agentsmesh.ai",
      logo: "https://agentsmesh.ai/icons/icon-512.png",
      sameAs: [
        "https://github.com/AgentsMesh/AgentsMesh",
        "https://x.com/agentsmeshai",
        "https://discord.gg/3RcX7VBbH9",
      ],
    },
  };

  return (
    <div className="azure-theme min-h-screen bg-background">
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }}
      />
      <Navbar />
      <main>
        <HeroSection />
        <AgentLogos />
        <ParadigmShift />
        <CoreFeatures />
        <HowItWorks />
        <PricingSection />
      </main>
      <Footer />
    </div>
  );
}
