import { Ticket, TicketStatus, useBoardColumns } from "@/stores/ticket";
import { useTicketStore } from "@/stores/ticket";
import { KanbanBoard } from "@/components/tickets";

interface BoardViewLayoutProps {
  tickets: Ticket[];
  onStatusChange: (slug: string, newStatus: TicketStatus) => Promise<void>;
  onTicketClick: (ticket: Ticket) => void;
  onCreatePodRequest?: (ticket: Ticket) => void;
}

export function BoardViewLayout({
  tickets,
  onStatusChange,
  onTicketClick,
  onCreatePodRequest,
}: BoardViewLayoutProps) {
  const boardColumns = useBoardColumns();
  const columnPagination = useTicketStore((s) => s.columnPagination);
  const doneCollapsed = useTicketStore((s) => s.doneCollapsed);
  const loadMoreColumn = useTicketStore((s) => s.loadMoreColumn);
  const setDoneCollapsed = useTicketStore((s) => s.setDoneCollapsed);

  return (
    <div className="h-full flex flex-col">
      <div className="flex-1 min-h-0 p-4">
        <KanbanBoard
          tickets={tickets}
          boardColumns={boardColumns.length > 0 ? boardColumns : undefined}
          columnPagination={columnPagination}
          doneCollapsed={doneCollapsed}
          onLoadMoreColumn={loadMoreColumn}
          onSetDoneCollapsed={setDoneCollapsed}
          onStatusChange={onStatusChange}
          onTicketClick={onTicketClick}
          onCreatePodRequest={onCreatePodRequest}
        />
      </div>
    </div>
  );
}
