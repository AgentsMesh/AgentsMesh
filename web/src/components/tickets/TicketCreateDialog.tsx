"use client";

import { useState, useCallback, lazy, Suspense } from "react";
import { useTranslations } from "@/lib/i18n/client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogBody,
  DialogFooter,
} from "@/components/ui/dialog";
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

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>
            {parentTicketId ? t("tickets.createDialog.createSubTicket") : t("tickets.createDialog.title")}
          </DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit}>
          <DialogBody className="space-y-4">
            {/* Title */}
            <div className="space-y-2">
              <label className="text-sm font-medium">
                {t("tickets.createDialog.titleLabel")} <span className="text-destructive">*</span>
              </label>
              <Input
                placeholder={t("tickets.createDialog.titlePlaceholder")}
                value={form.title}
                onChange={(e) => updateField("title", e.target.value)}
                autoFocus
              />
            </div>

            {/* Type & Priority */}
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">{t("tickets.filters.type")}</label>
                <Select
                  value={form.type}
                  onValueChange={(val) => updateField("type", val as TicketType)}
                >
                  <SelectTrigger>
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
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium">{t("tickets.filters.priority")}</label>
                <Select
                  value={form.priority}
                  onValueChange={(val) => updateField("priority", val as TicketPriority)}
                >
                  <SelectTrigger>
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
              </div>
            </div>

            {/* Repository */}
            <div className="space-y-2">
              <label className="text-sm font-medium">
                {t("tickets.createDialog.repository")}
              </label>
              <RepositorySelect
                value={form.repositoryId}
                onChange={(value) => updateField("repositoryId", value)}
                placeholder={t("tickets.createDialog.selectRepository")}
              />
            </div>

            {/* Summary */}
            <div className="space-y-2">
              <label className="text-sm font-medium">{t("tickets.createDialog.summary")}</label>
              <textarea
                className="w-full min-h-[60px] px-3 py-2 text-sm rounded-md border border-input bg-transparent shadow-sm placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring resize-none"
                placeholder={t("tickets.createDialog.summaryPlaceholder")}
                value={form.description}
                onChange={(e) => updateField("description", e.target.value)}
                rows={2}
              />
            </div>

            {/* Content - Rich Text Editor */}
            <div className="space-y-2">
              <label className="text-sm font-medium">{t("tickets.createDialog.content")}</label>
              <div className="border border-input rounded-md overflow-hidden min-h-[150px] bg-card">
                <Suspense fallback={<div className="h-[150px] animate-pulse bg-muted" />}>
                  <BlockEditor
                    initialContent={form.content}
                    onChange={(content) => updateField("content", content)}
                    editable={true}
                  />
                </Suspense>
              </div>
            </div>

            {/* Error Message */}
            {error && (
              <div className="text-sm text-destructive bg-destructive/10 px-3 py-2 rounded-md">
                {error}
              </div>
            )}
          </DialogBody>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose} disabled={loading}>
              {t("common.cancel")}
            </Button>
            <Button type="submit" loading={loading}>
              {t("tickets.createDialog.submit")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

export default TicketCreateDialog;
