import Link from "next/link";
import {
  Tag,
  Calendar,
  Users,
  Power,
  PowerOff,
  Trash2,
  MoreHorizontal,
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
import type { PromoCode, PromoCodeType } from "@/lib/api/admin";
import { formatDate } from "@/lib/utils";

const typeLabels: Record<PromoCodeType, string> = {
  media: "Media",
  partner: "Partner",
  campaign: "Campaign",
  internal: "Internal",
  referral: "Referral",
};

const typeColors: Record<PromoCodeType, "default" | "secondary" | "outline" | "destructive"> = {
  media: "default",
  partner: "secondary",
  campaign: "outline",
  internal: "destructive",
  referral: "default",
};

interface PromoCodesTableProps {
  promoCodes: PromoCode[];
  isLoading: boolean;
  onActivate: (id: number) => void;
  onDeactivate: (id: number) => void;
  onDelete: (code: PromoCode) => void;
}

function getRemainingUses(code: PromoCode) {
  if (code.max_uses === null) return "Unlimited";
  const remaining = code.max_uses - code.used_count;
  return `${remaining}/${code.max_uses}`;
}

function isExpired(code: PromoCode) {
  if (!code.expires_at) return false;
  return new Date(code.expires_at) < new Date();
}

export function PromoCodesTable({
  promoCodes,
  isLoading,
  onActivate,
  onDeactivate,
  onDelete,
}: PromoCodesTableProps) {
  return (
    <div className="overflow-hidden rounded-lg border border-border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Code</TableHead>
            <TableHead>Name</TableHead>
            <TableHead>Type</TableHead>
            <TableHead>Plan</TableHead>
            <TableHead>Duration</TableHead>
            <TableHead>Uses</TableHead>
            <TableHead>Expires</TableHead>
            <TableHead>Status</TableHead>
            <TableHead className="w-12"></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            Array.from({ length: 5 }).map((_, i) => (
              <TableRow key={i}>
                <TableCell colSpan={9}>
                  <div className="h-12 animate-pulse rounded bg-muted" />
                </TableCell>
              </TableRow>
            ))
          ) : promoCodes.length === 0 ? (
            <TableRow>
              <TableCell colSpan={9} className="py-8 text-center text-muted-foreground">
                No promo codes found
              </TableCell>
            </TableRow>
          ) : (
            promoCodes.map((code) => (
              <PromoCodeRow
                key={code.id}
                code={code}
                onActivate={onActivate}
                onDeactivate={onDeactivate}
                onDelete={onDelete}
              />
            ))
          )}
        </TableBody>
      </Table>
    </div>
  );
}

function PromoCodeRow({
  code,
  onActivate,
  onDeactivate,
  onDelete,
}: {
  code: PromoCode;
  onActivate: (id: number) => void;
  onDeactivate: (id: number) => void;
  onDelete: (code: PromoCode) => void;
}) {
  const expired = isExpired(code);

  return (
    <TableRow>
      <TableCell>
        <Link
          href={`/promo-codes/${code.id}`}
          className="flex items-center gap-2 font-mono font-medium hover:text-primary"
        >
          <Tag className="h-4 w-4 text-muted-foreground" />
          {code.code}
        </Link>
      </TableCell>
      <TableCell>{code.name}</TableCell>
      <TableCell>
        <Badge variant={typeColors[code.type]}>{typeLabels[code.type]}</Badge>
      </TableCell>
      <TableCell className="capitalize">{code.plan_name}</TableCell>
      <TableCell>{code.duration_months} months</TableCell>
      <TableCell>
        <div className="flex items-center gap-1">
          <Users className="h-3 w-3 text-muted-foreground" />
          {getRemainingUses(code)}
        </div>
      </TableCell>
      <TableCell>
        {code.expires_at ? (
          <div className="flex items-center gap-1">
            <Calendar className="h-3 w-3 text-muted-foreground" />
            <span className={expired ? "text-destructive" : ""}>
              {formatDate(code.expires_at)}
            </span>
          </div>
        ) : (
          <span className="text-muted-foreground">Never</span>
        )}
      </TableCell>
      <TableCell>
        {code.is_active && !expired ? (
          <Badge variant="success">Active</Badge>
        ) : expired ? (
          <Badge variant="destructive">Expired</Badge>
        ) : (
          <Badge variant="secondary">Inactive</Badge>
        )}
      </TableCell>
      <TableCell>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon">
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem asChild>
              <Link href={`/promo-codes/${code.id}`}>View Details</Link>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            {code.is_active ? (
              <DropdownMenuItem onClick={() => onDeactivate(code.id)}>
                <PowerOff className="mr-2 h-4 w-4" />
                Deactivate
              </DropdownMenuItem>
            ) : (
              <DropdownMenuItem onClick={() => onActivate(code.id)}>
                <Power className="mr-2 h-4 w-4" />
                Activate
              </DropdownMenuItem>
            )}
            <DropdownMenuSeparator />
            <DropdownMenuItem
              onClick={() => onDelete(code)}
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
