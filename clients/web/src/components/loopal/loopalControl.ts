import { relayPool } from "@/stores/relayConnection";

// Fire-and-forget Loopal control command over the relay. All Loopal controls
// share one envelope; UI state reflects the change only once the matching
// `_loopal/*` event flows back through loopalConsole (no optimistic write).
// Guards on connection like the prompt path (LoopalPromptInput) so a control
// issued on a disconnected pod is dropped deterministically instead of handed
// to a dead socket. Returns whether the command was dispatched.
export function loopalControl(
  podKey: string,
  subtype: string,
  payload: Record<string, unknown> = {},
): boolean {
  if (!relayPool.isConnected(podKey)) return false;
  relayPool.sendAcpCommand(podKey, { type: "control_request", subtype, payload });
  return true;
}
