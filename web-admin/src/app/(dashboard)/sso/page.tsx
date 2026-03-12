"use client";

import { useState, useEffect, useCallback } from "react";
import {
  Search,
  Plus,
  Power,
  PowerOff,
  Trash2,
  MoreHorizontal,
  ChevronLeft,
  ChevronRight,
  Pencil,
  FlaskConical,
  ShieldCheck,
} from "lucide-react";
import { toast } from "sonner";
import { Button, buttonVariants } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
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
import { formatDate } from "@/lib/utils";
import { SSOFormDialog } from "./sso-form-dialog";

const protocolLabels: Record<SSOProtocol, string> = {
  oidc: "OIDC",
  saml: "SAML",
  ldap: "LDAP",
};

const protocolColors: Record<SSOProtocol, "default" | "secondary" | "outline"> = {
  oidc: "default",
  saml: "secondary",
  ldap: "outline",
};

export default function SSOPage() {
  const [search, setSearch] = useState("");
  const [protocolFilter, setProtocolFilter] = useState<string>("all");
  const [configs, setConfigs] = useState<SSOConfig[]>([]);
  const [total, setTotal] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [page, setPage] = useState(1);
  const pageSize = 20;

  // Dialog state
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

  useEffect(() => {
    fetchConfigs();
  }, [fetchConfigs]);

  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  const handleCreate = () => {
    setEditingConfig(null);
    setDialogOpen(true);
  };

  const handleEdit = (config: SSOConfig) => {
    setEditingConfig(config);
    setDialogOpen(true);
  };

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
      throw err; // Prevent dialog close
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
        <Button onClick={handleCreate}>
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
            onChange={(e) => {
              setSearch(e.target.value);
              setPage(1);
            }}
            className="pl-10"
          />
        </div>
        <Select
          value={protocolFilter}
          onValueChange={(value) => {
            setProtocolFilter(value);
            setPage(1);
          }}
        >
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

      {/* Table */}
      <div className="overflow-hidden rounded-lg border border-border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Domain</TableHead>
              <TableHead>Name</TableHead>
              <TableHead>Protocol</TableHead>
              <TableHead>Enabled</TableHead>
              <TableHead>Enforce SSO</TableHead>
              <TableHead>Created</TableHead>
              <TableHead className="w-12"></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell colSpan={7}>
                    <div className="h-12 animate-pulse rounded bg-muted" />
                  </TableCell>
                </TableRow>
              ))
            ) : configs.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} className="py-8 text-center text-muted-foreground">
                  No SSO configs found
                </TableCell>
              </TableRow>
            ) : (
              configs.map((config) => (
                <TableRow key={config.id}>
                  <TableCell className="font-medium">{config.domain}</TableCell>
                  <TableCell>{config.name}</TableCell>
                  <TableCell>
                    <Badge variant={protocolColors[config.protocol]}>
                      {protocolLabels[config.protocol]}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    {config.is_enabled ? (
                      <Badge variant="success">Enabled</Badge>
                    ) : (
                      <Badge variant="secondary">Disabled</Badge>
                    )}
                  </TableCell>
                  <TableCell>
                    {config.enforce_sso ? (
                      <Badge variant="destructive" className="gap-1">
                        <ShieldCheck className="h-3 w-3" />
                        Enforced
                      </Badge>
                    ) : (
                      <span className="text-muted-foreground">-</span>
                    )}
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {formatDate(config.created_at)}
                  </TableCell>
                  <TableCell>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="icon">
                          <MoreHorizontal className="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem onClick={() => handleEdit(config)}>
                          <Pencil className="mr-2 h-4 w-4" />
                          Edit
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => handleTest(config)}>
                          <FlaskConical className="mr-2 h-4 w-4" />
                          Test Connection
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        {config.is_enabled ? (
                          <DropdownMenuItem onClick={() => handleDisable(config.id)}>
                            <PowerOff className="mr-2 h-4 w-4" />
                            Disable
                          </DropdownMenuItem>
                        ) : (
                          <DropdownMenuItem onClick={() => handleEnable(config.id)}>
                            <Power className="mr-2 h-4 w-4" />
                            Enable
                          </DropdownMenuItem>
                        )}
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                          onClick={() => setDeletingConfig(config)}
                          className="text-destructive focus:text-destructive"
                        >
                          <Trash2 className="mr-2 h-4 w-4" />
                          Delete
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <p className="text-sm text-muted-foreground">
            Showing {(page - 1) * pageSize + 1} to{" "}
            {Math.min(page * pageSize, total)} of {total} configs
          </p>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="icon"
              onClick={() => setPage(page - 1)}
              disabled={page <= 1}
            >
              <ChevronLeft className="h-4 w-4" />
            </Button>
            <span className="text-sm">
              Page {page} of {totalPages}
            </span>
            <Button
              variant="outline"
              size="icon"
              onClick={() => setPage(page + 1)}
              disabled={page >= totalPages}
            >
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
        </div>
      )}

      {/* Create/Edit Dialog */}
      <SSOFormDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        config={editingConfig}
        onSubmit={handleFormSubmit}
      />

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={!!deletingConfig} onOpenChange={(open) => !open && setDeletingConfig(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete SSO Config</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete SSO config &quot;{deletingConfig?.name}&quot; ({deletingConfig?.domain})?
              This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDeleteConfirm}
              className={buttonVariants({ variant: "destructive" })}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
