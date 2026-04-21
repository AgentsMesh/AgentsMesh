import { getFileService } from "@/lib/wasm-core";

export async function uploadImage(fileOrBlob: File | Blob): Promise<string> {
  const buf = new Uint8Array(await fileOrBlob.arrayBuffer());
  const name = fileOrBlob instanceof File ? fileOrBlob.name : "image.png";
  const type = fileOrBlob.type || "image/png";
  const result = await getFileService().upload_file(buf, name, type);
  // Rust returns the presigned GET url as a raw string; WASM service may
  // emit either a JSON envelope (`{ url }`) or the URL itself. Handle both.
  if (typeof result === "string" && /^https?:\/\//i.test(result.trim())) {
    return result.trim();
  }
  try {
    const parsed = JSON.parse(result);
    return parsed?.url ?? parsed?.file_url ?? result;
  } catch {
    return result;
  }
}
