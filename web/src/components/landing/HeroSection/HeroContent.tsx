"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";

interface HeroContentProps {
  t: (key: string) => string;
}

/**
 * HeroContent - Renders the hero section text content (centered)
 */
export function HeroContent({ t }: HeroContentProps) {
  return (
    <div className="text-center">
      {/* Badge */}
      <div className="inline-flex items-center gap-2 px-5 py-2 rounded-full bg-primary/10 border border-primary/20 text-primary text-sm font-medium mb-12 transition-all hover:bg-primary/15 hover:border-primary/30 animate-border-glow backdrop-blur-sm">
        <span className="relative flex h-2 w-2">
          <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary opacity-75"></span>
          <span className="relative inline-flex rounded-full h-2 w-2 bg-primary"></span>
        </span>
        {t("landing.hero.badge")}
      </div>

      {/* Headline */}
      <h1 className="text-3xl sm:text-4xl lg:text-5xl font-extrabold tracking-tight leading-[1.15] mb-8">
        <span className="text-foreground">{t("landing.hero.slogan1")}</span>
        <br className="hidden sm:block" />
        <span className="sm:mt-2 inline-block bg-gradient-to-r from-primary via-primary/80 to-primary/50 bg-clip-text text-transparent">{t("landing.hero.slogan2")}</span>
      </h1>

      {/* Description */}
      <p className="text-lg sm:text-xl lg:text-2xl text-muted-foreground/80 mb-14 max-w-2xl mx-auto leading-relaxed">
        {t("landing.hero.description")}
      </p>

      {/* CTA Buttons */}
      <div className="flex flex-col sm:flex-row gap-5 justify-center">
        <Link href="/register">
          <Button size="lg" className="relative w-full sm:w-auto bg-primary text-primary-foreground hover:bg-primary/90 text-lg px-10 h-14 rounded-full shadow-lg shadow-primary/25 transition-all hover:shadow-primary/50 hover:shadow-xl hover:-translate-y-0.5 overflow-hidden group">
            <span className="relative z-10">{t("landing.hero.getStartedFree")}</span>
            <div className="absolute inset-0 bg-gradient-to-r from-primary via-primary/80 to-primary opacity-0 group-hover:opacity-100 transition-opacity" />
          </Button>
        </Link>
        <Link href="/login">
          <Button size="lg" variant="outline" className="w-full sm:w-auto text-lg px-10 h-14 rounded-full hover:bg-secondary/50 transition-all hover:-translate-y-0.5 border-primary/20 hover:border-primary/40">
            {t("landing.hero.signInConsole")}
          </Button>
        </Link>
      </div>
      <p className="mt-4 text-sm text-muted-foreground/60">
        {t("landing.hero.enterpriseNote")}{" "}
        <Link href="/demo" className="underline hover:text-primary transition-colors">
          {t("landing.hero.contactUs")}
        </Link>
      </p>
    </div>
  );
}

export default HeroContent;
