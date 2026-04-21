"use client";

import { toast } from "sonner";

import type { ApplyOpsResult, OpEnvelope } from "@/lib/api/blockstoreTypes";
import { blockstoreApi } from "@/lib/api/blockstoreApi";
import { ApiError } from "@/lib/api/api-types";
import { reconnectRegistry } from "@/lib/realtime";
import { getErrorMessage } from "@/lib/utils";

import { useBlockstoreStore } from "./blockstore";

/**
 * Phase 3.4 — offline-friendly dispatch.
 *
 * dispatchOps first attempts a live HTTP call. If the call fails with a
 * network error (offline, 5xx, aborted) we enqueue the batch into localStorage
 * and return a synthetic "pending" result. A reconnect hook drains the queue
 * in FIFO order whenever the WebSocket connects back.
 *
 * Persistent replay is idempotency-safe because every queued batch carries a
 * stable idempotency_key — rerunning is a no-op on the server.
 */

const STORAGE_KEY = "blockstore:pending-ops:v1";

interface PendingBatch {
  workspaceID: string;
  ops: OpEnvelope[];
  idempotencyKey: string;
  parentOpID?: number;
  enqueuedAt: number;
}

export async function dispatchOps(
  workspaceID: string,
  ops: OpEnvelope[],
  opts?: { idempotencyKey?: string; parentOpID?: number },
): Promise<ApplyOpsResult> {
  if (ops.length === 0) return { op_ids: [], was_replay: false };
  const idempotencyKey = opts?.idempotencyKey ?? generateIdempotencyKey();

  try {
    const res = await blockstoreApi.applyOps({
      workspace_id: workspaceID,
      ops,
      idempotency_key: idempotencyKey,
      parent_op_id: opts?.parentOpID,
    });
    if (!res.was_replay) {
      await useBlockstoreStore.getState().actions.catchup(workspaceID);
    }
    return res;
  } catch (err) {
    if (!isTransientError(err)) {
      // Non-transient errors (400 / 403 / 409) won't succeed on retry —
      // surface them to the user. Most callers fire dispatch with `void`
      // so without this toast the write would fail silently.
      toast.error(getErrorMessage(err, "Action failed"));
      throw err;
    }
    enqueue({
      workspaceID,
      ops,
      idempotencyKey,
      parentOpID: opts?.parentOpID,
      enqueuedAt: Date.now(),
    });
    return { op_ids: [], was_replay: false };
  }
}

function isTransientError(err: unknown): boolean {
  if (err instanceof ApiError) {
    return err.status === 0 || err.status >= 500;
  }
  // fetch throws TypeError on network failures.
  return err instanceof TypeError;
}

function generateIdempotencyKey(): string {
  const rand = Math.random().toString(36).slice(2, 10);
  return `web-${Date.now()}-${rand}`;
}

function loadQueue(): PendingBatch[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

function saveQueue(batches: PendingBatch[]) {
  if (typeof window === "undefined") return;
  localStorage.setItem(STORAGE_KEY, JSON.stringify(batches));
}

function enqueue(batch: PendingBatch) {
  const q = loadQueue();
  q.push(batch);
  saveQueue(q);
}

/**
 * Attempts to flush every queued batch. Batches that still fail stay in the
 * queue and will be retried on the next reconnect event. Success drains from
 * the head.
 */
export async function flushPendingOps(): Promise<void> {
  const queue = loadQueue();
  if (queue.length === 0) return;
  const remaining: PendingBatch[] = [];
  for (const batch of queue) {
    try {
      await blockstoreApi.applyOps({
        workspace_id: batch.workspaceID,
        ops: batch.ops,
        idempotency_key: batch.idempotencyKey,
        parent_op_id: batch.parentOpID,
      });
      // Success → pull fresh state from server so the optimistic diff matches.
      await useBlockstoreStore.getState().actions.catchup(batch.workspaceID);
    } catch (err) {
      if (isTransientError(err)) {
        remaining.push(batch);
      }
      // Non-transient failure → drop the batch (it would fail deterministically)
    }
  }
  saveQueue(remaining);
}

export function pendingOpsCount(): number {
  return loadQueue().length;
}

// Drain the queue opportunistically whenever the WebSocket reconnects.
reconnectRegistry.register({
  name: "blockstore:flush-pending",
  fn: () => {
    void flushPendingOps();
  },
  priority: "deferred",
});
