import { randomUUID } from "@/lib/blockstore/uuid";
import type { JSONMap, OpEnvelope } from "@/lib/api/blockstoreTypes";

export function createBlockOp(args: {
  id?: string;
  type: string;
  data?: JSONMap;
  text?: string | null;
  meta?: JSONMap;
}): OpEnvelope {
  return {
    op: "createBlock",
    payload: {
      id: args.id ?? randomUUID(),
      type: args.type,
      data: args.data ?? {},
      text: args.text ?? null,
      meta: args.meta ?? {},
    },
  };
}

export function updateBlockOp(args: {
  id: string;
  data?: JSONMap;
  text?: string | null;
  meta?: JSONMap;
  expected_updated_at?: string;
}): OpEnvelope {
  const payload: JSONMap = { id: args.id };
  if (args.data !== undefined) payload.data = args.data;
  if (args.text !== undefined) payload.text = args.text;
  if (args.meta !== undefined) payload.meta = args.meta;
  if (args.expected_updated_at) payload.expected_updated_at = args.expected_updated_at;
  return { op: "updateBlock", payload };
}

export function deleteBlockOp(id: string): OpEnvelope {
  return { op: "deleteBlock", payload: { id } };
}

export function addRefOp(args: {
  from: string;
  to: string;
  rel: string;
  order_key?: string | null;
  anchor?: string | null;
  meta?: JSONMap;
}): OpEnvelope {
  const payload: JSONMap = { from: args.from, to: args.to, rel: args.rel };
  if (args.order_key !== undefined) payload.order_key = args.order_key;
  if (args.anchor !== undefined) payload.anchor = args.anchor;
  if (args.meta !== undefined) payload.meta = args.meta;
  return { op: "addRef", payload };
}

export function removeRefOp(refID: number): OpEnvelope {
  return { op: "removeRef", payload: { ref_id: refID } };
}

export function updateRefOp(args: {
  ref_id: number;
  from?: string;
  order_key?: string | null;
  anchor?: string | null;
  meta?: JSONMap;
}): OpEnvelope {
  const payload: JSONMap = { ref_id: args.ref_id };
  if (args.from !== undefined) payload.from = args.from;
  if (args.order_key !== undefined) payload.order_key = args.order_key;
  if (args.anchor !== undefined) payload.anchor = args.anchor;
  if (args.meta !== undefined) payload.meta = args.meta;
  return { op: "updateRef", payload };
}
