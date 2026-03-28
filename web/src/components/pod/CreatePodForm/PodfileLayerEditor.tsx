"use client";

import React from "react";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";

interface PodfileLayerEditorProps {
  generatedLayer: string;
  rawMode: boolean;
  rawText: string;
  onRawModeChange: (enabled: boolean) => void;
  onRawTextChange: (text: string) => void;
  t: (key: string) => string;
}

export function PodfileLayerEditor({
  generatedLayer,
  rawMode,
  rawText,
  onRawModeChange,
  onRawTextChange,
  t,
}: PodfileLayerEditorProps) {
  return (
    <div className="space-y-2 border-t pt-3">
      {/* Toggle: Form Mode / Source Mode */}
      <div className="flex items-center justify-between">
        <Label className="text-sm">{t("ide.createPod.podfileLayer")}</Label>
        <div className="flex items-center gap-2">
          <span className="text-xs text-muted-foreground">
            {t("ide.createPod.sourceMode")}
          </span>
          <Switch checked={rawMode} onCheckedChange={onRawModeChange} />
        </div>
      </div>

      {/* Layer preview or editor */}
      {rawMode ? (
        <Textarea
          value={rawText}
          onChange={(e) => onRawTextChange(e.target.value)}
          className="font-mono text-xs min-h-[120px] resize-y"
          placeholder={'CONFIG model = "opus"\nREPO "https://github.com/org/repo"'}
        />
      ) : (
        generatedLayer && (
          <pre className="bg-muted/50 rounded-md p-3 text-xs font-mono text-muted-foreground overflow-x-auto whitespace-pre-wrap">
            {generatedLayer}
          </pre>
        )
      )}
    </div>
  );
}
