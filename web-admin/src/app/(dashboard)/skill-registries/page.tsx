"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Plus,
  RefreshCw,
  Trash2,
  GitBranch,
  Loader2,
} from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  listSkillRegistries,
  createSkillRegistry,
  syncSkillRegistry,
  deleteSkillRegistry,
  SkillRegistry,
} from "@/lib/api/admin";
import { formatDate, formatRelativeTime } from "@/lib/utils";

function SyncStatusBadge({ status }: { status: string }) {
  switch (status) {
    case "success":
      return <Badge variant="success">Success</Badge>;
    case "syncing":
      return (
        <Badge variant="warning" className="gap-1">
          <Loader2 className="h-3 w-3 animate-spin" />
          Syncing
        </Badge>
      );
    case "failed":
      return <Badge variant="destructive">Failed</Badge>;
    case "pending":
      return <Badge variant="secondary">Pending</Badge>;
    default:
      return <Badge variant="secondary">{status}</Badge>;
  }
}

export default function SkillRegistriesPage() {
  const [dialogOpen, setDialogOpen] = useState(false);
  const [formUrl, setFormUrl] = useState("");
  const [formBranch, setFormBranch] = useState("");
  const [formSourceType, setFormSourceType] = useState("");
  const [syncingIds, setSyncingIds] = useState<Set<number>>(new Set());
  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ["skill-registries"],
    queryFn: listSkillRegistries,
  });

  const createMutation = useMutation({
    mutationFn: createSkillRegistry,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["skill-registries"] });
      toast.success("Skill registry added successfully");
      setDialogOpen(false);
      resetForm();
    },
    onError: (err: { error: string }) => {
      toast.error(err.error || "Failed to create skill registry");
    },
  });

  const syncMutation = useMutation({
    mutationFn: syncSkillRegistry,
    onSuccess: (_data, id) => {
      queryClient.invalidateQueries({ queryKey: ["skill-registries"] });
      setSyncingIds((prev) => {
        const next = new Set(prev);
        next.delete(id);
        return next;
      });
      toast.success("Sync triggered successfully");
    },
    onError: (err: { error: string }, id: number) => {
      setSyncingIds((prev) => {
        const next = new Set(prev);
        next.delete(id);
        return next;
      });
      toast.error(err.error || "Failed to sync skill registry");
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteSkillRegistry,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["skill-registries"] });
      toast.success("Skill registry deleted successfully");
    },
    onError: (err: { error: string }) => {
      toast.error(err.error || "Failed to delete skill registry");
    },
  });

  const resetForm = () => {
    setFormUrl("");
    setFormBranch("");
    setFormSourceType("");
  };

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault();
    if (!formUrl.trim()) {
      toast.error("Repository URL is required");
      return;
    }
    createMutation.mutate({
      repository_url: formUrl.trim(),
      branch: formBranch.trim() || undefined,
      source_type: formSourceType.trim() || undefined,
    });
  };

  const handleSync = (id: number) => {
    setSyncingIds((prev) => new Set(prev).add(id));
    syncMutation.mutate(id);
  };

  const handleDelete = (registry: SkillRegistry) => {
    if (
      confirm(
        `Are you sure you want to delete the skill registry "${registry.repository_url}"? This action cannot be undone.`
      )
    ) {
      deleteMutation.mutate(registry.id);
    }
  };

  const skillRegistries = data?.items || [];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Skill Registries</h1>
          <p className="text-sm text-muted-foreground">
            Manage platform-level skill registry repositories. Skills synced from
            these repos are available to all organizations.
          </p>
        </div>
        <Dialog open={dialogOpen} onOpenChange={(open) => {
          setDialogOpen(open);
          if (!open) resetForm();
        }}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="mr-2 h-4 w-4" />
              Add Registry
            </Button>
          </DialogTrigger>
          <DialogContent>
            <form onSubmit={handleCreate}>
              <DialogHeader>
                <DialogTitle>Add Skill Registry</DialogTitle>
                <DialogDescription>
                  Add a new skill registry repository. Skills will be synced
                  automatically after creation.
                </DialogDescription>
              </DialogHeader>
              <div className="grid gap-4 py-4">
                <div className="grid gap-2">
                  <Label htmlFor="repository_url">Repository URL</Label>
                  <Input
                    id="repository_url"
                    placeholder="https://github.com/org/repo"
                    value={formUrl}
                    onChange={(e) => setFormUrl(e.target.value)}
                    required
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="branch">Branch</Label>
                  <Input
                    id="branch"
                    placeholder="main"
                    value={formBranch}
                    onChange={(e) => setFormBranch(e.target.value)}
                  />
                  <p className="text-xs text-muted-foreground">
                    Leave empty to use the default branch.
                  </p>
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="source_type">Source Type</Label>
                  <Input
                    id="source_type"
                    placeholder="auto-detect"
                    value={formSourceType}
                    onChange={(e) => setFormSourceType(e.target.value)}
                  />
                  <p className="text-xs text-muted-foreground">
                    Leave empty for auto-detection.
                  </p>
                </div>
              </div>
              <DialogFooter>
                <Button
                  type="submit"
                  disabled={createMutation.isPending}
                >
                  {createMutation.isPending && (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  )}
                  Add Registry
                </Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      {/* Table */}
      <div className="rounded-lg border border-border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Repository URL</TableHead>
              <TableHead>Branch</TableHead>
              <TableHead>Type</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Skill Count</TableHead>
              <TableHead>Last Synced</TableHead>
              <TableHead className="w-28">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              Array.from({ length: 3 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell colSpan={7}>
                    <div className="h-12 animate-pulse rounded bg-muted" />
                  </TableCell>
                </TableRow>
              ))
            ) : skillRegistries.length === 0 ? (
              <TableRow>
                <TableCell
                  colSpan={7}
                  className="py-8 text-center text-muted-foreground"
                >
                  No skill registries configured
                </TableCell>
              </TableRow>
            ) : (
              skillRegistries.map((registry) => (
                <TableRow key={registry.id}>
                  <TableCell>
                    <div className="flex items-center gap-2 font-medium">
                      <GitBranch className="h-4 w-4 shrink-0 text-muted-foreground" />
                      <span className="truncate max-w-xs" title={registry.repository_url}>
                        {registry.repository_url}
                      </span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <code className="rounded bg-muted px-1.5 py-0.5 text-xs">
                      {registry.branch || "main"}
                    </code>
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline">{registry.source_type}</Badge>
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-col gap-1">
                      <SyncStatusBadge status={registry.sync_status} />
                      {registry.sync_status === "failed" && registry.sync_error && (
                        <span
                          className="text-xs text-destructive truncate max-w-[200px]"
                          title={registry.sync_error}
                        >
                          {registry.sync_error}
                        </span>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    <span className="font-medium">{registry.skill_count}</span>
                  </TableCell>
                  <TableCell>
                    {registry.last_synced_at ? (
                      <span
                        className="text-sm text-muted-foreground"
                        title={formatDate(registry.last_synced_at)}
                      >
                        {formatRelativeTime(registry.last_synced_at)}
                      </span>
                    ) : (
                      <span className="text-sm text-muted-foreground">Never</span>
                    )}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => handleSync(registry.id)}
                        disabled={
                          syncingIds.has(registry.id) ||
                          registry.sync_status === "syncing"
                        }
                        title="Sync now"
                      >
                        <RefreshCw
                          className={`h-4 w-4 ${
                            syncingIds.has(registry.id) ||
                            registry.sync_status === "syncing"
                              ? "animate-spin"
                              : ""
                          }`}
                        />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => handleDelete(registry)}
                        disabled={deleteMutation.isPending}
                        title="Delete skill registry"
                        className="text-destructive hover:text-destructive"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
