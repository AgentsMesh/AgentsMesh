"use client";

import { useTranslations } from "next-intl";
import { FeatureCard, type FeatureData } from "./FeatureCard";

export function CoreFeatures() {
  const t = useTranslations();

  const features: FeatureData[] = [
    buildAgentPodFeature(t),
    buildMeshFeature(t),
    buildTicketsFeature(t),
    buildScheduleFeature(t),
    buildRunnersFeature(t),
  ];

  return (
    <section className="py-24 relative" id="features">
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[800px] h-[800px] bg-primary/5 rounded-full blur-[150px]" />
      </div>

      <div className="container mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        <div className="text-center mb-16">
          <h2 className="text-3xl sm:text-4xl font-bold mb-4">
            {t("landing.coreFeatures.title")}{" "}
            <span className="bg-gradient-to-r from-primary to-primary/60 bg-clip-text text-transparent">
              {t("landing.coreFeatures.titleHighlight")}
            </span>
          </h2>
          <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
            {t("landing.coreFeatures.description")}
          </p>
        </div>

        <div className="space-y-24">
          {features.map((feature, index) => (
            <FeatureCard key={index} feature={feature} t={t} />
          ))}
        </div>
      </div>
    </section>
  );
}

type TFn = (key: string) => string;

function buildHighlights(t: TFn, prefix: string, count: number): string[] {
  return Array.from({ length: count }, (_, i) => t(`${prefix}.${i}`));
}

function buildAgentPodFeature(t: TFn): FeatureData {
  return {
    number: "01", title: t("landing.coreFeatures.agentpod.title"),
    subtitle: t("landing.coreFeatures.agentpod.subtitle"),
    description: t("landing.coreFeatures.agentpod.description"),
    highlights: buildHighlights(t, "landing.coreFeatures.agentpod.highlights", 5),
    podDemo: true, align: "left",
  };
}

function buildMeshFeature(t: TFn): FeatureData {
  return {
    number: "02", title: t("landing.coreFeatures.agentsmesh.title"),
    subtitle: t("landing.coreFeatures.agentsmesh.subtitle"),
    description: t("landing.coreFeatures.agentsmesh.description"),
    highlights: buildHighlights(t, "landing.coreFeatures.agentsmesh.highlights", 5),
    diagram: {
      nodes: [
        { id: "agent1", label: "Claude Code", x: 20, y: 30 },
        { id: "agent2", label: "Codex CLI", x: 70, y: 30 },
        { id: "channel", label: "Channel", x: 45, y: 70 },
      ],
      connections: [
        { from: "agent1", to: "channel" }, { from: "agent2", to: "channel" },
        { from: "agent1", to: "agent2", dashed: true },
      ],
    },
    align: "right",
  };
}

function buildTicketsFeature(t: TFn): FeatureData {
  return {
    number: "03", title: t("landing.coreFeatures.tickets.title"),
    subtitle: t("landing.coreFeatures.tickets.subtitle"),
    description: t("landing.coreFeatures.tickets.description"),
    highlights: buildHighlights(t, "landing.coreFeatures.tickets.highlights", 5),
    kanban: {
      columns: [
        { titleKey: "landing.coreDemo.kanban.backlog", cards: ["AUTH-1", "AUTH-3"] },
        { titleKey: "landing.coreDemo.kanban.inProgress", cards: ["AUTH-2"] },
        { titleKey: "landing.coreDemo.kanban.review", cards: ["AUTH-4"] },
        { titleKey: "landing.coreDemo.kanban.done", cards: [] },
      ],
    },
    align: "left",
  };
}

function buildScheduleFeature(t: TFn): FeatureData {
  return {
    number: "04", title: t("landing.coreFeatures.scheduledTasks.title"),
    subtitle: t("landing.coreFeatures.scheduledTasks.subtitle"),
    description: t("landing.coreFeatures.scheduledTasks.description"),
    highlights: buildHighlights(t, "landing.coreFeatures.scheduledTasks.highlights", 5),
    schedule: true, align: "left",
  };
}

function buildRunnersFeature(t: TFn): FeatureData {
  return {
    number: "05", title: t("landing.coreFeatures.runners.title"),
    subtitle: t("landing.coreFeatures.runners.subtitle"),
    description: t("landing.coreFeatures.runners.description"),
    highlights: buildHighlights(t, "landing.coreFeatures.runners.highlights", 4),
    architecture: true, align: "right",
  };
}
