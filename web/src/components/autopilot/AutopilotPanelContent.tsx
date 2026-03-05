"use client";

import * as React from "react";
import { cn } from "@/lib/utils";
import { Progress } from "@/components/ui/progress";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useAutopilotStore } from "@/stores/autopilot";
import { Brain, ListChecks, History, Bot, Terminal } from "lucide-react";
import {
  ThinkingTab,
  ProgressTab,
  HistoryTab,
  ControlButtons,
  normalizeDecisionType,
  phaseConfig,
} from "./panel";

interface AutopilotPanelContentProps {
  podKey: string | null;
  className?: string;
}

export function AutopilotPanelContent({ podKey, className }: AutopilotPanelContentProps) {
  const [activeTab, setActiveTab] = React.useState<"thinking" | "progress" | "history">("thinking");
  // Reactive selectors — re-render only when the matched controller or its thinking changes
  const activePhases = ["initializing", "running", "paused", "user_takeover", "waiting_approval"];
  const autopilotController = useAutopilotStore((s) =>
    podKey
      ? s.autopilotControllers.find((c) => c.pod_key === podKey && activePhases.includes(c.phase))
      : undefined
  );
  const autopilotControllerKey = autopilotController?.autopilot_controller_key;
  const thinking = useAutopilotStore((s) =>
    autopilotControllerKey ? s.thinking[autopilotControllerKey] ?? null : null
  );

  // Auto switch to thinking tab when help is needed
  const decisionType = thinking?.decision_type;
  React.useEffect(() => {
    if (!decisionType) return;
    const normalizedType = normalizeDecisionType(decisionType);
    if (normalizedType === "need_help") {
      setActiveTab("thinking");
    }
  }, [decisionType]);

  // No pod selected
  if (!podKey) {
    return (
      <div className={cn("flex items-center justify-center h-full text-muted-foreground text-sm", className)}>
        <Terminal className="w-4 h-4 mr-2" />
        <span>Select a Pod first</span>
      </div>
    );
  }

  // No autopilot for this pod
  if (!autopilotController) {
    return (
      <div className={cn("flex flex-col items-center justify-center h-full text-muted-foreground", className)}>
        <Bot className="w-8 h-8 mb-2 opacity-50" />
        <span className="text-sm">No active Autopilot for this Pod</span>
        <span className="text-xs mt-1">Click the Bot icon in terminal header to start</span>
      </div>
    );
  }

  const phaseInfo = phaseConfig[autopilotController.phase];
  const progress = (autopilotController.current_iteration / autopilotController.max_iterations) * 100;

  return (
    <div className={cn("flex flex-col h-full", className)}>
      {/* Header with status and controls */}
      <div className="flex items-center gap-3 px-3 py-2 border-b border-border/50">
        {/* Status */}
        <div className="flex items-center gap-2">
          <div className={cn("flex items-center gap-1.5", phaseInfo.color)}>
            {phaseInfo.icon}
            <span className="text-sm font-medium">{phaseInfo.label}</span>
          </div>
        </div>

        {/* Progress */}
        <div className="flex items-center gap-2 flex-1">
          <div className="w-24">
            <Progress value={progress} className="h-1.5" />
          </div>
          <span className="text-xs text-muted-foreground">
            {autopilotController.current_iteration}/{autopilotController.max_iterations}
          </span>
        </div>

        {/* Control buttons */}
        <ControlButtons autopilotController={autopilotController} />
      </div>

      {/* Tabs */}
      <Tabs
        value={activeTab}
        onValueChange={(v) => setActiveTab(v as typeof activeTab)}
        className="flex-1 flex flex-col min-h-0"
      >
        <TabsList className="h-8 bg-transparent px-2 justify-start gap-0 border-b border-border/50 rounded-none">
          <TabsTrigger
            value="thinking"
            className="h-7 px-3 text-xs data-[state=active]:bg-transparent data-[state=active]:shadow-none data-[state=active]:border-b-2 data-[state=active]:border-primary rounded-none"
          >
            <Brain className="h-3 w-3 mr-1.5" />
            Thinking
          </TabsTrigger>
          <TabsTrigger
            value="progress"
            className="h-7 px-3 text-xs data-[state=active]:bg-transparent data-[state=active]:shadow-none data-[state=active]:border-b-2 data-[state=active]:border-primary rounded-none"
          >
            <ListChecks className="h-3 w-3 mr-1.5" />
            Progress
          </TabsTrigger>
          <TabsTrigger
            value="history"
            className="h-7 px-3 text-xs data-[state=active]:bg-transparent data-[state=active]:shadow-none data-[state=active]:border-b-2 data-[state=active]:border-primary rounded-none"
          >
            <History className="h-3 w-3 mr-1.5" />
            History
          </TabsTrigger>
        </TabsList>

        <div className="flex-1 overflow-auto">
          <TabsContent value="thinking" className="mt-0 h-full">
            <ThinkingTab thinking={thinking} />
          </TabsContent>

          <TabsContent value="progress" className="mt-0 h-full">
            <ProgressTab thinking={thinking} />
          </TabsContent>

          <TabsContent value="history" className="mt-0 h-full">
            <HistoryTab autopilotControllerKey={autopilotController.autopilot_controller_key} />
          </TabsContent>
        </div>
      </Tabs>
    </div>
  );
}

export default AutopilotPanelContent;
