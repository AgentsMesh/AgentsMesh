"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";

interface HeroContentProps {
  t: (key: string) => string;
}

/**
 * HeroContent - Renders the hero section text content (left side)
 */
export function HeroContent({ t }: HeroContentProps) {
  return (
    <div className="text-center lg:text-left">
      {/* Badge */}
      <div className="inline-flex items-center gap-2 px-4 py-1.5 rounded-full bg-primary/10 border border-primary/20 text-primary text-sm font-medium mb-8 transition-all hover:bg-primary/15 hover:border-primary/30 animate-border-glow backdrop-blur-sm">
        <span className="relative flex h-2 w-2">
          <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary opacity-75"></span>
          <span className="relative inline-flex rounded-full h-2 w-2 bg-primary"></span>
        </span>
        {t("landing.hero.badge")}
      </div>

      {/* Headline */}
      <h1 className="text-4xl sm:text-5xl lg:text-7xl font-extrabold tracking-tight leading-[1.1] mb-6">
        <span className="text-foreground">{t("landing.hero.slogan1")}</span>
        <br />
        <span className="bg-gradient-to-r from-primary via-primary/80 to-primary/50 bg-clip-text text-transparent animate-gradient-shift bg-[length:200%_200%]">{t("landing.hero.slogan2")}</span>
      </h1>

      {/* Description */}
      <p className="text-lg sm:text-xl text-muted-foreground/90 mb-10 max-w-xl mx-auto lg:mx-0 leading-relaxed">
        {t("landing.hero.description")}
      </p>

      {/* CTA Buttons */}
      <div className="flex flex-col sm:flex-row gap-4 justify-center lg:justify-start">
        <Link href="/register">
          <Button size="lg" className="relative w-full sm:w-auto bg-primary text-primary-foreground hover:bg-primary/90 text-base px-8 h-12 rounded-full shadow-lg shadow-primary/25 transition-all hover:shadow-primary/50 hover:shadow-xl hover:-translate-y-0.5 overflow-hidden group">
            <span className="relative z-10">{t("landing.hero.getStartedFree")}</span>
            <div className="absolute inset-0 bg-gradient-to-r from-primary via-primary/80 to-primary opacity-0 group-hover:opacity-100 transition-opacity" />
          </Button>
        </Link>
        <Link href="/demo">
          <Button size="lg" variant="outline" className="w-full sm:w-auto text-base px-8 h-12 rounded-full hover:bg-secondary/50 transition-all hover:-translate-y-0.5 border-primary/20 hover:border-primary/40">
            {t("landing.hero.viewDocs")}
          </Button>
        </Link>
      </div>
    </div>
  );
}

export default HeroContent;
