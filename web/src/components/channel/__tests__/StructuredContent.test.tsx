import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { StructuredContent } from "../StructuredContent";
import type { MessageContent } from "@/lib/api/channel-message-types";

// Mock the pod store
const mockPods = [
  { pod_key: "pk-bot", alias: "MyBot", agent: { name: "Claude" } },
];

vi.mock("@/stores/pod", () => ({
  usePodStore: (selector: (s: { pods: typeof mockPods }) => unknown) =>
    selector({ pods: mockPods }),
  usePods: () => mockPods,
}));

vi.mock("@/lib/pod-display-name", () => ({
  getPodDisplayName: (pod: { alias?: string; pod_key: string }) =>
    pod.alias || pod.pod_key,
}));

describe("StructuredContent", () => {
  it("renders a paragraph with plain text", () => {
    const content: MessageContent = {
      kind: "text",
      blocks: [{ type: "paragraph", elements: [{ type: "text", text: "Hello world" }] }],
    };
    render(<StructuredContent content={content} />);
    expect(screen.getByText("Hello world")).toBeDefined();
  });

  it("renders multiple paragraphs", () => {
    const content: MessageContent = {
      kind: "text",
      blocks: [
        { type: "paragraph", elements: [{ type: "text", text: "First" }] },
        { type: "paragraph", elements: [{ type: "text", text: "Second" }] },
      ],
    };
    render(<StructuredContent content={content} />);
    expect(screen.getByText("First")).toBeDefined();
    expect(screen.getByText("Second")).toBeDefined();
  });

  it("renders a mention resolved from pod store", () => {
    const content: MessageContent = {
      kind: "text",
      blocks: [{
        type: "paragraph",
        elements: [
          { type: "text", text: "Hey " },
          { type: "mention", entity_type: "pod", entity_key: "pk-bot", display: "bot" },
        ],
      }],
    };
    render(<StructuredContent content={content} />);
    expect(screen.getByText("@MyBot")).toBeDefined();
  });

  it("falls back to display snapshot when pod not in store", () => {
    const content: MessageContent = {
      kind: "text",
      blocks: [{
        type: "paragraph",
        elements: [
          { type: "mention", entity_type: "pod", entity_key: "pk-unknown", display: "OldName" },
        ],
      }],
    };
    render(<StructuredContent content={content} />);
    expect(screen.getByText("@OldName")).toBeDefined();
  });

  it("renders a link element", () => {
    const content: MessageContent = {
      kind: "text",
      blocks: [{
        type: "paragraph",
        elements: [
          { type: "link", text: "click here", url: "https://example.com" },
        ],
      }],
    };
    render(<StructuredContent content={content} />);
    const link = screen.getByText("click here");
    expect(link.tagName).toBe("A");
    expect(link.getAttribute("href")).toBe("https://example.com");
  });

  it("renders bold and italic text", () => {
    const content: MessageContent = {
      kind: "text",
      blocks: [{
        type: "paragraph",
        elements: [
          { type: "text", text: "bold", bold: true },
          { type: "text", text: "italic", italic: true },
        ],
      }],
    };
    render(<StructuredContent content={content} />);
    const bold = screen.getByText("bold");
    expect(bold.closest("strong")).toBeDefined();
    const italic = screen.getByText("italic");
    expect(italic.closest("em")).toBeDefined();
  });

  it("renders code block", () => {
    const content: MessageContent = {
      kind: "text",
      blocks: [{ type: "code_block", text: "const x = 1;" }],
    };
    render(<StructuredContent content={content} />);
    expect(screen.getByText("const x = 1;")).toBeDefined();
  });

  it("returns null for empty blocks", () => {
    const content: MessageContent = { kind: "text", blocks: [] };
    const { container } = render(<StructuredContent content={content} />);
    expect(container.innerHTML).toBe("");
  });

  it("returns null for undefined blocks", () => {
    const content: MessageContent = { kind: "text" };
    const { container } = render(<StructuredContent content={content} />);
    expect(container.innerHTML).toBe("");
  });

  it("renders heading block", () => {
    const content: MessageContent = {
      kind: "text",
      blocks: [{ type: "heading", level: 2, elements: [{ type: "text", text: "Title" }] }],
    };
    render(<StructuredContent content={content} />);
    const el = screen.getByText("Title");
    expect(el.tagName).toBe("H2");
  });

  it("renders quote block", () => {
    const content: MessageContent = {
      kind: "text",
      blocks: [{ type: "quote", elements: [{ type: "text", text: "Quoted" }] }],
    };
    render(<StructuredContent content={content} />);
    const el = screen.getByText("Quoted");
    expect(el.closest("blockquote")).not.toBeNull();
  });

  it("renders ordered list block", () => {
    const content: MessageContent = {
      kind: "text",
      blocks: [{
        type: "list",
        ordered: true,
        items: [
          [{ type: "text", text: "First" }],
          [{ type: "text", text: "Second" }],
        ],
      }],
    };
    render(<StructuredContent content={content} />);
    expect(screen.getByText("First").closest("ol")).not.toBeNull();
    expect(screen.getByText("Second")).toBeDefined();
  });

  it("renders unordered list block", () => {
    const content: MessageContent = {
      kind: "text",
      blocks: [{
        type: "list",
        ordered: false,
        items: [[{ type: "text", text: "Bullet" }]],
      }],
    };
    render(<StructuredContent content={content} />);
    expect(screen.getByText("Bullet").closest("ul")).not.toBeNull();
  });

  it("renders linebreak element", () => {
    const content: MessageContent = {
      kind: "text",
      blocks: [{
        type: "paragraph",
        elements: [
          { type: "text", text: "before" },
          { type: "linebreak" },
          { type: "text", text: "after" },
        ],
      }],
    };
    const { container } = render(<StructuredContent content={content} />);
    expect(container.querySelector("br")).not.toBeNull();
  });

  it("handles unknown block type gracefully", () => {
    const content: MessageContent = {
      kind: "text",
      blocks: [{ type: "unknown_block" as "paragraph" }],
    };
    const { container } = render(<StructuredContent content={content} />);
    expect(container.querySelector("p")).toBeNull();
  });

  it("renders strikethrough and code inline text", () => {
    const content: MessageContent = {
      kind: "text",
      blocks: [{
        type: "paragraph",
        elements: [
          { type: "text", text: "deleted", strike: true },
          { type: "text", text: "monospace", code: true },
        ],
      }],
    };
    render(<StructuredContent content={content} />);
    expect(screen.getByText("deleted").closest("del")).not.toBeNull();
    expect(screen.getByText("monospace").closest("code")).not.toBeNull();
  });

  it("renders user mention with display fallback", () => {
    const content: MessageContent = {
      kind: "text",
      blocks: [{
        type: "paragraph",
        elements: [{ type: "mention", entity_type: "user", entity_key: "42", display: "alice" }],
      }],
    };
    render(<StructuredContent content={content} />);
    expect(screen.getByText("@alice")).toBeDefined();
  });
});
