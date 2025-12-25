#!/bin/bash
# Environment setup check script for eBPF demos

set -e

echo "========================================="
echo "eBPF Demo Environment Check"
echo "========================================="
echo ""

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check status
check_status=0

# Function to check command existence
check_command() {
    if command -v $1 &> /dev/null; then
        echo -e "${GREEN}✓${NC} $1 is installed"
        if [ -n "$2" ]; then
            version=$($1 $2 2>&1 | head -n1)
            echo "  Version: $version"
        fi
        return 0
    else
        echo -e "${RED}✗${NC} $1 is NOT installed"
        check_status=1
        return 1
    fi
}

# Function to check kernel version
check_kernel() {
    kernel_version=$(uname -r)
    echo -e "${GREEN}✓${NC} Linux Kernel: $kernel_version"

    major=$(echo $kernel_version | cut -d. -f1)
    minor=$(echo $kernel_version | cut -d. -f2)

    if [ "$major" -lt 5 ] || ([ "$major" -eq 5 ] && [ "$minor" -lt 10 ]); then
        echo -e "${RED}  Warning: Kernel version should be >= 5.10 for best eBPF support${NC}"
        check_status=1
    else
        echo -e "  ${GREEN}Kernel version is sufficient${NC}"
    fi
}

# Function to check BTF support
check_btf() {
    if [ -f /sys/kernel/btf/vmlinux ]; then
        echo -e "${GREEN}✓${NC} BTF (BPF Type Format) is available"
        btf_size=$(stat -c%s /sys/kernel/btf/vmlinux)
        echo "  BTF file size: $(numfmt --to=iec-i --suffix=B $btf_size)"
    else
        echo -e "${RED}✗${NC} BTF is NOT available"
        echo "  BTF is required for CO-RE (Compile Once - Run Everywhere)"
        check_status=1
    fi
}

# Function to check debugfs
check_debugfs() {
    if mount | grep -q debugfs; then
        echo -e "${GREEN}✓${NC} debugfs is mounted"
        tracefs_path=$(mount | grep debugfs | awk '{print $3}')
        if [ -f "${tracefs_path}/tracing/trace_pipe" ]; then
            echo "  trace_pipe available for bpf_printk() output"
        fi
    else
        echo -e "${YELLOW}!${NC} debugfs is NOT mounted"
        echo "  Mount it with: sudo mount -t debugfs none /sys/kernel/debug"
    fi
}

# Function to check bpffs
check_bpffs() {
    if mount | grep -q bpffs; then
        echo -e "${GREEN}✓${NC} bpffs is mounted"
    else
        echo -e "${YELLOW}!${NC} bpffs is NOT mounted"
        echo "  Mount it with: sudo mount -t bpf none /sys/fs/bpf"
    fi
}

echo "=== System Requirements ===="
echo ""

# 1. Kernel version
check_kernel
echo ""

# 2. BTF support
check_btf
echo ""

# 3. Required tools
echo "=== Required Tools ==="
echo ""

check_command clang "--version"
echo ""

check_command llc "--version"
echo ""

check_command bpftool "version"
echo ""

check_command go "version"
echo ""

# 4. Optional tools
echo "=== Optional Tools ==="
echo ""

check_command ip "-V"
echo ""

check_command tc "-V"
echo ""

check_command ping "-V"
echo ""

check_command nc "--version"
echo ""

# 5. Filesystem checks
echo "=== Filesystem Checks ==="
echo ""

check_debugfs
echo ""

check_bpffs
echo ""

# 6. Permissions check
echo "=== Permissions Check ==="
echo ""

if [ "$EUID" -eq 0 ]; then
    echo -e "${GREEN}✓${NC} Running as root"
else
    echo -e "${YELLOW}!${NC} Not running as root"
    echo "  eBPF operations require root privileges or CAP_BPF capability"
    echo "  Run demos with sudo"
fi
echo ""

# 7. Network interface check
echo "=== Network Interfaces ==="
echo ""

interfaces=$(ip link show | grep -E "^[0-9]+:" | awk -F': ' '{print $2}')
echo "Available interfaces:"
for iface in $interfaces; do
    echo "  - $iface"
done
echo ""

# Summary
echo "========================================="
if [ $check_status -eq 0 ]; then
    echo -e "${GREEN}✓ Environment check PASSED${NC}"
    echo "Your system is ready for eBPF demos!"
else
    echo -e "${RED}✗ Environment check FAILED${NC}"
    echo "Please install missing dependencies:"
    echo ""
    echo "Ubuntu/Debian:"
    echo "  sudo apt-get update"
    echo "  sudo apt-get install -y clang llvm libbpf-dev linux-headers-\$(uname -r)"
    echo "  sudo apt-get install -y bpftool golang-go iproute2 netcat-openbsd"
    echo ""
    echo "Fedora/RHEL:"
    echo "  sudo dnf install -y clang llvm libbpf-devel kernel-devel"
    echo "  sudo dnf install -y bpftool golang iproute netcat"
fi
echo "========================================="

exit $check_status
