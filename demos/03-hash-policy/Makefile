# Makefile for Demo 3: Hash Policy Match

CLANG ?= clang
IFACE ?= lo

BPF_OBJ := ebpf/policy.bpf.o
GO_BIN := userspace/policy

CLANG_FLAGS := -g -O2 -target bpf -D__TARGET_ARCH_x86 \
               -I../../common/headers

.PHONY: all
all: $(BPF_OBJ) $(GO_BIN)

# Compile eBPF program
$(BPF_OBJ): ebpf/policy.bpf.c
	@echo "Compiling eBPF program..."
	$(CLANG) $(CLANG_FLAGS) -c $< -o $@
	@echo "✓ Compiled eBPF: $@"

# Build Go userspace program
$(GO_BIN): userspace/main.go $(BPF_OBJ)
	@echo "Building Go userspace program..."
	cd userspace && go generate && go build -o policy main.go
	@echo "✓ Built userspace program: $@"

# Run controller (needs root)
.PHONY: run
run: all
	@echo "Starting controller on $(IFACE)..."
	sudo ./$(GO_BIN) $(IFACE)

# Clean
.PHONY: clean
clean:
	rm -f $(BPF_OBJ) $(GO_BIN)
	rm -f userspace/policy_bpfeb.go userspace/policy_bpfel.go userspace/policy_bpfeb.o userspace/policy_bpfel.o
	@echo "✓ Cleaned build artifacts"

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  make          - Compile eBPF and Go programs"
	@echo "  make run      - Run controller on $(IFACE)"
	@echo "  make clean    - Remove build artifacts"
	@echo ""
	@echo "Change interface: make run IFACE=eth0"
