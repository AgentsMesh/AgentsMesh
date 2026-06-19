// Facade re-export of the pod Connect-RPC adapter. Business code imports
// from here (or from the `@/lib/api` barrel) so the wire-shape layer stays
// internal to the facade boundary. Tests mock this path; the underlying
// `connect/podConnect.ts` remains the SSOT for proto encode/decode.

export {
  fromProtoPod,
  listPods,
  listPodsRaw,
  getPod,
  createPod,
  terminatePod,
  updatePodAlias,
  updatePodPerpetual,
  getPodConnection,
  sendPodPrompt,
  listPodsByTicket,
  type CreatePodInput,
  type PodConnectionInfo,
} from "../connect/podConnect";
