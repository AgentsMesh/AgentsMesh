import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { PromoCode } from "@/lib/api/admin";
import { PromoCodeRow } from "./promo-code-row";

interface PromoCodesTableProps {
  promoCodes: PromoCode[];
  isLoading: boolean;
  onActivate: (id: number) => void;
  onDeactivate: (id: number) => void;
  onDelete: (code: PromoCode) => void;
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
