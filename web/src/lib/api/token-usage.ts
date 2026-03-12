import { request, orgPath } from "./base";

// Token usage types — aligned with Backend domain/tokenusage structs

export interface TokenUsageSummary {
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_creation_tokens: number;
  total_tokens: number;
}

export interface TokenUsageTimeSeriesPoint {
  period: string; // ISO 8601 timestamp from date_trunc
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_creation_tokens: number;
}

export interface TokenUsageByAgent {
  agent_type: string; // agent_type_slug
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_creation_tokens: number;
  total_tokens: number;
}

export interface TokenUsageByUser {
  user_id: number;
  username: string;
  email: string;
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_creation_tokens: number;
  total_tokens: number;
}

export interface TokenUsageByModel {
  model: string;
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_creation_tokens: number;
  total_tokens: number;
}

export interface TokenUsageQueryParams {
  start_time?: string;
  end_time?: string;
  agent_type?: string;
  user_id?: number;
  model?: string;
  granularity?: "day" | "week" | "month";
}

// TokenUsageDashboard is the response from GET /token-usage/dashboard.
export interface TokenUsageDashboard {
  summary: TokenUsageSummary;
  time_series: TokenUsageTimeSeriesPoint[];
  by_agent: TokenUsageByAgent[];
  by_user: TokenUsageByUser[];
  by_model: TokenUsageByModel[];
}

function buildQueryString(params: TokenUsageQueryParams): string {
  const searchParams = new URLSearchParams();
  if (params.start_time) searchParams.append("start_time", params.start_time);
  if (params.end_time) searchParams.append("end_time", params.end_time);
  if (params.agent_type) searchParams.append("agent_type", params.agent_type);
  if (params.user_id !== undefined) searchParams.append("user_id", String(params.user_id));
  if (params.model) searchParams.append("model", params.model);
  if (params.granularity) searchParams.append("granularity", params.granularity);
  const qs = searchParams.toString();
  return qs ? `?${qs}` : "";
}

// Token Usage API — single dashboard endpoint.
export const tokenUsageApi = {
  getDashboard: (params: TokenUsageQueryParams = {}, signal?: AbortSignal) =>
    request<TokenUsageDashboard>(
      orgPath(`/token-usage/dashboard${buildQueryString(params)}`),
      { signal }
    ),
};
