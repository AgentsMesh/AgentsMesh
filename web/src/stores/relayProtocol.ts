/**
 * Relay binary protocol types and message encoding/decoding.
 *
 * The browser communicates with Relay using a binary protocol where each
 * message is prefixed with a 1-byte message type followed by a variable-length payload.
 *
 * Must match relay/internal/protocol/message.go
 */

/**
 * Relay message types (binary protocol)
 */
export const MsgType = {
  Snapshot: 0x01,           // Complete terminal snapshot
  Output: 0x02,             // Terminal output (raw pod data)
  Input: 0x03,              // User input to terminal
  Resize: 0x04,             // Terminal resize
  Ping: 0x05,               // Ping for keepalive
  Pong: 0x06,               // Pong response
  Control: 0x07,            // Control messages (JSON)
  RunnerDisconnected: 0x08, // Runner disconnected notification
  RunnerReconnected: 0x09,  // Runner reconnected notification
  Resync: 0x0a,             // Resync request: ask relay to replay buffered output
  AcpEvent: 0x0b,           // Runner → Browser, ACP event (JSON)
  AcpCommand: 0x0c,         // Browser → Runner, ACP command (JSON)
  AcpSnapshot: 0x0d,        // Runner → Browser, ACP session snapshot (JSON)
} as const;

/**
 * Encode a message with type prefix (Relay binary protocol).
 *
 * Wire format: [msgType: 1 byte][payload: N bytes]
 */
export function encodeMessage(msgType: number, payload: Uint8Array | string): Uint8Array {
  const payloadBytes = typeof payload === "string"
    ? new TextEncoder().encode(payload)
    : payload;
  const message = new Uint8Array(1 + payloadBytes.length);
  message[0] = msgType;
  message.set(payloadBytes, 1);
  return message;
}

/**
 * Decode a message with type prefix (Relay binary protocol).
 *
 * @returns Object with `type` (1-byte message type) and `payload` (remaining bytes).
 */
export function decodeMessage(data: Uint8Array): { type: number; payload: Uint8Array } {
  if (data.length < 1) {
    return { type: 0, payload: new Uint8Array(0) };
  }
  return {
    type: data[0],
    payload: data.slice(1),
  };
}

/**
 * Encode a resize payload as 4 bytes: cols (uint16 BE) + rows (uint16 BE).
 */
export function encodeResize(cols: number, rows: number): Uint8Array {
  const payload = new Uint8Array(4);
  payload[0] = (cols >> 8) & 0xff;
  payload[1] = cols & 0xff;
  payload[2] = (rows >> 8) & 0xff;
  payload[3] = rows & 0xff;
  return payload;
}

/**
 * Encode a JSON object as a typed relay message (e.g. AcpCommand).
 */
export function encodeJsonMessage(msgType: number, obj: unknown): Uint8Array {
  return encodeMessage(msgType, JSON.stringify(obj));
}

/**
 * Decode a JSON payload from a relay message.
 * Returns null if the payload cannot be parsed.
 */
export function decodeJsonPayload<T = unknown>(payload: Uint8Array): T | null {
  try {
    return JSON.parse(new TextDecoder().decode(payload)) as T;
  } catch {
    return null;
  }
}
