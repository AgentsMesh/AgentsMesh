import { apiClient } from "./base";
import { AdminUser } from "@/stores/auth";
import type { LoginRequest, LoginResponse } from "./adminTypesExtended";

export async function login(req: LoginRequest): Promise<LoginResponse> {
  return apiClient.post<LoginResponse>("/auth/login", req);
}

export async function getCurrentAdmin(): Promise<AdminUser> {
  return apiClient.get<AdminUser>("/me");
}
