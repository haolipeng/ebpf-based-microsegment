# Makefile for Demo 2: 5-Tuple Extractor

CLANG ?= clang
BPFTOOL ?= bpftool
IFACE ?= lo

BPF_OBJ := extractor.bpf.o

CLANG_FLAGS := -g -O2 -target bpf -D__TARGET_ARCH_x86 \
               -I../../common/headers

.PHONY: all
all: $(BPF_OBJ)

$(BPF_OBJ): extractor.bpf.c
	@echo "Compiling eBPF program..."
	$(CLANG) $(CLANG_FLAGS) -c $< -o $@
	@echo "✓ Compiled successfully: $@"

.PHONY: load
load: $(BPF_OBJ)
	@echo "Loading eBPF program to interface $(IFACE)..."
	-sudo tc qdisc del dev $(IFACE) clsact 2>/dev/null || true
	sudo tc qdisc add dev $(IFACE) clsact
	sudo tc filter add dev $(IFACE) ingress bpf direct-action obj $(BPF_OBJ) sec tc
	@echo "✓ eBPF program loaded on $(IFACE)"
	@echo ""
	@echo "View flow logs with:"
	@echo "  sudo cat /sys/kernel/debug/tracing/trace_pipe"
	@echo ""
	@echo "View protocol statistics with:"
	@echo "  sudo bpftool map dump name protocol_counter | grep -v ': 00 00'"

.PHONY: unload
unload:
	@echo "Unloading eBPF program from $(IFACE)..."
	-sudo tc qdisc del dev $(IFACE) clsact 2>/dev/null || true
	@echo "✓ eBPF program unloaded"

.PHONY: show
show:
	@echo "=== TC Filters on $(IFACE) ==="
	-sudo tc filter show dev $(IFACE) ingress
	@echo ""
	@echo "=== Protocol Statistics ==="
	@# Only show non-zero counters
	-sudo bpftool map dump name protocol_counter | grep -v ': 00 00'

.PHONY: logs
logs:
	@echo "=== Real-time Flow Logs ==="
	@echo "Press Ctrl+C to stop"
	@echo ""
	sudo cat /sys/kernel/debug/tracing/trace_pipe

.PHONY: clean
clean:
	rm -f $(BPF_OBJ)
	@echo "✓ Cleaned build artifacts"

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  make          - Compile eBPF program"
	@echo "  make load     - Load eBPF program to $(IFACE)"
	@echo "  make unload   - Unload eBPF program"
	@echo "  make show     - Show loaded program and stats"
	@echo "  make logs     - View real-time flow logs"
	@echo "  make clean    - Remove build artifacts"
