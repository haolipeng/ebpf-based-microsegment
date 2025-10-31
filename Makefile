# Makefile for eBPF Microsegmentation Project
.PHONY: all clean bpf agent test install help

# Variables
BIN_DIR := bin
AGENT_BIN := $(BIN_DIR)/microsegment-agent
SRC_BPF := src/bpf
SRC_AGENT := src/agent
GO := go
CLANG := clang

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[0;33m
NC := \033[0m # No Color

all: bpf agent ## Build everything

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}'

$(BIN_DIR):
	@mkdir -p $(BIN_DIR)

bpf: ## Generate eBPF Go bindings using bpf2go
	@echo "$(YELLOW)Generating eBPF Go bindings...$(NC)"
	cd $(SRC_AGENT)/pkg/dataplane && $(GO) generate
	@echo "$(GREEN)✓ eBPF bindings generated$(NC)"

agent: $(BIN_DIR) ## Build the microsegmentation agent
	@echo "$(YELLOW)Building agent...$(NC)"
	cd $(SRC_AGENT) && $(GO) build -o ../../$(AGENT_BIN) ./cmd
	@echo "$(GREEN)✓ Agent built: $(AGENT_BIN)$(NC)"

install: agent ## Install the agent to /usr/local/bin
	@echo "$(YELLOW)Installing agent...$(NC)"
	sudo install -m 755 $(AGENT_BIN) /usr/local/bin/
	@echo "$(GREEN)✓ Agent installed to /usr/local/bin/$(NC)"

test: ## Run unit tests
	@echo "$(YELLOW)Running tests...$(NC)"
	cd $(SRC_AGENT) && $(GO) test -v ./...
	@echo "$(GREEN)✓ Tests completed$(NC)"

test-integration: agent ## Run integration tests
	@echo "$(YELLOW)Running integration tests...$(NC)"
	sudo ./tests/integration_test.sh
	@echo "$(GREEN)✓ Integration tests completed$(NC)"

clean: ## Clean build artifacts
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

