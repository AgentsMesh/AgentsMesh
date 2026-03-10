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
        t("landing.coreFeatures.agentpod.highlights.4"),
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
        t("landing.coreFeatures.agentsmesh.highlights.4"),
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
        t("landing.coreFeatures.tickets.highlights.4"),
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
      title: t("landing.coreFeatures.scheduledTasks.title"),
      subtitle: t("landing.coreFeatures.scheduledTasks.subtitle"),
      description: t("landing.coreFeatures.scheduledTasks.description"),
      highlights: [
        t("landing.coreFeatures.scheduledTasks.highlights.0"),
        t("landing.coreFeatures.scheduledTasks.highlights.1"),
        t("landing.coreFeatures.scheduledTasks.highlights.2"),
        t("landing.coreFeatures.scheduledTasks.highlights.3"),
        t("landing.coreFeatures.scheduledTasks.highlights.4"),
      ],
      schedule: true,
      align: "left",
    },
    {
      number: "05",
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
    <section className="py-24 relative" id="features">
      {/* Section background glow */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[800px] h-[800px] bg-primary/5 rounded-full blur-[150px]" />
      </div>

      <div className="container mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        {/* Section header */}
        <div className="text-center mb-16">
          <h2 className="text-3xl sm:text-4xl font-bold mb-4">
            {t("landing.coreFeatures.title")} <span className="bg-gradient-to-r from-primary to-primary/60 bg-clip-text text-transparent">{t("landing.coreFeatures.titleHighlight")}</span>
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
                        <svg
                          className="w-3 h-3 text-primary"
                          fill="none"
                          viewBox="0 0 24 24"
                          stroke="currentColor"
                        >
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth={3}
                            d="M5 13l4 4L19 7"
                          />
                        </svg>
                      </div>
                      <span className="text-base text-foreground/80">{highlight}</span>
                    </li>
                  ))}
                </ul>
              </div>

              {/* Visual */}
              <div className={`relative group ${feature.align === "right" ? "lg:order-1" : ""}`}>
                {/* Glow effect - enhanced */}
                <div className="absolute -inset-6 bg-primary/20 blur-[50px] rounded-full opacity-0 group-hover:opacity-30 transition-opacity duration-700" />
                <div className="absolute -inset-1 bg-gradient-to-r from-primary/20 via-transparent to-primary/20 rounded-xl opacity-0 group-hover:opacity-100 transition-opacity duration-500 blur-sm" />

                <div className="relative transform transition-all duration-500 hover:scale-[1.02] hover:-rotate-1 scanline-overlay overflow-hidden rounded-xl">
                  {feature.terminal && (
                    <div className="bg-card/95 backdrop-blur rounded-xl border border-primary/20 shadow-2xl shadow-primary/10 overflow-hidden animate-border-glow">
                      <div className="flex items-center justify-between px-4 py-3 bg-muted/50 border-b border-border">
                        <div className="flex gap-2">
                          <div className="w-3 h-3 rounded-full bg-red-500/80 border border-red-600/20" />
                          <div className="w-3 h-3 rounded-full bg-yellow-500/80 border border-yellow-600/20" />
                          <div className="w-3 h-3 rounded-full bg-green-500/80 border border-green-600/20" />
                        </div>
                        <span className="text-xs text-muted-foreground font-mono font-medium opacity-70">
                          {t(feature.terminal.titleKey)}
                        </span>
                        <div className="w-12" /> {/* Spacer for centering */}
                      </div>
                      <div className="p-6 font-mono text-sm leading-relaxed overflow-x-auto">
                        {feature.terminal.lines.map((line, i) => (
                          <div
                            key={i}
                            className={`${
                              line.startsWith("$")
                                ? "text-primary font-bold"
                                : line.startsWith(">")
                                ? "text-green-500 dark:text-green-400"
                                : "text-muted-foreground/90"
                            } whitespace-pre`}
                          >
                            {line || "\u00A0"}
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  {feature.diagram && (
                    <div className="bg-card/95 backdrop-blur rounded-xl border border-primary/20 shadow-2xl shadow-primary/10 p-8 relative overflow-hidden animate-border-glow">
                      {/* Grid background */}
                      <div className="absolute inset-0 opacity-[0.03]" 
                           style={{ backgroundImage: 'radial-gradient(circle, currentColor 1px, transparent 1px)', backgroundSize: '20px 20px' }} 
                      />
                      
                      <div className="relative h-64">
                        {/* Agent nodes */}
                        <div className="absolute left-[10%] top-[20%] px-5 py-3 bg-card border border-primary/30 rounded-xl shadow-lg shadow-primary/5 z-10">
                          <div className="flex items-center gap-2 mb-1">
                            <div className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
                            <div className="text-sm font-bold text-foreground">Claude Code</div>
                          </div>
                          <div className="text-xs text-muted-foreground font-mono">{t("landing.coreDemo.podA")}</div>
                        </div>
                        
                        <div className="absolute right-[10%] top-[20%] px-5 py-3 bg-card border border-primary/30 rounded-xl shadow-lg shadow-primary/5 z-10">
                          <div className="flex items-center gap-2 mb-1">
                            <div className="w-2 h-2 rounded-full bg-green-500 animate-pulse delay-75" />
                            <div className="text-sm font-bold text-foreground">Codex CLI</div>
                          </div>
                          <div className="text-xs text-muted-foreground font-mono">{t("landing.coreDemo.podB")}</div>
                        </div>
                        
                        {/* Channel */}
                        <div className="absolute left-1/2 -translate-x-1/2 bottom-[20%] px-6 py-3 bg-secondary/80 backdrop-blur border border-border rounded-full shadow-lg z-10 flex items-center gap-3">
                          <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center text-primary">#</div>
                          <div>
                            <div className="text-sm font-bold">{t("landing.coreDemo.devChannel")}</div>
                            <div className="text-xs text-muted-foreground">{t("landing.coreDemo.members", { count: 3 })}</div>
                          </div>
                        </div>
                        
                        {/* Connection lines */}
                        <svg className="absolute inset-0 w-full h-full pointer-events-none" style={{ zIndex: 0 }}>
                          <path d="M120 80 Q 180 200 250 220" fill="none" stroke="currentColor" strokeWidth="2" strokeDasharray="4" className="text-primary/30" />
                          <path d="M380 80 Q 320 200 250 220" fill="none" stroke="currentColor" strokeWidth="2" strokeDasharray="4" className="text-primary/30" />
                          <path d="M140 60 Q 250 20 360 60" fill="none" stroke="currentColor" strokeWidth="1.5" strokeDasharray="6 4" className="text-primary/20" />
                          
                          {/* Animated dots */}
                          <circle r="3" fill="currentColor" className="text-primary animate-ping" style={{ animationDuration: '3s' }}>
                            <animateMotion dur="2s" repeatCount="indefinite" path="M120 80 Q 180 200 250 220" />
                          </circle>
                          <circle r="3" fill="currentColor" className="text-primary animate-ping" style={{ animationDuration: '3s', animationDelay: '1s' }}>
                            <animateMotion dur="2s" repeatCount="indefinite" path="M380 80 Q 320 200 250 220" />
                          </circle>
                        </svg>
                      </div>
                    </div>
                  )}

                  {feature.kanban && (
                    <div className="bg-card/95 backdrop-blur rounded-xl border border-primary/20 shadow-2xl shadow-primary/10 p-6 relative overflow-hidden animate-border-glow">
                      <div className="grid grid-cols-4 gap-4">
                        {feature.kanban.columns.map((col, i) => (
                          <div key={i} className="bg-secondary/30 rounded-xl p-3 flex flex-col h-48">
                            <div className="flex items-center justify-between mb-3 px-1">
                              <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
                                {t(col.titleKey)}
                              </span>
                              <span className="text-[10px] bg-background/50 px-1.5 py-0.5 rounded text-muted-foreground">
                                {col.cards.length}
                              </span>
                            </div>
                            <div className="space-y-2 flex-1 overflow-y-auto custom-scrollbar">
                              {col.cards.map((card, j) => (
                                <div
                                  key={j}
                                  className="bg-card border border-border/50 rounded-lg p-2.5 shadow-sm hover:shadow-md hover:border-primary/30 transition-all cursor-default group/card"
                                >
                                  <div className="flex items-center justify-between mb-1.5">
                                    <span className="font-mono text-[10px] text-primary bg-primary/5 px-1 rounded">
                                      {card}
                                    </span>
                                    <div className="w-1.5 h-1.5 rounded-full bg-green-500" />
                                  </div>
                                  <div className="text-[10px] text-foreground/80 font-medium leading-tight">
                                    {t("landing.coreDemo.kanban.authFeature")}
                                  </div>
                                  <div className="mt-2 flex items-center gap-1 opacity-50 group-hover/card:opacity-100 transition-opacity">
                                    <div className="w-3 h-3 rounded-full bg-primary/20" />
                                    <div className="h-1 w-8 bg-border rounded-full" />
                                  </div>
                                </div>
                              ))}
                              {col.cards.length === 0 && (
                                <div className="h-full flex items-center justify-center border-2 border-dashed border-border/30 rounded-lg">
                                  <span className="text-[10px] text-muted-foreground/40 font-medium">
                                    {t("landing.coreDemo.kanban.empty")}
                                  </span>
                                </div>
                              )}
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  {feature.architecture && (
                    <div className="bg-card/95 backdrop-blur rounded-xl border border-primary/20 shadow-2xl shadow-primary/10 p-8 relative overflow-hidden animate-border-glow">
                      <div className="relative z-10">
                        {/* Your Infrastructure box */}
                        <div className="border-2 border-dashed border-primary/20 bg-primary/5 rounded-2xl p-8 relative">
                          <div className="absolute -top-3 left-6 px-2 bg-card text-xs font-bold text-primary uppercase tracking-wider border border-primary/20 rounded">
                            {t("landing.coreDemo.architecture.yourInfrastructure")}
                          </div>

                          <div className="flex items-center justify-center gap-12">
                            {/* Runner */}
                            <div className="text-center group/node">
                              <div className="w-20 h-20 bg-card border-2 border-primary/20 rounded-2xl flex items-center justify-center mx-auto mb-3 shadow-lg group-hover/node:border-primary/50 group-hover/node:shadow-primary/20 transition-all">
                                <svg className="w-10 h-10 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2" />
                                </svg>
                              </div>
                              <div className="text-sm font-bold">{t("landing.coreDemo.architecture.runner")}</div>
                              <div className="text-[10px] text-muted-foreground mt-1 font-mono">Docker/K8s</div>
                            </div>

                            {/* Connection */}
                            <div className="flex flex-col items-center gap-1">
                              <div className="w-16 h-0.5 bg-gradient-to-r from-transparent via-primary/50 to-transparent" />
                              <div className="text-[10px] text-muted-foreground font-mono">gRPC mTLS</div>
                              <div className="w-16 h-0.5 bg-gradient-to-r from-transparent via-primary/50 to-transparent" />
                            </div>

                            {/* Agent */}
                            <div className="text-center group/node">
                              <div className="w-20 h-20 bg-secondary/80 border-2 border-border rounded-2xl flex items-center justify-center mx-auto mb-3 shadow-lg group-hover/node:border-primary/50 transition-all">
                                <svg className="w-10 h-10 text-foreground/70" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                                </svg>
                              </div>
                              <div className="text-sm font-bold">{t("landing.coreDemo.architecture.agent")}</div>
                              <div className="text-[10px] text-muted-foreground mt-1 font-mono">Isolated Pod</div>
                            </div>
                          </div>
                        </div>

                        {/* Cloud connection */}
                        <div className="h-16 w-px bg-gradient-to-b from-primary/20 to-transparent mx-auto my-2 relative">
                          <div className="absolute top-1/2 left-4 -translate-y-1/2 text-[10px] text-muted-foreground whitespace-nowrap font-mono">
                            {t("landing.coreDemo.architecture.websocket")} (Encrypted)
                          </div>
                        </div>
                      </div>

                      <div className="text-center relative z-10">
                        <div className="inline-flex items-center gap-2 px-6 py-2.5 bg-card/80 backdrop-blur rounded-full border border-border shadow-lg">
                          <div className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
                          <span className="text-sm font-medium">{t("landing.coreDemo.architecture.agentsmeshCloud")}</span>
                        </div>
                      </div>

                      {/* Background decoration */}
                      <div className="absolute inset-0 bg-gradient-to-b from-transparent via-primary/5 to-transparent opacity-50 pointer-events-none" />
                    </div>
                  )}

                  {feature.schedule && (
                    <div className="bg-card/95 backdrop-blur rounded-xl border border-primary/20 shadow-2xl shadow-primary/10 overflow-hidden animate-border-glow">
                      {/* Header */}
                      <div className="flex items-center justify-between px-5 py-3.5 bg-muted/50 border-b border-border">
                        <div className="flex items-center gap-2">
                          <svg className="w-4 h-4 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                          </svg>
                          <span className="text-sm font-bold text-foreground">{t("landing.coreDemo.schedule.title")}</span>
                        </div>
                        <div className="flex items-center gap-1.5 px-2.5 py-1 bg-primary/10 border border-primary/20 rounded-full">
                          <svg className="w-3 h-3 text-primary animate-spin" style={{ animationDuration: '3s' }} fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                          </svg>
                          <span className="text-[11px] font-medium text-primary">{t("landing.coreDemo.schedule.auto")}</span>
                        </div>
                      </div>

                      {/* Task rows */}
                      <div className="divide-y divide-border/50">
                        {[
                          { schedule: t("landing.coreDemo.schedule.daily"), task: t("landing.coreDemo.schedule.depUpdate"), status: "passed" },
                          { schedule: t("landing.coreDemo.schedule.weekly"), task: t("landing.coreDemo.schedule.secScan"), status: "running" },
                          { schedule: t("landing.coreDemo.schedule.friday"), task: t("landing.coreDemo.schedule.codeReview"), status: "pending" },
                          { schedule: t("landing.coreDemo.schedule.hourly"), task: t("landing.coreDemo.schedule.testSuite"), status: "passed" },
                        ].map((row, i) => (
                          <div key={i} className="flex items-center gap-4 px-5 py-4 hover:bg-muted/30 transition-colors group/row">
                            {/* Status indicator */}
                            <div className={`w-2.5 h-2.5 rounded-full flex-shrink-0 ${
                              row.status === "passed" ? "bg-green-500" :
                              row.status === "running" ? "bg-yellow-500 animate-pulse" :
                              "bg-muted-foreground/30"
                            }`} />

                            {/* Schedule */}
                            <div className="w-24 flex-shrink-0">
                              <span className="text-[11px] font-mono text-muted-foreground bg-secondary/50 px-2 py-0.5 rounded">
                                {row.schedule}
                              </span>
                            </div>

                            {/* Task name */}
                            <div className="flex-1 text-sm font-medium text-foreground/90 group-hover/row:text-foreground transition-colors">
                              {row.task}
                            </div>

                            {/* Status badge */}
                            <div className={`text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded-full ${
                              row.status === "passed" ? "bg-green-500/10 text-green-500 border border-green-500/20" :
                              row.status === "running" ? "bg-yellow-500/10 text-yellow-500 border border-yellow-500/20" :
                              "bg-muted text-muted-foreground border border-border"
                            }`}>
                              {row.status === "passed" ? t("landing.coreDemo.schedule.passed") :
                               row.status === "running" ? t("landing.coreDemo.schedule.running") :
                               t("landing.coreDemo.schedule.pending")}
                            </div>
                          </div>
                        ))}
                      </div>

                      {/* Footer */}
                      <div className="px-5 py-3 bg-muted/30 border-t border-border flex items-center justify-between">
                        <span className="text-[11px] text-muted-foreground">
                          {t("landing.coreDemo.schedule.nextRun")} <span className="font-mono text-foreground/70">12m 34s</span>
                        </span>
                        <div className="flex gap-1">
                          {[1, 2, 3, 4, 5, 6, 7].map((d) => (
                            <div key={d} className={`w-1.5 h-4 rounded-sm ${
                              d <= 5 ? "bg-green-500/60" : "bg-border"
                            }`} />
                          ))}
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
