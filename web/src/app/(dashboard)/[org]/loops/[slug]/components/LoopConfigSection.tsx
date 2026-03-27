import Link from "next/link";
import { Bot, Layers, Shield, Timer, Hash, Zap, ExternalLink } from "lucide-react";
import type { LoopData } from "@/stores/loop";
import { ConfigRow } from "./ConfigRow";

interface LoopConfigSectionProps {
  loop: LoopData;
  orgSlug: string;
  t: (key: string) => string;
}

export function LoopConfigSection({ loop, orgSlug, t }: LoopConfigSectionProps) {
  return (
    <div className="grid grid-cols-1 xl:grid-cols-[1fr_1fr] gap-3 mb-8">
      <ConfigPanel loop={loop} t={t} />
      <ApiTriggerPanel loop={loop} orgSlug={orgSlug} t={t} />
    </div>
  );
}

function ConfigPanel({ loop, t }: { loop: LoopData; t: (key: string) => string }) {
  const concurrencyLabel =
    loop.concurrency_policy === "skip"
      ? t("loops.policySkip")
      : loop.concurrency_policy === "queue"
        ? t("loops.policyQueue")
        : t("loops.policyReplace");

  return (
    <section>
      <h2 className="text-sm font-semibold mb-3">{t("loops.configuration")}</h2>
      <div className="border rounded-xl overflow-hidden h-[calc(100%-2rem)]">
        <div className="p-4 space-y-3">
          <ConfigRow icon={<Bot className="w-3.5 h-3.5" />} label={t("loops.mode")}
            value={loop.execution_mode === "autopilot" ? t("loops.modeAutopilot") : t("loops.modeDirect")} />
          <ConfigRow icon={<Layers className="w-3.5 h-3.5" />} label={t("loops.sandbox")}
            value={loop.sandbox_strategy === "persistent" ? t("loops.sandboxPersistent") : t("loops.sandboxFresh")} />
          <ConfigRow icon={<Shield className="w-3.5 h-3.5" />} label={t("loops.concurrency")} value={concurrencyLabel} />
          <ConfigRow icon={<Timer className="w-3.5 h-3.5" />} label={t("loops.timeout")}
            value={`${loop.timeout_minutes} ${t("loops.minutes")}`} />
          <ConfigRow icon={<Shield className="w-3.5 h-3.5" />} label={t("loops.sessionLabel")}
            value={loop.session_persistence ? t("loops.sessionKeep") : t("loops.sessionFresh")} />
          <ConfigRow icon={<Hash className="w-3.5 h-3.5" />} label={t("loops.maxConcurrent")}
            value={loop.max_concurrent_runs.toString()} />
          {loop.max_retained_runs > 0 && (
            <ConfigRow icon={<Hash className="w-3.5 h-3.5" />} label={t("loops.maxRetainedRuns")}
              value={loop.max_retained_runs.toString()} />
          )}
          <ConfigRow icon={<Timer className="w-3.5 h-3.5" />} label={t("loops.triggerLabel")}
            value={loop.cron_expression ? (
              <span className="px-1.5 py-0.5 rounded bg-amber-500/10 text-amber-600 dark:text-amber-400 text-[10px] font-medium font-mono">
                {loop.cron_expression}
              </span>
            ) : (
              <span className="text-muted-foreground">{t("loops.onDemand")}</span>
            )} />
        </div>
        {loop.callback_url && (
          <div className="px-4 pb-3">
            <ConfigRow icon={<Zap className="w-3.5 h-3.5" />} label={t("loops.webhookUrl")}
              value={<span className="text-xs font-mono truncate max-w-[140px] sm:max-w-[200px] md:max-w-[300px] inline-block align-bottom">{loop.callback_url}</span>} />
          </div>
        )}
        <div className="border-t p-4">
          <div className="text-xs font-medium text-muted-foreground mb-2">{t("loops.prompt")}</div>
          <pre className="p-3 bg-muted/50 rounded-lg text-sm whitespace-pre-wrap font-mono leading-relaxed max-h-32 overflow-y-auto text-foreground/80">
            {loop.prompt_template}
          </pre>
        </div>
      </div>
    </section>
  );
}

function ApiTriggerPanel({ loop, orgSlug, t }: { loop: LoopData; orgSlug: string; t: (key: string) => string }) {
  return (
    <section>
      <h2 className="text-sm font-semibold mb-3">{t("loops.apiTrigger")}</h2>
      <div className="border rounded-xl p-4 h-[calc(100%-2rem)]">
        <p className="text-xs text-muted-foreground mb-3">{t("loops.apiTriggerDesc")}</p>
        <div className="relative">
          <div className="absolute top-2 left-3 text-[10px] text-muted-foreground font-medium uppercase tracking-wider">
            {t("loops.curlExample")}
          </div>
          <pre suppressHydrationWarning className="pt-7 pb-3 px-3 bg-muted/50 rounded-lg text-xs font-mono overflow-x-auto whitespace-pre-wrap break-all text-foreground/70 leading-relaxed">
{`curl -X POST \\
  ${typeof window !== "undefined" ? window.location.origin : ""}/api/v1/ext/orgs/${orgSlug}/loops/${loop.slug}/trigger \\
  -H "X-API-Key: amk_your_api_key_here" \\
  -H "Content-Type: application/json"`}
          </pre>
        </div>
        <p className="text-[10px] text-muted-foreground mt-2">{t("loops.apiKeyHint")}</p>
        <Link href={`/${orgSlug}/settings`}
          className="inline-flex items-center gap-1 text-[10px] text-primary hover:underline mt-1">
          {t("loops.manageApiKeys")}
          <ExternalLink className="w-2.5 h-2.5" />
        </Link>
      </div>
    </section>
  );
}
