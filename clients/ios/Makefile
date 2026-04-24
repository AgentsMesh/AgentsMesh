# AgentsMesh iOS — build helpers.
#
# The XCFramework + Swift glue come from Bazel
# (//clients/core/crates/ffi:AgentsMeshCore). SPM `Package.swift` can't
# reach files outside its package root, so `link-core` symlinks the
# Bazel outputs into the expected SPM paths.
#
# CI uses the same recipes via `make ios-setup`.

WORKTREE  := $(shell cd ../.. && pwd)
BAZEL_OUT := $(WORKTREE)/bazel-bin/clients/core/crates/ffi
PKG_CORE  := Packages/AgentsMeshCore/Sources

.PHONY: help ios-setup xcframework link-core project check-sim clean

help:
	@echo "make xcframework   - build AgentsMeshCore.xcframework via Bazel"
	@echo "make link-core     - symlink xcframework + Swift bindings into SPM tree"
	@echo "make project       - generate AgentsMesh.xcodeproj via xcodegen"
	@echo "make ios-setup     - xcframework + link-core + project (one-shot)"
	@echo "make check-sim     - build for iOS Simulator via xcodebuild"
	@echo "make clean         - remove symlinks and build artifacts"

# Delegates to the Bazel pipeline in build_defs/ios/rust_xcframework.bzl.
# Tree-artifact output:
#   $(BAZEL_OUT)/AgentsMeshCore.xcframework/        (iOS slices + Info.plist)
#   $(BAZEL_OUT)/AgentsMeshCore_bindings_out/AgentsMeshCore.swift
xcframework:
	bazel build //clients/core/crates/ffi:AgentsMeshCore \
	            //clients/core/crates/ffi:AgentsMeshCore_swift

link-core:
	@mkdir -p $(PKG_CORE)/AgentsMeshCore/Generated
	@rm -f $(PKG_CORE)/AgentsMeshCore/Generated/AgentsMeshCore.swift
	@rm -rf $(PKG_CORE)/AgentsMeshCoreFFI/AgentsMeshCore.xcframework
	ln -s $(BAZEL_OUT)/AgentsMeshCore_bindings_out/AgentsMeshCore.swift \
	      $(PKG_CORE)/AgentsMeshCore/Generated/AgentsMeshCore.swift
	ln -s $(BAZEL_OUT)/AgentsMeshCore.xcframework \
	      $(PKG_CORE)/AgentsMeshCoreFFI/AgentsMeshCore.xcframework
	@echo "Linked Bazel outputs into SPM tree."

project:
	xcodegen generate

ios-setup: xcframework link-core project

check-sim:
	xcodebuild \
	  -project AgentsMesh.xcodeproj \
	  -scheme AgentsMesh \
	  -destination 'generic/platform=iOS Simulator' \
	  -configuration Debug \
	  build

clean:
	rm -f $(PKG_CORE)/AgentsMeshCore/Generated/AgentsMeshCore.swift
	rm -rf $(PKG_CORE)/AgentsMeshCoreFFI/AgentsMeshCore.xcframework
	rm -rf Packages/AgentsMeshCore/.build Packages/AgentsMeshFeatures/.build
	rm -rf AgentsMesh.xcodeproj DerivedData
