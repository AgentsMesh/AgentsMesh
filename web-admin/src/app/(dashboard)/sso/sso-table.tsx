"use client";

import {
  Power,
  PowerOff,
  Trash2,
  MoreHorizontal,
  Pencil,
  FlaskConical,
  ShieldCheck,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
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
import type { SSOConfig, SSOProtocol } from "@/lib/api/sso";
import { formatDate } from "@/lib/utils";

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

interface SSOTableProps {
  configs: SSOConfig[];
  isLoading: boolean;
  onEdit: (config: SSOConfig) => void;
  onTest: (config: SSOConfig) => void;
  onEnable: (id: number) => void;
  onDisable: (id: number) => void;
  onDelete: (config: SSOConfig) => void;
}

export function SSOTable({
  configs,
  isLoading,
  onEdit,
  onTest,
  onEnable,
  onDisable,
  onDelete,
}: SSOTableProps) {
  return (
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
              <SSOTableRow
                key={config.id}
                config={config}
                onEdit={onEdit}
                onTest={onTest}
                onEnable={onEnable}
                onDisable={onDisable}
                onDelete={onDelete}
              />
            ))
          )}
        </TableBody>
      </Table>
    </div>
  );
}

function SSOTableRow({
  config,
  onEdit,
  onTest,
  onEnable,
  onDisable,
  onDelete,
}: {
  config: SSOConfig;
  onEdit: (config: SSOConfig) => void;
  onTest: (config: SSOConfig) => void;
  onEnable: (id: number) => void;
  onDisable: (id: number) => void;
  onDelete: (config: SSOConfig) => void;
}) {
  return (
    <TableRow>
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
            <DropdownMenuItem onClick={() => onEdit(config)}>
              <Pencil className="mr-2 h-4 w-4" />
              Edit
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => onTest(config)}>
              <FlaskConical className="mr-2 h-4 w-4" />
              Test Connection
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            {config.is_enabled ? (
              <DropdownMenuItem onClick={() => onDisable(config.id)}>
                <PowerOff className="mr-2 h-4 w-4" />
                Disable
              </DropdownMenuItem>
            ) : (
              <DropdownMenuItem onClick={() => onEnable(config.id)}>
                <Power className="mr-2 h-4 w-4" />
                Enable
              </DropdownMenuItem>
            )}
            <DropdownMenuSeparator />
            <DropdownMenuItem
              onClick={() => onDelete(config)}
              className="text-destructive focus:text-destructive"
            >
              <Trash2 className="mr-2 h-4 w-4" />
              Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </TableCell>
    </TableRow>
  );
}
