import { describe, it, expect, vi, beforeEach } from "vitest";

// Connect-RPC transport mock — support-tickets fully migrated.
const mockCallConnect = vi.fn();
vi.mock("@/lib/connect/transport", () => ({
  callConnect: (...args: unknown[]) => mockCallConnect(...args),
  ConnectError: class ConnectError extends Error {
    code: string;
    status: number;
    constructor(msg: string, code: string, status: number) {
      super(msg);
      this.code = code;
      this.status = status;
    }
  },
}));

// REST apiClient still used by the multipart Reply fallback (Connect has
// no multipart story).
const mockPostFormData = vi.fn();
vi.mock("../base", () => ({
  apiClient: {
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

const SERVICE = "proto.support_ticket.v1.SupportTicketAdminService";

describe("Admin API - Support Tickets (Connect-RPC)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("listSupportTickets calls ListSupportTickets and converts to snake_case", async () => {
    mockCallConnect.mockResolvedValue({
      data: [
        {
          id: 1n,
          userId: 5n,
          title: "Help",
          category: "bug",
          status: "open",
          priority: "high",
          createdAt: "2026-01-01T00:00:00Z",
          updatedAt: "2026-01-02T00:00:00Z",
          assignedAdminId: 9n,
        },
      ],
      total: 1n,
      page: 1,
      pageSize: 20,
      totalPages: 1,
    });
    const result = await listSupportTickets({ status: "open", page: 1, page_size: 20 });
    expect(mockCallConnect.mock.calls[0][0]).toBe(SERVICE);
    expect(mockCallConnect.mock.calls[0][1]).toBe("ListSupportTickets");
    expect(mockCallConnect.mock.calls[0][4]).toMatchObject({
      status: "open",
      page: 1,
      pageSize: 20,
    });
    expect(result.total).toBe(1);
    expect(result.data[0]).toMatchObject({
      id: 1,
      user_id: 5,
      status: "open",
      assigned_admin_id: 9,
    });
  });

  it("getSupportTicketStats calls GetSupportTicketStats", async () => {
    mockCallConnect.mockResolvedValue({
      total: 10n,
      open: 3n,
      inProgress: 2n,
      resolved: 4n,
      closed: 1n,
    });
    const stats = await getSupportTicketStats();
    expect(mockCallConnect.mock.calls[0][1]).toBe("GetSupportTicketStats");
    expect(stats).toEqual({ total: 10, open: 3, in_progress: 2, resolved: 4, closed: 1 });
  });

  it("getSupportTicketDetail calls GetSupportTicket and unwraps detail", async () => {
    mockCallConnect.mockResolvedValue({
      ticket: {
        id: 7n,
        userId: 1n,
        title: "T",
        category: "bug",
        status: "open",
        priority: "low",
        createdAt: "2026-01-01T00:00:00Z",
        updatedAt: "2026-01-01T00:00:00Z",
      },
      messages: [],
    });
    const detail = await getSupportTicketDetail(7);
    expect(mockCallConnect.mock.calls[0][1]).toBe("GetSupportTicket");
    expect(mockCallConnect.mock.calls[0][4]).toEqual({ id: 7n });
    expect(detail.ticket.id).toBe(7);
    expect(detail.messages).toEqual([]);
  });

  it("getSupportTicketMessages calls ListSupportTicketMessages", async () => {
    mockCallConnect.mockResolvedValue({ data: [] });
    const out = await getSupportTicketMessages(7);
    expect(mockCallConnect.mock.calls[0][1]).toBe("ListSupportTicketMessages");
    expect(mockCallConnect.mock.calls[0][4]).toEqual({ id: 7n });
    expect(out.messages).toEqual([]);
  });

  it("replySupportTicket calls ReplySupportTicket on JSON-only path", async () => {
    mockCallConnect.mockResolvedValue({
      id: 100n,
      ticketId: 7n,
      userId: 1n,
      content: "hi",
      isAdminReply: true,
      createdAt: "2026-01-01T00:00:00Z",
      attachments: [],
    });
    const msg = await replySupportTicket(7, "hi");
    expect(mockCallConnect.mock.calls[0][1]).toBe("ReplySupportTicket");
    expect(mockCallConnect.mock.calls[0][4]).toEqual({ id: 7n, content: "hi" });
    expect(msg.id).toBe(100);
  });

  it("replySupportTicket falls back to multipart REST when files attached", async () => {
    mockPostFormData.mockResolvedValue({ id: 50 });
    const file = new File(["x"], "x.txt", { type: "text/plain" });
    await replySupportTicket(7, "hi", [file]);
    expect(mockPostFormData).toHaveBeenCalledWith(
      "/support-tickets/7/reply",
      expect.any(FormData),
    );
    expect(mockCallConnect).not.toHaveBeenCalled();
  });

  it("updateSupportTicketStatus calls UpdateSupportTicketStatus", async () => {
    mockCallConnect.mockResolvedValue({ message: "ok" });
    const out = await updateSupportTicketStatus(7, "in_progress");
    expect(mockCallConnect.mock.calls[0][1]).toBe("UpdateSupportTicketStatus");
    expect(mockCallConnect.mock.calls[0][4]).toEqual({ id: 7n, status: "in_progress" });
    expect(out.message).toBe("ok");
  });

  it("assignSupportTicket calls AssignSupportTicket", async () => {
    mockCallConnect.mockResolvedValue({ message: "assigned" });
    await assignSupportTicket(7, 42);
    expect(mockCallConnect.mock.calls[0][1]).toBe("AssignSupportTicket");
    expect(mockCallConnect.mock.calls[0][4]).toEqual({ id: 7n, adminId: 42n });
  });

  it("getSupportTicketAttachmentUrl calls GetSupportTicketAttachmentUrl", async () => {
    mockCallConnect.mockResolvedValue({ url: "https://s3.example.com/file.png" });
    const out = await getSupportTicketAttachmentUrl(99);
    expect(mockCallConnect.mock.calls[0][1]).toBe("GetSupportTicketAttachmentUrl");
    expect(mockCallConnect.mock.calls[0][4]).toEqual({ attachmentId: 99n });
    expect(out.url).toBe("https://s3.example.com/file.png");
  });
});
