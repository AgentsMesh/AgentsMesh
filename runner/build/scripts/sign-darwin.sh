#!/bin/bash
# Sign macOS binary with Developer ID certificate using rcodesign.
# Called by GoReleaser post-build hook for darwin targets.
# Requires: MACOS_CERTIFICATE (base64 .p12), MACOS_CERTIFICATE_PASSWORD
set -euo pipefail

BINARY="$1"

if [ -z "${MACOS_CERTIFICATE:-}" ]; then
  echo "MACOS_CERTIFICATE not set, skipping sign"
  exit 0
fi

RCODESIGN="${RCODESIGN:-rcodesign}"
ENTITLEMENTS="${ENTITLEMENTS:-./build/darwin/entitlements.plist}"

echo "$MACOS_CERTIFICATE" | base64 -d > /tmp/cert.p12
trap 'rm -f /tmp/cert.p12' EXIT

$RCODESIGN sign \
  --p12-file /tmp/cert.p12 \
  --p12-password "$MACOS_CERTIFICATE_PASSWORD" \
  --code-signature-flags runtime \
  --entitlements-xml-path "$ENTITLEMENTS" \
  "$BINARY"

echo "Signed: $BINARY"
