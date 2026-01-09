"use client";

const features = [
  {
    number: "01",
    title: "DevPod",
    subtitle: "Remote AI Workstation",
    description:
      "Run AI agents in the cloud, access a complete development environment through your browser. Each session runs on isolated Git worktrees with real-time terminal access.",
    highlights: [
      "Web Terminal with real-time interaction",
      "Git Worktree isolation per session",
      "Multiple concurrent sessions",
      "Auto-cleanup on completion",
    ],
    terminal: {
      title: "DevPod Session",
      lines: [
        "$ claude-code --session dev-42",
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
    title: "AgentMesh",
    subtitle: "Multi-Agent Collaboration",
    description:
      "Enable multiple AI agents to work together through session bindings and channel communication. Visualize the collaboration topology in real-time.",
    highlights: [
      "Session Binding with permission control",
      "Channel-based message broadcasting",
      "Real-time topology visualization",
      "Cross-agent task coordination",
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
    title: "Tickets",
    subtitle: "Task-Driven Development",
    description:
      "Integrate AI sessions with your task management workflow. Create tickets, bind them to sessions, and track progress through a Kanban board.",
    highlights: [
      "Kanban board visualization",
      "Ticket ↔ Session binding",
      "Progress tracking",
      "MR/PR integration",
    ],
    kanban: {
      columns: [
        { title: "Backlog", cards: ["AUTH-1", "AUTH-3"] },
        { title: "In Progress", cards: ["AUTH-2"] },
        { title: "Review", cards: ["AUTH-4"] },
        { title: "Done", cards: [] },
      ],
    },
    align: "left",
  },
  {
    number: "04",
    title: "Self-hosted Runners",
    subtitle: "Your Infrastructure, Your Data",
    description:
      "Deploy runners on your own infrastructure. Your code and data never leave your environment, meeting the strictest security and compliance requirements.",
    highlights: [
      "Docker/Kubernetes deployment",
      "Complete data localization",
      "Air-gapped environment support",
      "Full audit trail",
    ],
    architecture: true,
    align: "right",
  },
];

export function CoreFeatures() {
  return (
    <section className="py-24" id="features">
      <div className="container mx-auto px-4 sm:px-6 lg:px-8">
        {/* Section header */}
        <div className="text-center mb-16">
          <h2 className="text-3xl sm:text-4xl font-bold mb-4">
            Core <span className="text-primary">Features</span>
          </h2>
          <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
            Everything you need to manage and coordinate terminal-based AI coding agents at scale.
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
                  <div className="bg-[#0d0d0d] rounded-xl border border-border overflow-hidden">
                    <div className="flex items-center gap-2 px-4 py-3 bg-[#1a1a1a] border-b border-border">
                      <div className="flex gap-2">
                        <div className="w-3 h-3 rounded-full bg-red-500/80" />
                        <div className="w-3 h-3 rounded-full bg-yellow-500/80" />
                        <div className="w-3 h-3 rounded-full bg-green-500/80" />
                      </div>
                      <span className="text-xs text-muted-foreground font-mono ml-2">
                        {feature.terminal.title}
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
                              ? "text-green-400"
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
                  <div className="bg-[#0d0d0d] rounded-xl border border-border p-8">
                    <div className="relative h-64">
                      {/* Agent nodes */}
                      <div className="absolute left-[10%] top-[20%] px-4 py-2 bg-primary/10 border border-primary/30 rounded-lg">
                        <div className="text-sm font-medium text-primary">Claude Code</div>
                        <div className="text-xs text-muted-foreground">Session A</div>
                      </div>
                      <div className="absolute right-[10%] top-[20%] px-4 py-2 bg-primary/10 border border-primary/30 rounded-lg">
                        <div className="text-sm font-medium text-primary">Codex CLI</div>
                        <div className="text-xs text-muted-foreground">Session B</div>
                      </div>
                      {/* Channel */}
                      <div className="absolute left-1/2 -translate-x-1/2 bottom-[20%] px-6 py-3 bg-secondary/50 border border-border rounded-lg">
                        <div className="text-sm font-medium">#dev-channel</div>
                        <div className="text-xs text-muted-foreground">3 members</div>
                      </div>
                      {/* Connection lines */}
                      <svg className="absolute inset-0 w-full h-full" style={{ zIndex: -1 }}>
                        <line x1="25%" y1="40%" x2="50%" y2="70%" stroke="#22d3ee" strokeWidth="2" strokeDasharray="4" opacity="0.5" />
                        <line x1="75%" y1="40%" x2="50%" y2="70%" stroke="#22d3ee" strokeWidth="2" strokeDasharray="4" opacity="0.5" />
                        <line x1="30%" y1="30%" x2="70%" y2="30%" stroke="#22d3ee" strokeWidth="1" strokeDasharray="8" opacity="0.3" />
                      </svg>
                    </div>
                  </div>
                )}

                {feature.kanban && (
                  <div className="bg-[#0d0d0d] rounded-xl border border-border p-4">
                    <div className="grid grid-cols-4 gap-3">
                      {feature.kanban.columns.map((col, i) => (
                        <div key={i} className="bg-secondary/20 rounded-lg p-3">
                          <div className="text-xs font-medium text-muted-foreground mb-3">
                            {col.title}
                          </div>
                          <div className="space-y-2">
                            {col.cards.map((card, j) => (
                              <div
                                key={j}
                                className="bg-[#1a1a1a] border border-border rounded p-2 text-xs"
                              >
                                <div className="font-mono text-primary">{card}</div>
                                <div className="text-muted-foreground mt-1">Auth feature</div>
                              </div>
                            ))}
                            {col.cards.length === 0 && (
                              <div className="text-xs text-muted-foreground/50 text-center py-4">
                                Empty
                              </div>
                            )}
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {feature.architecture && (
                  <div className="bg-[#0d0d0d] rounded-xl border border-border p-8">
                    <div className="relative">
                      {/* Your Infrastructure box */}
                      <div className="border-2 border-dashed border-primary/30 rounded-xl p-6">
                        <div className="text-xs text-primary mb-4">Your Infrastructure</div>
                        <div className="flex items-center justify-center gap-8">
                          {/* Runner */}
                          <div className="text-center">
                            <div className="w-16 h-16 bg-primary/10 border border-primary/30 rounded-lg flex items-center justify-center mx-auto mb-2">
                              <svg className="w-8 h-8 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2" />
                              </svg>
                            </div>
                            <div className="text-sm font-medium">Runner</div>
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
                            <div className="text-sm font-medium">Agent</div>
                          </div>
                        </div>
                      </div>
                      {/* Cloud connection */}
                      <div className="absolute -bottom-8 left-1/2 -translate-x-1/2 flex flex-col items-center">
                        <div className="w-px h-6 bg-border" />
                        <div className="text-xs text-muted-foreground">WebSocket</div>
                      </div>
                    </div>
                    <div className="mt-12 text-center">
                      <div className="inline-flex items-center gap-2 px-4 py-2 bg-secondary/30 rounded-lg border border-border">
                        <svg className="w-4 h-4 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 10-9.78 2.096A4.001 4.001 0 003 15z" />
                        </svg>
                        <span className="text-sm">AgentMesh Cloud</span>
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
