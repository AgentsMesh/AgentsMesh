"use client";

import type { useTranslations } from "next-intl";
import { AgentPodDemo } from "./AgentPodDemo";

type TFunc = ReturnType<typeof useTranslations>;

interface FeatureVisualsProps {
  feature: {
    podDemo?: boolean;
    diagram?: { nodes: Array<{ id: string; label: string; x: number; y: number }>; connections: Array<{ from: string; to: string; dashed?: boolean }> };
    kanban?: { columns: Array<{ titleKey: string; cards: string[] }> };
    schedule?: boolean;
    architecture?: boolean;
  };
  t: TFunc;
}

export function FeatureVisuals({ feature, t }: FeatureVisualsProps) {
  if (feature.podDemo) return <AgentPodDemo />;
  if (feature.diagram) return <MeshDiagram t={t} />;
  if (feature.kanban) return <KanbanBoard columns={feature.kanban.columns} t={t} />;
  if (feature.schedule) return <SchedulePanel t={t} />;
  if (feature.architecture) return <ArchitectureDiagram t={t} />;
  return null;
}

function MeshDiagram({ t }: { t: TFunc }) {
  return (
    <div className="bg-[#0f0e2a] rounded-xl border border-[#2a2556] shadow-2xl p-8 relative overflow-hidden">
      <div className="absolute inset-0 opacity-[0.06]"
        style={{ backgroundImage: 'radial-gradient(circle, #818cf8 1px, transparent 1px)', backgroundSize: '20px 20px' }} />
      <div className="relative h-64">
        <div className="absolute left-[10%] top-[20%] px-5 py-3 bg-[#1a1745] border border-[#3730a3]/40 rounded-xl shadow-lg shadow-indigo-500/10 z-10">
          <div className="flex items-center gap-2 mb-1">
            <div className="w-2 h-2 rounded-full bg-[#34d399] animate-pulse" />
            <div className="text-sm font-bold text-[#e0e7ff]">Claude Code</div>
          </div>
          <div className="text-xs text-[#a5b4fc] font-mono">{t("landing.coreDemo.podA")}</div>
        </div>
        <div className="absolute right-[10%] top-[20%] px-5 py-3 bg-[#1a1745] border border-[#3730a3]/40 rounded-xl shadow-lg shadow-indigo-500/10 z-10">
          <div className="flex items-center gap-2 mb-1">
            <div className="w-2 h-2 rounded-full bg-[#34d399] animate-pulse delay-75" />
            <div className="text-sm font-bold text-[#e0e7ff]">Codex CLI</div>
          </div>
          <div className="text-xs text-[#a5b4fc] font-mono">{t("landing.coreDemo.podB")}</div>
        </div>
        <div className="absolute left-1/2 -translate-x-1/2 bottom-[20%] px-6 py-3 bg-[#1a1745] border border-[#3730a3]/40 rounded-full shadow-lg shadow-indigo-500/10 z-10 flex items-center gap-3">
          <div className="w-8 h-8 rounded-full bg-[#818cf8]/15 flex items-center justify-center text-[#818cf8]">#</div>
          <div>
            <div className="text-sm font-bold text-[#e0e7ff]">{t("landing.coreDemo.devChannel")}</div>
            <div className="text-xs text-[#a5b4fc]/70">{t("landing.coreDemo.members", { count: 3 })}</div>
          </div>
        </div>
        <svg className="absolute inset-0 w-full h-full pointer-events-none" style={{ zIndex: 0 }}>
          <path d="M120 80 Q 180 200 250 220" fill="none" stroke="#818cf8" strokeWidth="2" strokeDasharray="4" opacity="0.3" />
          <path d="M380 80 Q 320 200 250 220" fill="none" stroke="#818cf8" strokeWidth="2" strokeDasharray="4" opacity="0.3" />
          <path d="M140 60 Q 250 20 360 60" fill="none" stroke="#818cf8" strokeWidth="1.5" strokeDasharray="6 4" opacity="0.2" />
          <circle r="3" fill="#818cf8" className="animate-ping" style={{ animationDuration: '3s' }}>
            <animateMotion dur="2s" repeatCount="indefinite" path="M120 80 Q 180 200 250 220" />
          </circle>
          <circle r="3" fill="#818cf8" className="animate-ping" style={{ animationDuration: '3s', animationDelay: '1s' }}>
            <animateMotion dur="2s" repeatCount="indefinite" path="M380 80 Q 320 200 250 220" />
          </circle>
        </svg>
      </div>
    </div>
  );
}

function KanbanBoard({ columns, t }: { columns: Array<{ titleKey: string; cards: string[] }>; t: TFunc }) {
  return (
    <div className="bg-[#fafbfc] rounded-xl border border-[#d8dee4] shadow-2xl p-6 relative overflow-hidden">
      <div className="grid grid-cols-4 gap-4">
        {columns.map((col, i) => (
          <div key={i} className="bg-[#f0f3f6] rounded-xl p-3 flex flex-col h-48">
            <div className="flex items-center justify-between mb-3 px-1">
              <span className="text-[10px] font-bold uppercase tracking-wider text-[#57606a]">{t(col.titleKey)}</span>
              <span className="text-[10px] bg-[#d8dee4] px-1.5 py-0.5 rounded text-[#57606a]">{col.cards.length}</span>
            </div>
            <div className="space-y-2 flex-1 overflow-y-auto custom-scrollbar">
              {col.cards.map((card, j) => (
                <div key={j} className="bg-white border border-[#d8dee4] rounded-lg p-2.5 shadow-sm hover:shadow-md hover:border-[#0969da]/40 transition-all cursor-default group/card">
                  <div className="flex items-center justify-between mb-1.5">
                    <span className="font-mono text-[10px] text-[#0969da] bg-[#ddf4ff] px-1 rounded">{card}</span>
                    <div className="w-1.5 h-1.5 rounded-full bg-[#1a7f37]" />
                  </div>
                  <div className="text-[10px] text-[#24292f] font-medium leading-tight">{t("landing.coreDemo.kanban.authFeature")}</div>
                  <div className="mt-2 flex items-center gap-1 opacity-50 group-hover/card:opacity-100 transition-opacity">
                    <div className="w-3 h-3 rounded-full bg-[#0969da]/15" />
                    <div className="h-1 w-8 bg-[#d8dee4] rounded-full" />
                  </div>
                </div>
              ))}
              {col.cards.length === 0 && (
                <div className="h-full flex items-center justify-center border-2 border-dashed border-[#d8dee4] rounded-lg">
                  <span className="text-[10px] text-[#8c959f] font-medium">{t("landing.coreDemo.kanban.empty")}</span>
                </div>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function SchedulePanel({ t }: { t: TFunc }) {
  const rows = [
    { schedule: t("landing.coreDemo.schedule.daily"), task: t("landing.coreDemo.schedule.depUpdate"), status: "passed" },
    { schedule: t("landing.coreDemo.schedule.weekly"), task: t("landing.coreDemo.schedule.secScan"), status: "running" },
    { schedule: t("landing.coreDemo.schedule.friday"), task: t("landing.coreDemo.schedule.codeReview"), status: "pending" },
    { schedule: t("landing.coreDemo.schedule.hourly"), task: t("landing.coreDemo.schedule.testSuite"), status: "passed" },
  ];

  return (
    <div className="bg-[#071a12] rounded-xl border border-[#1a3a2a] shadow-2xl overflow-hidden">
      <div className="flex items-center justify-between px-5 py-3.5 bg-[#0c2618] border-b border-[#1a3a2a]">
        <div className="flex items-center gap-2">
          <svg className="w-4 h-4 text-[#34d399]" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <span className="text-sm font-bold text-[#d1fae5]">{t("landing.coreDemo.schedule.title")}</span>
        </div>
        <div className="flex items-center gap-1.5 px-2.5 py-1 bg-[#34d399]/10 border border-[#34d399]/20 rounded-full">
          <svg className="w-3 h-3 text-[#34d399] animate-spin" style={{ animationDuration: '3s' }} fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          <span className="text-[11px] font-medium text-[#34d399]">{t("landing.coreDemo.schedule.auto")}</span>
        </div>
      </div>
      <div className="divide-y divide-[#1a3a2a]">
        {rows.map((row, i) => (
          <div key={i} className="flex items-center gap-4 px-5 py-4 hover:bg-[#0c2618] transition-colors group/row">
            <div className={`w-2.5 h-2.5 rounded-full flex-shrink-0 ${row.status === "passed" ? "bg-[#34d399]" : row.status === "running" ? "bg-[#fbbf24] animate-pulse" : "bg-[#365a48]"}`} />
            <div className="w-24 flex-shrink-0">
              <span className="text-[11px] font-mono text-[#6ee7b7] bg-[#0c2618] px-2 py-0.5 rounded">{row.schedule}</span>
            </div>
            <div className="flex-1 text-sm font-medium text-[#a7f3d0]/80 group-hover/row:text-[#d1fae5] transition-colors">{row.task}</div>
            <div className={`text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded-full ${row.status === "passed" ? "bg-[#34d399]/10 text-[#34d399] border border-[#34d399]/20" : row.status === "running" ? "bg-[#fbbf24]/10 text-[#fbbf24] border border-[#fbbf24]/20" : "bg-[#1a3a2a] text-[#6ee7b7]/50 border border-[#365a48]"}`}>
              {row.status === "passed" ? t("landing.coreDemo.schedule.passed") : row.status === "running" ? t("landing.coreDemo.schedule.running") : t("landing.coreDemo.schedule.pending")}
            </div>
          </div>
        ))}
      </div>
      <div className="px-5 py-3 bg-[#0c2618] border-t border-[#1a3a2a] flex items-center justify-between">
        <span className="text-[11px] text-[#6ee7b7]/60">
          {t("landing.coreDemo.schedule.nextRun")} <span className="font-mono text-[#a7f3d0]/70">12m 34s</span>
        </span>
        <div className="flex gap-1">
          {[1, 2, 3, 4, 5, 6, 7].map((d) => (
            <div key={d} className={`w-1.5 h-4 rounded-sm ${d <= 5 ? "bg-[#34d399]/50" : "bg-[#1a3a2a]"}`} />
          ))}
        </div>
      </div>
    </div>
  );
}

function ArchitectureDiagram({ t }: { t: TFunc }) {
  return (
    <div className="bg-[#1c1917] rounded-xl border border-[#44403c] shadow-2xl p-8 relative overflow-hidden">
      <div className="relative z-10">
        <div className="border-2 border-dashed border-[#57534e] bg-[#292524] rounded-2xl p-8 relative">
          <div className="absolute -top-3 left-6 px-2 bg-[#1c1917] text-xs font-bold text-[#fb923c] uppercase tracking-wider border border-[#44403c] rounded">
            {t("landing.coreDemo.architecture.yourInfrastructure")}
          </div>
          <div className="flex items-center justify-center gap-12">
            <div className="text-center group/node">
              <div className="w-20 h-20 bg-[#1c1917] border-2 border-[#44403c] rounded-2xl flex items-center justify-center mx-auto mb-3 shadow-lg group-hover/node:border-[#fb923c]/50 transition-all">
                <svg className="w-10 h-10 text-[#fb923c]" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2" />
                </svg>
              </div>
              <div className="text-sm font-bold text-[#e7e5e4]">{t("landing.coreDemo.architecture.runner")}</div>
              <div className="text-[10px] text-[#a8a29e] mt-1 font-mono">Docker/K8s</div>
            </div>
            <div className="flex flex-col items-center gap-1">
              <div className="w-16 h-0.5 bg-gradient-to-r from-transparent via-[#fb923c]/50 to-transparent" />
              <div className="text-[10px] text-[#a8a29e] font-mono">gRPC mTLS</div>
              <div className="w-16 h-0.5 bg-gradient-to-r from-transparent via-[#fb923c]/50 to-transparent" />
            </div>
            <div className="text-center group/node">
              <div className="w-20 h-20 bg-[#292524] border-2 border-[#44403c] rounded-2xl flex items-center justify-center mx-auto mb-3 shadow-lg group-hover/node:border-[#fb923c]/50 transition-all">
                <svg className="w-10 h-10 text-[#e7e5e4]" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
              </div>
              <div className="text-sm font-bold text-[#e7e5e4]">{t("landing.coreDemo.architecture.agent")}</div>
              <div className="text-[10px] text-[#a8a29e] mt-1 font-mono">Isolated Pod</div>
            </div>
          </div>
        </div>
        <div className="h-16 w-px bg-gradient-to-b from-[#44403c] to-transparent mx-auto my-2 relative">
          <div className="absolute top-1/2 left-4 -translate-y-1/2 text-[10px] text-[#a8a29e] whitespace-nowrap font-mono">
            {t("landing.coreDemo.architecture.websocket")} (Encrypted)
          </div>
        </div>
      </div>
      <div className="text-center relative z-10">
        <div className="inline-flex items-center gap-2 px-6 py-2.5 bg-[#292524] rounded-full border border-[#44403c] shadow-lg">
          <div className="w-2 h-2 rounded-full bg-[#34d399] animate-pulse" />
          <span className="text-sm font-medium text-[#e7e5e4]">{t("landing.coreDemo.architecture.agentsmeshCloud")}</span>
        </div>
      </div>
      <div className="absolute inset-0 bg-gradient-to-b from-transparent via-[#fb923c]/5 to-transparent opacity-50 pointer-events-none" />
    </div>
  );
}
