"use client";

import type { useTranslations } from "next-intl";
import { FeatureVisuals } from "./FeatureVisuals";

export interface FeatureData {
  number: string;
  title: string;
  subtitle: string;
  description: string;
  highlights: string[];
  align: string;
  podDemo?: boolean;
  diagram?: { nodes: Array<{ id: string; label: string; x: number; y: number }>; connections: Array<{ from: string; to: string; dashed?: boolean }> };
  kanban?: { columns: Array<{ titleKey: string; cards: string[] }> };
  schedule?: boolean;
  architecture?: boolean;
}

interface FeatureCardProps {
  feature: FeatureData;
  t: ReturnType<typeof useTranslations>;
}

export function FeatureCard({ feature, t }: FeatureCardProps) {
  return (
    <div className={`grid lg:grid-cols-2 gap-12 items-center ${feature.align === "right" ? "lg:flex-row-reverse" : ""}`}>
      {/* Content */}
      <div className={feature.align === "right" ? "lg:order-2" : ""}>
        <div className="flex items-center gap-6 mb-6">
          <span className="text-6xl font-black bg-gradient-to-br from-primary/30 via-primary/15 to-transparent bg-clip-text text-transparent select-none drop-shadow-[0_0_15px_var(--primary)]">
            {feature.number}
          </span>
          <div>
            <p className="text-primary font-medium tracking-wide uppercase text-sm mb-1">{feature.subtitle}</p>
            <h3 className="text-3xl font-bold tracking-tight">{feature.title}</h3>
          </div>
        </div>
        <p className="text-muted-foreground text-lg leading-relaxed mb-8">{feature.description}</p>
        <ul className="space-y-4">
          {feature.highlights.map((highlight, i) => (
            <li key={i} className="flex items-start gap-3 group">
              <div className="mt-1 w-5 h-5 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0 group-hover:bg-primary/20 transition-colors">
                <svg className="w-3 h-3 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
                </svg>
              </div>
              <span className="text-base text-foreground/80">{highlight}</span>
            </li>
          ))}
        </ul>
      </div>

      {/* Visual */}
      <div className={`relative group ${feature.align === "right" ? "lg:order-1" : ""}`}>
        <div className="absolute -inset-6 bg-primary/20 blur-[50px] rounded-full opacity-0 group-hover:opacity-30 transition-opacity duration-700" />
        <div className="absolute -inset-1 bg-gradient-to-r from-primary/20 via-transparent to-primary/20 rounded-xl opacity-0 group-hover:opacity-100 transition-opacity duration-500 blur-sm" />
        <div className="relative transform transition-all duration-500 hover:scale-[1.02] hover:-rotate-1 scanline-overlay overflow-hidden rounded-xl">
          <FeatureVisuals feature={feature} t={t} />
        </div>
      </div>
    </div>
  );
}
