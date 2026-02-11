"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/common";
import { useTranslations } from "next-intl";

interface JobPosition {
  id: string;
  titleKey: string;
  descriptionKey: string;
  responsibilities: string[];
  icon: React.ReactNode;
}

const positions: JobPosition[] = [
  {
    id: "builder",
    titleKey: "careers.positions.builder.title",
    descriptionKey: "careers.positions.builder.description",
    responsibilities: [
      "careers.positions.builder.resp1",
      "careers.positions.builder.resp2",
      "careers.positions.builder.resp3",
    ],
    icon: (
      <svg
        className="w-8 h-8"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={2}
          d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"
        />
      </svg>
    ),
  },
  {
    id: "experiencer",
    titleKey: "careers.positions.experiencer.title",
    descriptionKey: "careers.positions.experiencer.description",
    responsibilities: [
      "careers.positions.experiencer.resp1",
      "careers.positions.experiencer.resp2",
      "careers.positions.experiencer.resp3",
    ],
    icon: (
      <svg
        className="w-8 h-8"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={2}
          d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z"
        />
      </svg>
    ),
  },
  {
    id: "growth",
    titleKey: "careers.positions.growth.title",
    descriptionKey: "careers.positions.growth.description",
    responsibilities: [
      "careers.positions.growth.resp1",
      "careers.positions.growth.resp2",
      "careers.positions.growth.resp3",
    ],
    icon: (
      <svg
        className="w-8 h-8"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={2}
          d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6"
        />
      </svg>
    ),
  },
];

export default function CareersPage() {
  const t = useTranslations();

  return (
    <div className="min-h-screen bg-background">
      <PageHeader />

      {/* Hero Section */}
      <section className="py-20 px-4 text-center">
        <div className="container mx-auto max-w-4xl">
          <h1 className="text-4xl md:text-5xl font-bold mb-6">
            {t("careers.hero.title")}
          </h1>
          <p className="text-xl text-muted-foreground max-w-2xl mx-auto mb-8">
            {t("careers.hero.subtitle")}
          </p>
        </div>
      </section>

      {/* Positions */}
      <section className="py-16 px-4">
        <div className="container mx-auto max-w-5xl">
          <h2 className="text-3xl font-bold mb-12 text-center">
            {t("careers.openPositions")}
          </h2>

          <div className="space-y-8">
            {positions.map((position) => (
              <div
                key={position.id}
                className="p-8 rounded-xl border border-border bg-card hover:border-primary/50 transition-colors"
              >
                <div className="flex items-start gap-6">
                  <div className="w-16 h-16 rounded-xl bg-primary/10 flex items-center justify-center text-primary flex-shrink-0">
                    {position.icon}
                  </div>
                  <div className="flex-1">
                    <h3 className="text-2xl font-bold mb-2">
                      {t(position.titleKey)}
                    </h3>
                    <p className="text-muted-foreground mb-4">
                      {t(position.descriptionKey)}
                    </p>
                    <div className="mb-6">
                      <h4 className="font-semibold mb-2">
                        {t("careers.responsibilities")}
                      </h4>
                      <ul className="space-y-1">
                        {position.responsibilities.map((resp, idx) => (
                          <li
                            key={idx}
                            className="text-muted-foreground flex items-start gap-2"
                          >
                            <span className="text-primary">•</span>
                            {t(resp)}
                          </li>
                        ))}
                      </ul>
                    </div>
                    <Link href="mailto:recruiter@agentsmesh.ai">
                      <Button>{t("careers.applyNow")}</Button>
                    </Link>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Why Join Us */}
      <section className="py-16 px-4 bg-muted/30">
        <div className="container mx-auto max-w-4xl text-center">
          <h2 className="text-3xl font-bold mb-8">{t("careers.whyJoin.title")}</h2>
          <div className="grid md:grid-cols-3 gap-8">
            <div>
              <div className="w-12 h-12 rounded-full bg-primary/10 flex items-center justify-center mx-auto mb-4">
                <svg
                  className="w-6 h-6 text-primary"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"
                  />
                </svg>
              </div>
              <h3 className="text-lg font-semibold mb-2">
                {t("careers.whyJoin.innovation.title")}
              </h3>
              <p className="text-muted-foreground">
                {t("careers.whyJoin.innovation.content")}
              </p>
            </div>
            <div>
              <div className="w-12 h-12 rounded-full bg-primary/10 flex items-center justify-center mx-auto mb-4">
                <svg
                  className="w-6 h-6 text-primary"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M3.055 11H5a2 2 0 012 2v1a2 2 0 002 2 2 2 0 012 2v2.945M8 3.935V5.5A2.5 2.5 0 0010.5 8h.5a2 2 0 012 2 2 2 0 104 0 2 2 0 012-2h1.064M15 20.488V18a2 2 0 012-2h3.064M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                  />
                </svg>
              </div>
              <h3 className="text-lg font-semibold mb-2">
                {t("careers.whyJoin.remote.title")}
              </h3>
              <p className="text-muted-foreground">
                {t("careers.whyJoin.remote.content")}
              </p>
            </div>
            <div>
              <div className="w-12 h-12 rounded-full bg-primary/10 flex items-center justify-center mx-auto mb-4">
                <svg
                  className="w-6 h-6 text-primary"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                  />
                </svg>
              </div>
              <h3 className="text-lg font-semibold mb-2">
                {t("careers.whyJoin.equity.title")}
              </h3>
              <p className="text-muted-foreground">
                {t("careers.whyJoin.equity.content")}
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="py-16 px-4">
        <div className="container mx-auto max-w-4xl text-center">
          <h2 className="text-2xl font-bold mb-4">{t("careers.cta.title")}</h2>
          <p className="text-muted-foreground mb-6">{t("careers.cta.content")}</p>
          <Link href="mailto:recruiter@agentsmesh.ai">
            <Button size="lg">{t("careers.cta.button")}</Button>
          </Link>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-border mt-16">
        <div className="container mx-auto px-4 py-8">
          <div className="flex flex-col md:flex-row justify-between items-center gap-4">
            <p className="text-sm text-muted-foreground">
              &copy; {new Date().getFullYear()} AgentsMesh. {t("common.allRightsReserved")}
            </p>
            <div className="flex gap-6">
              <Link
                href="/privacy"
                className="text-sm text-muted-foreground hover:text-foreground"
              >
                {t("landing.footer.legal.privacy")}
              </Link>
              <Link
                href="/terms"
                className="text-sm text-muted-foreground hover:text-foreground"
              >
                {t("landing.footer.legal.terms")}
              </Link>
              <Link
                href="/docs"
                className="text-sm text-muted-foreground hover:text-foreground"
              >
                {t("landing.footer.resources.documentation")}
              </Link>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}
