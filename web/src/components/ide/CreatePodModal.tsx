"use client";

import React, { useMemo } from "react";
import { PodData } from "@/lib/api";
import { useTranslations } from "next-intl";
import { useFocusTrap } from "@/components/pod/hooks";
import { CreatePodForm, CreatePodFormConfig, TicketContext } from "@/components/pod/CreatePodForm";

interface CreatePodModalProps {
  open: boolean;
  onClose: () => void;
  onCreated: (pod?: PodData) => void;
  /** Optional ticket context for creating pod from ticket */
  ticketContext?: TicketContext;
}

/**
 * Modal wrapper for CreatePodForm
 *
 * This component provides the modal container and delegates all form logic
 * to the shared CreatePodForm component.
 */
export function CreatePodModal({ open, onClose, onCreated, ticketContext }: CreatePodModalProps) {
  const t = useTranslations();

  // Focus trap for modal accessibility
  const modalRef = useFocusTrap<HTMLDivElement>(open, onClose);

  // Build form configuration based on ticket context
  const formConfig: CreatePodFormConfig = useMemo(() => ({
    scenario: ticketContext ? "ticket" : "workspace",
    context: ticketContext ? { ticket: ticketContext } : undefined,
    promptGenerator: ticketContext
      ? (ctx) => {
          if (!ctx.ticket) return "";
          return `Work on ticket ${ctx.ticket.identifier}: ${ctx.ticket.title}${
            ctx.ticket.description ? `\n\n${ctx.ticket.description}` : ""
          }`;
        }
      : undefined,
    onSuccess: (pod) => {
      onCreated(pod);
      onClose();
    },
    onCancel: onClose,
  }), [ticketContext, onCreated, onClose]);

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="create-pod-title"
    >
      <div
        ref={modalRef}
        className="bg-background border border-border rounded-lg w-full max-w-md p-4 md:p-6 max-h-[90vh] overflow-y-auto"
      >
        <h2 id="create-pod-title" className="text-lg md:text-xl font-semibold mb-4">
          {t("ide.createPod.title")}
        </h2>

        <CreatePodForm
          config={formConfig}
          enabled={open}
        />
      </div>
    </div>
  );
}

export default CreatePodModal;

// Re-export TicketContext type for backward compatibility
export type { TicketContext };
