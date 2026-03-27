"use client";

import { useState, useEffect, useCallback } from "react";
import { Plus, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
import { SkillRegistriesTable } from "./skill-registries-table";

export default function SkillRegistriesPage() {
  const [dialogOpen, setDialogOpen] = useState(false);
  const [formUrl, setFormUrl] = useState("");
  const [formBranch, setFormBranch] = useState("");
  const [formSourceType, setFormSourceType] = useState("");
  const [syncingIds, setSyncingIds] = useState<Set<number>>(new Set());
  const [isCreating, setIsCreating] = useState(false);

  const [data, setData] = useState<{ items: SkillRegistry[]; total: number } | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const fetchRegistries = useCallback(async () => {
    try {
      const result = await listSkillRegistries();
      setData(result);
    } catch {
      // Keep previous data on error
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchRegistries();
  }, [fetchRegistries]);

  const resetForm = () => {
    setFormUrl("");
    setFormBranch("");
    setFormSourceType("");
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formUrl.trim()) {
      toast.error("Repository URL is required");
      return;
    }
    setIsCreating(true);
    try {
      await createSkillRegistry({
        repository_url: formUrl.trim(),
        branch: formBranch.trim() || undefined,
        source_type: formSourceType.trim() || undefined,
      });
      toast.success("Skill registry added successfully");
      setDialogOpen(false);
      resetForm();
      await fetchRegistries();
    } catch (err: unknown) {
      toast.error((err as { error?: string })?.error || "Failed to create skill registry");
    } finally {
      setIsCreating(false);
    }
  };

  const handleSync = async (id: number) => {
    setSyncingIds((prev) => new Set(prev).add(id));
    try {
      await syncSkillRegistry(id);
      toast.success("Sync triggered successfully");
      await fetchRegistries();
    } catch (err: unknown) {
      toast.error((err as { error?: string })?.error || "Failed to sync skill registry");
    } finally {
      setSyncingIds((prev) => { const next = new Set(prev); next.delete(id); return next; });
    }
  };

  const handleDelete = async (registry: SkillRegistry) => {
    if (!confirm(`Are you sure you want to delete "${registry.repository_url}"? This action cannot be undone.`)) return;
    try {
      await deleteSkillRegistry(registry.id);
      toast.success("Skill registry deleted successfully");
      await fetchRegistries();
    } catch (err: unknown) {
      toast.error((err as { error?: string })?.error || "Failed to delete skill registry");
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold">Skill Registries</h1>
          <p className="text-sm text-muted-foreground">
            Manage platform-level skill registry repositories. Skills synced from
            these repos are available to all organizations.
          </p>
        </div>
        <Dialog open={dialogOpen} onOpenChange={(open) => { setDialogOpen(open); if (!open) resetForm(); }}>
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
                  Add a new skill registry repository. Skills will be synced automatically after creation.
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
                  <Input id="branch" placeholder="main" value={formBranch} onChange={(e) => setFormBranch(e.target.value)} />
                  <p className="text-xs text-muted-foreground">Leave empty to use the default branch.</p>
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="source_type">Source Type</Label>
                  <Input id="source_type" placeholder="auto-detect" value={formSourceType} onChange={(e) => setFormSourceType(e.target.value)} />
                  <p className="text-xs text-muted-foreground">Leave empty for auto-detection.</p>
                </div>
              </div>
              <DialogFooter>
                <Button type="submit" disabled={isCreating}>
                  {isCreating && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                  Add Registry
                </Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      <SkillRegistriesTable
        registries={data?.items || []}
        isLoading={isLoading}
        syncingIds={syncingIds}
        onSync={handleSync}
        onDelete={handleDelete}
      />
    </div>
  );
}
