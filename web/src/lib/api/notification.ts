import { request, orgPath } from "./base";

export interface NotificationPreference {
  source: string;
  entity_id?: string;
  is_muted: boolean;
  channels: Record<string, boolean>;
}

export const notificationApi = {
  // Get all notification preferences for the current user
  getPreferences: () =>
    request<{ preferences: NotificationPreference[] }>(orgPath("/notifications/preferences")),

  // Set a notification preference
  setPreference: (pref: {
    source: string;
    entity_id?: string;
    is_muted: boolean;
    channels: Record<string, boolean>;
  }) =>
    request<{ status: string }>(orgPath("/notifications/preferences"), {
      method: "PUT",
      body: pref,
    }),
};
