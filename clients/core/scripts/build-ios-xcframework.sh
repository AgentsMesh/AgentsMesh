#!/usr/bin/env bash
# Build AgentsMeshCore.xcframework for iOS (device + simulator universal).
#
# Invocation:
#   bazel run //clients/core/crates/ffi:AgentsMeshCore        (primary)
#   bash     clients/core/scripts/build-ios-xcframework.sh     (direct)
#
# Output:
#   clients/core/ios-framework/AgentsMeshCore.xcframework
#   clients/core/ios-framework/Generated/AgentsMeshCore.swift   (Swift glue)
#
# Requirements:
#   - macOS with Xcode 15+
#   - rustup targets: aarch64-apple-ios aarch64-apple-ios-sim x86_64-apple-ios
#
# Consumed by clients/ios/ via SPM binaryTarget + Swift source inclusion.
set -euo pipefail

CRATE=agentsmesh-ffi
LIB=libagentsmesh_ffi.a
MODULE=AgentsMeshCore

# Resolve the crate workspace:
#  - `bazel run` sets BUILD_WORKSPACE_DIRECTORY to the repo root; the
#    runfiles execroot doesn't contain Cargo.toml, so we jump back to
#    the real checkout.
#  - Direct invocation computes the workspace from the script path.
if [ -n "${BUILD_WORKSPACE_DIRECTORY:-}" ]; then
    WORKSPACE="$BUILD_WORKSPACE_DIRECTORY/clients/core"
else
    WORKSPACE="$(cd "$(dirname "$0")/.." && pwd)"
fi

TARGET_DIR="$WORKSPACE/target"
OUT="$WORKSPACE/ios-framework"
GENERATED="$OUT/Generated"
HEADERS="$OUT/Headers"

cd "$WORKSPACE"

echo "==> 1/5 Building Rust staticlib for 3 iOS targets (release)"
for t in aarch64-apple-ios aarch64-apple-ios-sim x86_64-apple-ios; do
    cargo build -p "$CRATE" --release --target "$t"
done

echo "==> 2/5 Lipo'ing simulator slices into universal"
mkdir -p "$OUT/sim"
lipo -create \
    "$TARGET_DIR/aarch64-apple-ios-sim/release/$LIB" \
    "$TARGET_DIR/x86_64-apple-ios/release/$LIB" \
    -output "$OUT/sim/$LIB"

echo "==> 3/5 Generating Swift bindings + headers + modulemap"
rm -rf "$GENERATED" "$HEADERS"
mkdir -p "$GENERATED" "$HEADERS"
cargo run -p uniffi-bindgen -- generate \
    --library "$TARGET_DIR/aarch64-apple-ios/release/$LIB" \
    --language swift \
    --out-dir "$GENERATED"

# Xcode convention: modulemap must be named `module.modulemap` and sit beside
# the C headers so xcframework's -headers path can expose both to Swift.
mv "$GENERATED/${CRATE//-/_}FFI.modulemap" "$HEADERS/module.modulemap"
cp "$GENERATED/${CRATE//-/_}FFI.h" "$HEADERS/"
# Normalize Swift source filename to match the module we expose.
mv "$GENERATED/${CRATE//-/_}.swift" "$GENERATED/${MODULE}.swift"

echo "==> 4/5 Assembling XCFramework"
rm -rf "$OUT/$MODULE.xcframework"
xcodebuild -create-xcframework \
    -library "$TARGET_DIR/aarch64-apple-ios/release/$LIB" -headers "$HEADERS" \
    -library "$OUT/sim/$LIB"                              -headers "$HEADERS" \
    -output "$OUT/$MODULE.xcframework"

echo "==> 5/5 Done"
echo "    XCFramework : $OUT/$MODULE.xcframework"
echo "    Swift glue  : $GENERATED/${MODULE}.swift"
