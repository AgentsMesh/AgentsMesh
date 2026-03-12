"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { formatDistanceToNow } from "date-fns";
import { FileText, Upload, Download, Loader2 } from "lucide-react";
import { runnerApi } from "@/lib/api";
import type { RunnerLogData } from "@/lib/api";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

interface RunnerLogsCardProps {
  runnerId: number;
  runnerStatus: string;
}

const ACTIVE_STATUSES = ["pending", "collecting", "uploading"];
const POLL_INTERVAL = 5000;

/**
 * Diagnostic logs card showing log upload history with polling for active uploads
 */
export function RunnerLogsCard({ runnerId, runnerStatus }: RunnerLogsCardProps) {
  const t = useTranslations();
  const [logs, setLogs] = useState<RunnerLogData[]>([]);
  const [uploading, setUploading] = useState(false);
  const mountedRef = useRef(true);

  const loadLogs = useCallback(async () => {
    try {
      const res = await runnerApi.listLogs(runnerId);
      if (mountedRef.current) {
        setLogs(res.logs || []);
      }
    } catch {
      // Silently ignore polling errors
    }
  }, [runnerId]);

  const hasActiveLogs = logs.some((log) => ACTIVE_STATUSES.includes(log.status));

  // Track mount state for safe setState
  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  // Initial load
  useEffect(() => {
    loadLogs();
  }, [loadLogs]);

  // Poll when any log is in an active state
  useEffect(() => {
    if (!hasActiveLogs) return;
    const id = setInterval(loadLogs, POLL_INTERVAL);
    return () => clearInterval(id);
  }, [hasActiveLogs, loadLogs]);

  const handleUpload = async () => {
    setUploading(true);
    try {
      await runnerApi.requestLogUpload(runnerId);
      toast.success(t("runners.logs.uploadSuccess"));
      await loadLogs();
    } catch {
      toast.error(t("runners.logs.uploadFailed"));
    } finally {
      if (mountedRef.current) setUploading(false);
    }
  };

  const isOnline = runnerStatus === "online";

  return (
    <div className="bg-card rounded-lg border border-border p-6 md:col-span-2">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-medium text-foreground flex items-center">
          <FileText className="w-5 h-5 mr-2 text-muted-foreground" />
          {t("runners.logs.title")}
        </h3>
        <button
          onClick={handleUpload}
          disabled={uploading || !isOnline}
          title={!isOnline ? t("runners.logs.offlineHint") : undefined}
          className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          {uploading ? (
            <Loader2 className="w-4 h-4 animate-spin" />
          ) : (
            <Upload className="w-4 h-4" />
          )}
          {uploading ? t("runners.logs.uploading") : t("runners.logs.upload")}
        </button>
      </div>

      {logs.length === 0 ? (
        <p className="text-sm text-muted-foreground">{t("runners.logs.noLogs")}</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-border">
            <thead>
              <tr>
                <th scope="col" className="px-4 py-2 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  {t("runners.detail.status")}
                </th>
                <th scope="col" className="px-4 py-2 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  {t("runners.logs.size")}
                </th>
                <th scope="col" className="px-4 py-2 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  {t("runners.logs.requestedAt")}
                </th>
                <th scope="col" className="px-4 py-2 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  {t("runners.detail.actions")}
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {logs.map((log) => (
                <tr key={log.id}>
                  <td className="px-4 py-3">
                    <LogStatusBadge status={log.status} />
                  </td>
                  <td className="px-4 py-3 text-sm text-muted-foreground">
                    {log.size_bytes > 0 ? formatBytes(log.size_bytes) : "-"}
                  </td>
                  <td className="px-4 py-3 text-sm text-muted-foreground">
                    {formatDistanceToNow(new Date(log.created_at), { addSuffix: true })}
                  </td>
                  <td className="px-4 py-3">
                    {log.status === "completed" && log.download_url ? (
                      <DownloadLink url={log.download_url} label={t("runners.logs.download")} />
                    ) : log.status === "failed" && log.error_message ? (
                      <span className="text-sm text-destructive truncate max-w-[200px] inline-block" title={log.error_message}>
                        {log.error_message}
                      </span>
                    ) : (
                      <span className="text-sm text-muted-foreground">-</span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

/** M10: validate URL protocol before rendering download link */
function isValidDownloadUrl(url: string): boolean {
  try {
    const parsed = new URL(url);
    return parsed.protocol === "http:" || parsed.protocol === "https:";
  } catch {
    return false;
  }
}

function DownloadLink({ url, label }: { url: string; label: string }) {
  if (!isValidDownloadUrl(url)) {
    return <span className="text-sm text-muted-foreground">-</span>;
  }

  return (
    <a
      href={url}
      target="_blank"
      rel="noopener noreferrer"
      className="inline-flex items-center gap-1 text-sm text-primary hover:underline"
    >
      <Download className="w-3.5 h-3.5" />
      {label}
    </a>
  );
}

function LogStatusBadge({ status }: { status: string }) {
  const t = useTranslations();

  const styles: Record<string, string> = {
    pending: "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400",
    collecting: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400",
    uploading: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400",
    completed: "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400",
    failed: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
  };

  const key = `runners.logs.status.${status}` as Parameters<typeof t>[0];
  const label = t(key);

  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${styles[status] || "bg-muted text-muted-foreground"}`}>
      {ACTIVE_STATUSES.includes(status) && (
        <Loader2 className="w-3 h-3 mr-1 animate-spin" />
      )}
      {label}
    </span>
  );
}

function formatBytes(bytes: number): string {
  if (bytes <= 0) return "0 B";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
  return `${(bytes / 1024 / 1024 / 1024).toFixed(1)} GB`;
}
