import { describe, expect, it } from "vitest";
import {
  parsePodAcpEvent,
  parsePodDriverDisconnected,
  parsePodStatusEvent,
} from "./relay_listener_events";

describe("relay listener event parsing", () => {
  it("accepts well-formed driver, status, and ACP envelopes", () => {
    expect(parsePodDriverDisconnected('{"podKey":"pod-1","generation":2}')).toEqual({
      podKey: "pod-1",
      generation: 2,
    });
    expect(parsePodStatusEvent('{"generation":2,"revision":0,"status":"connected"}')).toEqual({
      generation: 2,
      revision: 0,
    });
    expect(parsePodAcpEvent('{"generation":2,"msgType":13}')).toEqual({ generation: 2 });
  });

  it.each([
    "not-json",
    "{}",
    '{"podKey":1,"generation":1}',
    '{"podKey":"pod-1","generation":0}',
    '{"podKey":"pod-1","generation":1.5}',
  ])("rejects malformed driver disconnect envelope %s", (raw) => {
    expect(parsePodDriverDisconnected(raw)).toBeNull();
  });

  it.each([
    "not-json",
    "{}",
    '{"generation":0,"revision":0}',
    '{"generation":1.5,"revision":0}',
    '{"generation":1,"revision":-1}',
    '{"generation":1,"revision":1.5}',
  ])("rejects malformed status envelope %s", (raw) => {
    expect(parsePodStatusEvent(raw)).toBeNull();
  });

  it.each([
    "not-json",
    "{}",
    '{"generation":0}',
    '{"generation":1.5}',
  ])("rejects malformed ACP envelope %s", (raw) => {
    expect(parsePodAcpEvent(raw)).toBeNull();
  });
});
