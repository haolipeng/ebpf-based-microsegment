# Makefile for Demo 1: Hello Packet Counter

# Variables
CLANG ?= clang
BPFTOOL ?= bpftool
IFACE ?= lo
VMLINUX_H := /sys/kernel/btf/vmlinux

# Output files
BPF_OBJ := hello.bpf.o

# Clang flags for eBPF compilation
CLANG_FLAGS := -g -O2 -target bpf -D__TARGET_ARCH_x86 \
               -I../../common/headers

# Default target
.PHONY: all
all: $(BPF_OBJ)

# Compile eBPF program
$(BPF_OBJ): hello.bpf.c
	@echo "Compiling eBPF program..."
	$(CLANG) $(CLANG_FLAGS) -c $< -o $@
	@echo "✓ Compiled successfully: $@"

# Load eBPF program to TC
.PHONY: load
load: $(BPF_OBJ)
	@echo "Loading eBPF program to interface $(IFACE)..."
	@# Remove existing qdisc (if any)
	-sudo tc qdisc del dev $(IFACE) clsact 2>/dev/null || true
	@# Create clsact qdisc (required for TC eBPF)
	sudo tc qdisc add dev $(IFACE) clsact
	@# Attach eBPF program to ingress
	sudo tc filter add dev $(IFACE) ingress bpf direct-action obj $(BPF_OBJ) sec tc
	@echo "✓ eBPF program loaded on $(IFACE)"
	@echo ""
	@echo "View packet count with:"
	@echo "  sudo bpftool map dump name packet_counter"
	@echo ""
	@echo "View debug logs with:"
	@echo "  sudo cat /sys/kernel/debug/tracing/trace_pipe"

# Unload eBPF program
.PHONY: unload
unload:
	@echo "Unloading eBPF program from $(IFACE)..."
	-sudo tc qdisc del dev $(IFACE) clsact 2>/dev/null || true
	@echo "✓ eBPF program unloaded"

# Show loaded eBPF programs
.PHONY: show
show:
	@echo "=== TC Filters on $(IFACE) ==="
	-sudo tc filter show dev $(IFACE) ingress
	@echo ""
	@echo "=== eBPF Programs ==="
	-sudo bpftool prog show
	@echo ""
	@echo "=== eBPF Maps ==="
	-sudo bpftool map show

# Dump packet counter map
.PHONY: dump
dump:
	@echo "=== Packet Counter ==="
	sudo bpftool map dump name packet_counter

# Clean build artifacts
.PHONY: clean
clean:
	rm -f $(BPF_OBJ)
	@echo "✓ Cleaned build artifacts"

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  make          - Compile eBPF program"
	@echo "  make load     - Load eBPF program to $(IFACE)"
	@echo "  make unload   - Unload eBPF program from $(IFACE)"
	@echo "  make show     - Show loaded eBPF programs and maps"
	@echo "  make dump     - Dump packet counter value"
	@echo "  make clean    - Remove build artifacts"
	@echo ""
	@echo "Change interface: make load IFACE=eth0"
