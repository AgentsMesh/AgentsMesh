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
    name: "Loopal",
    descriptionKey: "landing.agentLogos.descriptions.selfBuilt",
    icon: (
      <svg viewBox="0 0 24 24" className="w-8 h-8" fill="currentColor">
        <circle cx="12" cy="12" r="9" stroke="currentColor" strokeWidth="2" fill="none" />
        <path d="M12 6l4 4-4 4-4-4z" stroke="currentColor" strokeWidth="2" fill="none" />
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
    <section className="py-16 bg-[var(--azure-bg-deeper)]">
      <div className="container mx-auto px-4 sm:px-6 lg:px-8">
        <p className="text-center font-headline text-xs uppercase tracking-[0.25em] text-[var(--azure-text-muted)] mb-10">
          {t("landing.agentLogos.title")}
        </p>

        <div className="relative overflow-hidden">
          <div className="absolute left-0 top-0 bottom-0 w-24 bg-gradient-to-r from-[var(--azure-bg-deeper)] to-transparent z-10 pointer-events-none" />
          <div className="absolute right-0 top-0 bottom-0 w-24 bg-gradient-to-l from-[var(--azure-bg-deeper)] to-transparent z-10 pointer-events-none" />

          <div className="flex animate-scroll gap-6">
            {[...agentConfigs, ...agentConfigs].map((agent, index) => (
              <div
                key={index}
                className="flex-shrink-0 flex items-center gap-4 px-6 py-4 azure-glass rounded-2xl border border-white/5 hover:border-[var(--azure-cyan)]/30 transition-all cursor-default group"
              >
                <div className="text-[var(--azure-text-muted)] group-hover:text-[var(--azure-cyan)] transition-colors">
                  {agent.icon}
                </div>
                <div>
                  <div className="font-headline font-semibold text-sm text-foreground">{agent.name}</div>
                  <div className="text-xs text-[var(--azure-text-muted)]">{t(agent.descriptionKey)}</div>
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
