export interface RelayInvokeContract {
  "relay:subscribe": {
    args: [podKey: string, subId: string, attemptId: string, url: string, token: string];
    result: boolean | undefined;
  };
  "relay:unsubscribe": { args: [podKey: string, subId: string]; result: void };
  "relay:disconnect": { args: [podKey: string]; result: void };
  "relay:disconnectAll": { args: []; result: void };
  "relay:send": { args: [podKey: string, data: string]; result: void };
  "relay:resize": { args: [podKey: string, cols: number, rows: number]; result: void };
  "relay:forceResize": { args: [podKey: string, cols: number, rows: number]; result: void };
  "relay:acpCommand": { args: [podKey: string, command: string]; result: void };
  "relay:getStatus": { args: [podKey: string]; result: string };
  "relay:isRunnerDisconnected": { args: [podKey: string]; result: boolean };
  "relay:getPodSize": { args: [podKey: string]; result: number[] };
}

export type RelayInvokeChannel = keyof RelayInvokeContract;
export type RelayInvokeArgs<C extends RelayInvokeChannel> = RelayInvokeContract[C]["args"];
export type RelayInvokeResult<C extends RelayInvokeChannel> = RelayInvokeContract[C]["result"];
export type RelaySubscribeArgs = RelayInvokeArgs<"relay:subscribe">;
export type RelaySubscribeResult = RelayInvokeResult<"relay:subscribe">;

export interface RelayOutputPayload {
  podKey: string;
  subId: string;
  attemptId: string;
  data: Uint8Array;
}

export interface RelayStatusPayload {
  podKey: string;
  json: string;
}

export interface RelayAcpPayload {
  podKey: string;
  json: string;
}

export interface RelayPodDisconnectedPayload {
  podKey: string;
  generation: number;
}

export interface RelayPushPayloads {
  "relay:output": RelayOutputPayload;
  "relay:status": RelayStatusPayload;
  "relay:acp": RelayAcpPayload;
  "relay:pod-disconnected": RelayPodDisconnectedPayload;
}

export type RelayPushChannel = keyof RelayPushPayloads;
export type RelayPushPayload<C extends RelayPushChannel> = RelayPushPayloads[C];

type RelayPushSubscription<P> = (handler: (payload: P) => void) => () => void;

export interface RelayPushApi {
  onRelayOutput: RelayPushSubscription<RelayOutputPayload>;
  onRelayStatus: RelayPushSubscription<RelayStatusPayload>;
  onRelayAcp: RelayPushSubscription<RelayAcpPayload>;
  onRelayPodDisconnected: RelayPushSubscription<RelayPodDisconnectedPayload>;
}
