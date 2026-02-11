"use client";

import { useTranslations } from "next-intl";

export function CoreFeatures() {
  const t = useTranslations();

  const features = [
    {
      number: "01",
      title: t("landing.coreFeatures.agentpod.title"),
      subtitle: t("landing.coreFeatures.agentpod.subtitle"),
      description: t("landing.coreFeatures.agentpod.description"),
      highlights: [
        t("landing.coreFeatures.agentpod.highlights.0"),
        t("landing.coreFeatures.agentpod.highlights.1"),
        t("landing.coreFeatures.agentpod.highlights.2"),
        t("landing.coreFeatures.agentpod.highlights.3"),
      ],
      terminal: {
        titleKey: "landing.coreDemo.terminalTitle",
        lines: [
          "$ claude-code --pod dev-42",
          "> Workspace: /projects/my-app",
          "> Branch: feature/auth-system",
          "> Status: Running",
          "",
          "$ git status",
          "On branch feature/auth-system",
          "Changes to be committed:",
          "  new file: src/auth/login.ts",
          "  new file: src/auth/oauth.ts",
        ],
      },
      align: "left",
    },
    {
      number: "02",
      title: t("landing.coreFeatures.agentsmesh.title"),
      subtitle: t("landing.coreFeatures.agentsmesh.subtitle"),
      description: t("landing.coreFeatures.agentsmesh.description"),
      highlights: [
        t("landing.coreFeatures.agentsmesh.highlights.0"),
        t("landing.coreFeatures.agentsmesh.highlights.1"),
        t("landing.coreFeatures.agentsmesh.highlights.2"),
        t("landing.coreFeatures.agentsmesh.highlights.3"),
      ],
      diagram: {
        nodes: [
          { id: "agent1", label: "Claude Code", x: 20, y: 30 },
          { id: "agent2", label: "Codex CLI", x: 70, y: 30 },
          { id: "channel", label: "Channel", x: 45, y: 70 },
        ],
        connections: [
          { from: "agent1", to: "channel" },
          { from: "agent2", to: "channel" },
          { from: "agent1", to: "agent2", dashed: true },
        ],
      },
      align: "right",
    },
    {
      number: "03",
      title: t("landing.coreFeatures.tickets.title"),
      subtitle: t("landing.coreFeatures.tickets.subtitle"),
      description: t("landing.coreFeatures.tickets.description"),
      highlights: [
        t("landing.coreFeatures.tickets.highlights.0"),
        t("landing.coreFeatures.tickets.highlights.1"),
        t("landing.coreFeatures.tickets.highlights.2"),
        t("landing.coreFeatures.tickets.highlights.3"),
      ],
      kanban: {
        columns: [
          { titleKey: "landing.coreDemo.kanban.backlog", cards: ["AUTH-1", "AUTH-3"] },
          { titleKey: "landing.coreDemo.kanban.inProgress", cards: ["AUTH-2"] },
          { titleKey: "landing.coreDemo.kanban.review", cards: ["AUTH-4"] },
          { titleKey: "landing.coreDemo.kanban.done", cards: [] },
        ],
      },
      align: "left",
    },
    {
      number: "04",
      title: t("landing.coreFeatures.runners.title"),
      subtitle: t("landing.coreFeatures.runners.subtitle"),
      description: t("landing.coreFeatures.runners.description"),
      highlights: [
        t("landing.coreFeatures.runners.highlights.0"),
        t("landing.coreFeatures.runners.highlights.1"),
        t("landing.coreFeatures.runners.highlights.2"),
        t("landing.coreFeatures.runners.highlights.3"),
      ],
      architecture: true,
      align: "right",
    },
  ];

  return (
    <section className="py-24" id="features">
      <div className="container mx-auto px-4 sm:px-6 lg:px-8">
        {/* Section header */}
        <div className="text-center mb-16">
          <h2 className="text-3xl sm:text-4xl font-bold mb-4">
            {t("landing.coreFeatures.title")} <span className="text-primary">{t("landing.coreFeatures.titleHighlight")}</span>
          </h2>
          <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
            {t("landing.coreFeatures.description")}
          </p>
        </div>

        {/* Features */}
        <div className="space-y-24">
          {features.map((feature, index) => (
            <div
              key={index}
              className={`grid lg:grid-cols-2 gap-12 items-center ${
                feature.align === "right" ? "lg:flex-row-reverse" : ""
              }`}
            >
              {/* Content */}
              <div className={feature.align === "right" ? "lg:order-2" : ""}>
                <div className="flex items-center gap-4 mb-4">
                  <span className="text-5xl font-bold text-primary/20">{feature.number}</span>
                  <div>
                    <h3 className="text-2xl font-bold">{feature.title}</h3>
                    <p className="text-primary">{feature.subtitle}</p>
                  </div>
                </div>

                <p className="text-muted-foreground mb-6">{feature.description}</p>

                <ul className="space-y-3">
                  {feature.highlights.map((highlight, i) => (
                    <li key={i} className="flex items-center gap-3">
                      <svg
                        className="w-5 h-5 text-primary flex-shrink-0"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M5 13l4 4L19 7"
                        />
                      </svg>
                      <span className="text-sm">{highlight}</span>
                    </li>
                  ))}
                </ul>
              </div>

              {/* Visual */}
              <div className={feature.align === "right" ? "lg:order-1" : ""}>
                {feature.terminal && (
                  <div className="bg-card rounded-xl border border-border overflow-hidden">
                    <div className="flex items-center gap-2 px-4 py-3 bg-muted border-b border-border">
                      <div className="flex gap-2">
                        <div className="w-3 h-3 rounded-full bg-red-500/80" />
                        <div className="w-3 h-3 rounded-full bg-yellow-500/80" />
                        <div className="w-3 h-3 rounded-full bg-green-500/80" />
                      </div>
                      <span className="text-xs text-muted-foreground font-mono ml-2">
                        {t(feature.terminal.titleKey)}
                      </span>
                    </div>
                    <div className="p-4 font-mono text-sm">
                      {feature.terminal.lines.map((line, i) => (
                        <div
                          key={i}
                          className={
                            line.startsWith("$")
                              ? "text-primary"
                              : line.startsWith(">")
                              ? "text-green-500 dark:text-green-400"
                              : "text-muted-foreground"
                          }
                        >
                          {line || "\u00A0"}
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {feature.diagram && (
                  <div className="bg-card rounded-xl border border-border p-8">
                    <div className="relative h-64">
                      {/* Agent nodes */}
                      <div className="absolute left-[10%] top-[20%] px-4 py-2 bg-primary/10 border border-primary/30 rounded-lg">
                        <div className="text-sm font-medium text-primary">Claude Code</div>
                        <div className="text-xs text-muted-foreground">{t("landing.coreDemo.podA")}</div>
                      </div>
                      <div className="absolute right-[10%] top-[20%] px-4 py-2 bg-primary/10 border border-primary/30 rounded-lg">
                        <div className="text-sm font-medium text-primary">Codex CLI</div>
                        <div className="text-xs text-muted-foreground">{t("landing.coreDemo.podB")}</div>
                      </div>
                      {/* Channel */}
                      <div className="absolute left-1/2 -translate-x-1/2 bottom-[20%] px-6 py-3 bg-secondary/50 border border-border rounded-lg">
                        <div className="text-sm font-medium">{t("landing.coreDemo.devChannel")}</div>
                        <div className="text-xs text-muted-foreground">{t("landing.coreDemo.members", { count: 3 })}</div>
                      </div>
                      {/* Connection lines */}
                      <svg className="absolute inset-0 w-full h-full" style={{ zIndex: -1 }}>
                        <line x1="25%" y1="40%" x2="50%" y2="70%" stroke="currentColor" strokeWidth="2" strokeDasharray="4" opacity="0.5" className="text-primary" />
                        <line x1="75%" y1="40%" x2="50%" y2="70%" stroke="currentColor" strokeWidth="2" strokeDasharray="4" opacity="0.5" className="text-primary" />
                        <line x1="30%" y1="30%" x2="70%" y2="30%" stroke="currentColor" strokeWidth="1" strokeDasharray="8" opacity="0.3" className="text-primary" />
                      </svg>
                    </div>
                  </div>
                )}

                {feature.kanban && (
                  <div className="bg-card rounded-xl border border-border p-4">
                    <div className="grid grid-cols-4 gap-3">
                      {feature.kanban.columns.map((col, i) => (
                        <div key={i} className="bg-secondary/20 rounded-lg p-3">
                          <div className="text-xs font-medium text-muted-foreground mb-3">
                            {t(col.titleKey)}
                          </div>
                          <div className="space-y-2">
                            {col.cards.map((card, j) => (
                              <div
                                key={j}
                                className="bg-muted border border-border rounded p-2 text-xs"
                              >
                                <div className="font-mono text-primary">{card}</div>
                                <div className="text-muted-foreground mt-1">{t("landing.coreDemo.kanban.authFeature")}</div>
                              </div>
                            ))}
                            {col.cards.length === 0 && (
                              <div className="text-xs text-muted-foreground/50 text-center py-4">
                                {t("landing.coreDemo.kanban.empty")}
                              </div>
                            )}
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {feature.architecture && (
                  <div className="bg-card rounded-xl border border-border p-8">
                    <div className="relative">
                      {/* Your Infrastructure box */}
                      <div className="border-2 border-dashed border-primary/30 rounded-xl p-6">
                        <div className="text-xs text-primary mb-4">{t("landing.coreDemo.architecture.yourInfrastructure")}</div>
                        <div className="flex items-center justify-center gap-8">
                          {/* Runner */}
                          <div className="text-center">
                            <div className="w-16 h-16 bg-primary/10 border border-primary/30 rounded-lg flex items-center justify-center mx-auto mb-2">
                              <svg className="w-8 h-8 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2" />
                              </svg>
                            </div>
                            <div className="text-sm font-medium">{t("landing.coreDemo.architecture.runner")}</div>
                          </div>
                          {/* Arrow */}
                          <svg className="w-8 h-8 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 8l4 4m0 0l-4 4m4-4H3" />
                          </svg>
                          {/* Agent */}
                          <div className="text-center">
                            <div className="w-16 h-16 bg-secondary/50 border border-border rounded-lg flex items-center justify-center mx-auto mb-2">
                              <svg className="w-8 h-8 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                              </svg>
                            </div>
                            <div className="text-sm font-medium">{t("landing.coreDemo.architecture.agent")}</div>
                          </div>
                        </div>
                      </div>
                      {/* Cloud connection */}
                      <div className="absolute -bottom-8 left-1/2 -translate-x-1/2 flex flex-col items-center">
                        <div className="w-px h-6 bg-border" />
                        <div className="text-xs text-muted-foreground">{t("landing.coreDemo.architecture.websocket")}</div>
                      </div>
                    </div>
                    <div className="mt-12 text-center">
                      <div className="inline-flex items-center gap-2 px-4 py-2 bg-secondary/30 rounded-lg border border-border">
                        <svg className="w-4 h-4 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 10-9.78 2.096A4.001 4.001 0 003 15z" />
                        </svg>
                        <span className="text-sm">{t("landing.coreDemo.architecture.agentsmeshCloud")}</span>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
