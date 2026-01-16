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
