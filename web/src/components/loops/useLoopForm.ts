"use client";

import { useState, useEffect } from "react";
import type { LoopData } from "@/lib/api/loop";
import { RUNNER_HOST_PROFILE_ID } from "./types";

/**
 * Manages all Loop form field state and syncs values when dialog opens / editLoop changes.
 */
export function useLoopForm(open: boolean, editLoop?: LoopData) {
  const [loading, setLoading] = useState(false);

  // --- Basic fields ---
  const [name, setName] = useState(editLoop?.name || "");
  const [description, setDescription] = useState(editLoop?.description || "");
  const [promptTemplate, setPromptTemplate] = useState(editLoop?.prompt_template || "");

  // --- Pod configuration fields ---
  const [selectedAgentSlug, setSelectedAgentSlug] = useState<string | null>(editLoop?.agent_slug || null);
  const [selectedRunnerId, setSelectedRunnerId] = useState<number | null>(editLoop?.runner_id || null);
  const [selectedRepositoryId, setSelectedRepositoryId] = useState<number | null>(
    editLoop?.repository_id || null
  );
  const [selectedBranch, setSelectedBranch] = useState(editLoop?.branch_name || "");

  // --- Loop-specific fields ---
  const [executionMode, setExecutionMode] = useState<string>(editLoop?.execution_mode || "autopilot");
  const [cronEnabled, setCronEnabled] = useState(!!editLoop?.cron_expression);
  const [cronExpression, setCronExpression] = useState(editLoop?.cron_expression || "");
  const [sandboxStrategy, setSandboxStrategy] = useState<string>(editLoop?.sandbox_strategy || "persistent");
  const [concurrencyPolicy, setConcurrencyPolicy] = useState<string>(editLoop?.concurrency_policy || "skip");
  const [timeoutMinutes, setTimeoutMinutes] = useState(editLoop?.timeout_minutes || 60);
  const [callbackUrl, setCallbackUrl] = useState(editLoop?.callback_url || "");
  const [sessionPersistence, setSessionPersistence] = useState(editLoop?.session_persistence ?? true);
  const [maxConcurrentRuns, setMaxConcurrentRuns] = useState(editLoop?.max_concurrent_runs || 1);
  const [maxRetainedRuns, setMaxRetainedRuns] = useState(editLoop?.max_retained_runs || 0);

  // Sync form state when dialog opens or editLoop changes
  useEffect(() => {
    if (!open) return;
    setName(editLoop?.name || "");
    setDescription(editLoop?.description || "");
    setPromptTemplate(editLoop?.prompt_template || "");
    setSelectedAgentSlug(editLoop?.agent_slug || null);
    setSelectedRunnerId(editLoop?.runner_id || null);
    setSelectedRepositoryId(editLoop?.repository_id || null);
    setSelectedBranch(editLoop?.branch_name || "");
    setExecutionMode(editLoop?.execution_mode || "autopilot");
    setCronEnabled(!!editLoop?.cron_expression);
    setCronExpression(editLoop?.cron_expression || "");
    setSandboxStrategy(editLoop?.sandbox_strategy || "persistent");
    setConcurrencyPolicy(editLoop?.concurrency_policy || "skip");
    setTimeoutMinutes(editLoop?.timeout_minutes || 60);
    setCallbackUrl(editLoop?.callback_url || "");
    setSessionPersistence(editLoop?.session_persistence ?? true);
    setMaxConcurrentRuns(editLoop?.max_concurrent_runs || 1);
    setMaxRetainedRuns(editLoop?.max_retained_runs || 0);
    setLoading(false);
  }, [open, editLoop]);

  return {
    loading, setLoading,
    // Basic fields
    name, setName,
    description, setDescription,
    promptTemplate, setPromptTemplate,
    // Pod config
    selectedAgentSlug, setSelectedAgentSlug,
    selectedRunnerId, setSelectedRunnerId,
    selectedRepositoryId, setSelectedRepositoryId,
    selectedBranch, setSelectedBranch,
    // Loop settings
    executionMode, setExecutionMode,
    cronEnabled, setCronEnabled,
    cronExpression, setCronExpression,
    sandboxStrategy, setSandboxStrategy,
    concurrencyPolicy, setConcurrencyPolicy,
    timeoutMinutes, setTimeoutMinutes,
    callbackUrl, setCallbackUrl,
    sessionPersistence, setSessionPersistence,
    maxConcurrentRuns, setMaxConcurrentRuns,
    maxRetainedRuns, setMaxRetainedRuns,
  };
}

export type LoopFormState = ReturnType<typeof useLoopForm>;
