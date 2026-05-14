// Minimal Connect-RPC JSON client for web-admin. Mirrors the
// `@connectrpc/connect-web` shape we need without taking a dep, since
// web-admin doesn't run the wasm bridge that clients/web uses.
//
// Wire format: HTTP POST to /proto.<package>.<Service>/<Method>
//   Content-Type: application/json
//   Authorization: Bearer <jwt>            (when present)
//   Body: protojson-encoded request
//   Response 200: protojson-encoded response
//   Response 4xx/5xx: { code, message } (Connect error envelope)
//
// We use binary-format (toBinary/fromBinary) instead of JSON because the
// server's Connect handlers accept binary by default and the gen TS uses
// the @bufbuild/protobuf v2 runtime that exposes both. JSON also works,
// but binary keeps wire parity with clients/web.
import { create, toBinary, fromBinary, type DescMessage, type MessageInitShape, type MessageShape } from "@bufbuild/protobuf";

import { getAuthToken, useAuthStore } from "@/stores/auth";

function getOrigin(): string {
  if (typeof window !== "undefined") {
    return window.location.origin;
  }
  const primary = process.env.NEXT_PUBLIC_PRIMARY_DOMAIN;
  if (primary && !primary.startsWith("__")) {
    const useHttps = process.env.NEXT_PUBLIC_USE_HTTPS === "true";
    return `${useHttps ? "https" : "http"}://${primary}`;
  }
  return "http://localhost:10000";
}

export class ConnectError extends Error {
  readonly code: string;
  readonly status: number;
  constructor(message: string, code: string, status: number) {
    super(message);
    this.code = code;
    this.status = status;
  }
}

// callConnect invokes a Connect-RPC unary method. `service` is the proto
// fully-qualified name (e.g. "proto.admin.v1.AdminService"); `method` is
// the PascalCase RPC name (e.g. "ListUsers"). Schemas come from the
// generated TS — pass the *Schema constants directly.
export async function callConnect<I extends DescMessage, O extends DescMessage>(
  service: string,
  method: string,
  inputSchema: I,
  outputSchema: O,
  input: MessageInitShape<I>,
): Promise<MessageShape<O>> {
  const msg = create(inputSchema, input);
  const body = toBinary(inputSchema, msg);

  const headers: Record<string, string> = {
    "Content-Type": "application/proto",
  };
  const token = getAuthToken();
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const url = `${getOrigin()}/${service}/${method}`;
  const resp = await fetch(url, {
    method: "POST",
    headers,
    body,
  });

  if (resp.status === 401) {
    useAuthStore.getState().logout();
    throw new ConnectError("Session expired. Please login again.", "unauthenticated", 401);
  }

  if (!resp.ok) {
    // Connect error envelope is JSON regardless of request codec.
    let detail = `HTTP ${resp.status}`;
    let code = "unknown";
    try {
      const err = (await resp.json()) as { code?: string; message?: string };
      detail = err.message || detail;
      code = err.code || code;
    } catch {
      // body wasn't JSON — keep default detail/code
    }
    throw new ConnectError(detail, code, resp.status);
  }

  const buf = new Uint8Array(await resp.arrayBuffer());
  return fromBinary(outputSchema, buf);
}
