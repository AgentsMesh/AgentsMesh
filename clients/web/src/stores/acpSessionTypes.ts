export interface AcpContentChunk { text: string; role: string; timestamp: number; complete?: boolean }
export interface AcpToolCall { toolCallId: string; toolName: string; status: string; argumentsJson: string; resultText?: string; errorMessage?: string; success?: boolean; timestamp: number }
export interface AcpPlanStep { title: string; status: string }
export interface AcpThinking { text: string; timestamp: number; complete?: boolean }
export interface AcpPermissionRequest { requestId: string; toolName: string; argumentsJson: string; description: string }
export interface AcpLog { level: string; message: string; timestamp: number }
export interface AcpConfiguration { permissionMode: string; model: string; supportedPermissionModes: string[] }

export interface AcpSessionState {
  messages: AcpContentChunk[]; toolCalls: Record<string, AcpToolCall>; plan: AcpPlanStep[];
  thinkings: AcpThinking[]; logs: AcpLog[]; state: string; pendingPermissions: AcpPermissionRequest[];
  configuration: AcpConfiguration;
}

export const EMPTY_CONFIGURATION: AcpConfiguration = { permissionMode: "", model: "", supportedPermissionModes: [] };

export const EMPTY_SESSION: AcpSessionState = {
  messages: [], toolCalls: {}, plan: [], thinkings: [], logs: [], state: "idle", pendingPermissions: [],
  configuration: EMPTY_CONFIGURATION,
};

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function mapState(raw: any): string {
  if (typeof raw === "string") return raw;
  if (typeof raw === "object" && raw !== null) return Object.keys(raw)[0] || "idle";
  return "idle";
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function toolCallFromWasm(id: string, tc: any): AcpToolCall {
  return {
    toolCallId: tc.id ?? id,
    toolName: tc.name ?? "",
    status: tc.status ?? "",
    argumentsJson: tc.args ? JSON.stringify(tc.args) : "",
    resultText: tc.result_text,
    errorMessage: tc.error_message,
    success: tc.success,
    timestamp: tc.timestamp ?? 0,
  };
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function permReqFromWasm(p: any): AcpPermissionRequest {
  return {
    requestId: p.id ?? "",
    toolName: p.tool_name ?? "",
    argumentsJson: p.args ? JSON.stringify(p.args) : "",
    description: p.description ?? "",
  };
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function configurationFromWasm(raw: any): AcpConfiguration {
  if (!raw || typeof raw !== "object") return { ...EMPTY_CONFIGURATION };
  return {
    permissionMode: typeof raw.permission_mode === "string" ? raw.permission_mode : "",
    model: typeof raw.model === "string" ? raw.model : "",
    supportedPermissionModes: Array.isArray(raw.supported_permission_modes) ? raw.supported_permission_modes : [],
  };
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function configurationToWasmObj(c: AcpConfiguration): Record<string, any> {
  return { permission_mode: c.permissionMode, model: c.model, supported_permission_modes: c.supportedPermissionModes };
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function sessionFromWasm(raw: any): AcpSessionState {
  const tcEntries = Object.entries(raw.tool_calls || {});
  const toolCalls: Record<string, AcpToolCall> = {};
  for (const [k, v] of tcEntries) toolCalls[k] = toolCallFromWasm(k, v);
  return {
    messages: raw.messages || [],
    toolCalls,
    plan: raw.plan || [],
    thinkings: raw.thinkings || [],
    logs: raw.logs || [],
    state: mapState(raw.state),
    pendingPermissions: (raw.pending_permissions || []).map(permReqFromWasm),
    configuration: configurationFromWasm(raw.configuration),
  };
}

export function toolCallToWasm(tc: AcpToolCall): string {
  return JSON.stringify({
    id: tc.toolCallId,
    name: tc.toolName,
    status: tc.status,
    args: tc.argumentsJson ? JSON.parse(tc.argumentsJson) : null,
    result_text: tc.resultText,
    error_message: tc.errorMessage,
    success: tc.success,
    timestamp: tc.timestamp,
  });
}

export function permReqToWasm(req: AcpPermissionRequest): string {
  return JSON.stringify({
    id: req.requestId,
    tool_name: req.toolName,
    args: req.argumentsJson ? JSON.parse(req.argumentsJson) : null,
    description: req.description,
  });
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function toolCallToWasmObj(tc: AcpToolCall): Record<string, any> {
  return {
    id: tc.toolCallId,
    name: tc.toolName,
    status: tc.status,
    args: tc.argumentsJson ? JSON.parse(tc.argumentsJson) : null,
    result_text: tc.resultText,
    error_message: tc.errorMessage,
    success: tc.success,
    timestamp: tc.timestamp,
  };
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function permReqToWasmObj(req: AcpPermissionRequest): Record<string, any> {
  return {
    id: req.requestId,
    tool_name: req.toolName,
    args: req.argumentsJson ? JSON.parse(req.argumentsJson) : null,
    description: req.description,
  };
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function wasmFromSession(s: AcpSessionState): Record<string, any> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const toolCalls: Record<string, any> = {};
  for (const [k, v] of Object.entries(s.toolCalls)) toolCalls[k] = toolCallToWasmObj(v);
  return {
    messages: s.messages,
    tool_calls: toolCalls,
    plan: s.plan,
    thinkings: s.thinkings,
    logs: s.logs,
    state: s.state,
    pending_permissions: s.pendingPermissions.map(permReqToWasmObj),
    configuration: configurationToWasmObj(s.configuration),
  };
}
