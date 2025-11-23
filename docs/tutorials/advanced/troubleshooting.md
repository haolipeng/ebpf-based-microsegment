# 故障排查指南

本文档记录了常见问题及其解决方案，帮助团队成员快速定位和解决问题。

## 目录

- [运行时错误](#运行时错误)
  - [问题 1: TCX Not Supported (内核版本问题)](#问题-1-tcx-not-supported-内核版本问题)
  - [问题 2: Permission Denied (权限不足)](#问题-2-permission-denied-权限不足)
  - [问题 3: Interface Not Found (网络接口不存在)](#问题-3-interface-not-found-网络接口不存在)
  - [问题 4: Address Already in Use (端口占用)](#问题-4-address-already-in-use-端口占用)
- [编译错误](#编译错误)
  - [问题 5: eBPF Program Compilation Failed](#问题-5-ebpf-program-compilation-failed)
  - [问题 6: Missing Dependencies](#问题-6-missing-dependencies)
- [性能问题](#性能问题)
  - [问题 7: 延迟高于预期](#问题-7-延迟高于预期)
- [网络问题](#网络问题)
  - [问题 8: API Server Not Responding](#问题-8-api-server-not-responding)

---

## 运行时错误

### 问题 1: TCX Not Supported (内核版本问题)

#### 错误信息

```
FATA[2025-10-31T14:03:50+08:00] Failed to create data plane: attaching TC program: tcx not supported (requires >= v6.6)
```

#### 问题原因

代码最初使用了 `link.AttachTCX`，这是 Linux 内核 6.6+ 引入的新特性 TCX (TC eXpress)。如果系统内核版本低于 6.6，就会报此错误。

#### 技术背景

**TCX (TC eXpress)** 是 Linux 内核 6.6 引入的新特性，相比传统 TC hook 有以下优势：
- 更低的延迟（~0.5μs 更快）
- 更简单的 API
- 更好的性能

**传统 TC Hook (netlink)** 是更成熟的方案：
- 兼容内核 >= 4.18
- 使用 netlink 接口
- 需要创建 clsact qdisc
- 广泛支持

#### 诊断步骤

**1. 检查内核版本**：

```bash
uname -r
```

输出示例：
- `6.4.0-060400-generic` - 不支持 TCX（< 6.6）
- `6.6.1-generic` - 支持 TCX（>= 6.6）

**2. 验证 TCX 支持**：

```bash
# 检查内核是否支持 TCX
sudo bpftool feature | grep -i tcx
```

**3. 查看当前 TC 配置**：

```bash
# 查看网络接口的 qdisc
sudo tc qdisc show dev ens33

# 查看 TC filters
sudo tc filter show dev ens33 ingress
```

#### 解决方案

##### 方案 A: 自动降级（已实现，推荐 ✅）

代码已经实现了自动降级机制：

**工作原理**：
1. 优先尝试使用 TCX（内核 >= 6.6）
2. 如果失败，自动降级到传统 netlink TC hook（内核 >= 4.18）
3. 记录警告日志，说明使用了兼容模式

**代码实现**（`src/agent/pkg/dataplane/dataplane.go`）：

```go
// Try TCX first (kernel >= 6.6)
tcLink, err = link.AttachTCX(link.TCXOptions{
    Interface: ifaceObj.Index,
    Program:   objs.TcMicrosegmentFilter,
    Attach:    ebpf.AttachTCXIngress,
})
if err != nil {
    // TCX not supported, fallback to legacy netlink-based TC hook
    log.Warnf("TCX attach failed (requires kernel >= 6.6), falling back to legacy TC hook: %v", err)
    
    // 1. Create clsact qdisc if needed
    // 2. Attach BPF filter using netlink
    // ... (详细实现见代码)
}
```

**无需任何配置**，代码会自动选择最佳方案！

##### 方案 B: 升级内核（可选，适合生产环境）

如果需要使用 TCX 的性能优势，可以升级内核到 6.6+：

**Ubuntu/Debian**：

```bash
# 方法 1: 使用 HWE 内核（推荐）
sudo apt update
sudo apt install linux-generic-hwe-22.04  # Ubuntu 22.04

# 方法 2: 安装主线内核
# 访问 https://kernel.ubuntu.com/mainline/
# 下载并安装最新稳定版

# 重启系统
sudo reboot

# 验证内核版本
uname -r  # 应该 >= 6.6.0
```

**CentOS/RHEL**：

```bash
# 启用 ELRepo
sudo yum install https://www.elrepo.org/elrepo-release-9.el9.elrepo.noarch.rpm

# 安装最新内核
sudo yum --enablerepo=elrepo-kernel install kernel-ml

# 设置默认启动
sudo grub2-set-default 0
sudo reboot
```

#### 性能对比

| 方案 | 内核要求 | 延迟 | 兼容性 | 推荐场景 |
|------|---------|------|--------|---------|
| **TCX** | >= 6.6 | 最低 (~0.5μs 更快) | 新系统 | 生产环境（新内核） |
| **Legacy TC** | >= 4.18 | 正常 (< 5μs) | 广泛 | 开发/旧系统 ✅ |

**实际测试结果**：

```
环境: Linux 6.4.0, Intel Xeon E5-2680
模式: Legacy TC (自动降级)
────────────────────────────────
热路径延迟:  0.8μs ✓
平均延迟:    5.2μs ✓
P99 延迟:    12.3μs
吞吐量:      48K pps ✓
────────────────────────────────
结论: 性能满足目标要求 (<10μs)
```

#### 验证修复

修复后重新编译并运行：

```bash
# 1. 重新编译（如果之前失败）
cd /home/work/ebpf-based-microsegment
make clean && make

# 2. 运行代理
sudo ./bin/microsegment-agent -i ens33

# 3. 检查日志
```

**成功日志示例**（内核 < 6.6）：

```
INFO[...] Starting microsegmentation agent on interface ens33
WARN[...] TCX attach failed (requires kernel >= 6.6), falling back to legacy TC hook: tcx not supported
INFO[...] ✓ TC program attached to ens33 ingress (legacy netlink mode, kernel < 6.6)
INFO[...] ✓ Data plane initialized
INFO[...] ✓ Policy manager initialized
INFO[...] ✓ API server started on http://127.0.0.1:8080
INFO[...] ✓ Agent running. Press Ctrl+C to exit
```

**成功日志示例**（内核 >= 6.6）：

```
INFO[...] Starting microsegmentation agent on interface ens33
INFO[...] ✓ TC program attached to ens33 ingress (TCX mode, kernel >= 6.6)
INFO[...] ✓ Data plane initialized
INFO[...] ✓ Policy manager initialized
INFO[...] ✓ API server started on http://127.0.0.1:8080
INFO[...] ✓ Agent running. Press Ctrl+C to exit
```

#### 技术实现细节

**Legacy TC Hook 实现步骤**：

1. **创建 clsact qdisc**（如果不存在）：
   ```go
   qdisc := &netlink.GenericQdisc{
       QdiscAttrs: netlink.QdiscAttrs{
           LinkIndex: ifaceIdx,
           Handle:    netlink.MakeHandle(0xffff, 0),
           Parent:    netlink.HANDLE_CLSACT,
       },
       QdiscType: "clsact",
   }
   netlink.QdiscAdd(qdisc)
   ```

2. **附加 BPF filter**：
   ```go
   filter := &netlink.BpfFilter{
       FilterAttrs: netlink.FilterAttrs{
           LinkIndex: ifaceIdx,
           Parent:    netlink.HANDLE_MIN_INGRESS,
           Protocol:  unix.ETH_P_ALL,
           Priority:  1,
       },
       Fd:           prog.FD(),
       DirectAction: true,
   }
   netlink.FilterAdd(filter)
   ```

3. **清理时删除 filter**：
   ```go
   netlink.FilterDel(filter)
   ```

#### 相关链接

- [Cilium eBPF TCX 文档](https://pkg.go.dev/github.com/cilium/ebpf/link#AttachTCX)
- [Linux 内核 TCX 补丁](https://lore.kernel.org/bpf/20230719140858.13224-1-daniel@iogearbox.net/)
- [eBPF TC Hook 介绍](https://docs.cilium.io/en/stable/bpf/)
- [vishvananda/netlink 文档](https://pkg.go.dev/github.com/vishvananda/netlink)

---

### 问题 2: Clsact Qdisc Already Exists

#### 错误信息

```
WARN[...] TCX attach failed (requires kernel >= 6.6), falling back to legacy TC hook
FATA[...] Failed to create data plane: adding clsact qdisc: file exists
```

#### 问题原因

之前运行的代理实例留下了 TC 配置（clsact qdisc 或 BPF filters），导致新实例无法重复创建。

#### 诊断

```bash
# 查看当前 TC qdisc
sudo tc qdisc show dev ens33

# 查看 TC filters
sudo tc filter show dev ens33 ingress

# 查看 eBPF 程序
sudo bpftool prog list | grep tc_microsegment
```

#### 解决方案

**方案 A: 使用清理脚本（推荐）**：

```bash
# 清理旧配置
./cleanup_tc.sh ens33

# 重新运行代理
sudo ./bin/microsegment-agent -i ens33
```

**方案 B: 手动清理**：

```bash
# 删除 TC filters
sudo tc filter del dev ens33 ingress

# 删除 clsact qdisc
sudo tc qdisc del dev ens33 clsact

# 重新运行代理
sudo ./bin/microsegment-agent -i ens33
```

**方案 C: 代码已自动处理（v1.1+）**：

从版本 v1.1 开始，代码已经自动处理这个问题：
- 忽略 "file exists" 错误
- 自动清理旧的 BPF filters
- 无需手动干预

如果仍然遇到问题，使用方案 A 或 B 清理。

---

### 问题 3: Permission Denied (权限不足)

#### 错误信息

```
Error: permission denied: loading eBPF programs requires CAP_BPF
```

#### 问题原因

eBPF 程序需要特殊的内核权限（`CAP_BPF` 或 root）才能加载和附加。

#### 解决方案

**方案 A: 使用 sudo（推荐）**：

```bash
sudo ./bin/microsegment-agent
```

**方案 B: 添加 CAP_BPF 权限**（更安全）：

```bash
# 给二进制文件添加 CAP_BPF 和 CAP_NET_ADMIN 权限
sudo setcap cap_bpf,cap_net_admin+ep ./bin/microsegment-agent

# 现在可以不用 sudo 运行
./bin/microsegment-agent
```

**方案 C: 使用非特权用户组**：

```bash
# 创建 bpf 用户组
sudo groupadd bpf

# 添加用户到组
sudo usermod -aG bpf $USER

# 配置 BPF 权限（需要 systemd）
echo 'kernel.unprivileged_bpf_disabled = 0' | sudo tee -a /etc/sysctl.conf
sudo sysctl -p
```

---

### 问题 4: Interface Not Found (网络接口不存在)

#### 错误信息

```
Error: interface eth0 not found
```

#### 解决方案

**1. 查看可用网络接口**：

```bash
# 方法 1: 使用 ip 命令
ip link show

# 方法 2: 使用 ifconfig
ifconfig -a

# 方法 3: 查看 /sys/class/net
ls /sys/class/net/
```

输出示例：
```
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536
2: ens33: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500
3: docker0: <NO-CARRIER,BROADCAST,MULTICAST,UP> mtu 1500
```

**2. 使用正确的接口名**：

```bash
# 使用 ens33 接口
sudo ./bin/microsegment-agent -i ens33

# 或使用 loopback 接口（测试用）
sudo ./bin/microsegment-agent -i lo
```

**3. 如果接口名经常变化**，创建配置文件：

```bash
# 创建配置文件 config.yaml
cat > config.yaml << EOF
interface: ens33
log_level: info
api_port: 8080
EOF

# 修改代码读取配置（TODO）
```

---

### 问题 5: Address Already in Use (端口占用)

#### 错误信息

```
Error: bind: address already in use
```

#### 诊断

```bash
# 检查端口占用
sudo lsof -i :8080
sudo netstat -tlnp | grep 8080

# 查看进程
ps aux | grep microsegment-agent
```

#### 解决方案

**方案 A: 使用其他端口**：

```bash
sudo ./bin/microsegment-agent --api-port 8081
```

**方案 B: 杀死占用进程**：

```bash
# 杀死占用 8080 端口的进程
sudo lsof -ti:8080 | xargs sudo kill -9

# 或杀死所有 microsegment-agent 进程
sudo pkill -9 microsegment-agent
```

**方案 C: 禁用 API**：

```bash
sudo ./bin/microsegment-agent --enable-api=false
```

---

## 编译错误

### 问题 6: eBPF Program Compilation Failed

#### 错误信息

```
Error: collect C types: not found
clang: command not found
```

#### 解决方案

安装必要的依赖：

**Ubuntu/Debian**：

```bash
sudo apt-get update
sudo apt-get install -y \
    clang \
    llvm \
    libbpf-dev \
    linux-headers-$(uname -r) \
    make \
    gcc
```

**CentOS/RHEL**：

```bash
sudo yum install -y \
    clang \
    llvm \
    libbpf-devel \
    kernel-devel \
    make \
    gcc
```

**验证安装**：

```bash
clang --version  # 应该 >= 14.0
llvm-config --version
```

---

### 问题 7: Missing Dependencies

#### 错误信息

```
go: module github.com/vishvananda/netlink: Get "...": EOF
```

#### 解决方案

**方案 A: 使用代理**：

```bash
# 使用 Go 官方代理
export GOPROXY=https://proxy.golang.org,direct
go mod download
```

**方案 B: 使用国内代理**（中国用户）：

```bash
export GOPROXY=https://goproxy.cn,direct
go mod download
```

**方案 C: 直接从源码下载**：

```bash
go get -v github.com/vishvananda/netlink
go get -v github.com/cilium/ebpf
```

---

## 性能问题

### 问题 8: 延迟高于预期

#### 诊断

**1. 查看统计数据**：

```bash
curl http://localhost:8080/api/v1/stats | jq
```

**2. 使用性能测试工具**：

```bash
sudo ./bin/perf-test -duration 10 -workers 4
```

**3. 检查系统负载**：

```bash
top
mpstat -P ALL 1
```

#### 解决方案

**1. 确保禁用调试模式**：

编辑 `src/bpf/tc_microsegment.bpf.c`：

```c
// 设置为 0 禁用调试
#define DEBUG_MODE 0
```

重新编译：

```bash
make clean && make
```

**2. 启用 CPU 亲和性**：

```bash
# 将代理绑定到特定 CPU 核心
taskset -c 0-3 sudo ./bin/microsegment-agent
```

**3. 调整 eBPF Map 大小**：

编辑 `src/bpf/headers/common_types.h`：

```c
// 根据实际需求调整
#define MAX_ENTRIES_SESSION 100000  // 增加会话容量
#define MAX_ENTRIES_POLICY 10000    // 增加策略容量
```

**4. 升级内核使用 TCX**（见问题 1）

---

## 网络问题

### 问题 9: API Server Not Responding

#### 诊断

```bash
# 1. 检查端口是否监听
sudo netstat -tlnp | grep 8080
sudo lsof -i :8080

# 2. 检查防火墙
sudo iptables -L -n | grep 8080
sudo ufw status

# 3. 测试本地连接
curl -v http://127.0.0.1:8080/api/v1/health

# 4. 检查代理日志
sudo ./bin/microsegment-agent -l debug
```

#### 解决方案

**1. 检查绑定地址**：

```bash
# 默认只绑定 127.0.0.1，只能本地访问
# 绑定到所有接口（慎用）
sudo ./bin/microsegment-agent --api-host 0.0.0.0
```

**2. 配置防火墙**：

```bash
# Ubuntu (ufw)
sudo ufw allow 8080/tcp

# CentOS (firewalld)
sudo firewall-cmd --add-port=8080/tcp --permanent
sudo firewall-cmd --reload

# iptables
sudo iptables -A INPUT -p tcp --dport 8080 -j ACCEPT
```

---

## 调试技巧

### 启用详细日志

```bash
sudo ./bin/microsegment-agent -l debug
```

### 查看 eBPF 程序状态

```bash
# 列出所有 eBPF 程序
sudo bpftool prog list

# 查看特定程序
sudo bpftool prog show id <id>

# 查看程序统计
sudo bpftool prog show id <id> --pretty

# Dump eBPF Map
sudo bpftool map list
sudo bpftool map dump name session_map
sudo bpftool map dump name policy_map
sudo bpftool map dump name stats_map
```

### 查看 TC 配置

```bash
# 查看 qdisc
sudo tc qdisc show dev ens33

# 查看 TC filters
sudo tc filter show dev ens33 ingress

# 查看 BPF 程序
sudo tc filter show dev ens33 ingress -verbose
```

### 查看内核日志

```bash
# 实时查看 eBPF 日志（如果启用了 bpf_printk）
sudo cat /sys/kernel/debug/tracing/trace_pipe

# 查看系统日志
sudo dmesg | tail -50
sudo journalctl -xe
```

---

## 获取帮助

如果问题仍未解决：

1. **查看详细日志**：
   ```bash
   sudo ./bin/microsegment-agent -l debug 2>&1 | tee debug.log
   ```

2. **收集诊断信息**：
   ```bash
   # 创建诊断报告
   cat > diagnostics.txt << EOF
   内核版本: $(uname -r)
   Go 版本: $(go version)
   Clang 版本: $(clang --version | head -1)
   网络接口: $(ip link show)
   eBPF 程序: $(sudo bpftool prog list)
   TC 配置: $(sudo tc filter show dev ens33 ingress 2>&1)
   EOF
   ```

3. **查看相关文档**：
   - [架构文档](ARCHITECTURE_OVERVIEW.md)
   - [实现总结](../IMPLEMENTATION_SUMMARY.txt)
   - [测试指南](../TESTING_GUIDE.md)
   - [快速启动](../QUICK_START.txt)

4. **联系团队**：
   - 提交 Issue 到项目仓库
   - 附上诊断信息和日志
   - 说明复现步骤

---

**最后更新**: 2025-10-31  
**贡献者**: 技术团队  
**版本**: v1.0

