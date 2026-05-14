// Dashboard stats stays on REST until proto.admin.v1.AdminService.GetDashboardStats
// gets a Connect handler in a follow-up PR (the unified scaffold proto already
// declares the RPC; the handler hasn't been implemented yet).
import { apiClient } from "./base";
import type { DashboardStats } from "./adminTypes";

export async function getDashboardStats(): Promise<DashboardStats> {
  return apiClient.get<DashboardStats>("/dashboard/stats");
}
