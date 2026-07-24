import { mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { AppState } from "@agentsmesh/node-bridge";
import { afterEach, describe, expect, it } from "vitest";

const createdDirectories: string[] = [];

async function withTimeout<T>(pending: Promise<T>, label: string): Promise<T> {
  let timer: ReturnType<typeof setTimeout> | undefined;
  const timeout = new Promise<never>((_resolve, reject) => {
    timer = setTimeout(() => reject(new Error(`${label} timed out`)), 10_000);
  });
  try {
    return await Promise.race([pending, timeout]);
  } finally {
    if (timer) clearTimeout(timer);
  }
}

describe("node-bridge relay readiness ABI", () => {
  afterEach(async () => {
    await Promise.all(createdDirectories.splice(0).map((directory) =>
      rm(directory, { recursive: true, force: true }),
    ));
  });

  it("rejects the N-API subscribe promise when a pending generation is removed", async () => {
    const storageDir = await mkdtemp(join(tmpdir(), "agentsmesh-node-relay-"));
    createdDirectories.push(storageDir);
    const state = new AppState("http://127.0.0.1:1", storageDir);
    let sawStatus!: () => void;
    const firstStatus = new Promise<void>((resolve) => { sawStatus = resolve; });
    const subscribe = state.relaySubscribe(
      "napi-readiness-pod",
      "napi-readiness-sub",
      "ws://127.0.0.1:1",
      "invalid-token",
      () => undefined,
      (error) => { if (!error) sawStatus(); },
      () => undefined,
      () => undefined,
      "napi-readiness-listeners",
    );
    const outcome = subscribe.then(
      () => ({ resolved: true as const, error: undefined }),
      (error: unknown) => ({ resolved: false as const, error }),
    );

    const settled = await (async () => {
      try {
        await withTimeout(firstStatus, "initial N-API status callback");
        await state.relayUnsubscribe("napi-readiness-pod", "napi-readiness-sub");
        return await withTimeout(outcome, "cancelled N-API subscribe");
      } finally {
        await state.relayDisconnectAll();
      }
    })();

    expect(settled.resolved, "subscribe must not report readiness without a baseline").toBe(false);
    expect(String(settled.error)).toMatch(/cancel|closed|removed|readiness|subscription/i);
  });
});
