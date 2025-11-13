#!/bin/bash

# Protocol Buffers Code Generation Script
# Generates Go code from .proto files

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Protocol Buffers Code Generation ===${NC}"

# Check for required tools
echo "Checking for required tools..."

if ! command -v protoc &> /dev/null; then
    echo -e "${RED}Error: protoc not found. Please install Protocol Buffers compiler.${NC}"
    echo "  Ubuntu/Debian: sudo apt-get install -y protobuf-compiler"
    echo "  macOS: brew install protobuf"
    exit 1
fi

if ! command -v protoc-gen-go &> /dev/null; then
    echo -e "${RED}Error: protoc-gen-go not found.${NC}"
    echo "  Install with: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"
    exit 1
fi

if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo -e "${RED}Error: protoc-gen-go-grpc not found.${NC}"
    echo "  Install with: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"
    exit 1
fi

echo -e "${GREEN}✓ All required tools found${NC}"

# Print versions
echo "Tool versions:"
protoc --version
protoc-gen-go --version
echo -n "protoc-gen-go-grpc: "
protoc-gen-go-grpc --version 2>&1 | head -n 1 || echo "version unknown"

# Project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROTO_DIR="${PROJECT_ROOT}/api/proto"
OUTPUT_DIR="${PROJECT_ROOT}/api/proto"

echo ""
echo "Project root: ${PROJECT_ROOT}"
echo "Proto files:  ${PROTO_DIR}"
echo "Output dir:   ${OUTPUT_DIR}"

# Create output directories
echo ""
echo "Creating output directories..."
mkdir -p "${OUTPUT_DIR}/common"
mkdir -p "${OUTPUT_DIR}/flow"
mkdir -p "${OUTPUT_DIR}/policy"
mkdir -p "${OUTPUT_DIR}/agent"

# Generate code for each proto file
echo ""
echo -e "${YELLOW}Generating Go code from proto files...${NC}"

# List of proto files to compile
PROTO_FILES=(
    "common/common.proto"
    "flow/flow.proto"
    "policy/policy.proto"
    "agent/agent.proto"
)

for proto_file in "${PROTO_FILES[@]}"; do
    echo ""
    echo "Processing ${proto_file}..."

    if [ ! -f "${PROTO_DIR}/${proto_file}" ]; then
        echo -e "${RED}Error: ${proto_file} not found in ${PROTO_DIR}${NC}"
        exit 1
    fi

    # Determine package directory from proto file path (e.g., common/common.proto -> common)
    package_name=$(dirname "${proto_file}")
    output_package_dir="${OUTPUT_DIR}/${package_name}"

    # Run protoc
    if [[ "${proto_file}" == "common/common.proto" ]]; then
        # common.proto doesn't have gRPC services
        protoc \
            --proto_path="${PROTO_DIR}" \
            --go_out="${OUTPUT_DIR}" \
            --go_opt=paths=source_relative \
            "${PROTO_DIR}/${proto_file}"
    else
        # flow, policy, agent have gRPC services
        protoc \
            --proto_path="${PROTO_DIR}" \
            --go_out="${OUTPUT_DIR}" \
            --go_opt=paths=source_relative \
            --go-grpc_out="${OUTPUT_DIR}" \
            --go-grpc_opt=paths=source_relative \
            "${PROTO_DIR}/${proto_file}"
    fi

    echo -e "${GREEN}✓ Generated code for ${proto_file}${NC}"
done

# List generated files
echo ""
echo -e "${GREEN}=== Generated Files ===${NC}"
find "${OUTPUT_DIR}" -name "*.pb.go" -o -name "*_grpc.pb.go" | sort

# Count files
pb_count=$(find "${OUTPUT_DIR}" -name "*.pb.go" | wc -l)
grpc_count=$(find "${OUTPUT_DIR}" -name "*_grpc.pb.go" | wc -l)

echo ""
echo -e "${GREEN}✓ Code generation complete!${NC}"
echo "  Generated ${pb_count} protobuf files"
echo "  Generated ${grpc_count} gRPC files"
echo ""
echo "Next steps:"
echo "  1. Run 'cd src/proto && go build ./...' to verify compilation"
echo "  2. Import packages in your code, e.g.:"
echo "     import pb \"github.com/yourusername/ebpf-based-microsegment/src/proto/flow\""
