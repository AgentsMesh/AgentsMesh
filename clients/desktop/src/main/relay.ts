import { ipcMain, type IpcMainInvokeEvent } from "electron";
import type {
  RelayInvokeArgs,
  RelayInvokeChannel,
  RelayInvokeResult,
  RelaySubscribeArgs,
  RelaySubscribeResult,
} from "@agentsmesh/electron-adapter/relay-ipc-contract";
import { logEvent, type AppState } from "@agentsmesh/node-bridge";
import { type WindowRegistry } from "./window_registry";
import { RelayListenerWiring } from "./relay_listener_wiring";
import { RelayOutputSubscriptions } from "./relay_output_subscriptions";

// Bridges the Rust `RelayConnectionPool` (terminal data plane SSOT, owned by the
// main process) to renderers. Renderer → main: `relay:*` invoke handlers map onto
// `appState.relay*` NAPI methods. Main → renderer: output/status/acp fan out via
// the registry. PTY output retains its Rust subscription identity all the way
// to one renderer callback; status and ACP remain pod-scoped fan-out.
//
// Rust still owns one transport per pod, but every renderer subscription gets a
// distinct pool subscriber. Its renderer attempt id remains attached to every
// output event, so a replacement baseline cannot land in another generation.

export interface RelayBridge {
  releaseWebContents: (wcId: number) => void;
  dispose: () => void;
}

type RelayInvokeHandler<C extends RelayInvokeChannel> = (
  event: IpcMainInvokeEvent,
  ...args: RelayInvokeArgs<C>
) => RelayInvokeResult<C> | Promise<RelayInvokeResult<C>>;

function registerRelayHandler<C extends RelayInvokeChannel>(
  channel: C,
  handler: RelayInvokeHandler<C>,
): void {
  ipcMain.handle(channel, (event, ...args) =>
    handler(event, ...(args as RelayInvokeArgs<C>)),
  );
}

export function setupRelayBridge(appState: AppState, registry: WindowRegistry): RelayBridge {
  const subscriptions = new RelayOutputSubscriptions(registry);
  const listenerWiring = new RelayListenerWiring(appState, subscriptions, registry);

  const subscribe = async (
    wcId: number,
    ...[podKey, subId, attemptId, url, token]: RelaySubscribeArgs
  ): Promise<RelaySubscribeResult> => {
    const attempt = subscriptions.begin(wcId, podKey, subId, attemptId);
    const listeners = listenerWiring.forPod(podKey);
    const { coreSubId: id } = attempt;
    logEvent("info", "relay", `subscribe ${podKey}/${subId} (wc ${wcId})`);
    if (attempt.supersededCoreSubId) {
      void appState.relayUnsubscribe(podKey, attempt.supersededCoreSubId).catch((error: unknown) =>
        logEvent("warn", "relay", `supersede ${podKey}/${attempt.supersededCoreSubId} failed: ${error}`),
      );
    }
    try {
      await appState.relaySubscribe(
        podKey,
        id,
        url,
        token,
        subscriptions.onOutput(attempt),
        listeners.onStatus,
        listeners.onAcp,
        listeners.onBound,
        listeners.leaseId,
      );
    } catch (error) {
      const wasCurrent = subscriptions.rollback(attempt);
      listenerWiring.dropPodIfUnused(podKey);
      void appState.relayUnsubscribe(podKey, id).catch((cleanupError: unknown) =>
        logEvent("warn", "relay", `rollback ${podKey}/${id} failed: ${cleanupError}`),
      );
      if (!wasCurrent) return false;
      throw error;
    }
    const { committed, replaced } = subscriptions.commit(attempt);
    if (!committed) {
      void appState.relayUnsubscribe(podKey, id).catch((error: unknown) =>
        logEvent("warn", "relay", `superseded ${podKey}/${id} failed: ${error}`),
      );
      return false;
    }
    if (replaced) {
      void appState.relayUnsubscribe(podKey, replaced).catch((error: unknown) =>
        logEvent("warn", "relay", `replace ${podKey}/${replaced} failed: ${error}`),
      );
    }
    return true;
  };

  const unsubscribeMany = (podKey: string, ids: string[]) =>
    Promise.all(ids.map((id) => appState.relayUnsubscribe(podKey, id))).then(() => undefined);

  const unsubscribe = (wcId: number, podKey: string, subId: string) => {
    const removed = subscriptions.take(wcId, podKey, subId);
    if (!removed) return undefined;
    if (removed.podUnused) listenerWiring.dropPod(podKey);
    logEvent("debug", "relay", `unsubscribe ${podKey}/${subId} (wc ${wcId})`);
    return unsubscribeMany(podKey, removed.coreSubIds);
  };

  // Drop one window's subscriptions without disturbing other windows sharing the
  // pod. If it was the final window, preserve disconnect's immediate teardown.
  const disconnectPodForWc = (wcId: number, podKey: string) => {
    const removed = subscriptions.takePod(wcId, podKey);
    if (!removed) return undefined;
    if (!removed.podUnused) return unsubscribeMany(podKey, removed.coreSubIds);
    listenerWiring.dropPod(podKey);
    return appState.relayDisconnect(podKey);
  };

  const releaseWebContents = (wcId: number) => {
    for (const removed of subscriptions.takeWindow(wcId)) {
      if (removed.podUnused) listenerWiring.dropPod(removed.podKey);
      for (const id of removed.coreSubIds) {
        const { podKey } = removed;
        void appState.relayUnsubscribe(podKey, id).catch((error: unknown) =>
          logEvent("warn", "relay", `relayUnsubscribe ${podKey}/${id} failed: ${error}`),
        );
      }
    }
  };

  registerRelayHandler("relay:subscribe", (e, ...args) =>
    subscribe(e.sender.id, ...args),
  );
  registerRelayHandler("relay:unsubscribe", (e, podKey, subId) =>
    unsubscribe(e.sender.id, podKey, subId),
  );
  registerRelayHandler("relay:disconnect", (e, podKey) => disconnectPodForWc(e.sender.id, podKey));
  // beforeunload fires this from the unloading window — release just its subs.
  registerRelayHandler("relay:disconnectAll", (e) => releaseWebContents(e.sender.id));
  registerRelayHandler("relay:send", (_e, podKey, data) => appState.relaySend(podKey, data));
  registerRelayHandler("relay:resize", (_e, podKey, cols, rows) =>
    appState.relaySendResize(podKey, cols, rows));
  registerRelayHandler("relay:forceResize", (_e, podKey, cols, rows) =>
    appState.relayForceResize(podKey, cols, rows));
  registerRelayHandler("relay:acpCommand", (_e, podKey, command) =>
    appState.relaySendAcpCommand(podKey, command));
  registerRelayHandler("relay:getStatus", (_e, podKey) => appState.relayGetStatus(podKey));
  registerRelayHandler("relay:isRunnerDisconnected", (_e, podKey) =>
    appState.relayIsRunnerDisconnected(podKey));
  registerRelayHandler("relay:getPodSize", (_e, podKey) => appState.relayGetPodSize(podKey));

  void appState.relayOnPodDisconnected((_e: unknown, raw: string) =>
    listenerWiring.handleDriverDisconnected(raw));

  const allChannels = [
    "relay:subscribe", "relay:unsubscribe", "relay:disconnect", "relay:disconnectAll",
    "relay:send", "relay:resize", "relay:forceResize", "relay:acpCommand",
    "relay:getStatus", "relay:isRunnerDisconnected", "relay:getPodSize",
  ] satisfies RelayInvokeChannel[];
  return {
    releaseWebContents,
    dispose: () => {
      for (const channel of allChannels) ipcMain.removeHandler(channel);
      subscriptions.clear();
      listenerWiring.clear();
      void appState.relayDisconnectAll().catch((e: unknown) =>
        logEvent("warn", "relay", `relayDisconnectAll failed: ${e}`),
      );
    },
  };
}
