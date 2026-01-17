#!/bin/bash
# Proto code generation script
# Supports both buf and protoc

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GEN_DIR="${SCRIPT_DIR}/gen/go"

# Create output directory
mkdir -p "${GEN_DIR}"

# Check for buf first (preferred)
if command -v buf &> /dev/null; then
    echo "Using buf for code generation..."
    cd "${SCRIPT_DIR}"
    buf generate
    echo "Done!"
    exit 0
fi

# Fallback to protoc
if command -v protoc &> /dev/null; then
    echo "Using protoc for code generation..."

    # Check for required plugins
    if ! command -v protoc-gen-go &> /dev/null; then
        echo "Error: protoc-gen-go not found. Install with:"
        echo "  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"
        exit 1
    fi

    if ! command -v protoc-gen-go-grpc &> /dev/null; then
        echo "Error: protoc-gen-go-grpc not found. Install with:"
        echo "  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"
        exit 1
    fi

    # Generate code
    protoc \
        --proto_path="${SCRIPT_DIR}" \
        --go_out="${GEN_DIR}" \
        --go_opt=paths=source_relative \
        --go-grpc_out="${GEN_DIR}" \
        --go-grpc_opt=paths=source_relative \
        "${SCRIPT_DIR}/runner/v1/runner.proto"

    echo "Done!"
    exit 0
fi

echo "Error: Neither buf nor protoc is installed."
echo ""
echo "Install buf (recommended):"
echo "  brew install bufbuild/buf/buf"
echo ""
echo "Or install protoc:"
echo "  brew install protobuf"
echo "  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"
echo "  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"
exit 1
