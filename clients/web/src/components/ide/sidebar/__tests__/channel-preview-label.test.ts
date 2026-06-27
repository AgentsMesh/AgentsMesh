import { describe, it, expect } from "vitest";
import { previewLabel } from "../channel-preview-label";
import type { ChannelLastMessage } from "@/stores/channel";

const t = (k: string) => k; // stub translator returns the key
function lm(o: Partial<ChannelLastMessage>): ChannelLastMessage {
  return { sender_name: "", content_preview: "", timestamp: "", ...o };
}

describe("previewLabel", () => {
  it("attachment → localized label, not raw body", () => {
    expect(previewLabel(lm({ message_type: "attachment", content_preview: "file.png" }), t)).toBe("typeAttachment");
  });

  it("code → localized label (overrides Rust's raw content_preview)", () => {
    expect(previewLabel(lm({ message_type: "code", content_preview: "fn main(){}" }), t)).toBe("typeCode");
  });

  it("command → localized label", () => {
    expect(previewLabel(lm({ message_type: "command", content_preview: "/deploy" }), t)).toBe("typeCommand");
  });

  it("text → the authoritative Rust content_preview", () => {
    expect(previewLabel(lm({ message_type: "text", content_preview: "hello world" }), t)).toBe("hello world");
  });

  it("unknown / missing type → content_preview fallback", () => {
    expect(previewLabel(lm({ content_preview: "fallback" }), t)).toBe("fallback");
  });
});
