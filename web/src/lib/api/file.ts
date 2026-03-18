import { useAuthStore } from "@/stores/auth";
import { getApiBaseUrl } from "@/lib/env";

/**
 * Upload an image file directly to S3 via presigned URL.
 * 1. POST /files/presign → get put_url + get_url
 * 2. PUT put_url → upload file directly to S3
 * 3. Return get_url for accessing the image
 */
export async function uploadImage(file: File): Promise<string> {
  const { token, currentOrg } = useAuthStore.getState();

  if (!currentOrg) {
    throw new Error("No organization selected");
  }

  const API_BASE_URL = getApiBaseUrl();

  // Step 1: Get presigned URLs from backend
  const presignRes = await fetch(
    `${API_BASE_URL}/api/v1/orgs/${currentOrg.slug}/files/presign`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        "X-Organization-Slug": currentOrg.slug,
      },
      body: JSON.stringify({
        filename: file.name,
        content_type: file.type || "application/octet-stream",
        size: file.size,
      }),
    }
  );

  if (!presignRes.ok) {
    const errorData = await presignRes
      .json()
      .catch(() => ({ error: "Failed to get upload URL" }));
    throw new Error(errorData.error || "Failed to get upload URL");
  }

  const { put_url, get_url } = await presignRes.json();

  // Step 2: Upload file directly to S3 via presigned PUT URL
  const uploadRes = await fetch(put_url, {
    method: "PUT",
    headers: {
      "Content-Type": file.type || "application/octet-stream",
    },
    body: file,
  });

  if (!uploadRes.ok) {
    throw new Error("Failed to upload file to storage");
  }

  // Step 3: Return the GET URL for accessing the uploaded file
  return get_url;
}
