import { describe, it, expect } from "vitest";
import { MsgType, encodeMessage, decodeMessage, encodeResize } from "../relayProtocol";

describe("MsgType", () => {
  it("defines expected message type constants", () => {
    expect(MsgType.Snapshot).toBe(0x01);
    expect(MsgType.Output).toBe(0x02);
    expect(MsgType.Input).toBe(0x03);
    expect(MsgType.Resize).toBe(0x04);
    expect(MsgType.Ping).toBe(0x05);
    expect(MsgType.Pong).toBe(0x06);
    expect(MsgType.Control).toBe(0x07);
    expect(MsgType.RunnerDisconnected).toBe(0x08);
    expect(MsgType.RunnerReconnected).toBe(0x09);
    expect(MsgType.Resync).toBe(0x0a);
  });
});

describe("encodeMessage", () => {
  it("encodes a Uint8Array payload with type prefix", () => {
    const payload = new Uint8Array([0x48, 0x69]); // "Hi"
    const result = encodeMessage(MsgType.Output, payload);
    expect(result).toEqual(new Uint8Array([0x02, 0x48, 0x69]));
  });

  it("encodes a string payload with type prefix", () => {
    const result = encodeMessage(MsgType.Input, "A");
    expect(result[0]).toBe(0x03);
    expect(result[1]).toBe(0x41); // "A"
    expect(result.length).toBe(2);
  });

  it("encodes empty payload", () => {
    const result = encodeMessage(MsgType.Resync, new Uint8Array(0));
    expect(result).toEqual(new Uint8Array([0x0a]));
  });

  it("encodes multi-byte UTF-8 string", () => {
    const result = encodeMessage(MsgType.Input, "你");
    expect(result[0]).toBe(0x03);
    // "你" is 3 bytes in UTF-8
    expect(result.length).toBe(4);
  });
});

describe("decodeMessage", () => {
  it("decodes a message with type and payload", () => {
    const data = new Uint8Array([0x02, 0x48, 0x69]);
    const { type, payload } = decodeMessage(data);
    expect(type).toBe(0x02);
    expect(payload).toEqual(new Uint8Array([0x48, 0x69]));
  });

  it("decodes a type-only message (no payload)", () => {
    const data = new Uint8Array([0x06]);
    const { type, payload } = decodeMessage(data);
    expect(type).toBe(0x06);
    expect(payload.length).toBe(0);
  });

  it("returns type 0 and empty payload for empty data", () => {
    const data = new Uint8Array(0);
    const { type, payload } = decodeMessage(data);
    expect(type).toBe(0);
    expect(payload.length).toBe(0);
  });
});

describe("encodeResize", () => {
  it("encodes cols and rows as uint16 big-endian", () => {
    const result = encodeResize(80, 24);
    expect(result).toEqual(new Uint8Array([0x00, 0x50, 0x00, 0x18]));
  });

  it("encodes large dimensions correctly", () => {
    const result = encodeResize(300, 100);
    // 300 = 0x012C, 100 = 0x0064
    expect(result).toEqual(new Uint8Array([0x01, 0x2c, 0x00, 0x64]));
  });

  it("encodes zero dimensions", () => {
    const result = encodeResize(0, 0);
    expect(result).toEqual(new Uint8Array([0x00, 0x00, 0x00, 0x00]));
  });

  it("produces a valid resize message when combined with encodeMessage", () => {
    const msg = encodeMessage(MsgType.Resize, encodeResize(80, 24));
    expect(msg[0]).toBe(0x04); // MsgType.Resize
    expect(msg[1]).toBe(0x00); // cols high byte
    expect(msg[2]).toBe(0x50); // cols low byte (80)
    expect(msg[3]).toBe(0x00); // rows high byte
    expect(msg[4]).toBe(0x18); // rows low byte (24)
    expect(msg.length).toBe(5);
  });
});
