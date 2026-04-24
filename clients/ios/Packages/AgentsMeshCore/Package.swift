// swift-tools-version: 5.9
// `AgentsMeshCore` — Swift façade over the Rust-powered XCFramework.
//
// The binary target points at the xcframework produced by Bazel:
// `bazel build //clients/core/crates/ffi:AgentsMeshCore`. The SPM
// symlink under Sources/AgentsMeshCoreFFI/ is populated by
// `make -C clients/ios link-core` (or `make ios-setup` one-shot).
import PackageDescription

let package = Package(
    name: "AgentsMeshCore",
    platforms: [.iOS(.v16)],
    products: [
        .library(name: "AgentsMeshCore", targets: ["AgentsMeshCore"]),
    ],
    targets: [
        // Auto-generated Swift glue (from uniffi-bindgen-swift) lives at
        // bazel-bin/clients/core/crates/ffi/AgentsMeshCore_bindings_out/
        // AgentsMeshCore.swift. SPM doesn't let us add files from outside
        // the package, so we symlink it into Sources/AgentsMeshCore/
        // Generated/ via `make link-core`.
        .target(
            name: "AgentsMeshCore",
            dependencies: ["AgentsMeshCoreFFI"],
            path: "Sources/AgentsMeshCore"
        ),
        .binaryTarget(
            name: "AgentsMeshCoreFFI",
            // Relative to package root; symlink → bazel-bin created by
            // `make link-core`.
            path: "Sources/AgentsMeshCoreFFI/AgentsMeshCore.xcframework"
        ),
        .testTarget(
            name: "AgentsMeshCoreTests",
            dependencies: ["AgentsMeshCore"],
            path: "Tests/AgentsMeshCoreTests"
        ),
    ]
)
