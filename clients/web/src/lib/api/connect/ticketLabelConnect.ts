// Label ops for proto.ticket.v1.TicketService — labels are an independent
// CRUD surface (labels are first-class entities the ticket model references),
// split from ticketConnect.ts for SRP.

import {
  AddLabelRequestSchema,
  CreateLabelRequestSchema,
  DeleteLabelRequestSchema,
  LabelSchema,
  ListLabelsRequestSchema,
  ListLabelsResponseSchema,
  RemoveLabelRequestSchema,
  UpdateLabelRequestSchema,
} from "@proto/ticket/v1/ticket_pb";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import { getTicketService } from "@/lib/wasm-core";
import { fromProtoLabel } from "./ticketConnect";

export async function listLabels(
  orgSlug: string,
  opts: { repository_id?: number } = {},
): Promise<Array<{ id: number; name: string; color: string }>> {
  const req = create(ListLabelsRequestSchema, {
    orgSlug,
    repositoryId: opts.repository_id !== undefined ? BigInt(opts.repository_id) : undefined,
  });
  const bytes = toBinary(ListLabelsRequestSchema, req);
  const respBytes = await getTicketService().list_labels_connect(bytes);
  const resp = fromBinary(ListLabelsResponseSchema, new Uint8Array(respBytes));
  return resp.items.map(fromProtoLabel);
}

// Raw wire bytes for the fetch→state path: response (ListLabelsResponse) →
// apply_fetched_labels (Rust set_labels), no TS labelsToProto.
export async function listLabelsRaw(
  orgSlug: string, opts: { repository_id?: number } = {},
): Promise<Uint8Array> {
  const req = create(ListLabelsRequestSchema, {
    orgSlug,
    repositoryId: opts.repository_id !== undefined ? BigInt(opts.repository_id) : undefined,
  });
  return new Uint8Array(
    await getTicketService().list_labels_connect(toBinary(ListLabelsRequestSchema, req)),
  );
}

export async function createLabel(
  orgSlug: string,
  name: string,
  color: string,
  opts: { repository_id?: number } = {},
): Promise<{ id: number; name: string; color: string }> {
  const req = create(CreateLabelRequestSchema, {
    orgSlug,
    name,
    color,
    repositoryId: opts.repository_id !== undefined ? BigInt(opts.repository_id) : undefined,
  });
  const bytes = toBinary(CreateLabelRequestSchema, req);
  const respBytes = await getTicketService().create_label_connect(bytes);
  return fromProtoLabel(fromBinary(LabelSchema, new Uint8Array(respBytes)));
}

export async function updateLabel(
  orgSlug: string,
  id: number,
  patch: { name?: string; color?: string },
): Promise<{ id: number; name: string; color: string }> {
  const req = create(UpdateLabelRequestSchema, {
    orgSlug,
    id: BigInt(id),
    name: patch.name,
    color: patch.color,
  });
  const bytes = toBinary(UpdateLabelRequestSchema, req);
  const respBytes = await getTicketService().update_label_connect(bytes);
  return fromProtoLabel(fromBinary(LabelSchema, new Uint8Array(respBytes)));
}

export async function deleteLabel(orgSlug: string, id: number): Promise<void> {
  const req = create(DeleteLabelRequestSchema, { orgSlug, id: BigInt(id) });
  await getTicketService().delete_label_connect(toBinary(DeleteLabelRequestSchema, req));
}

export async function addLabel(
  orgSlug: string,
  ticketSlug: string,
  labelId: number,
): Promise<void> {
  const req = create(AddLabelRequestSchema, {
    orgSlug,
    ticketSlug,
    labelId: BigInt(labelId),
  });
  await getTicketService().add_label_connect(toBinary(AddLabelRequestSchema, req));
}

export async function removeLabel(
  orgSlug: string,
  ticketSlug: string,
  labelId: number,
): Promise<void> {
  const req = create(RemoveLabelRequestSchema, {
    orgSlug,
    ticketSlug,
    labelId: BigInt(labelId),
  });
  await getTicketService().remove_label_connect(toBinary(RemoveLabelRequestSchema, req));
}
