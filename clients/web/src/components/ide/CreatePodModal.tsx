"use client";

import React, { useMemo } from "react";
import type { PodData } from "@/lib/api/pod";
import { useTranslations } from "next-intl";
import { useFocusTrap } from "@/components/pod/hooks";
import { CreatePodForm, CreatePodFormConfig, TicketContext } from "@/components/pod/CreatePodForm";

interface CreatePodModalProps {
  open: boolean;
  onClose: () => void;
  onCreated: (pod?: PodData) => void;
  ticketContext?: TicketContext;
}

export function CreatePodModal({ open, onClose, onCreated, ticketContext }: CreatePodModalProps) {
  const t = useTranslations();

  const modalRef = useFocusTrap<HTMLDivElement>(open, onClose);

  const formConfig: CreatePodFormConfig = useMemo(() => ({
    scenario: ticketContext ? "ticket" : "workspace",
    context: ticketContext ? { ticket: ticketContext } : undefined,
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
