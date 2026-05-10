#!/usr/bin/env node
// Pure-stdlib HTTP healthcheck — distroless/nodejs has no curl/wget.
// docker-compose invokes `node /app/healthcheck.mjs`. Returns exit 0 on
// 2xx/3xx, 1 otherwise. PORT/HEALTHCHECK_PATH override via env.
import http from "node:http";

const port = process.env.PORT || "3000";
const path = process.env.HEALTHCHECK_PATH || "/";

const req = http.request(
  { host: "127.0.0.1", port, path, method: "GET", timeout: 3000 },
  (res) => process.exit(res.statusCode >= 200 && res.statusCode < 400 ? 0 : 1),
);
req.on("error", () => process.exit(1));
req.on("timeout", () => {
  req.destroy();
  process.exit(1);
});
req.end();
