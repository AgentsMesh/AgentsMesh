"use client";

import { useCallback } from "react";
import { useLoopStore } from "@/stores/loop";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import type { LoopData } from "@/lib/api/loop";
import type { LoopFormState } from "./useLoopForm";

interface UseLoopSubmitParams {
  form: LoopFormState;
  editLoop?: LoopData;
  onCreated?: (createdLoop?: LoopData) => void;
  configValues: Record<string, unknown>;
  selectedCredentialProfileId: number;
}

/**
 * Handles Loop create/update submission logic.
 */
export function useLoopSubmit({
  form,
  editLoop,
  onCreated,
  configValues,
  selectedCredentialProfileId,
}: UseLoopSubmitParams) {
  const t = useTranslations();
  const createLoop = useLoopStore((s) => s.createLoop);
  const updateLoop = useLoopStore((s) => s.updateLoop);
  const isEdit = !!editLoop;

  const handleSubmit = useCallback(async () => {
    if (!form.name.trim() || !form.promptTemplate.trim() || !form.selectedAgentSlug) return;

    form.setLoading(true);
    try {
      const data = {
        name: form.name.trim(),
        description: form.description || undefined,
        agent_slug: form.selectedAgentSlug,
        prompt_template: form.promptTemplate,
        runner_id: form.selectedRunnerId || undefined,
        repository_id: form.selectedRepositoryId || undefined,
        branch_name: form.selectedBranch || undefined,
        credential_profile_id: selectedCredentialProfileId > 0 ? selectedCredentialProfileId : undefined,
        config_overrides: Object.keys(configValues).length > 0 ? configValues : undefined,
        execution_mode: form.executionMode,
        cron_expression: form.cronEnabled && form.cronExpression ? form.cronExpression : "",
        sandbox_strategy: form.sandboxStrategy,
        concurrency_policy: form.concurrencyPolicy,
        timeout_minutes: form.timeoutMinutes,
        callback_url: form.callbackUrl || undefined,
        session_persistence: form.sessionPersistence,
        max_concurrent_runs: form.maxConcurrentRuns,
        max_retained_runs: form.maxRetainedRuns,
      };

      if (isEdit && editLoop) {
        await updateLoop(editLoop.slug, data);
        toast.success(t("loops.updated"));
        onCreated?.();
      } else {
        const res = await createLoop(data);
        toast.success(t("loops.created"));
        onCreated?.(res.loop);
      }
    } catch (err) {
      toast.error(isEdit ? t("loops.updateFailed") : t("loops.createFailed"), {
        description: (err as Error).message,
      });
    } finally {
      form.setLoading(false);
    }
  }, [
    form, selectedCredentialProfileId, configValues,
    isEdit, editLoop, createLoop, updateLoop, onCreated, t,
  ]);

  return { handleSubmit, isEdit, t };
}
