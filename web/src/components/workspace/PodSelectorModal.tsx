"use client";

import React, { useMemo } from "react";
import { usePodStore } from "@/stores/pod";
import { Button } from "@/components/ui/button";
import { getPodDisplayName } from "@/lib/pod-utils";

interface PodSelectorModalProps {
  openPodKeys: string[];
  onSelect: (podKey: string) => void;
  onClose: () => void;
}

export function PodSelectorModal({ openPodKeys, onSelect, onClose }: PodSelectorModalProps) {
  // Subscribe to raw pods array — filter in useMemo to avoid creating
  // a new array reference inside the selector (causes infinite re-render loop).
  const allPods = usePodStore((s) => s.pods);
  const pods = useMemo(
    () => allPods.filter(
      (pod) => pod.status === "running" && !openPodKeys.includes(pod.pod_key)
    ),
    [allPods, openPodKeys]
  );

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-background border border-border rounded-lg w-full max-w-md max-h-[80vh] overflow-hidden">
        <div className="p-4 border-b border-border">
          <h2 className="text-lg font-semibold">Select a Pod</h2>
          <p className="text-sm text-muted-foreground">
            Choose a running pod to open a terminal
          </p>
        </div>

        <div className="overflow-y-auto max-h-96">
          {pods.length === 0 ? (
            <div className="p-8 text-center text-muted-foreground">
              <p>No running pods available</p>
              <p className="text-sm mt-1">Create a pod to start a terminal</p>
            </div>
          ) : (
            <div className="divide-y divide-border">
              {pods.map((pod) => (
                <button
                  key={pod.pod_key}
                  className="w-full p-4 text-left hover:bg-muted transition-colors"
                  onClick={() => onSelect(pod.pod_key)}
                >
                  <div className="flex items-center justify-between">
                    <code className="text-sm font-mono bg-muted px-2 py-0.5 rounded">
                      {getPodDisplayName(pod)}
                    </code>
                    <span className="text-xs text-green-500 dark:text-green-400">{pod.status}</span>
                  </div>
                  <div className="mt-1 text-xs text-muted-foreground">
                    <span>Agent: {pod.agent_status}</span>
                    {pod.runner && (
                      <span className="ml-2">• Runner: {pod.runner.node_id}</span>
                    )}
                  </div>
                </button>
              ))}
            </div>
          )}
        </div>

        <div className="p-4 border-t border-border">
          <Button variant="outline" className="w-full" onClick={onClose}>
            Cancel
          </Button>
        </div>
      </div>
    </div>
  );
}

export default PodSelectorModal;
