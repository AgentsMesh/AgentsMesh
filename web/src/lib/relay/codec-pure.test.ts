import { describe, it, expect } from "vitest";
import {
  relay_encode_input,
  relay_encode_resize,
  relay_encode_ping,
  relay_encode_control,
  relay_encode_resync,
  relay_encode_acp_command,
  relay_decode_message,
} from "./codec-pure";

// Contract test: desktop renderer owns the relay WebSocket directly, so
// its pure-JS codec must byte-align with core/crates/protocol/src/codec.rs.
// Regression: a previous stub returned `{type: 0, data: empty}` for every
// inbound frame, which the dispatcher rejected as "Unknown message type 0"
// — 226 times per workspace open. Any drift from Rust protocol (msg_type
// byte values, frame layout) immediately surfaces here.

describe("desktop relay-codec · byte contract", () => {
  it("decode_message extracts [type, payload]", () => {
    const frame = new Uint8Array([0x02, 1, 2, 3]);
    const out = relay_decode_message(frame);
    expect(out?.type).toBe(0x02);
    expect(Array.from(out!.payload)).toEqual([1, 2, 3]);
  });

  it("decode_message returns null on empty input", () => {
    expect(relay_decode_message(new Uint8Array(0))).toBeNull();
  });

  it("encode_input prefixes 0x03", () => {
    const out = relay_encode_input(new Uint8Array([0x41, 0x42]));
    expect(Array.from(out)).toEqual([0x03, 0x41, 0x42]);
  });

  it("encode_resize encodes cols/rows as big-endian u16", () => {
    const out = relay_encode_resize(80, 24);
    expect(Array.from(out)).toEqual([0x04, 0, 80, 0, 24]);

    const wide = relay_encode_resize(500, 300);
    // 500 = 0x01F4, 300 = 0x012C
    expect(Array.from(wide)).toEqual([0x04, 0x01, 0xF4, 0x01, 0x2C]);
  });

  it("single-byte encoders emit the expected marker", () => {
    expect(Array.from(relay_encode_ping())).toEqual([0x05]);
    expect(Array.from(relay_encode_resync())).toEqual([0x0a]);
  });

  it("prefixed encoders preserve payload bytes", () => {
    expect(Array.from(relay_encode_control(new Uint8Array([9])))).toEqual([0x07, 9]);
    expect(Array.from(relay_encode_acp_command(new Uint8Array([1, 2])))).toEqual([0x0c, 1, 2]);
  });
});
