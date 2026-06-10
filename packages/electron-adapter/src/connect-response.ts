// Normalize a `connectCall` IPC response to bytes. The main process returns a
// number[] (`Array.from(bytes)`); in-process fakes may hand back a Uint8Array.
// Anything else (null / undefined / plain object) is a malformed payload — fail
// closed so it surfaces here instead of decoding to a corrupt all-defaults proto.
export function coerceConnectResponse(resp: number[] | Uint8Array): Uint8Array {
  if (resp instanceof Uint8Array) return resp;
  if (Array.isArray(resp)) return new Uint8Array(resp);
  throw new Error("connect-fallback: IPC response was not binary");
}
