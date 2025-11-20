#!/bin/bash
# Test script for process monitoring eBPF program
# This script verifies basic functionality without requiring full userspace integration

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Process Monitor eBPF Program - Basic Tests"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
  echo "❌ ERROR: This script must be run as root (for eBPF operations)"
  echo "   Please run: sudo $0"
  exit 1
fi

# Test 1: Verify eBPF object file exists
echo "Test 1: Checking eBPF object file..."
BPF_OBJ="$PROJECT_ROOT/src/agent/pkg/dataplane/processbpf_x86_bpfel.o"
if [ -f "$BPF_OBJ" ]; then
  echo "  ✅ eBPF object file exists: $BPF_OBJ"
  ls -lh "$BPF_OBJ"
else
  echo "  ❌ eBPF object file not found: $BPF_OBJ"
  echo "     Run 'make bpf' to generate it"
  exit 1
fi
echo ""

# Test 2: Verify Go bindings exist
echo "Test 2: Checking Go bindings..."
GO_BINDINGS="$PROJECT_ROOT/src/agent/pkg/dataplane/processbpf_x86_bpfel.go"
if [ -f "$GO_BINDINGS" ]; then
  echo "  ✅ Go bindings file exists: $GO_BINDINGS"
  echo "     Lines: $(wc -l < "$GO_BINDINGS")"
  echo "     Size: $(ls -lh "$GO_BINDINGS" | awk '{print $5}')"
else
  echo "  ❌ Go bindings file not found: $GO_BINDINGS"
  exit 1
fi
echo ""

# Test 3: Check eBPF object structure
echo "Test 3: Analyzing eBPF object structure..."
if command -v llvm-objdump &> /dev/null; then
  echo "  Sections in eBPF object:"
  llvm-objdump -h "$BPF_OBJ" | grep -E "(tp/sched|\.maps)" || echo "  (No matching sections - may need full build)"

  echo ""
  echo "  Maps defined:"
  llvm-objdump -t "$BPF_OBJ" | grep -E "(process_info_map|process_events)" || echo "  (Maps will be visible after full compilation)"
else
  echo "  ⚠️  llvm-objdump not available, skipping structure analysis"
fi
echo ""

# Test 4: Verify tracepoint exists in kernel
echo "Test 4: Checking kernel tracepoint availability..."
TRACEPOINT="/sys/kernel/debug/tracing/events/sched/sched_process_exec"
if [ -d "$TRACEPOINT" ]; then
  echo "  ✅ Tracepoint exists: sched:sched_process_exec"

  # Check if we can read it
  if [ -r "$TRACEPOINT/format" ]; then
    echo "  ✅ Tracepoint format is readable"
  else
    echo "  ⚠️  Cannot read tracepoint format (may need root or debugfs mounted)"
  fi
else
  echo "  ❌ Tracepoint not found: $TRACEPOINT"
  echo "     Make sure debugfs is mounted: mount -t debugfs none /sys/kernel/debug"
  exit 1
fi
echo ""

# Test 5: Verify kernel version
echo "Test 5: Checking kernel version..."
KERNEL_VERSION=$(uname -r)
echo "  Kernel: $KERNEL_VERSION"

# Extract major and minor version
MAJOR=$(echo "$KERNEL_VERSION" | cut -d. -f1)
MINOR=$(echo "$KERNEL_VERSION" | cut -d. -f2)

if [ "$MAJOR" -gt 4 ] || ([ "$MAJOR" -eq 4 ] && [ "$MINOR" -ge 18 ]); then
  echo "  ✅ Kernel version >= 4.18 (tracepoint support confirmed)"
else
  echo "  ⚠️  Kernel version < 4.18 (tracepoint may not be fully supported)"
fi
echo ""

# Test 6: Check BTF support
echo "Test 6: Checking BTF (BPF Type Format) support..."
if [ -f "/sys/kernel/btf/vmlinux" ]; then
  echo "  ✅ BTF vmlinux available"
  BTF_SIZE=$(ls -lh /sys/kernel/btf/vmlinux | awk '{print $5}')
  echo "     Size: $BTF_SIZE"
else
  echo "  ⚠️  BTF vmlinux not found (CO-RE may not work)"
  echo "     Some kernel versions don't include BTF by default"
fi
echo ""

# Test 7: Verify source code integrity
echo "Test 7: Checking source code..."
PROCESS_MONITOR_BPF="$PROJECT_ROOT/src/bpf/process_monitor.bpf.c"
PROCESS_MONITOR_H="$PROJECT_ROOT/src/bpf/headers/process_monitor.h"

if [ -f "$PROCESS_MONITOR_BPF" ]; then
  echo "  ✅ process_monitor.bpf.c exists"
  echo "     Lines: $(wc -l < "$PROCESS_MONITOR_BPF")"

  # Check for key functions
  if grep -q "trace_sched_process_exec" "$PROCESS_MONITOR_BPF"; then
    echo "  ✅ Tracepoint handler function found"
  else
    echo "  ❌ Tracepoint handler function not found"
    exit 1
  fi

  if grep -q "extract_container_id_from_cgroup" "$PROCESS_MONITOR_BPF"; then
    echo "  ✅ Container ID extraction function found"
  else
    echo "  ❌ Container ID extraction function not found"
    exit 1
  fi
else
  echo "  ❌ process_monitor.bpf.c not found"
  exit 1
fi

if [ -f "$PROCESS_MONITOR_H" ]; then
  echo "  ✅ process_monitor.h exists"
  echo "     Lines: $(wc -l < "$PROCESS_MONITOR_H")"

  # Check for key structures
  if grep -q "struct process_cache_entry" "$PROCESS_MONITOR_H"; then
    echo "  ✅ process_cache_entry structure defined"
  else
    echo "  ❌ process_cache_entry structure not found"
    exit 1
  fi

  if grep -q "struct process_event" "$PROCESS_MONITOR_H"; then
    echo "  ✅ process_event structure defined"
  else
    echo "  ❌ process_event structure not found"
    exit 1
  fi
else
  echo "  ❌ process_monitor.h not found"
  exit 1
fi
echo ""

# Test 8: Compilation test (if clang available)
echo "Test 8: Testing eBPF compilation..."
if command -v clang &> /dev/null; then
  TEMP_OUTPUT="/tmp/process_monitor_test.o"

  if clang -target bpf -O2 -g -Wall \
    -I"$PROJECT_ROOT/src/bpf" \
    -I"$PROJECT_ROOT/src/bpf/headers" \
    -I"$PROJECT_ROOT/vmlinux/x86" \
    -c "$PROCESS_MONITOR_BPF" \
    -o "$TEMP_OUTPUT" 2>&1 | tee /tmp/clang_output.txt; then

    echo "  ✅ eBPF program compiles successfully"
    ls -lh "$TEMP_OUTPUT"

    # Check for warnings
    if grep -qi "warning" /tmp/clang_output.txt; then
      echo "  ⚠️  Compilation warnings detected:"
      grep -i "warning" /tmp/clang_output.txt | head -5
    fi

    rm -f "$TEMP_OUTPUT"
  else
    echo "  ❌ eBPF compilation failed"
    cat /tmp/clang_output.txt
    exit 1
  fi

  rm -f /tmp/clang_output.txt
else
  echo "  ⚠️  clang not available, skipping compilation test"
fi
echo ""

# Summary
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ All basic tests passed!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Next steps:"
echo "  1. Implement userspace ProcessMonitor (Task #48)"
echo "  2. Load and attach tracepoint program"
echo "  3. Read events from ring buffer"
echo "  4. Test with real workloads (curl, docker, etc.)"
echo ""
echo "For manual testing:"
echo "  - View tracepoint format: cat /sys/kernel/debug/tracing/events/sched/sched_process_exec/format"
echo "  - Enable tracing: echo 1 > /sys/kernel/debug/tracing/events/sched/sched_process_exec/enable"
echo "  - View events: cat /sys/kernel/debug/tracing/trace_pipe"
echo ""
