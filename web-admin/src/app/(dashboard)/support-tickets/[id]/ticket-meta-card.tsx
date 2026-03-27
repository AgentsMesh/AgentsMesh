import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { SupportTicket } from "@/lib/api/admin";
import { categoryLabels, priorityLabels } from "@/lib/support-constants";

interface TicketMetaCardProps {
  ticket: SupportTicket;
  isUpdatingStatus: boolean;
  isAssigning: boolean;
  onStatusChange: (status: string) => void;
  onAssign: () => void;
}

export function TicketMetaCard({
  ticket,
  isUpdatingStatus,
  isAssigning,
  onStatusChange,
  onAssign,
}: TicketMetaCardProps) {
  return (
    <Card>
      <CardContent className="pt-6">
        <div className="flex flex-wrap items-center gap-4">
          <div className="flex items-center gap-2">
            <span className="text-sm text-muted-foreground">Status:</span>
            <Select
              value={ticket.status}
              onValueChange={onStatusChange}
              disabled={isUpdatingStatus}
            >
              <SelectTrigger className="w-36 h-8">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="open">Open</SelectItem>
                <SelectItem value="in_progress">In Progress</SelectItem>
                <SelectItem value="resolved">Resolved</SelectItem>
                <SelectItem value="closed">Closed</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="flex items-center gap-2">
            <span className="text-sm text-muted-foreground">Category:</span>
            <Badge variant="outline">{categoryLabels[ticket.category] || ticket.category}</Badge>
          </div>

          <div className="flex items-center gap-2">
            <span className="text-sm text-muted-foreground">Priority:</span>
            <Badge variant={ticket.priority === "high" ? "destructive" : ticket.priority === "medium" ? "warning" : "secondary"}>
              {priorityLabels[ticket.priority] || ticket.priority}
            </Badge>
          </div>

          <div className="flex items-center gap-2">
            <span className="text-sm text-muted-foreground">Assigned:</span>
            {ticket.assigned_admin ? (
              <span className="text-sm">{ticket.assigned_admin.name || ticket.assigned_admin.email}</span>
            ) : (
              <Button
                variant="outline"
                size="sm"
                onClick={onAssign}
                disabled={isAssigning}
              >
                Assign to me
              </Button>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
