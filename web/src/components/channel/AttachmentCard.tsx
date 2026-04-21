"use client";

import { useState } from "react";
import { File, Download } from "lucide-react";
import { useTranslations } from "next-intl";
import { cn } from "@/lib/utils";

interface AttachmentCardProps {
  url: string;
  className?: string;
}

const IMAGE_EXTS = ["jpg", "jpeg", "png", "gif", "webp", "svg", "bmp", "avif"];

function extOf(url: string): string {
  try {
    const clean = url.split("?")[0].split("#")[0];
    const dot = clean.lastIndexOf(".");
    if (dot < 0) return "";
    return clean.slice(dot + 1).toLowerCase();
  } catch {
    return "";
  }
}

function fileNameOf(url: string): string {
  try {
    const parsed = new URL(url, "http://x");
    const path = parsed.pathname;
    const segments = path.split("/").filter(Boolean);
    return segments[segments.length - 1] || url;
  } catch {
    return url;
  }
}

/**
 * Inline attachment rendered below a message body. Images show a capped
 * thumbnail (click → open in new tab); other files show a download link.
 */
export function AttachmentCard({ url, className }: AttachmentCardProps) {
  const t = useTranslations("channels.attachment");
  const ext = extOf(url);
  const isImage = IMAGE_EXTS.includes(ext);
  const [errored, setErrored] = useState(false);

  if (isImage && !errored) {
    return (
      <a
        href={url}
        target="_blank"
        rel="noreferrer"
        data-testid="message-attachment"
        className={cn(
          "mt-1.5 inline-block max-w-[320px] overflow-hidden rounded-md border border-border",
          className,
        )}
        aria-label={t("preview")}
      >
        <img
          src={url}
          alt=""
          className="block max-h-[240px] w-full object-cover"
          onError={() => setErrored(true)}
        />
      </a>
    );
  }

  const name = fileNameOf(url);
  return (
    <a
      href={url}
      target="_blank"
      rel="noreferrer"
      download
      data-testid="message-attachment"
      className={cn(
        "mt-1.5 inline-flex items-center gap-1.5 rounded-md border border-border bg-muted/40 px-2 py-1 text-xs text-foreground hover:bg-muted",
        className,
      )}
      aria-label={t("download")}
    >
      <File className="h-3.5 w-3.5 text-muted-foreground" />
      <span className="max-w-[220px] truncate">{name}</span>
      <Download className="h-3 w-3 text-muted-foreground" />
    </a>
  );
}

export default AttachmentCard;
