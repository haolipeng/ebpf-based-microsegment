# Makefile for eBPF Microsegmentation Project
.PHONY: all clean bpf agent server test install help proto generate-proto install-proto-tools clean-proto

# Variables
BIN_DIR := bin
AGENT_BIN := $(BIN_DIR)/microsegment-agent
SERVER_BIN := $(BIN_DIR)/microsegment-server
SRC_BPF := src/bpf
SRC_AGENT := src/agent
SRC_SERVER := src/server
PROTO_DIR := api/proto
PROTO_OUT := api/proto
GO := go
CLANG := clang

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[0;33m
NC := \033[0m # No Color

all: proto bpf agent server ## Build everything

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}'

$(BIN_DIR):
	@mkdir -p $(BIN_DIR)

proto: generate-proto ## Alias for generate-proto

generate-proto: ## Generate Go code from Protocol Buffers
	@echo "$(YELLOW)Generating Go code from Protocol Buffers...$(NC)"
	@./scripts/generate-proto.sh
	@echo "$(GREEN)✓ Protocol Buffers code generated$(NC)"

install-proto-tools: ## Install Protocol Buffers code generation tools
	@echo "$(YELLOW)Installing Protocol Buffers tools...$(NC)"
	@echo "Installing protoc-gen-go..."
	$(GO) install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@echo "Installing protoc-gen-go-grpc..."
	$(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "$(GREEN)✓ Protocol Buffers tools installed$(NC)"
	@echo ""
	@echo "Note: Make sure protoc is installed on your system:"
	@echo "  Ubuntu/Debian: sudo apt-get install -y protobuf-compiler"
	@echo "  macOS: brew install protobuf"

clean-proto: ## Clean generated Protocol Buffers code
	@echo "$(YELLOW)Cleaning generated proto code...$(NC)"
	rm -rf $(PROTO_OUT)/common/*.pb.go
	rm -rf $(PROTO_OUT)/flow/*.pb.go $(PROTO_OUT)/flow/*_grpc.pb.go
	rm -rf $(PROTO_OUT)/policy/*.pb.go $(PROTO_OUT)/policy/*_grpc.pb.go
	rm -rf $(PROTO_OUT)/agent/*.pb.go $(PROTO_OUT)/agent/*_grpc.pb.go
	@echo "$(GREEN)✓ Cleaned proto code$(NC)"

bpf: ## Generate eBPF Go bindings using bpf2go
	@echo "$(YELLOW)Generating eBPF Go bindings...$(NC)"
	cd $(SRC_AGENT)/pkg/dataplane && $(GO) generate
	@echo "$(GREEN)✓ eBPF bindings generated$(NC)"

agent: $(BIN_DIR) ## Build the microsegmentation agent
	@echo "$(YELLOW)Building agent...$(NC)"
	cd $(SRC_AGENT) && $(GO) build -o ../../$(AGENT_BIN) ./cmd
	@echo "$(GREEN)✓ Agent built: $(AGENT_BIN)$(NC)"

server: $(BIN_DIR) proto ## Build the microsegmentation server
	@echo "$(YELLOW)Building server...$(NC)"
	cd $(SRC_SERVER) && $(GO) build -o ../../$(SERVER_BIN) ./cmd
	@echo "$(GREEN)✓ Server built: $(SERVER_BIN)$(NC)"

install: agent server ## Install binaries to /usr/local/bin
	@echo "$(YELLOW)Installing agent...$(NC)"
	sudo install -m 755 $(AGENT_BIN) /usr/local/bin/
	@echo "$(YELLOW)Installing server...$(NC)"
	sudo install -m 755 $(SERVER_BIN) /usr/local/bin/
	@echo "$(GREEN)✓ Binaries installed to /usr/local/bin/$(NC)"

test: ## Run unit tests
	@echo "$(YELLOW)Running tests...$(NC)"
	cd $(SRC_AGENT) && $(GO) test -v ./...
	@echo "$(GREEN)✓ Tests completed$(NC)"

test-integration: agent ## Run integration tests
	@echo "$(YELLOW)Running integration tests...$(NC)"
	sudo ./tests/integration_test.sh
	@echo "$(GREEN)✓ Integration tests completed$(NC)"

clean: clean-proto ## Clean build artifacts
	@echo "$(YELLOW)Cleaning...$(NC)"
	rm -rf $(BIN_DIR)
	rm -f $(SRC_AGENT)/pkg/dataplane/bpf_*.go
	rm -f $(SRC_AGENT)/pkg/dataplane/bpf_*.o
	@echo "$(GREEN)✓ Cleaned$(NC)"

fmt: ## Format Go code
	@echo "$(YELLOW)Formatting code...$(NC)"
	cd $(SRC_AGENT) && $(GO) fmt ./...
	@echo "$(GREEN)✓ Code formatted$(NC)"

lint: ## Run linters
	@echo "$(YELLOW)Running linters...$(NC)"
	cd $(SRC_AGENT) && golangci-lint run ./...
	@echo "$(GREEN)✓ Linting completed$(NC)"

deps: ## Download Go dependencies
	@echo "$(YELLOW)Downloading dependencies...$(NC)"
	$(GO) mod download
	$(GO) mod tidy
	@echo "$(GREEN)✓ Dependencies downloaded$(NC)"

run: agent ## Run the agent (requires sudo)
	@echo "$(YELLOW)Starting agent...$(NC)"
	sudo $(AGENT_BIN) --interface lo --log-level info

.DEFAULT_GOAL := help

