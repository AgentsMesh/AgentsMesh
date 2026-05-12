// Connect-RPC adapter for proto.file.v1.FileService.
//
// Encodes requests via @bufbuild/protobuf .toBinary(), passes the Uint8Array
// to the wasm bridge (binary in / binary out — conventions §2.5), decodes
// responses via .fromBinary().
//
// Only the presign-URL RPC is on Connect; the actual PUT to S3 is direct
// from the browser via fetch(put_url). The legacy `getFileService().upload_file`
// wraps both steps (presign + PUT) using the REST path — we expose
// `presignUpload` here so callers can opt into the Connect lane and then
// do the PUT themselves.

import {
  PresignUploadRequestSchema,
  PresignUploadResponseSchema,
} from "@proto/file/v1/file_pb";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import { getFileService } from "@/lib/wasm-core";

export interface PresignResult {
  put_url: string;
  get_url: string;
}

export async function presignUpload(
  orgSlug: string,
  filename: string,
  contentType: string,
  size: number,
): Promise<PresignResult> {
  const req = create(PresignUploadRequestSchema, {
    orgSlug,
    filename,
    contentType,
    size: BigInt(size),
  });
  const bytes = toBinary(PresignUploadRequestSchema, req);
  const respBytes = await getFileService().presignUploadConnect(bytes);
  const resp = fromBinary(PresignUploadResponseSchema, new Uint8Array(respBytes));
  return { put_url: resp.putUrl, get_url: resp.getUrl };
}
