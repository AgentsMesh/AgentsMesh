"use client";

import { useState, useEffect, useCallback } from "react";
import { Search, Plus, ChevronLeft, ChevronRight } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  listSSOConfigs,
  enableSSOConfig,
  disableSSOConfig,
  deleteSSOConfig,
  createSSOConfig,
  updateSSOConfig,
  testSSOConfig,
  SSOConfig,
  SSOProtocol,
  CreateSSOConfigRequest,
} from "@/lib/api/sso";
import { SSOFormDialog } from "./sso-form-dialog";
import { SSOTable } from "./sso-table";
import { SSODeleteDialog } from "./sso-delete-dialog";

export default function SSOPage() {
  const [search, setSearch] = useState("");
  const [protocolFilter, setProtocolFilter] = useState<string>("all");
  const [configs, setConfigs] = useState<SSOConfig[]>([]);
  const [total, setTotal] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [page, setPage] = useState(1);
  const pageSize = 20;

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingConfig, setEditingConfig] = useState<SSOConfig | null>(null);
  const [deletingConfig, setDeletingConfig] = useState<SSOConfig | null>(null);

  const fetchConfigs = useCallback(async () => {
    setIsLoading(true);
    try {
      const result = await listSSOConfigs({
        search: search || undefined,
        protocol: protocolFilter !== "all" ? (protocolFilter as SSOProtocol) : undefined,
        page,
        page_size: pageSize,
      });
      setConfigs(result.data || []);
      setTotal(result.total || 0);
    } catch {
      // Keep previous data on error
    } finally {
      setIsLoading(false);
    }
  }, [search, protocolFilter, page]);

  useEffect(() => { fetchConfigs(); }, [fetchConfigs]);

  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  const handleFormSubmit = async (data: CreateSSOConfigRequest) => {
    try {
      if (editingConfig) {
        await updateSSOConfig(editingConfig.id, data);
        toast.success("SSO config updated");
      } else {
        await createSSOConfig(data);
        toast.success("SSO config created");
      }
      await fetchConfigs();
    } catch (err: unknown) {
      const message = (err as { error?: string })?.error || "Failed to save SSO config";
      toast.error(message);
      throw err;
    }
  };

  const handleEnable = async (id: number) => {
    try {
      await enableSSOConfig(id);
      toast.success("SSO config enabled");
      await fetchConfigs();
    } catch (err: unknown) {
      toast.error((err as { error?: string })?.error || "Failed to enable SSO config");
    }
  };

  const handleDisable = async (id: number) => {
    try {
      await disableSSOConfig(id);
      toast.success("SSO config disabled");
      await fetchConfigs();
    } catch (err: unknown) {
      toast.error((err as { error?: string })?.error || "Failed to disable SSO config");
    }
  };

  const handleDeleteConfirm = async () => {
    if (!deletingConfig) return;
    try {
      await deleteSSOConfig(deletingConfig.id);
      toast.success("SSO config deleted");
      setDeletingConfig(null);
      await fetchConfigs();
    } catch (err: unknown) {
      toast.error((err as { error?: string })?.error || "Failed to delete SSO config");
    }
  };

  const handleTest = async (config: SSOConfig) => {
    try {
      const result = await testSSOConfig(config.id);
      if (result.success) {
        toast.success(result.message || "Connection test passed");
      } else {
        toast.error(result.error || result.message || "Connection test failed");
      }
    } catch (err: unknown) {
      toast.error((err as { error?: string })?.error || "Connection test failed");
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold">SSO Configs</h1>
          <p className="text-sm text-muted-foreground">
            Manage single sign-on configurations for domains
          </p>
        </div>
        <Button onClick={() => { setEditingConfig(null); setDialogOpen(true); }}>
          <Plus className="mr-2 h-4 w-4" />
          Create SSO Config
        </Button>
      </div>

      {/* Filters */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search by domain or name..."
            value={search}
            onChange={(e) => { setSearch(e.target.value); setPage(1); }}
            className="pl-10"
          />
        </div>
        <Select value={protocolFilter} onValueChange={(value) => { setProtocolFilter(value); setPage(1); }}>
          <SelectTrigger className="w-40">
            <SelectValue placeholder="All Protocols" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Protocols</SelectItem>
            <SelectItem value="oidc">OIDC</SelectItem>
            <SelectItem value="saml">SAML</SelectItem>
            <SelectItem value="ldap">LDAP</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <SSOTable
        configs={configs}
        isLoading={isLoading}
        onEdit={(config) => { setEditingConfig(config); setDialogOpen(true); }}
        onTest={handleTest}
        onEnable={handleEnable}
        onDisable={handleDisable}
        onDelete={setDeletingConfig}
      />

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <p className="text-sm text-muted-foreground">
            Showing {(page - 1) * pageSize + 1} to {Math.min(page * pageSize, total)} of {total} configs
          </p>
          <div className="flex items-center gap-2">
            <Button variant="outline" size="icon" onClick={() => setPage(page - 1)} disabled={page <= 1}>
              <ChevronLeft className="h-4 w-4" />
            </Button>
            <span className="text-sm">Page {page} of {totalPages}</span>
            <Button variant="outline" size="icon" onClick={() => setPage(page + 1)} disabled={page >= totalPages}>
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
        </div>
      )}

      <SSOFormDialog open={dialogOpen} onOpenChange={setDialogOpen} config={editingConfig} onSubmit={handleFormSubmit} />
      <SSODeleteDialog config={deletingConfig} onOpenChange={() => setDeletingConfig(null)} onConfirm={handleDeleteConfirm} />
    </div>
  );
}
