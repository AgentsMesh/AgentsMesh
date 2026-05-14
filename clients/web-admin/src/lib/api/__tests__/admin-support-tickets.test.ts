import { describe, it, expect, vi, beforeEach } from "vitest";

// Mock the apiClient
const mockGet = vi.fn();
const mockPost = vi.fn();
const mockPut = vi.fn();
const mockPatch = vi.fn();
const mockPostFormData = vi.fn();

vi.mock("../base", () => ({
  apiClient: {
    get: (...args: unknown[]) => mockGet(...args),
    post: (...args: unknown[]) => mockPost(...args),
    put: (...args: unknown[]) => mockPut(...args),
    patch: (...args: unknown[]) => mockPatch(...args),
    postFormData: (...args: unknown[]) => mockPostFormData(...args),
  },
}));

import {
  listSupportTickets,
  getSupportTicketStats,
  getSupportTicketDetail,
  getSupportTicketMessages,
  replySupportTicket,
  updateSupportTicketStatus,
  assignSupportTicket,
  getSupportTicketAttachmentUrl,
} from "../admin";

describe("Admin API - Support Tickets", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("listSupportTickets calls GET /support-tickets with params", async () => {
    mockGet.mockResolvedValue({ data: [], total: 0 });
    await listSupportTickets({ status: "open", page: 1 });
    expect(mockGet).toHaveBeenCalledWith(
      "/support-tickets",
      expect.objectContaining({ status: "open", page: 1 })
    );
  });

  it("getSupportTicketStats calls GET /support-tickets/stats", async () => {
    mockGet.mockResolvedValue({ total: 10, open: 3 });
    await getSupportTicketStats();
    expect(mockGet).toHaveBeenCalledWith("/support-tickets/stats");
  });

  it("getSupportTicketDetail calls GET /support-tickets/:id", async () => {
    mockGet.mockResolvedValue({ ticket: {}, messages: [] });
    await getSupportTicketDetail(7);
    expect(mockGet).toHaveBeenCalledWith("/support-tickets/7");
  });

  it("getSupportTicketMessages calls GET /support-tickets/:id/messages", async () => {
    mockGet.mockResolvedValue({ data: [] });
    await getSupportTicketMessages(7);
    expect(mockGet).toHaveBeenCalledWith("/support-tickets/7/messages");
  });

  it("replySupportTicket calls postFormData /support-tickets/:id/reply", async () => {
    mockPostFormData.mockResolvedValue({ id: 1 });
    await replySupportTicket(7, "hello");
    expect(mockPostFormData).toHaveBeenCalledWith(
      "/support-tickets/7/reply",
      expect.any(FormData)
    );
  });

  it("updateSupportTicketStatus calls PATCH /support-tickets/:id/status", async () => {
    mockPatch.mockResolvedValue({ id: 1 });
    await updateSupportTicketStatus(7, "in_progress");
    expect(mockPatch).toHaveBeenCalledWith(
      "/support-tickets/7/status",
      { status: "in_progress" }
    );
  });

  it("assignSupportTicket calls POST /support-tickets/:id/assign", async () => {
    mockPost.mockResolvedValue({ id: 1 });
    await assignSupportTicket(7, 42);
    expect(mockPost).toHaveBeenCalledWith(
      "/support-tickets/7/assign",
      { admin_id: 42 }
    );
  });

  it("getSupportTicketAttachmentUrl calls GET /support-tickets/attachments/:id/url", async () => {
    mockGet.mockResolvedValue({ url: "https://s3.example.com/file.png" });
    const result = await getSupportTicketAttachmentUrl(99);
    expect(mockGet).toHaveBeenCalledWith("/support-tickets/attachments/99/url");
    expect(result.url).toBe("https://s3.example.com/file.png");
  });
});
