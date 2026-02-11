"use client";

import { useTranslations } from "next-intl";

const agentConfigs = [
  {
    name: "Claude Code",
    descriptionKey: "landing.agentLogos.descriptions.anthropic",
    icon: (
      <svg viewBox="0 0 24 24" className="w-8 h-8" fill="currentColor">
        <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5" stroke="currentColor" strokeWidth="2" fill="none" />
      </svg>
    ),
  },
  {
    name: "Codex CLI",
    descriptionKey: "landing.agentLogos.descriptions.openai",
    icon: (
      <svg viewBox="0 0 24 24" className="w-8 h-8" fill="currentColor">
        <circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="2" fill="none" />
        <path d="M12 6v6l4 2" stroke="currentColor" strokeWidth="2" fill="none" />
      </svg>
    ),
  },
  {
    name: "Gemini CLI",
    descriptionKey: "landing.agentLogos.descriptions.google",
    icon: (
      <svg viewBox="0 0 24 24" className="w-8 h-8" fill="currentColor">
        <polygon points="12,2 22,8.5 22,15.5 12,22 2,15.5 2,8.5" stroke="currentColor" strokeWidth="2" fill="none" />
      </svg>
    ),
  },
  {
    name: "Aider",
    descriptionKey: "landing.agentLogos.descriptions.openSource",
    icon: (
      <svg viewBox="0 0 24 24" className="w-8 h-8" fill="currentColor">
        <rect x="3" y="3" width="18" height="18" rx="2" stroke="currentColor" strokeWidth="2" fill="none" />
        <path d="M9 9l6 6M15 9l-6 6" stroke="currentColor" strokeWidth="2" />
      </svg>
    ),
  },
  {
    name: "OpenCode",
    descriptionKey: "landing.agentLogos.descriptions.community",
    icon: (
      <svg viewBox="0 0 24 24" className="w-8 h-8" fill="currentColor">
        <path d="M12 2a10 10 0 110 20 10 10 0 010-20z" stroke="currentColor" strokeWidth="2" fill="none" />
        <path d="M8 12l2 2 4-4" stroke="currentColor" strokeWidth="2" fill="none" />
      </svg>
    ),
  },
  {
    name: "Custom Agent",
    descriptionKey: "landing.agentLogos.descriptions.yourOwn",
    icon: (
      <svg viewBox="0 0 24 24" className="w-8 h-8" fill="currentColor">
        <path d="M12 5v14M5 12h14" stroke="currentColor" strokeWidth="2" />
      </svg>
    ),
  },
];

export function AgentLogos() {
  const t = useTranslations();

  return (
    <section className="py-12 border-y border-border bg-muted/50">
      <div className="container mx-auto px-4 sm:px-6 lg:px-8">
        <p className="text-center text-sm text-muted-foreground mb-8">
          {t("landing.agentLogos.title")}
        </p>

        <div className="relative overflow-hidden">
          {/* Gradient masks */}
          <div className="absolute left-0 top-0 bottom-0 w-20 bg-gradient-to-r from-background to-transparent z-10" />
          <div className="absolute right-0 top-0 bottom-0 w-20 bg-gradient-to-l from-background to-transparent z-10" />

          {/* Scrolling container */}
          <div className="flex animate-scroll gap-8">
            {/* First set */}
            {agentConfigs.map((agent, index) => (
              <div
                key={`first-${index}`}
                className="flex-shrink-0 flex items-center gap-4 px-6 py-4 bg-secondary/30 rounded-xl border border-border hover:border-primary/50 transition-colors cursor-pointer group"
              >
                <div className="text-muted-foreground group-hover:text-primary transition-colors">
                  {agent.icon}
                </div>
                <div>
                  <div className="font-semibold text-sm">{agent.name}</div>
                  <div className="text-xs text-muted-foreground">{t(agent.descriptionKey)}</div>
                </div>
              </div>
            ))}
            {/* Duplicate for seamless loop */}
            {agentConfigs.map((agent, index) => (
              <div
                key={`second-${index}`}
                className="flex-shrink-0 flex items-center gap-4 px-6 py-4 bg-secondary/30 rounded-xl border border-border hover:border-primary/50 transition-colors cursor-pointer group"
              >
                <div className="text-muted-foreground group-hover:text-primary transition-colors">
                  {agent.icon}
                </div>
                <div>
                  <div className="font-semibold text-sm">{agent.name}</div>
                  <div className="text-xs text-muted-foreground">{t(agent.descriptionKey)}</div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      <style jsx>{`
        @keyframes scroll {
          0% {
            transform: translateX(0);
          }
          100% {
            transform: translateX(-50%);
          }
        }
        .animate-scroll {
          animation: scroll 30s linear infinite;
        }
        .animate-scroll:hover {
          animation-play-state: paused;
        }
      `}</style>
    </section>
  );
}
