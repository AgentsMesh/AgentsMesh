import { MsgType } from "@/stores/relayProtocol";
import { useLoopalConsoleStore } from "@/stores/loopalConsole";

// Routes Loopal control-panel signals (`loopal.*` carried over MsgTypeAcpEvent)
// to the Loopal store. Returns true when the payload was a Loopal signal so the
// caller can skip the standard ACP session dispatch. `loopal.snapshot` rebuilds
// full state; all other `loopal.*` are incremental events.
export function dispatchLoopalRelayEvent(
  podKey: string,
  msgType: number,
  payload: unknown,
): boolean {
  if (msgType !== MsgType.AcpEvent) return false;
  const data = payload as Record<string, unknown>;
  const type = data?.type;
  if (typeof type !== "string" || !type.startsWith("loopal.")) return false;

  const store = useLoopalConsoleStore.getState();
  if (type === "loopal.snapshot") {
    store.dispatchSnapshot(podKey, data);
  } else {
    store.dispatchEvent(podKey, type, data);
  }
  return true;
}
