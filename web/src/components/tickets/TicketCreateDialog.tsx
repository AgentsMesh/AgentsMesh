"use client";

import { useState, useCallback, useEffect, useRef, lazy, Suspense } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { FormField } from "@/components/ui/form-field";
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
import { TicketPriority } from "@/lib/api/ticket";
import { ticketApi } from "@/lib/api";
import { organizationApi, OrganizationMember } from "@/lib/api/organization";
import { useAuthStore } from "@/stores/auth";
import { useBreakpoint } from "@/components/layout/useBreakpoint";
import { cn } from "@/lib/utils";
import { ChevronDown, Users, Check } from "lucide-react";

const BlockEditor = lazy(() => import("@/components/ui/block-editor"));

export interface TicketCreateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated?: (ticketId: number, slug: string) => void;
  parentTicketSlug?: string;
}

const priorityOptions: { value: TicketPriority }[] = [
  { value: "urgent" },
  { value: "high" },
  { value: "medium" },
  { value: "low" },
  { value: "none" },
];

interface FormData {
  title: string;
  content: string;
  priority: TicketPriority;
  assigneeIds: number[];
}

export function TicketCreateDialog({
  open,
  onOpenChange,
  onCreated,
  parentTicketSlug,
}: TicketCreateDialogProps) {
  const t = useTranslations();
  const { isMobile } = useBreakpoint();
  const { currentOrg } = useAuthStore();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [members, setMembers] = useState<OrganizationMember[]>([]);
  const [assigneeDropdownOpen, setAssigneeDropdownOpen] = useState(false);
  const assigneeDropdownRef = useRef<HTMLDivElement>(null);
  const [form, setForm] = useState<FormData>({
    title: "",
    content: "",
    priority: "medium",
    assigneeIds: [],
  });

  useEffect(() => {
    if (open && currentOrg?.slug && members.length === 0) {
      organizationApi.listMembers(currentOrg.slug)
        .then((res) => setMembers(res.members || []))
        .catch(() => {});
    }
  }, [open, currentOrg?.slug, members.length]);

  useEffect(() => {
    if (!assigneeDropdownOpen) return;
    const handleClickOutside = (e: MouseEvent) => {
      if (assigneeDropdownRef.current && !assigneeDropdownRef.current.contains(e.target as Node)) {
        setAssigneeDropdownOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [assigneeDropdownOpen]);

  const resetForm = useCallback(() => {
    setForm({
      title: "",
      content: "",
      priority: "medium",
      assigneeIds: [],
    });
    setError(null);
  }, []);

  const handleClose = useCallback(() => {
    onOpenChange(false);
    resetForm();
  }, [onOpenChange, resetForm]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!form.title.trim()) {
      setError(t("tickets.createDialog.titleRequired"));
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const response = await ticketApi.create({
        title: form.title.trim(),
        content: form.content || undefined,
        priority: form.priority,
        parentSlug: parentTicketSlug,
        assigneeIds: form.assigneeIds.length > 0 ? form.assigneeIds : undefined,
      });

      onCreated?.(response.id, response.slug);
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

  const toggleAssignee = (userId: number) => {
    setForm((prev) => ({
      ...prev,
      assigneeIds: prev.assigneeIds.includes(userId)
        ? prev.assigneeIds.filter((id) => id !== userId)
        : [...prev.assigneeIds, userId],
    }));
  };

  const dialogTitle = parentTicketSlug
    ? t("tickets.createDialog.createSubTicket")
    : t("tickets.createDialog.title");

  return (
    <ResponsiveDialog open={open} onOpenChange={onOpenChange}>
      <ResponsiveDialogContent className="max-w-lg">
        <ResponsiveDialogHeader onClose={() => onOpenChange(false)}>
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

            {/* Priority */}
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

            {/* Assignees */}
            <FormField label={t("tickets.detail.assignees")}>
              <div ref={assigneeDropdownRef} className="relative">
                <button
                  type="button"
                  onClick={() => setAssigneeDropdownOpen(!assigneeDropdownOpen)}
                  className={cn(
                    "flex items-center justify-between w-full h-9 rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-xs",
                    "focus:outline-none focus:ring-2 focus:ring-primary/20",
                    form.assigneeIds.length === 0 && "text-muted-foreground"
                  )}
                >
                  <span className="truncate">
                    {form.assigneeIds.length === 0
                      ? t("tickets.detail.noAssignees")
                      : members
                          .filter((m) => form.assigneeIds.includes(m.user_id))
                          .map((m) => m.user?.name || m.user?.username || m.user?.email)
                          .join(", ")}
                  </span>
                  <ChevronDown className={cn("h-4 w-4 shrink-0 opacity-50 transition-transform", assigneeDropdownOpen && "rotate-180")} />
                </button>
                {assigneeDropdownOpen && (
                  <div className="absolute z-50 mt-1 w-full rounded-md border bg-popover p-1 text-popover-foreground shadow-md max-h-48 overflow-y-auto">
                    {members.length > 0 ? (
                      members.map((member) => {
                        const isSelected = form.assigneeIds.includes(member.user_id);
                        return (
                          <button
                            key={member.user_id}
                            type="button"
                            className="flex items-center gap-2 w-full rounded-sm px-2 py-1.5 text-sm hover:bg-accent hover:text-accent-foreground transition-colors"
                            onClick={() => toggleAssignee(member.user_id)}
                          >
                            <span className={cn("flex h-4 w-4 items-center justify-center shrink-0", !isSelected && "opacity-0")}>
                              <Check className="h-3.5 w-3.5" />
                            </span>
                            {member.user?.avatar_url ? (
                              /* eslint-disable-next-line @next/next/no-img-element */
                              <img src={member.user.avatar_url} alt="" className="w-5 h-5 rounded-full shrink-0" />
                            ) : (
                              <Users className="w-4 h-4 text-muted-foreground shrink-0" />
                            )}
                            <span className="truncate">{member.user?.name || member.user?.username || member.user?.email}</span>
                          </button>
                        );
                      })
                    ) : (
                      <p className="px-2 py-1.5 text-sm text-muted-foreground">
                        {t("tickets.detail.noAssignees")}
                      </p>
                    )}
                  </div>
                )}
              </div>
            </FormField>

            {/* Content */}
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
