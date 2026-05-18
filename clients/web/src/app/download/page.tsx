import { Navbar, Footer } from "@/components/landing";
import {
  DownloadHero,
  FallbackHero,
  PlatformGrid,
  RunnerSection,
  ResourcesSection,
} from "@/components/download";
import { fetchLatestRelease } from "@/lib/download/github-release";

export const revalidate = 3600;

export default async function DownloadPage() {
  const release = await fetchLatestRelease();

  const jsonLd = {
    "@context": "https://schema.org",
    "@type": "SoftwareApplication",
    name: "AgentsMesh Desktop",
    applicationCategory: "DeveloperApplication",
    operatingSystem: "macOS, Windows, Linux",
    softwareVersion: release?.version,
    downloadUrl: release?.htmlUrl,
    offers: { "@type": "Offer", price: "0", priceCurrency: "USD" },
  };

  return (
    <div className="azure-theme min-h-screen bg-background">
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }}
      />
      <Navbar />
      <main>
        {release ? (
          <>
            <DownloadHero release={release} />
            <PlatformGrid desktop={release.desktop} />
            <RunnerSection runner={release.runner} />
            <ResourcesSection checksumsUrl={release.checksumsUrl} />
          </>
        ) : (
          <FallbackHero />
        )}
      </main>
      <Footer />
    </div>
  );
}
