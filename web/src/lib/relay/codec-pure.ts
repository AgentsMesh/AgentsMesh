// Pure-JS relay protocol codec. Mirrors core/crates/protocol/src/codec.rs.
// Desktop renderer connects directly to the relay WebSocket (no IPC hop),
// so it needs real encode/decode — the previous "no-op / return type=0"
// stub caused every inbound relay frame to log "Unknown message type: 0"
// and all outbound frames to be dropped.

const Msg = {
  Input: 0x03,
  Resize: 0x04,
  Ping: 0x05,
  Control: 0x07,
  Resync: 0x0a,
  AcpCommand: 0x0c,
} as const;

function prefixed(type: number, data: Uint8Array = new Uint8Array(0)): Uint8Array {
  const out = new Uint8Array(1 + data.length);
  out[0] = type;
  out.set(data, 1);
  return out;
}

export const relay_encode_input = (data: Uint8Array): Uint8Array => prefixed(Msg.Input, data);
export const relay_encode_ping = (): Uint8Array => prefixed(Msg.Ping);
export const relay_encode_control = (data: Uint8Array): Uint8Array => prefixed(Msg.Control, data);
export const relay_encode_resync = (): Uint8Array => prefixed(Msg.Resync);
export const relay_encode_acp_command = (data: Uint8Array): Uint8Array => prefixed(Msg.AcpCommand, data);

export function relay_encode_resize(cols: number, rows: number): Uint8Array {
  const buf = new Uint8Array(5);
  buf[0] = Msg.Resize;
  buf[1] = (cols >> 8) & 0xff;
  buf[2] = cols & 0xff;
  buf[3] = (rows >> 8) & 0xff;
  buf[4] = rows & 0xff;
  return buf;
}

export function relay_decode_message(data: Uint8Array): { type: number; payload: Uint8Array } | null {
  if (!data || data.length === 0) return null;
  return { type: data[0], payload: data.slice(1) };
}
