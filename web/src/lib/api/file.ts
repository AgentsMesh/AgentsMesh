import { useAuthStore } from "@/stores/auth";
import { getApiBaseUrl } from "@/lib/env";

/**
 * Upload an image file to the backend and return its presigned URL.
 * Reusable across block-editor and terminal image paste.
 */
export async function uploadImage(file: File): Promise<string> {
  const { token, currentOrg } = useAuthStore.getState();

  if (!currentOrg) {
    throw new Error("No organization selected");
  }

  const formData = new FormData();
  formData.append("file", file);

  const API_BASE_URL = getApiBaseUrl();
  const res = await fetch(`${API_BASE_URL}/api/v1/orgs/${currentOrg.slug}/files/upload`, {
    method: "POST",
    headers: {
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      "X-Organization-Slug": currentOrg.slug,
    },
    body: formData,
  });

  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: "Upload failed" }));
    throw new Error(errorData.error || "Upload failed");
  }

  const data = await res.json();
  return data.url;
}
