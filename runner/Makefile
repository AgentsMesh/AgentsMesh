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
# Build
# =============================================================================

.PHONY: build
build:
	@echo "Building runner..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(RUNNER) ./cmd/runner
	@echo "Build complete: $(RUNNER)"

# =============================================================================
# Run
# =============================================================================

.PHONY: run
run: build
	$(RUNNER) run

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

.PHONY: build-darwin
build-darwin:
	@echo "Building for macOS..."
	@mkdir -p $(BIN_DIR)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/runner-darwin-amd64 ./cmd/runner
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/runner-darwin-arm64 ./cmd/runner
	@echo "macOS builds complete"

.PHONY: build-linux
build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/runner-linux-amd64 ./cmd/runner
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/runner-linux-arm64 ./cmd/runner
	@echo "Linux builds complete"

.PHONY: build-windows
build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(BIN_DIR)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/runner-windows-amd64.exe ./cmd/runner
	GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/runner-windows-arm64.exe ./cmd/runner
	@echo "Windows builds complete"

.PHONY: build-all
build-all: build-darwin build-linux build-windows
	@echo "All platform builds complete"

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

LOCAL_BIN := $(HOME)/.local/bin

.PHONY: install
install: build
	@echo "Installing to $(LOCAL_BIN)..."
	@mkdir -p $(LOCAL_BIN)
	@cp $(RUNNER) $(LOCAL_BIN)/runner
	@echo "Installed: $(LOCAL_BIN)/runner"

.PHONY: uninstall
uninstall:
	@echo "Removing from $(LOCAL_BIN)..."
	@rm -f $(LOCAL_BIN)/runner
	@echo "Uninstalled"

# =============================================================================
# Service management (for development)
# =============================================================================

.PHONY: service-install
service-install: build
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
	@echo "  make build            - Build runner (no CGO required)"
	@echo "  make run              - Build and run"
	@echo "  make test             - Run tests"
	@echo "  make lint             - Run linter"
	@echo "  make clean            - Clean build artifacts"
	@echo ""
	@echo "Cross-platform:"
	@echo "  make build-darwin     - Build for macOS (amd64 + arm64)"
	@echo "  make build-linux      - Build for Linux (amd64 + arm64)"
	@echo "  make build-windows    - Build for Windows (amd64 + arm64)"
	@echo "  make build-all        - Build for all platforms"
	@echo ""
	@echo "Release:"
	@echo "  make release-snapshot - Create snapshot release with GoReleaser"
	@echo ""
	@echo "Install:"
	@echo "  make install          - Install to ~/.local/bin"
	@echo "  make uninstall        - Remove from ~/.local/bin"
	@echo ""
	@echo "Service:"
	@echo "  make service-install  - Install as system service"
	@echo "  make service-start    - Start service"
	@echo "  make service-stop     - Stop service"
	@echo "  make service-status   - Show service status"
	@echo "  make service-uninstall- Uninstall service"
