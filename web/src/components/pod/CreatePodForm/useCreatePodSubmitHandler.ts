import { useCallback } from "react";
import { estimateWorkspaceTerminalSize } from "@/lib/terminal-size";
import type { CreatePodFormState } from "../hooks";

/**
 * Extracted submit handler for the CreatePodForm.
 * Builds terminal size estimation and delegates to form.submit().
 */
export function useCreatePodSubmitHandler(
  form: CreatePodFormState,
  selectedRunner: { id: number } | null,
  configValues: Record<string, unknown>,
  context?: { ticket?: { slug?: string } } | null,
  onError?: (err: Error) => void,
) {
  return useCallback(async () => {
    if (!form.selectedAgent) return;
    try {
      const { cols, rows } = estimateWorkspaceTerminalSize();
      await form.submit(selectedRunner?.id ?? null, configValues, {
        ticketSlug: context?.ticket?.slug,
        cols,
        rows,
      });
    } catch (err) {
      const error = err instanceof Error ? err : new Error("Unknown error");
      onError?.(error);
    }
  }, [form, selectedRunner, configValues, context, onError]);
}
