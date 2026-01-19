# AgentsMesh Runner Makefile
# Provides build, test, and development commands

# Version info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)

# Directories
BIN_DIR := ./bin
DIST_DIR := ./dist

# Binary
RUNNER := $(BIN_DIR)/runner

# Default target
.PHONY: all
all: build

# =============================================================================
# Development Builds
# =============================================================================

# Desktop build (with system tray support, requires CGO)
.PHONY: build
build:
	@echo "Building runner with desktop support (CGO required)..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=1 go build -tags desktop -ldflags "$(LDFLAGS)" -o $(RUNNER) ./cmd/runner
	@echo "Build complete: $(RUNNER)"
	@echo "  - Desktop mode: enabled"
	@echo "  - Web console:  enabled"

# CLI build (no desktop support, no CGO required)
.PHONY: build-cli
build-cli:
	@echo "Building runner CLI (no desktop support)..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(RUNNER) ./cmd/runner
	@echo "Build complete: $(RUNNER)"
	@echo "  - Desktop mode: disabled (use 'runner run' + 'runner webconsole')"
	@echo "  - Web console:  enabled"

# =============================================================================
# Run
# =============================================================================

.PHONY: run
run: build-cli
	$(RUNNER) run

.PHONY: run-desktop
run-desktop: build
	$(RUNNER) desktop

# =============================================================================
# Test & Lint
# =============================================================================

.PHONY: test
test:
	go test -v ./...

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: clean
clean:
	rm -rf $(BIN_DIR) $(DIST_DIR)
	go clean

# =============================================================================
# Cross-platform builds
# =============================================================================

# Cross-platform CLI builds (no CGO, for servers)
.PHONY: build-darwin-cli
build-darwin-cli:
	@echo "Building CLI for macOS..."
	@mkdir -p $(BIN_DIR)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -tags nocgo -o $(BIN_DIR)/runner-darwin-amd64 ./cmd/runner
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -tags nocgo -o $(BIN_DIR)/runner-darwin-arm64 ./cmd/runner
	@echo "macOS CLI builds complete"

.PHONY: build-linux-cli
build-linux-cli:
	@echo "Building CLI for Linux..."
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -tags nocgo -o $(BIN_DIR)/runner-linux-amd64 ./cmd/runner
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -tags nocgo -o $(BIN_DIR)/runner-linux-arm64 ./cmd/runner
	@echo "Linux CLI builds complete"

.PHONY: build-windows-cli
build-windows-cli:
	@echo "Building CLI for Windows..."
	@mkdir -p $(BIN_DIR)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -tags nocgo -o $(BIN_DIR)/runner-windows-amd64.exe ./cmd/runner
	GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -tags nocgo -o $(BIN_DIR)/runner-windows-arm64.exe ./cmd/runner
	@echo "Windows CLI builds complete"

.PHONY: build-all-cli
build-all-cli: build-darwin-cli build-linux-cli build-windows-cli
	@echo "All platform CLI builds complete (no desktop support)"

# Native desktop build (with CGO, for current platform only)
.PHONY: build-desktop
build-desktop: build
	@echo "Desktop build complete (native platform with CGO)"

# =============================================================================
# Packaging
# =============================================================================

.PHONY: package
package: package-darwin package-linux package-windows
	@echo "All packages created in $(DIST_DIR)"

# macOS .app bundle and .dmg
.PHONY: package-darwin
package-darwin: build-desktop
	@echo "Creating macOS .app bundle..."
	@mkdir -p $(DIST_DIR)/AgentsMesh\ Runner.app/Contents/MacOS
	@mkdir -p $(DIST_DIR)/AgentsMesh\ Runner.app/Contents/Resources
	@cp $(BUILD_DIR)/darwin/Info.plist $(DIST_DIR)/AgentsMesh\ Runner.app/Contents/
	@cp $(RUNNER_DESKTOP) $(DIST_DIR)/AgentsMesh\ Runner.app/Contents/MacOS/
	@# Copy icon if exists
	@if [ -f $(BUILD_DIR)/darwin/AppIcon.icns ]; then \
		cp $(BUILD_DIR)/darwin/AppIcon.icns $(DIST_DIR)/AgentsMesh\ Runner.app/Contents/Resources/; \
	fi
	@echo "macOS .app bundle created"
	@# Create DMG if create-dmg is available
	@if command -v create-dmg >/dev/null 2>&1; then \
		echo "Creating DMG..."; \
		create-dmg \
			--volname "AgentsMesh Runner" \
			--volicon "$(BUILD_DIR)/darwin/AppIcon.icns" \
			--window-pos 200 120 \
			--window-size 600 400 \
			--icon-size 100 \
			--icon "AgentsMesh Runner.app" 150 185 \
			--hide-extension "AgentsMesh Runner.app" \
			--app-drop-link 450 185 \
			$(DIST_DIR)/AgentsMesh-Runner-$(VERSION).dmg \
			$(DIST_DIR)/AgentsMesh\ Runner.app || true; \
	else \
		echo "Skipping DMG creation (install create-dmg: brew install create-dmg)"; \
	fi

# Linux .deb and .rpm packages
.PHONY: package-linux
package-linux: build-linux
	@echo "Creating Linux packages..."
	@mkdir -p $(DIST_DIR)
	@# Use nfpm if available
	@if command -v nfpm >/dev/null 2>&1; then \
		echo "Building .deb package..."; \
		nfpm package --packager deb --target $(DIST_DIR)/; \
		echo "Building .rpm package..."; \
		nfpm package --packager rpm --target $(DIST_DIR)/; \
	else \
		echo "nfpm not found. Install: go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest"; \
		echo "Creating tarball instead..."; \
		tar -czvf $(DIST_DIR)/agentsmesh-runner-$(VERSION)-linux-amd64.tar.gz \
			-C $(BIN_DIR) runner-linux-amd64 \
			-C ../$(BUILD_DIR)/linux runner.service; \
	fi
	@echo "Linux packages complete"

# Windows .msi installer
.PHONY: package-windows
package-windows: build-windows
	@echo "Creating Windows packages..."
	@mkdir -p $(DIST_DIR)
	@# Create zip archive
	@cd $(BIN_DIR) && zip -j ../$(DIST_DIR)/agentsmesh-runner-$(VERSION)-windows-amd64.zip runner-windows-amd64.exe
	@echo "Windows zip created"
	@# Build MSI if WiX is available
	@if command -v candle.exe >/dev/null 2>&1; then \
		echo "Building MSI installer..."; \
		cd $(BUILD_DIR)/windows && \
		candle.exe installer.wxs -dVersion=$(VERSION) && \
		light.exe -ext WixUIExtension installer.wixobj -o ../../$(DIST_DIR)/AgentsMesh-Runner-$(VERSION).msi; \
	else \
		echo "WiX Toolset not found. Skipping MSI creation."; \
	fi

# =============================================================================
# Release (using GoReleaser)
# =============================================================================

.PHONY: release-snapshot
release-snapshot:
	@echo "Creating snapshot release..."
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser release --snapshot --clean; \
	else \
		echo "GoReleaser not found. Install: go install github.com/goreleaser/goreleaser@latest"; \
		exit 1; \
	fi

# =============================================================================
# Install / Uninstall (for local development)
# =============================================================================

.PHONY: install
install: build-cli
	@echo "Installing to /usr/local/bin..."
	@sudo cp $(RUNNER) /usr/local/bin/runner
	@echo "Installed: /usr/local/bin/runner"

.PHONY: uninstall
uninstall:
	@echo "Removing from /usr/local/bin..."
	@sudo rm -f /usr/local/bin/runner
	@echo "Uninstalled"

# =============================================================================
# Service management (for development)
# =============================================================================

.PHONY: service-install
service-install: build-cli
	@echo "Installing as system service..."
	$(RUNNER) service install

.PHONY: service-start
service-start:
	$(RUNNER) service start

.PHONY: service-stop
service-stop:
	$(RUNNER) service stop

.PHONY: service-status
service-status:
	$(RUNNER) service status

.PHONY: service-uninstall
service-uninstall:
	$(RUNNER) service uninstall

# =============================================================================
# Help
# =============================================================================

.PHONY: help
help:
	@echo "AgentsMesh Runner Makefile"
	@echo ""
	@echo "Development:"
	@echo "  make build          - Build with desktop support (CGO required)"
	@echo "  make build-cli      - Build CLI only (no CGO, cross-compile friendly)"
	@echo "  make run            - Build CLI and run in CLI mode"
	@echo "  make run-desktop    - Build and run in desktop tray mode"
	@echo "  make test           - Run tests"
	@echo "  make lint           - Run linter"
	@echo "  make clean          - Clean build artifacts"
	@echo ""
	@echo "Release:"
	@echo "  make release-snapshot - Create snapshot release with GoReleaser"
	@echo ""
	@echo "Install:"
	@echo "  make install        - Install CLI to /usr/local/bin"
	@echo "  make uninstall      - Remove from /usr/local/bin"
	@echo ""
	@echo "Service:"
	@echo "  make service-install  - Install as system service"
	@echo "  make service-start    - Start service"
	@echo "  make service-stop     - Stop service"
	@echo "  make service-status   - Show service status"
	@echo "  make service-uninstall- Uninstall service"
