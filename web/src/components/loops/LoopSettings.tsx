"use client";

import React from "react";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";

interface LoopSettingsProps {
  executionMode: string;
  setExecutionMode: (v: string) => void;
  cronEnabled: boolean;
  setCronEnabled: (v: boolean) => void;
  cronExpression: string;
  setCronExpression: (v: string) => void;
  sandboxStrategy: string;
  setSandboxStrategy: (v: string) => void;
  concurrencyPolicy: string;
  setConcurrencyPolicy: (v: string) => void;
  timeoutMinutes: number;
  setTimeoutMinutes: (v: number) => void;
  maxConcurrentRuns: number;
  setMaxConcurrentRuns: (v: number) => void;
  maxRetainedRuns: number;
  setMaxRetainedRuns: (v: number) => void;
  sessionPersistence: boolean;
  setSessionPersistence: (v: boolean) => void;
  callbackUrl: string;
  setCallbackUrl: (v: string) => void;
  t: (key: string) => string;
}

export function LoopSettings({
  executionMode, setExecutionMode,
  cronEnabled, setCronEnabled,
  cronExpression, setCronExpression,
  sandboxStrategy, setSandboxStrategy,
  concurrencyPolicy, setConcurrencyPolicy,
  timeoutMinutes, setTimeoutMinutes,
  maxConcurrentRuns, setMaxConcurrentRuns,
  maxRetainedRuns, setMaxRetainedRuns,
  sessionPersistence, setSessionPersistence,
  callbackUrl, setCallbackUrl,
  t,
}: LoopSettingsProps) {
  return (
    <div className="border-t border-border pt-4 space-y-4">
      {/* Cron scheduling (optional, API trigger is always available) */}
      <CronSection
        cronEnabled={cronEnabled} setCronEnabled={setCronEnabled}
        cronExpression={cronExpression} setCronExpression={setCronExpression}
        t={t}
      />

      {/* Execution Mode */}
      <div className="space-y-1.5">
        <Label>{t("loops.executionMode")}</Label>
        <Select value={executionMode} onValueChange={setExecutionMode}>
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="autopilot">{t("loops.modeAutopilot")}</SelectItem>
            <SelectItem value="direct">{t("loops.modeDirect")}</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {/* Sandbox & Concurrency */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div className="space-y-1.5">
          <Label>{t("loops.sandboxStrategy")}</Label>
          <Select value={sandboxStrategy} onValueChange={setSandboxStrategy}>
            <SelectTrigger><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="persistent">{t("loops.sandboxPersistent")}</SelectItem>
              <SelectItem value="fresh">{t("loops.sandboxFresh")}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-1.5">
          <Label>{t("loops.concurrency")}</Label>
          <Select value={concurrencyPolicy} onValueChange={setConcurrencyPolicy}>
            <SelectTrigger><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="skip">{t("loops.policySkip")}</SelectItem>
              <SelectItem value="queue" disabled>
                {t("loops.policyQueue")} ({t("loops.comingSoon")})
              </SelectItem>
              <SelectItem value="replace" disabled>
                {t("loops.policyReplace")} ({t("loops.comingSoon")})
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      {/* Timeout & Max Concurrent */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div className="space-y-1.5">
          <Label>{t("loops.timeout")}</Label>
          <div className="flex items-center gap-2">
            <Input
              type="number" min={1} max={1440}
              value={timeoutMinutes}
              onChange={(e) => setTimeoutMinutes(Math.max(1, parseInt(e.target.value) || 60))}
              className="w-24"
            />
            <span className="text-sm text-muted-foreground">{t("loops.minutes")}</span>
          </div>
        </div>
        <div className="space-y-1.5">
          <Label>{t("loops.maxConcurrentRuns")}</Label>
          <Input
            type="number" min={1} max={10}
            value={maxConcurrentRuns}
            onChange={(e) => setMaxConcurrentRuns(parseInt(e.target.value) || 1)}
            className="w-24"
          />
        </div>
      </div>

      {/* Run History Limit */}
      <div className="space-y-1.5">
        <Label>{t("loops.maxRetainedRuns")}</Label>
        <div className="flex items-center gap-2">
          <Input
            type="number" min={0} max={10000}
            value={maxRetainedRuns}
            onChange={(e) => setMaxRetainedRuns(Math.max(0, parseInt(e.target.value) || 0))}
            className="w-24"
          />
          <span className="text-sm text-muted-foreground">
            {maxRetainedRuns === 0 ? t("loops.unlimited") : ""}
          </span>
        </div>
        <p className="text-xs text-muted-foreground">{t("loops.maxRetainedRunsHelp")}</p>
      </div>

      {/* Session Persistence */}
      {sandboxStrategy === "persistent" && (
        <div className="flex items-center justify-between">
          <div>
            <Label>{t("loops.sessionPersistence")}</Label>
            <p className="text-xs text-muted-foreground">{t("loops.sessionPersistenceHelp")}</p>
          </div>
          <Switch checked={sessionPersistence} onCheckedChange={setSessionPersistence} />
        </div>
      )}

      {/* Webhook URL */}
      <div className="space-y-1.5">
        <Label>{t("loops.callbackUrl")}</Label>
        <Input
          type="url"
          value={callbackUrl}
          onChange={(e) => setCallbackUrl(e.target.value)}
          placeholder={t("loops.callbackUrlPlaceholder")}
        />
        <p className="text-xs text-muted-foreground">{t("loops.callbackUrlHelp")}</p>
      </div>
    </div>
  );
}

/** Cron scheduling toggle + expression input */
function CronSection({
  cronEnabled, setCronEnabled,
  cronExpression, setCronExpression,
  t,
}: Pick<LoopSettingsProps, "cronEnabled" | "setCronEnabled" | "cronExpression" | "setCronExpression" | "t">) {
  return (
    <>
      <div className="flex items-center justify-between">
        <div>
          <Label>{t("loops.enableCron")}</Label>
          <p className="text-xs text-muted-foreground">{t("loops.apiAlwaysAvailable")}</p>
        </div>
        <Switch checked={cronEnabled} onCheckedChange={setCronEnabled} />
      </div>

      {cronEnabled && (
        <div className="space-y-1.5">
          <Label>{t("loops.cronExpression")}</Label>
          <Input
            value={cronExpression}
            onChange={(e) => setCronExpression(e.target.value)}
            placeholder="0 9 * * *"
            className="font-mono"
          />
          <p className="text-xs text-muted-foreground">
            {t("loops.cronHelp")}
          </p>
        </div>
      )}
    </>
  );
}
