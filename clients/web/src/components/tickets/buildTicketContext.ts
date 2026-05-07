import type { Ticket } from "@/stores/ticket";
import type { TicketContext } from "@/components/pod/CreatePodForm";

type TicketLike = Partial<Pick<Ticket, "id" | "title" | "content" | "repository_id">>;

export function buildTicketContext(
  ticket: TicketLike,
  slug: string,
): TicketContext | undefined {
  if (!ticket.id) return undefined;
  return {
    id: ticket.id,
    slug,
    title: ticket.title ?? "",
    description: ticket.content,
    repositoryId: ticket.repository_id,
  };
}
