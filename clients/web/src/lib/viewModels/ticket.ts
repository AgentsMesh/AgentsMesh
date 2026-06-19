// Ticket view-model types moved to the zero-dep @agentsmesh/service-interface
// contract layer so the web fromProtoTicket projection and the desktop
// electron-adapter projection share one definition. Re-exported here to
// preserve existing `@/lib/viewModels/ticket` import paths.
export type {
  TicketStatus,
  TicketPriority,
  TicketData,
  TicketRelation,
  TicketCommit,
  TicketComment,
  BoardColumn,
} from "@agentsmesh/service-interface";
