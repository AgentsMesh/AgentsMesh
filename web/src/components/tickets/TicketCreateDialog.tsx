"use client";

import { useState, useCallback, lazy, Suspense } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { FormField, FormRow } from "@/components/ui/form-field";
import {
  ResponsiveDialog,
  ResponsiveDialogContent,
  ResponsiveDialogHeader,
  ResponsiveDialogTitle,
  ResponsiveDialogBody,
  ResponsiveDialogFooter,
} from "@/components/ui/responsive-dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { TicketType, TicketPriority } from "@/lib/api/ticket";
import { ticketApi } from "@/lib/api";
import { RepositorySelect } from "@/components/common/RepositorySelect";
import { useBreakpoint } from "@/components/layout/useBreakpoint";
import { cn } from "@/lib/utils";

// Lazy load BlockEditor to avoid SSR issues
const BlockEditor = lazy(() => import("@/components/ui/block-editor"));

export interface TicketCreateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated?: (ticketId: number, identifier: string) => void;
  defaultRepositoryId?: number;
  parentTicketId?: number;
}

const typeOptions: { value: TicketType; label: string }[] = [
  { value: "task", label: "Task" },
  { value: "bug", label: "Bug" },
  { value: "feature", label: "Feature" },
  { value: "improvement", label: "Improvement" },
  { value: "epic", label: "Epic" },
];

const priorityOptions: { value: TicketPriority; label: string }[] = [
  { value: "urgent", label: "Urgent" },
  { value: "high", label: "High" },
  { value: "medium", label: "Medium" },
  { value: "low", label: "Low" },
  { value: "none", label: "None" },
];

interface FormData {
  title: string;
  description: string;
  content: string;
  type: TicketType;
  priority: TicketPriority;
  repositoryId: number | null;
}

export function TicketCreateDialog({
  open,
  onOpenChange,
  onCreated,
  defaultRepositoryId,
  parentTicketId,
}: TicketCreateDialogProps) {
  const t = useTranslations();
  const { isMobile } = useBreakpoint();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [form, setForm] = useState<FormData>({
    title: "",
    description: "",
    content: "",
    type: "task",
    priority: "medium",
    repositoryId: defaultRepositoryId || null,
  });

  const resetForm = useCallback(() => {
    setForm({
      title: "",
      description: "",
      content: "",
      type: "task",
      priority: "medium",
      repositoryId: defaultRepositoryId || null,
    });
    setError(null);
  }, [defaultRepositoryId]);

  const handleClose = useCallback(() => {
    onOpenChange(false);
    resetForm();
  }, [onOpenChange, resetForm]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    // Validation
    if (!form.title.trim()) {
      setError(t("tickets.createDialog.titleRequired"));
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const response = await ticketApi.create({
        repositoryId: form.repositoryId || undefined,
        title: form.title.trim(),
        description: form.description.trim() || undefined,
        content: form.content || undefined,
        type: form.type,
        priority: form.priority,
        parentId: parentTicketId,
      });

      onCreated?.(response.id, response.identifier);
      handleClose();
    } catch (err: unknown) {
      console.error("Failed to create ticket:", err);
      setError(err instanceof Error ? err.message : t("tickets.createDialog.createFailed"));
    } finally {
      setLoading(false);
    }
  };

  const updateField = <K extends keyof FormData>(key: K, value: FormData[K]) => {
    setForm((prev) => ({ ...prev, [key]: value }));
    if (error) setError(null);
  };

  const dialogTitle = parentTicketId
    ? t("tickets.createDialog.createSubTicket")
    : t("tickets.createDialog.title");

  return (
    <ResponsiveDialog open={open} onOpenChange={onOpenChange}>
      <ResponsiveDialogContent className="max-w-lg" title={dialogTitle}>
        <ResponsiveDialogHeader>
          <ResponsiveDialogTitle>{dialogTitle}</ResponsiveDialogTitle>
        </ResponsiveDialogHeader>

        <form onSubmit={handleSubmit} className="flex flex-col flex-1 min-h-0">
          <ResponsiveDialogBody className="space-y-4">
            {/* Title */}
            <FormField
              label={t("tickets.createDialog.titleLabel")}
              htmlFor="ticket-title"
              required
            >
              <Input
                id="ticket-title"
                placeholder={t("tickets.createDialog.titlePlaceholder")}
                value={form.title}
                onChange={(e) => updateField("title", e.target.value)}
                autoFocus
              />
            </FormField>

            {/* Type & Priority */}
            <FormRow>
              <FormField label={t("tickets.filters.type")} htmlFor="ticket-type">
                <Select
                  value={form.type}
                  onValueChange={(val) => updateField("type", val as TicketType)}
                >
                  <SelectTrigger id="ticket-type">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {typeOptions.map((opt) => (
                      <SelectItem key={opt.value} value={opt.value}>
                        {t(`tickets.type.${opt.value}`)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>

              <FormField label={t("tickets.filters.priority")} htmlFor="ticket-priority">
                <Select
                  value={form.priority}
                  onValueChange={(val) => updateField("priority", val as TicketPriority)}
                >
                  <SelectTrigger id="ticket-priority">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {priorityOptions.map((opt) => (
                      <SelectItem key={opt.value} value={opt.value}>
                        {t(`tickets.priority.${opt.value}`)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
            </FormRow>

            {/* Repository */}
            <FormField label={t("tickets.createDialog.repository")} htmlFor="ticket-repo">
              <RepositorySelect
                value={form.repositoryId}
                onChange={(value) => updateField("repositoryId", value)}
                placeholder={t("tickets.createDialog.selectRepository")}
              />
            </FormField>

            {/* Summary */}
            <FormField label={t("tickets.createDialog.summary")} htmlFor="ticket-summary">
              <textarea
                id="ticket-summary"
                className="w-full min-h-[60px] px-3 py-2 text-sm rounded-md border border-input bg-transparent shadow-sm placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring resize-none"
                placeholder={t("tickets.createDialog.summaryPlaceholder")}
                value={form.description}
                onChange={(e) => updateField("description", e.target.value)}
                rows={2}
              />
            </FormField>

            {/* Content - Rich Text Editor */}
            <FormField label={t("tickets.createDialog.content")}>
              <div className={cn(
                "border border-input rounded-md overflow-hidden bg-card",
                isMobile ? "min-h-[100px]" : "min-h-[150px]"
              )}>
                <Suspense fallback={<div className={cn("animate-pulse bg-muted", isMobile ? "h-[100px]" : "h-[150px]")} />}>
                  <BlockEditor
                    initialContent={form.content}
                    onChange={(content) => updateField("content", content)}
                    editable={true}
                  />
                </Suspense>
              </div>
            </FormField>

            {/* Error Message */}
            {error && (
              <div className="text-sm text-destructive bg-destructive/10 px-3 py-2 rounded-md">
                {error}
              </div>
            )}
          </ResponsiveDialogBody>

          <ResponsiveDialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={handleClose}
              disabled={loading}
              className="w-full sm:w-auto"
            >
              {t("common.cancel")}
            </Button>
            <Button type="submit" loading={loading} className="w-full sm:w-auto">
              {t("tickets.createDialog.submit")}
            </Button>
          </ResponsiveDialogFooter>
        </form>
      </ResponsiveDialogContent>
    </ResponsiveDialog>
  );
}

export default TicketCreateDialog;
