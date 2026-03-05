"use client";

import * as React from "react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useAutopilotStore } from "@/stores/autopilot";
import { Brain, ListChecks, History, X } from "lucide-react";
import { ThinkingTab } from "./ThinkingTab";
import { ProgressTab } from "./ProgressTab";
import { HistoryTab } from "./HistoryTab";
import type { AutopilotFloatingPanelProps } from "./types";

/**
 * AutopilotFloatingPanel - Displays autopilot status, thinking, and history
 *
 * Shows three tabs:
 * - Thinking: Current decision and reasoning
 * - Progress: Task completion status
 * - History: Past iterations
 */
export function AutopilotFloatingPanel({
  autopilotController,
  className,
  onClose,
}: AutopilotFloatingPanelProps) {
  const [activeTab, setActiveTab] = React.useState<"thinking" | "progress" | "history">("thinking");

  // Reactive thinking selector — re-renders when this controller's thinking changes
  const thinking = useAutopilotStore(
    (s) => s.thinking[autopilotController.autopilot_controller_key] ?? null
  );

  // Auto switch to thinking tab when help is needed
  React.useEffect(() => {
    if (thinking?.decision_type === "need_help") {
      setActiveTab("thinking");
    }
  }, [thinking?.decision_type]);

  return (
    <div
      className={cn(
        "border-b bg-card shadow-sm",
        className
      )}
    >
      <Tabs
        value={activeTab}
        onValueChange={(v) => setActiveTab(v as typeof activeTab)}
        className="w-full"
      >
        <div className="flex items-center justify-between px-2 border-b">
          <TabsList className="h-8 bg-transparent p-0 gap-0">
            <TabsTrigger
              value="thinking"
              className="h-8 px-3 text-xs data-[state=active]:bg-transparent data-[state=active]:shadow-none data-[state=active]:border-b-2 data-[state=active]:border-primary rounded-none"
            >
              <Brain className="h-3 w-3 mr-1.5" />
              Thinking
            </TabsTrigger>
            <TabsTrigger
              value="progress"
              className="h-8 px-3 text-xs data-[state=active]:bg-transparent data-[state=active]:shadow-none data-[state=active]:border-b-2 data-[state=active]:border-primary rounded-none"
            >
              <ListChecks className="h-3 w-3 mr-1.5" />
              Progress
            </TabsTrigger>
            <TabsTrigger
              value="history"
              className="h-8 px-3 text-xs data-[state=active]:bg-transparent data-[state=active]:shadow-none data-[state=active]:border-b-2 data-[state=active]:border-primary rounded-none"
            >
              <History className="h-3 w-3 mr-1.5" />
              History
            </TabsTrigger>
          </TabsList>

          {onClose && (
            <Button
              size="sm"
              variant="ghost"
              className="h-6 w-6 p-0"
              onClick={onClose}
            >
              <X className="h-3.5 w-3.5 text-muted-foreground" />
            </Button>
          )}
        </div>

        <TabsContent value="thinking" className="mt-0">
          <ThinkingTab thinking={thinking} />
        </TabsContent>

        <TabsContent value="progress" className="mt-0">
          <ProgressTab thinking={thinking} />
        </TabsContent>

        <TabsContent value="history" className="mt-0 py-2">
          <HistoryTab autopilotControllerKey={autopilotController.autopilot_controller_key} />
        </TabsContent>
      </Tabs>
    </div>
  );
}

export default AutopilotFloatingPanel;
