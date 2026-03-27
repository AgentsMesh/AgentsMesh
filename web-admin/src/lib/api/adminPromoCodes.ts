import { apiClient, PaginatedResponse } from "./base";
import type {
  PromoCode,
  PromoCodeListParams,
  CreatePromoCodeRequest,
  UpdatePromoCodeRequest,
  PromoCodeRedemption,
} from "./adminTypes";

export async function listPromoCodes(params?: PromoCodeListParams): Promise<PaginatedResponse<PromoCode>> {
  const queryParams: Record<string, string | number | undefined> = {};
  if (params) {
    if (params.search) queryParams.search = params.search;
    if (params.type) queryParams.type = params.type;
    if (params.plan_name) queryParams.plan_name = params.plan_name;
    if (params.is_active !== undefined) queryParams.is_active = params.is_active ? "true" : "false";
    if (params.page) queryParams.page = params.page;
    if (params.page_size) queryParams.page_size = params.page_size;
  }
  return apiClient.get<PaginatedResponse<PromoCode>>("/promo-codes", queryParams);
}

export async function getPromoCode(id: number): Promise<PromoCode> {
  return apiClient.get<PromoCode>(`/promo-codes/${id}`);
}

export async function createPromoCode(data: CreatePromoCodeRequest): Promise<PromoCode> {
  return apiClient.post<PromoCode>("/promo-codes", data);
}

export async function updatePromoCode(id: number, data: UpdatePromoCodeRequest): Promise<PromoCode> {
  return apiClient.put<PromoCode>(`/promo-codes/${id}`, data);
}

export async function activatePromoCode(id: number): Promise<{ message: string }> {
  return apiClient.post<{ message: string }>(`/promo-codes/${id}/activate`);
}

export async function deactivatePromoCode(id: number): Promise<{ message: string }> {
  return apiClient.post<{ message: string }>(`/promo-codes/${id}/deactivate`);
}

export async function deletePromoCode(id: number): Promise<{ message: string }> {
  return apiClient.delete<{ message: string }>(`/promo-codes/${id}`);
}

export async function listPromoCodeRedemptions(id: number, params?: { page?: number; page_size?: number }): Promise<PaginatedResponse<PromoCodeRedemption>> {
  return apiClient.get<PaginatedResponse<PromoCodeRedemption>>(`/promo-codes/${id}/redemptions`, params as Record<string, string | number | undefined>);
}
