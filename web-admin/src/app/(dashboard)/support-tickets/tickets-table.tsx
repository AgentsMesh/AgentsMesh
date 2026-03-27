import Link from "next/link";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { SupportTicket } from "@/lib/api/admin";
import { formatDate, formatRelativeTime } from "@/lib/utils";
import {
  statusLabels,
  statusVariants,
  categoryLabels,
  categoryVariants,
  priorityLabels,
  priorityVariants,
} from "@/lib/support-constants";

interface TicketsTableProps {
  tickets: SupportTicket[];
  isLoading: boolean;
}

export function TicketsTable({ tickets, isLoading }: TicketsTableProps) {
  return (
    <div className="overflow-hidden rounded-lg border border-border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Title</TableHead>
            <TableHead>User</TableHead>
            <TableHead>Category</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Priority</TableHead>
            <TableHead>Assigned To</TableHead>
            <TableHead>Created</TableHead>
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
          ) : tickets.length === 0 ? (
            <TableRow>
              <TableCell colSpan={7} className="py-8 text-center text-muted-foreground">
                No support tickets found
              </TableCell>
            </TableRow>
          ) : (
            tickets.map((ticket) => <TicketRow key={ticket.id} ticket={ticket} />)
          )}
        </TableBody>
      </Table>
    </div>
  );
}

function TicketRow({ ticket }: { ticket: SupportTicket }) {
  return (
    <TableRow className="cursor-pointer">
      <TableCell>
        <Link href={`/support-tickets/${ticket.id}`} className="font-medium hover:text-primary">
          {ticket.title}
        </Link>
      </TableCell>
      <TableCell>
        {ticket.user ? (
          <div className="flex items-center gap-2">
            {ticket.user.avatar_url ? (
              <img src={ticket.user.avatar_url} alt={ticket.user.name} className="h-6 w-6 rounded-full" />
            ) : (
              <div className="flex h-6 w-6 items-center justify-center rounded-full bg-primary/20 text-xs font-medium text-primary">
                {(ticket.user.name || ticket.user.email)[0].toUpperCase()}
              </div>
            )}
            <span className="text-sm">{ticket.user.email}</span>
          </div>
        ) : (
          <span className="text-muted-foreground">-</span>
        )}
      </TableCell>
      <TableCell>
        <Badge variant={categoryVariants[ticket.category] || "secondary"}>
          {categoryLabels[ticket.category] || ticket.category}
        </Badge>
      </TableCell>
      <TableCell>
        <Badge variant={statusVariants[ticket.status] || "secondary"}>
          {statusLabels[ticket.status] || ticket.status}
        </Badge>
      </TableCell>
      <TableCell>
        <Badge variant={priorityVariants[ticket.priority] || "secondary"}>
          {priorityLabels[ticket.priority] || ticket.priority}
        </Badge>
      </TableCell>
      <TableCell>
        {ticket.assigned_admin ? (
          <span className="text-sm">{ticket.assigned_admin.name || ticket.assigned_admin.email}</span>
        ) : (
          <span className="text-sm text-muted-foreground">Unassigned</span>
        )}
      </TableCell>
      <TableCell>
        <span className="text-sm text-muted-foreground" title={formatDate(ticket.created_at)}>
          {formatRelativeTime(ticket.created_at)}
        </span>
      </TableCell>
    </TableRow>
  );
}
