"use client";

import { useTranslations } from "next-intl";

/**
 * Architecture diagram for docs Introduction page.
 * Pure CSS/HTML — no external diagram library needed.
 *
 * Layout (top → bottom):
 *   Client Layer → Cloud Layer → [Control Plane / Data Plane split] → Runner Layer → Agent Layer
 */
export default function ArchitectureDiagram() {
  const t = useTranslations("docs.architecture");

  return (
    <div className="my-8 overflow-x-auto">
      <div className="min-w-[600px] max-w-3xl mx-auto flex flex-col items-center gap-0">
        {/* ── Client Layer ── */}
        <LayerBox
          label={t("clientLayer")}
          color="blue"
          items={[
            { icon: "🖥️", text: "Web" },
            { icon: "📱", text: "Mobile" },
            { icon: "📟", text: "Tablet" },
          ]}
          subtitle={t("clientSubtitle")}
        />

        {/* Arrow: HTTPS / WebSocket */}
        <Arrow label="HTTPS / WebSocket" />

        {/* ── Cloud Layer ── */}
        <div className="w-full border-2 border-emerald-500/40 rounded-xl bg-emerald-500/5 overflow-hidden">
          <div className="bg-emerald-500/10 px-4 py-1.5">
            <span className="text-[11px] font-semibold text-emerald-600 dark:text-emerald-400 uppercase tracking-wider">
              {t("cloudLayer")}
            </span>
          </div>
          <div className="p-4 pt-2">
            <div className="text-center mb-3">
              <span className="text-sm font-semibold text-emerald-700 dark:text-emerald-300">
                AgentsMesh Cloud
              </span>
              <p className="text-xs text-muted-foreground mt-0.5">
                {t("cloudDesc")}
              </p>
            </div>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-2 text-center text-xs">
              {(["orchestration", "monitoring", "collaboration", "security"] as const).map(
                (key) => (
                  <div
                    key={key}
                    className="bg-emerald-500/10 rounded-md px-2 py-1.5 font-medium text-emerald-700 dark:text-emerald-300"
                  >
                    {t(`cloud.${key}`)}
                  </div>
                )
              )}
            </div>
          </div>
        </div>

        {/* ── Split: Control Plane + Data Plane ── */}
        <div className="w-full grid grid-cols-2 gap-4 my-1">
          {/* Control Plane */}
          <div className="flex flex-col items-center">
            <div className="h-6 w-px border-l-2 border-dashed border-amber-500/60" />
            <div className="border border-amber-500/40 rounded-lg bg-amber-500/5 px-3 py-2.5 text-center w-full">
              <div className="text-[11px] font-bold text-amber-700 dark:text-amber-300 uppercase tracking-wider mb-1">
                {t("controlPlane")}
              </div>
              <div className="text-xs text-muted-foreground">{t("controlPlaneDesc")}</div>
              <div className="mt-1.5 inline-flex items-center gap-1 rounded-full bg-amber-500/15 px-2.5 py-0.5 text-[10px] font-semibold text-amber-700 dark:text-amber-300">
                🔒 gRPC + mTLS
              </div>
            </div>
            <div className="h-6 w-px border-l-2 border-dashed border-amber-500/60" />
          </div>

          {/* Data Plane */}
          <div className="flex flex-col items-center">
            <div className="h-6 w-px border-l-2 border-dashed border-violet-500/60" />
            <div className="border border-violet-500/40 rounded-lg bg-violet-500/5 px-3 py-2.5 text-center w-full">
              <div className="text-[11px] font-bold text-violet-700 dark:text-violet-300 uppercase tracking-wider mb-1">
                {t("dataPlane")}
              </div>
              <div className="text-xs text-muted-foreground">{t("dataPlaneDesc")}</div>
              <div className="mt-1.5 inline-flex items-center gap-1 rounded-full bg-violet-500/15 px-2.5 py-0.5 text-[10px] font-semibold text-violet-700 dark:text-violet-300">
                ⚡ Relay {t("cluster")}
              </div>
            </div>
            <div className="h-6 w-px border-l-2 border-dashed border-violet-500/60" />
          </div>
        </div>

        {/* ── Runner Layer ── */}
        <div className="w-full border-2 border-sky-500/40 rounded-xl bg-sky-500/5 overflow-hidden">
          <div className="bg-sky-500/10 px-4 py-1.5">
            <span className="text-[11px] font-semibold text-sky-600 dark:text-sky-400 uppercase tracking-wider">
              {t("runnerLayer")}
            </span>
          </div>
          <div className="p-4 pt-2">
            <div className="text-center mb-3">
              <span className="text-sm font-semibold text-sky-700 dark:text-sky-300">
                {t("selfHostedRunners")}
              </span>
              <p className="text-xs text-muted-foreground mt-0.5">
                {t("runnerDesc")}
              </p>
            </div>
            <div className="grid grid-cols-3 gap-2 text-center text-xs">
              {[
                { icon: "🖥️", label: t("runnerMac") },
                { icon: "🐧", label: t("runnerLinux") },
                { icon: "☁️", label: t("runnerCloud") },
              ].map((r) => (
                <div
                  key={r.label}
                  className="bg-sky-500/10 rounded-md px-2 py-1.5 font-medium text-sky-700 dark:text-sky-300"
                >
                  {r.icon} {r.label}
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Arrow: PTY / Sandbox */}
        <Arrow label="PTY + Sandbox + Git Worktree" />

        {/* ── Agent Layer ── */}
        <LayerBox
          label={t("agentLayer")}
          color="rose"
          items={[
            { icon: "🤖", text: "Claude Code" },
            { icon: "🤖", text: "Codex CLI" },
            { icon: "🤖", text: "Gemini CLI" },
            { icon: "🤖", text: "Aider" },
          ]}
          subtitle={t("agentSubtitle")}
        />

        {/* ── Security callout ── */}
        <div className="mt-4 w-full border border-amber-500/30 rounded-lg bg-amber-500/5 px-4 py-3">
          <div className="flex items-start gap-2">
            <span className="text-base mt-0.5">🔐</span>
            <div>
              <div className="text-xs font-semibold text-amber-700 dark:text-amber-300 mb-1">
                {t("securityTitle")}
              </div>
              <p className="text-xs text-muted-foreground leading-relaxed">
                {t("securityDesc")}
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

/* ── Helper: Arrow ── */
function Arrow({ label }: { label: string }) {
  return (
    <div className="flex flex-col items-center py-1">
      <div className="h-4 w-px bg-border" />
      <div className="text-[10px] text-muted-foreground font-medium px-2 py-0.5 rounded-full bg-muted/50 border border-border">
        {label}
      </div>
      <div className="h-2 w-px bg-border" />
      <div className="w-0 h-0 border-l-[5px] border-l-transparent border-r-[5px] border-r-transparent border-t-[6px] border-t-border" />
    </div>
  );
}

/* ── Helper: Layer Box ── */
function LayerBox({
  label,
  color,
  items,
  subtitle,
}: {
  label: string;
  color: "blue" | "rose";
  items: { icon: string; text: string }[];
  subtitle: string;
}) {
  const colorMap = {
    blue: {
      border: "border-blue-500/40",
      bg: "bg-blue-500/5",
      headerBg: "bg-blue-500/10",
      tag: "text-blue-600 dark:text-blue-400",
      itemBg: "bg-blue-500/10",
      itemText: "text-blue-700 dark:text-blue-300",
    },
    rose: {
      border: "border-rose-500/40",
      bg: "bg-rose-500/5",
      headerBg: "bg-rose-500/10",
      tag: "text-rose-600 dark:text-rose-400",
      itemBg: "bg-rose-500/10",
      itemText: "text-rose-700 dark:text-rose-300",
    },
  };
  const c = colorMap[color];

  return (
    <div className={`w-full border-2 ${c.border} rounded-xl ${c.bg} overflow-hidden`}>
      <div className={`${c.headerBg} px-4 py-1.5`}>
        <span
          className={`text-[11px] font-semibold ${c.tag} uppercase tracking-wider`}
        >
          {label}
        </span>
      </div>
      <div className="p-4 pt-2">
        <div className="text-center mb-2">
          <p className="text-xs text-muted-foreground">{subtitle}</p>
        </div>
        <div
          className="grid gap-2 text-center text-xs"
          style={{ gridTemplateColumns: `repeat(${items.length}, 1fr)` }}
        >
          {items.map((item) => (
            <div
              key={item.text}
              className={`${c.itemBg} rounded-md px-2 py-1.5 font-medium ${c.itemText}`}
            >
              {item.icon} {item.text}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
