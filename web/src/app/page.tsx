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
