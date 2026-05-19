import { useAuthStore } from "@/stores/auth";
import { useBlockstoreStore } from "@/stores/blockstore";
import { useWorkspaceStore } from "@/stores/workspace";
import { usePodStore } from "@/stores/pod";
import { logger } from "@/lib/logger";

// Burst detector for store update loops. When wired up in dev/debug builds
// it warns (with a stack trace) the moment any store exceeds N setStates per
// second — typically the signature of a render-effect-store update cycle
// (React #185). Each detector is a passive subscribe; zero runtime cost
// beyond a counter increment per setState.
// Output goes through `logger` so the warning lands in the Rust subscriber's
// rolling file on Desktop (and stays in console on Web). The trace string
// is captured here rather than at console.trace time because IPC strips the
// implicit `console` stack.
interface SubscribableStore {
  subscribe: (cb: () => void) => () => void;
}

function attach(name: string, store: SubscribableStore, threshold: number): () => void {
  let count = 0;
  let windowStart = Date.now();
  let warned = false;
  return store.subscribe(() => {
    const now = Date.now();
    if (now - windowStart > 1000) {
      count = 0;
      windowStart = now;
      warned = false;
    }
    count++;
    if (!warned && count > threshold) {
      warned = true;
      const stack = new Error("storeBurst trace").stack ?? "<no stack>";
      logger.warn(
        "storeBurst",
        `${name} exceeded ${threshold} setStates/s — possible render loop\n${stack}`,
      );
    }
  });
}

export function installStoreBurstDetector(threshold = 30): () => void {
  const disposers = [
    attach("auth", useAuthStore, threshold),
    attach("blockstore", useBlockstoreStore, threshold),
    attach("workspace", useWorkspaceStore, threshold),
    attach("pod", usePodStore, threshold),
  ];
  return () => disposers.forEach((fn) => fn());
}
