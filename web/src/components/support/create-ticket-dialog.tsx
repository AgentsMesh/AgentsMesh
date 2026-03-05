"use client";

import { useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Paperclip, X, Upload } from "lucide-react";
import { toast } from "sonner";
import { createSupportTicket } from "@/lib/api/support-ticket";

interface CreateTicketDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated?: () => void;
}

const categories = ["bug", "feature_request", "usage_question", "account", "other"];

const priorities = ["low", "medium", "high"];

const MAX_FILE_SIZE = 10 * 1024 * 1024; // 10MB, matches backend default

export function CreateTicketDialog({
  open,
  onOpenChange,
  onCreated,
}: CreateTicketDialogProps) {
  const router = useRouter();
  const t = useTranslations();
  const [title, setTitle] = useState("");
  const [category, setCategory] = useState("other");
  const [priority, setPriority] = useState("medium");
  const [content, setContent] = useState("");
  const [files, setFiles] = useState<File[]>([]);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState("");

  const handleSubmit = useCallback(async () => {
    if (!title.trim() || !content.trim()) {
      setError(t("support.validation.required"));
      return;
    }
    setError("");
    setIsSubmitting(true);
    try {
      const ticket = await createSupportTicket({
        title: title.trim(),
        category,
        content: content.trim(),
        priority,
        files: files.length > 0 ? files : undefined,
      });
      setTitle("");
      setCategory("other");
      setPriority("medium");
      setContent("");
      setFiles([]);
      onOpenChange(false);
      onCreated?.();
      router.push(`/support/${ticket.id}`);
    } catch {
      setError(t("support.error.createFailed"));
    } finally {
      setIsSubmitting(false);
    }
  }, [title, category, priority, content, files, onOpenChange, onCreated, router, t]);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files) {
      const newFiles = Array.from(e.target.files);
      const oversized = newFiles.filter((f) => f.size > MAX_FILE_SIZE);
      if (oversized.length > 0) {
        toast.error(t("support.fileTooLarge", { max: "10MB" }));
        return;
      }
      setFiles((prev) => [...prev, ...newFiles]);
    }
  };

  const removeFile = (index: number) => {
    setFiles((prev) => prev.filter((_, i) => i !== index));
  };

  const reset = () => {
    setTitle("");
    setCategory("other");
    setPriority("medium");
    setContent("");
    setFiles([]);
    setError("");
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(v) => {
        if (!v) reset();
        onOpenChange(v);
      }}
    >
      <DialogContent
        className="max-w-lg"
        title={t("support.create")}
        description={t("support.createDescription")}
      >
        <div className="px-6 py-4 space-y-4">
          {/* Title */}
          <div>
            <label className="text-sm font-medium mb-1 block">
              {t("support.fields.title")}
            </label>
            <Input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder={t("support.fields.titlePlaceholder")}
              maxLength={255}
            />
          </div>

          {/* Category + Priority */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-sm font-medium mb-1 block">
                {t("support.fields.category")}
              </label>
              <select
                value={category}
                onChange={(e) => setCategory(e.target.value)}
                className="w-full h-9 rounded-md border border-input bg-background px-3 text-sm"
              >
                {categories.map((value) => (
                  <option key={value} value={value}>
                    {t(`support.category.${value}`)}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="text-sm font-medium mb-1 block">
                {t("support.fields.priority")}
              </label>
              <select
                value={priority}
                onChange={(e) => setPriority(e.target.value)}
                className="w-full h-9 rounded-md border border-input bg-background px-3 text-sm"
              >
                {priorities.map((value) => (
                  <option key={value} value={value}>
                    {t(`support.priority.${value}`)}
                  </option>
                ))}
              </select>
            </div>
          </div>

          {/* Content */}
          <div>
            <label className="text-sm font-medium mb-1 block">
              {t("support.fields.content")}
            </label>
            <Textarea
              value={content}
              onChange={(e) => setContent(e.target.value)}
              placeholder={t("support.fields.contentPlaceholder")}
              rows={5}
            />
          </div>

          {/* File Upload */}
          <div>
            <label className="text-sm font-medium mb-1 block">
              {t("support.fields.attachments")}
            </label>
            <div className="space-y-2">
              {files.length > 0 && (
                <div className="flex flex-wrap gap-2">
                  {files.map((file, i) => (
                    <div
                      key={i}
                      className="flex items-center gap-1 rounded-md bg-muted px-2 py-1 text-xs"
                    >
                      <Paperclip className="h-3 w-3" />
                      <span className="max-w-[150px] truncate">{file.name}</span>
                      <button
                        onClick={() => removeFile(i)}
                        className="ml-1 text-muted-foreground hover:text-foreground"
                      >
                        <X className="h-3 w-3" />
                      </button>
                    </div>
                  ))}
                </div>
              )}
              <label className="flex cursor-pointer items-center gap-2 rounded-md border border-dashed border-input px-3 py-2 text-sm text-muted-foreground hover:bg-muted/50 transition-colors">
                <Upload className="h-4 w-4" />
                {t("support.fields.addFiles")}
                <input
                  type="file"
                  multiple
                  accept="image/*,.pdf,.txt,.log"
                  onChange={handleFileChange}
                  className="hidden"
                />
              </label>
            </div>
          </div>

          {error && (
            <p className="text-sm text-destructive">{error}</p>
          )}
        </div>

        {/* Footer */}
        <div className="flex justify-end gap-2 border-t px-6 py-3">
          <Button
            variant="outline"
            onClick={() => {
              reset();
              onOpenChange(false);
            }}
            disabled={isSubmitting}
          >
            {t("support.cancel")}
          </Button>
          <Button onClick={handleSubmit} disabled={isSubmitting}>
            {isSubmitting ? t("support.submitting") : t("support.submit")}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
