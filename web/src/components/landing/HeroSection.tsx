"use client";

import Link from "next/link";
import { useEffect, useState, useMemo } from "react";
import { Button } from "@/components/ui/button";
import { useTranslations } from "@/lib/i18n/client";

interface TuiFrame {
  time: number;
  content: {
    header: string;
    podInfoKey: "initializing" | "running" | "observing";
    mainContent: { type: string; textKey?: string; text?: string }[];
    input: string;
    topology: {
      nodes: { id: string; label: string; agent: string; status: string; x: number; y: number }[];
      connections: { from: string; to: string; label: string; type?: string; animated?: boolean }[];
    };
  };
}

// Frame template - text will be filled from translations
const getTuiFrames = (t: (key: string) => string): TuiFrame[] => [
  {
    time: 0,
    content: {
      header: "Claude Code",
      podInfoKey: "initializing",
      mainContent: [
        { type: "system", textKey: "landing.heroDemo.connecting" },
      ],
      input: "",
      topology: { nodes: [], connections: [] }
    }
  },
  {
    time: 1500,
    content: {
      header: "Claude Code",
      podInfoKey: "running",
      mainContent: [
        { type: "system", text: "✓ " + t("landing.heroDemo.connected") },
        { type: "system", text: "✓ " + t("landing.heroDemo.workspace") },
        { type: "system", text: "" },
        { type: "user", textKey: "landing.heroDemo.monitorPod" },
      ],
      input: "",
      topology: {
        nodes: [{ id: "alpha", label: "alpha-dev", agent: "Claude Code", status: "running", x: 25, y: 40 }],
        connections: []
      }
    }
  },
  {
    time: 3500,
    content: {
      header: "Claude Code",
      podInfoKey: "running",
      mainContent: [
        { type: "system", text: "✓ " + t("landing.heroDemo.connected") },
        { type: "system", text: "✓ " + t("landing.heroDemo.workspace") },
        { type: "system", text: "" },
        { type: "user", textKey: "landing.heroDemo.monitorPod" },
        { type: "system", text: "" },
        { type: "assistant", textKey: "landing.heroDemo.bindingPod" },
        { type: "tool", text: "⚡ mesh_bind_pod beta-dev --scopes observe,message" },
      ],
      input: "",
      topology: {
        nodes: [
          { id: "alpha", label: "alpha-dev", agent: "Claude Code", status: "running", x: 25, y: 40 },
          { id: "beta", label: "beta-dev", agent: "Codex CLI", status: "running", x: 75, y: 40 },
        ],
        connections: []
      }
    }
  },
  {
    time: 5500,
    content: {
      header: "Claude Code",
      podInfoKey: "observing",
      mainContent: [
        { type: "user", textKey: "landing.heroDemo.monitorPod" },
        { type: "system", text: "" },
        { type: "assistant", textKey: "landing.heroDemo.bindingPod" },
        { type: "tool", text: "⚡ mesh_bind_pod beta-dev --scopes observe,message" },
        { type: "success", text: "✓ " + t("landing.heroDemo.boundToPod") },
        { type: "system", text: "" },
        { type: "observe-header", text: "━━━ " + t("landing.heroDemo.observingHeader") + " ━━━" },
        { type: "observe", text: "│ " + t("landing.heroDemo.analyzing") },
      ],
      input: "",
      topology: {
        nodes: [
          { id: "alpha", label: "alpha-dev", agent: "Claude Code", status: "running", x: 25, y: 40 },
          { id: "beta", label: "beta-dev", agent: "Codex CLI", status: "running", x: 75, y: 40 },
        ],
        connections: [
          { from: "alpha", to: "beta", label: "observing", animated: true }
        ]
      }
    }
  },
  {
    time: 7500,
    content: {
      header: "Claude Code",
      podInfoKey: "observing",
      mainContent: [
        { type: "tool", text: "⚡ mesh_bind_pod beta-dev --scopes observe,message" },
        { type: "success", text: "✓ " + t("landing.heroDemo.boundToPod") },
        { type: "system", text: "" },
        { type: "observe-header", text: "━━━ " + t("landing.heroDemo.observingHeader") + " ━━━" },
        { type: "observe", text: "│ " + t("landing.heroDemo.analyzing") },
        { type: "observe", text: "│ " + t("landing.heroDemo.creatingOAuth") },
        { type: "observe", text: "│ " + t("landing.heroDemo.writingOAuth") },
        { type: "system", text: "" },
        { type: "assistant", textKey: "landing.heroDemo.noticedOAuth" },
      ],
      input: "",
      topology: {
        nodes: [
          { id: "alpha", label: "alpha-dev", agent: "Claude Code", status: "running", x: 25, y: 40 },
          { id: "beta", label: "beta-dev", agent: "Codex CLI", status: "running", x: 75, y: 40 },
        ],
        connections: [
          { from: "alpha", to: "beta", label: "observing", animated: true }
        ]
      }
    }
  },
  {
    time: 9500,
    content: {
      header: "Claude Code",
      podInfoKey: "observing",
      mainContent: [
        { type: "observe-header", text: "━━━ " + t("landing.heroDemo.observingHeader") + " ━━━" },
        { type: "observe", text: "│ " + t("landing.heroDemo.creatingOAuth") },
        { type: "observe", text: "│ " + t("landing.heroDemo.writingOAuth") },
        { type: "system", text: "" },
        { type: "assistant", textKey: "landing.heroDemo.noticedOAuth" },
        { type: "tool", text: "⚡ mesh_send_message beta-dev" },
        { type: "message-sent", text: "📤 " + t("landing.heroDemo.messageSent") },
        { type: "system", text: "" },
        { type: "observe", text: "│ " + t("landing.heroDemo.receivedSuggestion") },
      ],
      input: "",
      topology: {
        nodes: [
          { id: "alpha", label: "alpha-dev", agent: "Claude Code", status: "running", x: 25, y: 40 },
          { id: "beta", label: "beta-dev", agent: "Codex CLI", status: "running", x: 75, y: 40 },
        ],
        connections: [
          { from: "alpha", to: "beta", label: "observing", animated: true },
          { from: "alpha", to: "beta", label: "message", type: "message", animated: true }
        ]
      }
    }
  },
  {
    time: 12000,
    content: {
      header: "Claude Code",
      podInfoKey: "observing",
      mainContent: [
        { type: "tool", text: "⚡ mesh_send_message beta-dev" },
        { type: "message-sent", text: "📤 " + t("landing.heroDemo.messageSent") },
        { type: "system", text: "" },
        { type: "observe", text: "│ " + t("landing.heroDemo.receivedSuggestion") },
        { type: "observe", text: "│ " + t("landing.heroDemo.writingRateLimit") },
        { type: "observe", text: "│ ✓ " + t("landing.heroDemo.rateLimitAdded") },
        { type: "system", text: "" },
        { type: "assistant", textKey: "landing.heroDemo.reviewChanges" },
        { type: "tool", text: "⚡ mesh_read_file beta-dev:src/middleware/rateLimit.ts" },
      ],
      input: "",
      topology: {
        nodes: [
          { id: "alpha", label: "alpha-dev", agent: "Claude Code", status: "running", x: 25, y: 40 },
          { id: "beta", label: "beta-dev", agent: "Codex CLI", status: "running", x: 75, y: 40 },
        ],
        connections: [
          { from: "alpha", to: "beta", label: "collaborating", animated: true }
        ]
      }
    }
  },
];

export function HeroSection() {
  const [currentFrameIndex, setCurrentFrameIndex] = useState(0);
  const [displayedLines, setDisplayedLines] = useState<number>(0);
  const t = useTranslations();

  // Memoize the translated frames
  const tuiFrames = useMemo(() => getTuiFrames(t), [t]);

  // Cycle through frames
  useEffect(() => {
    const frame = tuiFrames[currentFrameIndex];
    const nextFrame = tuiFrames[currentFrameIndex + 1];

    if (nextFrame) {
      const delay = nextFrame.time - frame.time;
      const timer = setTimeout(() => {
        setCurrentFrameIndex(prev => prev + 1);
        setDisplayedLines(0);
      }, delay);
      return () => clearTimeout(timer);
    } else {
      // Reset after last frame
      const timer = setTimeout(() => {
        setCurrentFrameIndex(0);
        setDisplayedLines(0);
      }, 4000);
      return () => clearTimeout(timer);
    }
  }, [currentFrameIndex, tuiFrames]);

  // Animate lines appearing
  useEffect(() => {
    const frame = tuiFrames[currentFrameIndex];
    const totalLines = frame.content.mainContent.length;

    if (displayedLines < totalLines) {
      const timer = setTimeout(() => {
        setDisplayedLines(prev => prev + 1);
      }, 150);
      return () => clearTimeout(timer);
    }
  }, [currentFrameIndex, displayedLines, tuiFrames]);

  const currentFrame = tuiFrames[currentFrameIndex];
  const topology = currentFrame.content.topology;

  const getLineStyle = (type: string) => {
    switch (type) {
      case "user": return "text-blue-400";
      case "assistant": return "text-foreground";
      case "system": return "text-muted-foreground";
      case "tool": return "text-yellow-400";
      case "success": return "text-green-400";
      case "observe-header": return "text-primary font-bold";
      case "observe": return "text-cyan-300";
      case "message-sent": return "text-purple-400";
      default: return "text-muted-foreground";
    }
  };

  return (
    <section className="relative min-h-screen flex items-center pt-16">
      {/* Background gradient */}
      <div className="absolute inset-0 bg-gradient-to-b from-primary/5 via-transparent to-transparent" />

      {/* Grid pattern */}
      <div
        className="absolute inset-0 opacity-[0.02]"
        style={{
          backgroundImage: `linear-gradient(rgba(255,255,255,.1) 1px, transparent 1px),
                           linear-gradient(90deg, rgba(255,255,255,.1) 1px, transparent 1px)`,
          backgroundSize: '50px 50px'
        }}
      />

      <div className="container mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        <div className="grid lg:grid-cols-2 gap-8 lg:gap-12 items-center">
          {/* Left: Text Content */}
          <div className="text-center lg:text-left">
            <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-primary/10 border border-primary/20 text-primary text-sm mb-6">
              <span className="relative flex h-2 w-2">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary opacity-75"></span>
                <span className="relative inline-flex rounded-full h-2 w-2 bg-primary"></span>
              </span>
              {t("landing.hero.badge")}
            </div>

            <h1 className="text-4xl sm:text-5xl lg:text-6xl font-extrabold leading-tight mb-4">
              <span className="text-foreground">{t("landing.hero.slogan1")}</span>
              <br />
              <span className="text-primary">{t("landing.hero.slogan2")}</span>
            </h1>

            <p className="text-lg sm:text-xl text-muted-foreground/80 mb-6 font-medium italic">
              {t("landing.hero.tagline")}
            </p>

            <p className="text-lg sm:text-xl text-muted-foreground mb-8 max-w-xl mx-auto lg:mx-0">
              {t("landing.hero.description")}
            </p>

            <div className="flex flex-col sm:flex-row gap-4 justify-center lg:justify-start">
              <Link href="/register">
                <Button size="lg" className="w-full sm:w-auto bg-primary text-primary-foreground hover:bg-primary/90 text-base px-8">
                  {t("landing.hero.getStartedFree")}
                </Button>
              </Link>
              <Link href="/docs">
                <Button size="lg" variant="outline" className="w-full sm:w-auto text-base px-8">
                  {t("landing.hero.viewDocs")}
                </Button>
              </Link>
            </div>

            {/* Trust badges */}
            <div className="mt-10 pt-8 border-t border-border/50">
              <p className="text-sm text-muted-foreground mb-4">{t("landing.hero.trustedBy")}</p>
              <div className="flex items-center justify-center lg:justify-start gap-6 opacity-50">
                <div className="text-sm font-medium">{t("landing.hero.teams")}</div>
                <div className="w-px h-4 bg-border" />
                <div className="text-sm font-medium">{t("landing.hero.pods")}</div>
                <div className="w-px h-4 bg-border" />
                <div className="text-sm font-medium">{t("landing.hero.openSource")}</div>
              </div>
            </div>
          </div>

          {/* Right: TUI + Topology */}
          <div className="relative space-y-4">
            {/* Glow effect */}
            <div className="absolute -inset-4 bg-primary/20 blur-3xl rounded-full opacity-20" />

            {/* Claude Code TUI Window */}
            <div className="relative bg-[#0d0d0d] rounded-xl border border-border overflow-hidden shadow-2xl">
              {/* TUI Header */}
              <div className="flex items-center justify-between px-4 py-2 bg-[#1a1a1a] border-b border-border">
                <div className="flex items-center gap-2">
                  <div className="flex gap-2">
                    <div className="w-3 h-3 rounded-full bg-red-500/80" />
                    <div className="w-3 h-3 rounded-full bg-yellow-500/80" />
                    <div className="w-3 h-3 rounded-full bg-green-500/80" />
                  </div>
                  <span className="text-sm font-semibold text-foreground ml-2">{currentFrame.content.header}</span>
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-xs px-2 py-0.5 bg-primary/20 text-primary rounded">MCP</span>
                  <span className="text-xs px-2 py-0.5 bg-green-500/20 text-green-400 rounded">AgentsMesh</span>
                </div>
              </div>

              {/* Pod Info Bar */}
              <div className="px-4 py-1.5 bg-[#141414] border-b border-border text-xs font-mono text-muted-foreground">
                {t(`landing.heroDemo.podInfo.${currentFrame.content.podInfoKey}`)}
              </div>

              {/* Main Content Area */}
              <div className="p-4 font-mono text-sm h-[260px] overflow-hidden">
                {currentFrame.content.mainContent.slice(0, displayedLines).map((line, index) => {
                  const lineText = line.textKey ? t(line.textKey) : line.text;
                  return (
                    <div
                      key={`${currentFrameIndex}-${index}`}
                      className={`${getLineStyle(line.type)} ${lineText ? '' : 'h-4'}`}
                    >
                      {line.type === "user" && <span className="text-blue-500 mr-2">❯</span>}
                      {line.type === "assistant" && <span className="text-primary mr-2">●</span>}
                      {lineText}
                    </div>
                  );
                })}
                {displayedLines < currentFrame.content.mainContent.length && (
                  <span className="animate-pulse text-primary">▋</span>
                )}
              </div>

              {/* Input Area */}
              <div className="px-4 py-2 bg-[#141414] border-t border-border">
                <div className="flex items-center gap-2 text-sm font-mono">
                  <span className="text-primary">❯</span>
                  <span className="text-muted-foreground">{t("landing.heroDemo.typeMessage")}</span>
                  <span className="animate-pulse text-primary">▋</span>
                </div>
              </div>
            </div>

            {/* Topology visualization */}
            <div className="relative bg-[#0d0d0d] rounded-xl border border-border overflow-hidden">
              <div className="px-4 py-2 bg-[#1a1a1a] border-b border-border flex items-center justify-between">
                <span className="text-xs font-mono text-muted-foreground">{t("landing.heroDemo.podTopology")}</span>
                <div className="flex items-center gap-2">
                  <span className="text-xs text-green-400">● {t("landing.heroDemo.live")}</span>
                </div>
              </div>

              <div className="relative h-[140px] p-4">
                {/* Connection lines */}
                <svg className="absolute inset-0 w-full h-full pointer-events-none" style={{ zIndex: 1 }}>
                  <defs>
                    <marker id="arrowhead" markerWidth="10" markerHeight="7" refX="9" refY="3.5" orient="auto">
                      <polygon points="0 0, 10 3.5, 0 7" fill="#22d3ee" opacity="0.8" />
                    </marker>
                  </defs>
                  {topology.connections.map((conn, i) => {
                    const fromNode = topology.nodes.find(n => n.id === conn.from);
                    const toNode = topology.nodes.find(n => n.id === conn.to);
                    if (!fromNode || !toNode) return null;

                    const midX = (fromNode.x + toNode.x) / 2;
                    const midY = (fromNode.y + toNode.y) / 2;

                    return (
                      <g key={i}>
                        <line
                          x1={`${fromNode.x + 8}%`}
                          y1={`${fromNode.y}%`}
                          x2={`${toNode.x - 8}%`}
                          y2={`${toNode.y}%`}
                          stroke="#22d3ee"
                          strokeWidth="2"
                          strokeDasharray={conn.type === "message" ? "5,5" : "0"}
                          markerEnd="url(#arrowhead)"
                          opacity="0.6"
                          className={conn.animated ? "animate-pulse" : ""}
                        />
                        <text
                          x={`${midX}%`}
                          y={`${midY - 8}%`}
                          fill="#a1a1aa"
                          fontSize="10"
                          textAnchor="middle"
                          className="font-mono"
                        >
                          {conn.label}
                        </text>
                      </g>
                    );
                  })}
                </svg>

                {/* Pod nodes */}
                {topology.nodes.map((node) => (
                  <div
                    key={node.id}
                    className="absolute transform -translate-x-1/2 -translate-y-1/2 transition-all duration-500 ease-out"
                    style={{ left: `${node.x}%`, top: `${node.y}%`, zIndex: 2 }}
                  >
                    <div className="px-3 py-2 bg-[#1a1a1a] border border-primary/40 rounded-lg shadow-lg shadow-primary/10 min-w-[110px]">
                      <div className="flex items-center gap-2 mb-1">
                        <div className={`w-2 h-2 rounded-full ${node.status === "running" ? "bg-green-500 animate-pulse" : "bg-gray-500"}`} />
                        <span className="text-xs font-mono text-primary font-semibold">{node.label}</span>
                      </div>
                      <div className="text-[10px] text-muted-foreground">{node.agent}</div>
                    </div>
                  </div>
                ))}

                {/* Empty state */}
                {topology.nodes.length === 0 && (
                  <div className="flex items-center justify-center h-full text-muted-foreground text-sm">
                    <span className="animate-pulse">{t("landing.heroDemo.initializingPod")}</span>
                  </div>
                )}
              </div>
            </div>
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
