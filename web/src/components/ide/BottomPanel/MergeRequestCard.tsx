import { cn } from "@/lib/utils";
import {
  GitPullRequest,
  GitMerge,
  XCircle,
  GitBranch,
  ArrowRight,
  ExternalLink,
  Loader2,
} from "lucide-react";

export interface MergeRequestInfo {
  id: number;
  mr_iid: number;
  title: string;
  state: "opened" | "merged" | "closed" | string;
  mr_url: string;
  source_branch: string;
  target_branch: string;
  pipeline_status?: string;
  pipeline_url?: string;
}

export function MergeRequestCard({
  mr, providerType, t,
}: {
  mr: MergeRequestInfo;
  providerType?: string;
  t: (key: string) => string;
}) {
  return (
    <a href={mr.mr_url} target="_blank" rel="noopener noreferrer"
      className="block px-3 py-2 rounded bg-muted/50 hover:bg-muted transition-colors">
      <div className="flex items-center gap-2">
        <MRStateIcon state={mr.state} />
        <span className="text-xs font-medium flex-1 truncate" title={mr.title}>
          {mr.title || t("ide.bottomPanel.deliveryTab.untitled")}
        </span>
        <span className="text-xs text-muted-foreground">
          {providerType === "github" ? `#${mr.mr_iid}` : `!${mr.mr_iid}`}
        </span>
        <ExternalLink className="w-3 h-3 text-muted-foreground flex-shrink-0" />
      </div>

      <div className="flex items-center gap-1 mt-1.5 text-[10px] text-muted-foreground">
        <GitBranch className="w-3 h-3 flex-shrink-0" />
        <span className="font-mono truncate">{mr.source_branch}</span>
        <ArrowRight className="w-3 h-3 flex-shrink-0" />
        <span className="font-mono truncate">{mr.target_branch}</span>
      </div>

      {mr.pipeline_status && (
        <div className="flex items-center gap-1 mt-1.5">
          <PipelineStatusBadge status={mr.pipeline_status} url={mr.pipeline_url} />
        </div>
      )}
    </a>
  );
}

function MRStateIcon({ state }: { state: string }) {
  switch (state) {
    case "opened":
      return <GitPullRequest className="w-3.5 h-3.5 text-green-500 flex-shrink-0" />;
    case "merged":
      return <GitMerge className="w-3.5 h-3.5 text-purple-500 flex-shrink-0" />;
    case "closed":
      return <XCircle className="w-3.5 h-3.5 text-red-500 flex-shrink-0" />;
    default:
      return <GitPullRequest className="w-3.5 h-3.5 text-muted-foreground flex-shrink-0" />;
  }
}

function PipelineStatusBadge({ status, url }: { status: string; url?: string }) {
  const styleMap: Record<string, string> = {
    success: "bg-green-500/10 text-green-500",
    failed: "bg-red-500/10 text-red-500",
    running: "bg-blue-500/10 text-blue-500",
    pending: "bg-yellow-500/10 text-yellow-500",
    canceled: "bg-gray-500/10 text-gray-500",
  };

  const content = (
    <span className={cn("px-1.5 py-0.5 rounded text-[10px] flex items-center gap-1",
      styleMap[status] || "bg-muted text-muted-foreground")}>
      {status === "running" && <Loader2 className="w-2.5 h-2.5 animate-spin" />}
      <span>Pipeline: {status}</span>
    </span>
  );

  return url ? (
    <a href={url} target="_blank" rel="noopener noreferrer"
      onClick={(e) => e.stopPropagation()} className="hover:opacity-80">{content}</a>
  ) : content;
}
