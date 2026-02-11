"use client";

import { useTranslations } from "next-intl";
import { useTuiAnimation } from "./useTuiAnimation";
import { HeroContent } from "./HeroContent";
import { TuiWindow } from "./TuiWindow";
import { TopologyVisualization } from "./TopologyVisualization";

/**
 * HeroSection - Landing page hero section with animated TUI demonstration
 *
 * Combines hero text content with an interactive demonstration showing
 * multi-agent collaboration through the Claude Code TUI interface.
 */
export function HeroSection() {
  const t = useTranslations();
  const { currentFrame, currentFrameIndex, displayedLines, isTyping } = useTuiAnimation(t);

  return (
    <section className="relative min-h-screen flex items-center pt-16">
      {/* Background gradient */}
      <div className="absolute inset-0 bg-gradient-to-b from-primary/5 via-transparent to-transparent" />

      {/* Grid pattern */}
      <div
        className="absolute inset-0 opacity-[0.03] dark:opacity-[0.02]"
        style={{
          backgroundImage: `linear-gradient(var(--foreground) 1px, transparent 1px),
                           linear-gradient(90deg, var(--foreground) 1px, transparent 1px)`,
          backgroundSize: '50px 50px'
        }}
      />

      <div className="container mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        <div className="grid lg:grid-cols-2 gap-8 lg:gap-12 items-center">
          {/* Left: Text Content */}
          <HeroContent t={t} />

          {/* Right: TUI + Topology */}
          <div className="relative space-y-4">
            {/* Glow effect */}
            <div className="absolute -inset-4 bg-primary/20 blur-3xl rounded-full opacity-20" />

            {/* Claude Code TUI Window */}
            <TuiWindow
              frame={currentFrame}
              frameIndex={currentFrameIndex}
              displayedLines={displayedLines}
              isTyping={isTyping}
              t={t}
            />

            {/* Topology visualization */}
            <TopologyVisualization
              topology={currentFrame.content.topology}
              t={t}
            />
          </div>
        </div>
      </div>

      {/* Scroll indicator */}
      <div className="absolute bottom-8 left-1/2 -translate-x-1/2 animate-bounce">
        <svg
          className="w-6 h-6 text-muted-foreground"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M19 14l-7 7m0 0l-7-7m7 7V3"
          />
        </svg>
      </div>
    </section>
  );
}

export default HeroSection;
